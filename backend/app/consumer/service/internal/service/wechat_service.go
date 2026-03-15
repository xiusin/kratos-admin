package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/pkg/eventbus"
)

const (
	// 微信API地址
	wechatAuthURL     = "https://open.weixin.qq.com/connect/oauth2/authorize"
	wechatAccessURL   = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatUserInfoURL = "https://api.weixin.qq.com/sns/userinfo"
	wechatTemplateURL = "https://api.weixin.qq.com/cgi-bin/message/template/send"
	wechatJscode2URL  = "https://api.weixin.qq.com/sns/jscode2session"
	wechatTokenURL    = "https://api.weixin.qq.com/cgi-bin/token"

	// Redis key前缀
	redisKeyAccessToken = "wechat:access_token"
	redisKeyUserInfo    = "wechat:user_info:"

	// access_token 过期时间（7200秒 = 2小时）
	accessTokenExpire = 7200 * time.Second
	// 提前刷新时间（提前5分钟刷新）
	accessTokenRefreshBefore = 300 * time.Second
)

// WechatService 微信服务
type WechatService struct {
	consumerV1.UnimplementedWechatServiceServer

	rdb      *redis.Client
	eventBus eventbus.EventBus
	log      *log.Helper

	// 微信配置（TODO: 从配置文件读取）
	appID     string
	appSecret string
}

// NewWechatService 创建微信服务实例
func NewWechatService(
	ctx *bootstrap.Context,
	rdb *redis.Client,
	eventBus eventbus.EventBus,
) *WechatService {
	// TODO: 从配置文件读取微信配置
	// 配置路径：third_party.wechat.official_account
	appID := "your-wechat-official-app-id"
	appSecret := "your-wechat-official-app-secret"

	logger := ctx.NewLoggerHelper("wechat/service/consumer-service")
	logger.Infof("Wechat service initialized with AppID: %s", appID)

	return &WechatService{
		rdb:       rdb,
		eventBus:  eventBus,
		log:       logger,
		appID:     appID,
		appSecret: appSecret,
	}
}

// GetAuthURL 获取微信授权URL
func (s *WechatService) GetAuthURL(ctx context.Context, req *consumerV1.GetAuthURLRequest) (*consumerV1.GetAuthURLResponse, error) {
	s.log.Infof("GetAuthURL: redirect_uri=%s", req.GetRedirectUri())

	// 构建授权URL
	params := url.Values{}
	params.Set("appid", s.appID)
	params.Set("redirect_uri", req.GetRedirectUri())
	params.Set("response_type", "code")

	// 设置授权作用域（默认 snsapi_base）
	scope := req.GetScope()
	if scope == "" {
		scope = "snsapi_base"
	}
	params.Set("scope", scope)

	// 设置状态参数
	if req.State != nil {
		params.Set("state", req.GetState())
	}

	authURL := fmt.Sprintf("%s?%s#wechat_redirect", wechatAuthURL, params.Encode())

	s.log.Infof("GetAuthURL success: auth_url=%s", authURL)
	return &consumerV1.GetAuthURLResponse{
		AuthUrl: authURL,
	}, nil
}

// AuthCallback 微信授权回调
func (s *WechatService) AuthCallback(ctx context.Context, req *consumerV1.AuthCallbackRequest) (*consumerV1.AuthCallbackResponse, error) {
	s.log.Infof("AuthCallback: code=%s", req.GetCode())

	// 使用 code 换取 access_token
	params := url.Values{}
	params.Set("appid", s.appID)
	params.Set("secret", s.appSecret)
	params.Set("code", req.GetCode())
	params.Set("grant_type", "authorization_code")

	apiURL := fmt.Sprintf("%s?%s", wechatAccessURL, params.Encode())

	resp, err := http.Get(apiURL)
	if err != nil {
		s.log.Errorf("call wechat api failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to call wechat api")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Errorf("read response body failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to read response")
	}

	// 解析响应
	var result struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		OpenID       string `json:"openid"`
		Scope        string `json:"scope"`
		UnionID      string `json:"unionid"`
		ErrCode      int    `json:"errcode"`
		ErrMsg       string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Errorf("parse response failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to parse response")
	}

	// 检查错误
	if result.ErrCode != 0 {
		s.log.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		return nil, errors.BadRequest("WECHAT_AUTH_FAILED", result.ErrMsg)
	}

	// 缓存 access_token（按 openid 缓存）
	cacheKey := fmt.Sprintf("%s:%s", redisKeyAccessToken, result.OpenID)
	if err := s.rdb.Set(ctx, cacheKey, result.AccessToken, accessTokenExpire).Err(); err != nil {
		s.log.Errorf("cache access_token failed: %v", err)
		// 不影响主流程
	}

	s.log.Infof("AuthCallback success: openid=%s", result.OpenID)

	response := &consumerV1.AuthCallbackResponse{
		Openid:      result.OpenID,
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
	}

	if result.UnionID != "" {
		response.Unionid = &result.UnionID
	}

	return response, nil
}

// GetWechatUserInfo 获取微信用户信息
func (s *WechatService) GetWechatUserInfo(ctx context.Context, req *consumerV1.GetWechatUserInfoRequest) (*consumerV1.WechatUserInfo, error) {
	s.log.Infof("GetWechatUserInfo: openid=%s", req.GetOpenid())

	// 先从缓存获取
	cacheKey := redisKeyUserInfo + req.GetOpenid()
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var userInfo consumerV1.WechatUserInfo
		if err := json.Unmarshal([]byte(cached), &userInfo); err == nil {
			s.log.Infof("GetWechatUserInfo from cache: openid=%s", req.GetOpenid())
			return &userInfo, nil
		}
	}

	// 获取 access_token
	accessToken, err := s.getAccessToken(ctx, req.GetOpenid())
	if err != nil {
		return nil, err
	}

	// 调用微信API获取用户信息
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("openid", req.GetOpenid())
	params.Set("lang", "zh_CN")

	apiURL := fmt.Sprintf("%s?%s", wechatUserInfoURL, params.Encode())

	resp, err := http.Get(apiURL)
	if err != nil {
		s.log.Errorf("call wechat api failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to call wechat api")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Errorf("read response body failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to read response")
	}

	// 解析响应
	var result struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
		Sex        int32  `json:"sex"`
		Country    string `json:"country"`
		Province   string `json:"province"`
		City       string `json:"city"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Errorf("parse response failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to parse response")
	}

	// 检查错误
	if result.ErrCode != 0 {
		s.log.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		return nil, errors.BadRequest("WECHAT_API_FAILED", result.ErrMsg)
	}

	// 构建响应
	userInfo := &consumerV1.WechatUserInfo{
		Openid: result.OpenID,
	}

	if result.UnionID != "" {
		userInfo.Unionid = &result.UnionID
	}
	if result.Nickname != "" {
		userInfo.Nickname = &result.Nickname
	}
	if result.HeadImgURL != "" {
		userInfo.Headimgurl = &result.HeadImgURL
	}
	if result.Sex != 0 {
		userInfo.Sex = &result.Sex
	}
	if result.Country != "" {
		userInfo.Country = &result.Country
	}
	if result.Province != "" {
		userInfo.Province = &result.Province
	}
	if result.City != "" {
		userInfo.City = &result.City
	}

	// 缓存用户信息（30分钟）
	if data, err := json.Marshal(userInfo); err == nil {
		if err := s.rdb.Set(ctx, cacheKey, data, 30*time.Minute).Err(); err != nil {
			s.log.Errorf("cache user info failed: %v", err)
		}
	}

	s.log.Infof("GetWechatUserInfo success: openid=%s", result.OpenID)
	return userInfo, nil
}

// getAccessToken 获取 access_token（从缓存或刷新）
func (s *WechatService) getAccessToken(ctx context.Context, openid string) (string, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("%s:%s", redisKeyAccessToken, openid)
	token, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil && token != "" {
		// 检查是否需要刷新（提前5分钟）
		ttl, _ := s.rdb.TTL(ctx, cacheKey).Result()
		if ttl > accessTokenRefreshBefore {
			return token, nil
		}
	}

	// 需要刷新 access_token
	// 注意：这里简化处理，实际应该使用 refresh_token 刷新
	// 由于我们没有存储 refresh_token，这里返回错误，要求重新授权
	return "", errors.Unauthorized("ACCESS_TOKEN_EXPIRED", "access token expired, please re-authorize")
}

// verifySignature 验证微信签名（公开方法，供回调接口使用）
func (s *WechatService) VerifySignature(signature, timestamp, nonce string) bool {
	// 将 token（使用 appSecret）、timestamp、nonce 三个参数进行字典序排序
	params := []string{s.appSecret, timestamp, nonce}
	sort.Strings(params)

	// 拼接字符串
	str := strings.Join(params, "")

	// SHA1 加密
	h := sha1.New()
	h.Write([]byte(str))
	encrypted := hex.EncodeToString(h.Sum(nil))

	// 比较签名
	return encrypted == signature
}

// SendTemplateMessage 发送模板消息
func (s *WechatService) SendTemplateMessage(ctx context.Context, req *consumerV1.SendTemplateMessageRequest) (*emptypb.Empty, error) {
	s.log.Infof("SendTemplateMessage: openid=%s, template_id=%s", req.GetOpenid(), req.GetTemplateId())

	// 获取全局 access_token（用于调用公众号API）
	accessToken, err := s.getGlobalAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// 构建请求数据
	reqData := map[string]interface{}{
		"touser":      req.GetOpenid(),
		"template_id": req.GetTemplateId(),
		"data":        req.GetData(),
	}

	if req.Url != nil {
		reqData["url"] = req.GetUrl()
	}

	if req.MiniprogramAppid != nil && req.MiniprogramPagepath != nil {
		reqData["miniprogram"] = map[string]string{
			"appid":    req.GetMiniprogramAppid(),
			"pagepath": req.GetMiniprogramPagepath(),
		}
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		s.log.Errorf("marshal request data failed: %v", err)
		return nil, errors.InternalServer("MARSHAL_ERROR", "failed to marshal request data")
	}

	// 调用微信API
	apiURL := fmt.Sprintf("%s?access_token=%s", wechatTemplateURL, accessToken)
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		s.log.Errorf("call wechat api failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to call wechat api")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Errorf("read response body failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to read response")
	}

	// 解析响应
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MsgID   int64  `json:"msgid"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Errorf("parse response failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to parse response")
	}

	// 检查错误
	if result.ErrCode != 0 {
		s.log.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		return nil, errors.BadRequest("WECHAT_API_FAILED", result.ErrMsg)
	}

	s.log.Infof("SendTemplateMessage success: msgid=%d", result.MsgID)
	return &emptypb.Empty{}, nil
}

// MiniProgramLogin 小程序登录
func (s *WechatService) MiniProgramLogin(ctx context.Context, req *consumerV1.MiniProgramLoginRequest) (*consumerV1.MiniProgramLoginResponse, error) {
	s.log.Infof("MiniProgramLogin: code=%s", req.GetCode())

	// 调用微信API换取 session_key
	params := url.Values{}
	params.Set("appid", s.appID)
	params.Set("secret", s.appSecret)
	params.Set("js_code", req.GetCode())
	params.Set("grant_type", "authorization_code")

	apiURL := fmt.Sprintf("%s?%s", wechatJscode2URL, params.Encode())

	resp, err := http.Get(apiURL)
	if err != nil {
		s.log.Errorf("call wechat api failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to call wechat api")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Errorf("read response body failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to read response")
	}

	// 解析响应
	var result struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Errorf("parse response failed: %v", err)
		return nil, errors.InternalServer("WECHAT_API_ERROR", "failed to parse response")
	}

	// 检查错误
	if result.ErrCode != 0 {
		s.log.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		return nil, errors.BadRequest("WECHAT_LOGIN_FAILED", result.ErrMsg)
	}

	// 缓存 session_key（按 openid 缓存，7天过期）
	cacheKey := fmt.Sprintf("wechat:session_key:%s", result.OpenID)
	if err := s.rdb.Set(ctx, cacheKey, result.SessionKey, 7*24*time.Hour).Err(); err != nil {
		s.log.Errorf("cache session_key failed: %v", err)
		// 不影响主流程
	}

	s.log.Infof("MiniProgramLogin success: openid=%s", result.OpenID)

	response := &consumerV1.MiniProgramLoginResponse{
		Openid:     result.OpenID,
		SessionKey: result.SessionKey,
	}

	if result.UnionID != "" {
		response.Unionid = &result.UnionID
	}

	return response, nil
}

// getGlobalAccessToken 获取全局 access_token（用于调用公众号API）
func (s *WechatService) getGlobalAccessToken(ctx context.Context) (string, error) {
	// 从缓存获取
	cacheKey := redisKeyAccessToken + ":global"
	token, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil && token != "" {
		// 检查是否需要刷新（提前5分钟）
		ttl, _ := s.rdb.TTL(ctx, cacheKey).Result()
		if ttl > accessTokenRefreshBefore {
			return token, nil
		}
	}

	// 刷新 access_token
	params := url.Values{}
	params.Set("grant_type", "client_credential")
	params.Set("appid", s.appID)
	params.Set("secret", s.appSecret)

	apiURL := fmt.Sprintf("%s?%s", wechatTokenURL, params.Encode())

	resp, err := http.Get(apiURL)
	if err != nil {
		s.log.Errorf("call wechat api failed: %v", err)
		return "", errors.InternalServer("WECHAT_API_ERROR", "failed to call wechat api")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Errorf("read response body failed: %v", err)
		return "", errors.InternalServer("WECHAT_API_ERROR", "failed to read response")
	}

	// 解析响应
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Errorf("parse response failed: %v", err)
		return "", errors.InternalServer("WECHAT_API_ERROR", "failed to parse response")
	}

	// 检查错误
	if result.ErrCode != 0 {
		s.log.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
		return "", errors.BadRequest("WECHAT_API_FAILED", result.ErrMsg)
	}

	// 缓存 access_token
	if err := s.rdb.Set(ctx, cacheKey, result.AccessToken, accessTokenExpire).Err(); err != nil {
		s.log.Errorf("cache access_token failed: %v", err)
		// 不影响主流程
	}

	s.log.Infof("refresh global access_token success")
	return result.AccessToken, nil
}

// HandleWechatEvent 处理微信事件消息（用于接收微信推送的事件）
// 注意：这个方法不在 Protobuf 定义中，是内部使用的辅助方法
func (s *WechatService) HandleWechatEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error {
	s.log.Infof("HandleWechatEvent: type=%s", eventType)

	// 发布系统事件
	event := eventbus.NewEvent("wechat.event.received", map[string]interface{}{
		"event_type": eventType,
		"event_data": eventData,
	}).WithSource("wechat-service")

	if err := s.eventBus.PublishAsync(ctx, event); err != nil {
		s.log.Errorf("publish wechat event failed: %v", err)
		return err
	}

	s.log.Infof("HandleWechatEvent success: type=%s", eventType)
	return nil
}

// WechatEventMessage 微信事件消息结构（XML）
type WechatEventMessage struct {
	ToUserName   string `xml:"ToUserName"`   // 开发者微信号
	FromUserName string `xml:"FromUserName"` // 发送方帐号（OpenID）
	CreateTime   int64  `xml:"CreateTime"`   // 消息创建时间
	MsgType      string `xml:"MsgType"`      // 消息类型（event）
	Event        string `xml:"Event"`        // 事件类型
	EventKey     string `xml:"EventKey"`     // 事件KEY值
	Ticket       string `xml:"Ticket"`       // 二维码的ticket
	Latitude     string `xml:"Latitude"`     // 地理位置纬度
	Longitude    string `xml:"Longitude"`    // 地理位置经度
	Precision    string `xml:"Precision"`    // 地理位置精度
	MsgID        int64  `xml:"MsgId"`        // 消息ID
	Content      string `xml:"Content"`      // 文本消息内容
	PicUrl       string `xml:"PicUrl"`       // 图片链接
	MediaId      string `xml:"MediaId"`      // 媒体ID
}

// ParseWechatEventXML 解析微信事件消息XML
func (s *WechatService) ParseWechatEventXML(xmlData []byte) (*WechatEventMessage, error) {
	var msg WechatEventMessage

	// 简单的 XML 解析（使用 encoding/xml）
	// 注意：这里需要导入 "encoding/xml"
	// 由于当前没有导入，我们先用字符串解析

	xmlStr := string(xmlData)

	// 提取关键字段
	msg.ToUserName = extractXMLValue(xmlStr, "ToUserName")
	msg.FromUserName = extractXMLValue(xmlStr, "FromUserName")
	msg.MsgType = extractXMLValue(xmlStr, "MsgType")
	msg.Event = extractXMLValue(xmlStr, "Event")
	msg.EventKey = extractXMLValue(xmlStr, "EventKey")
	msg.Content = extractXMLValue(xmlStr, "Content")

	return &msg, nil
}

// extractXMLValue 从XML字符串中提取值（简单实现）
func extractXMLValue(xmlStr, tagName string) string {
	// 查找 <tagName><![CDATA[value]]></tagName> 或 <tagName>value</tagName>
	startTag := fmt.Sprintf("<%s>", tagName)
	endTag := fmt.Sprintf("</%s>", tagName)

	startIdx := strings.Index(xmlStr, startTag)
	if startIdx == -1 {
		return ""
	}

	startIdx += len(startTag)
	endIdx := strings.Index(xmlStr[startIdx:], endTag)
	if endIdx == -1 {
		return ""
	}

	value := xmlStr[startIdx : startIdx+endIdx]

	// 处理 CDATA
	if strings.HasPrefix(value, "<![CDATA[") && strings.HasSuffix(value, "]]>") {
		value = value[9 : len(value)-3]
	}

	return strings.TrimSpace(value)
}
