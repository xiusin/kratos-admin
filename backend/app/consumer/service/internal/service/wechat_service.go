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

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/pkg/eventbus"
)

// WechatService 微信服务
type WechatService struct {
	consumerV1.UnimplementedWechatServiceServer

	redis     *redis.Client
	eventbus  eventbus.EventBus
	log       *log.Helper
	appID     string
	appSecret string
}

// NewWechatService 创建微信服务
func NewWechatService(
	ctx *bootstrap.Context,
	redis *redis.Client,
	eventbus eventbus.EventBus,
) *WechatService {
	// 从配置中获取微信配置
	appID := ctx.Config().GetString("third_party.wechat.official_account.app_id")
	appSecret := ctx.Config().GetString("third_party.wechat.official_account.app_secret")

	return &WechatService{
		redis:     redis,
		eventbus:  eventbus,
		log:       ctx.NewLoggerHelper("consumer/service/wechat-service"),
		appID:     appID,
		appSecret: appSecret,
	}
}

// GetAuthURL 获取微信授权URL
func (s *WechatService) GetAuthURL(ctx context.Context, req *consumerV1.GetAuthURLRequest) (*consumerV1.GetAuthURLResponse, error) {
	s.log.WithContext(ctx).Infof("GetAuthURL: redirect_uri=%s, state=%s, scope=%s",
		req.RedirectUri, req.GetState(), req.GetScope())

	// 默认scope为snsapi_userinfo（获取用户信息）
	scope := "snsapi_userinfo"
	if req.Scope != nil && *req.Scope != "" {
		scope = *req.Scope
	}

	// 默认state为空字符串
	state := ""
	if req.State != nil {
		state = *req.State
	}

	// 构建微信授权URL
	// https://open.weixin.qq.com/connect/oauth2/authorize?appid=APPID&redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
	authURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		s.appID,
		url.QueryEscape(req.RedirectUri),
		scope,
		url.QueryEscape(state),
	)

	return &consumerV1.GetAuthURLResponse{
		AuthUrl: authURL,
	}, nil
}

// AuthCallback 微信授权回调
func (s *WechatService) AuthCallback(ctx context.Context, req *consumerV1.AuthCallbackRequest) (*consumerV1.AuthCallbackResponse, error) {
	s.log.WithContext(ctx).Infof("AuthCallback: code=%s, state=%s", req.Code, req.GetState())

	// 1. 通过code换取access_token
	accessTokenResp, err := s.getAccessTokenByCode(ctx, req.Code)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get access token failed: %v", err)
		return nil, consumerV1.ErrorWechatAuthFailed("微信授权失败: %v", err)
	}

	// 2. 缓存access_token
	if err := s.cacheAccessToken(ctx, accessTokenResp.OpenID, accessTokenResp.AccessToken, accessTokenResp.ExpiresIn); err != nil {
		s.log.WithContext(ctx).Warnf("cache access token failed: %v", err)
	}

	// 3. 返回授权结果
	return &consumerV1.AuthCallbackResponse{
		Openid:      accessTokenResp.OpenID,
		Unionid:     strPtrToPtr(accessTokenResp.UnionID),
		AccessToken: accessTokenResp.AccessToken,
		ExpiresIn:   int64(accessTokenResp.ExpiresIn),
	}, nil
}

// GetWechatUserInfo 获取微信用户信息
func (s *WechatService) GetWechatUserInfo(ctx context.Context, req *consumerV1.GetWechatUserInfoRequest) (*consumerV1.WechatUserInfo, error) {
	s.log.WithContext(ctx).Infof("GetWechatUserInfo: openid=%s", req.Openid)

	// 1. 从缓存获取access_token
	accessToken, err := s.getAccessTokenFromCache(ctx, req.Openid)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get access token from cache failed: %v", err)
		return nil, consumerV1.ErrorWechatAccessTokenExpired("access_token已过期或不存在")
	}

	// 2. 调用微信API获取用户信息
	userInfo, err := s.getUserInfoFromWechat(ctx, accessToken, req.Openid)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get user info from wechat failed: %v", err)
		return nil, consumerV1.ErrorWechatAPIFailed("获取微信用户信息失败: %v", err)
	}

	return userInfo, nil
}

// getAccessTokenByCode 通过code换取access_token
func (s *WechatService) getAccessTokenByCode(ctx context.Context, code string) (*WechatAccessTokenResponse, error) {
	// 构建请求URL
	// https://api.weixin.qq.com/sns/oauth2/access_token?appid=APPID&secret=SECRET&code=CODE&grant_type=authorization_code
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		s.appID,
		s.appSecret,
		code,
	)

	// 发送HTTP请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 解析响应
	var result WechatAccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// getUserInfoFromWechat 从微信获取用户信息
func (s *WechatService) getUserInfoFromWechat(ctx context.Context, accessToken string, openID string) (*consumerV1.WechatUserInfo, error) {
	// 构建请求URL
	// https://api.weixin.qq.com/sns/userinfo?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		accessToken,
		openID,
	)

	// 发送HTTP请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 解析响应
	var result WechatUserInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	// 转换为Protobuf格式
	userInfo := &consumerV1.WechatUserInfo{
		Openid:     result.OpenID,
		Unionid:    strPtrToPtr(result.UnionID),
		Nickname:   strPtrToPtr(result.Nickname),
		Headimgurl: strPtrToPtr(result.HeadImgURL),
		Sex:        int32PtrToPtr(result.Sex),
		Country:    strPtrToPtr(result.Country),
		Province:   strPtrToPtr(result.Province),
		City:       strPtrToPtr(result.City),
	}

	return userInfo, nil
}

// cacheAccessToken 缓存access_token
func (s *WechatService) cacheAccessToken(ctx context.Context, openID string, accessToken string, expiresIn int) error {
	key := fmt.Sprintf("wechat:access_token:%s", openID)

	// 缓存时间设置为过期时间-60秒（提前刷新）
	ttl := time.Duration(expiresIn-60) * time.Second
	if ttl <= 0 {
		ttl = 7140 * time.Second // 默认7140秒（7200-60）
	}

	return s.redis.Set(ctx, key, accessToken, ttl).Err()
}

// getAccessTokenFromCache 从缓存获取access_token
func (s *WechatService) getAccessTokenFromCache(ctx context.Context, openID string) (string, error) {
	key := fmt.Sprintf("wechat:access_token:%s", openID)

	accessToken, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("access_token not found")
	}
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// verifySignature 验证微信签名
func (s *WechatService) verifySignature(signature string, timestamp string, nonce string, token string) bool {
	// 1. 将token、timestamp、nonce三个参数进行字典序排序
	params := []string{token, timestamp, nonce}
	sort.Strings(params)

	// 2. 将三个参数字符串拼接成一个字符串进行sha1加密
	str := strings.Join(params, "")
	h := sha1.New()
	h.Write([]byte(str))
	encrypted := hex.EncodeToString(h.Sum(nil))

	// 3. 将加密后的字符串与signature对比
	return encrypted == signature
}

// WechatAccessTokenResponse 微信access_token响应
type WechatAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	UnionID      string `json:"unionid,omitempty"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

// WechatUserInfoResponse 微信用户信息响应
type WechatUserInfoResponse struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid,omitempty"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	Sex        int32  `json:"sex"`
	Country    string `json:"country"`
	Province   string `json:"province"`
	City       string `json:"city"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// strPtrToPtr 字符串转指针
func strPtrToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int32PtrToPtr int32转指针
func int32PtrToPtr(i int32) *int32 {
	return &i
}

// SendTemplateMessage 发送模板消息
func (s *WechatService) SendTemplateMessage(ctx context.Context, req *consumerV1.SendTemplateMessageRequest) (*emptypb.Empty, error) {
	s.log.WithContext(ctx).Infof("SendTemplateMessage: openid=%s, template_id=%s", req.Openid, req.TemplateId)

	// 1. 获取公众号access_token
	accessToken, err := s.getPublicAccessToken(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get public access token failed: %v", err)
		return nil, consumerV1.ErrorWechatAccessTokenExpired("获取access_token失败: %v", err)
	}

	// 2. 构建模板消息数据
	templateMsg := WechatTemplateMessage{
		ToUser:     req.Openid,
		TemplateID: req.TemplateId,
		Data:       make(map[string]WechatTemplateData),
	}

	// 转换模板数据
	for key, value := range req.Data {
		templateMsg.Data[key] = WechatTemplateData{
			Value: value.Value,
			Color: value.GetColor(),
		}
	}

	// 设置跳转URL
	if req.Url != nil && *req.Url != "" {
		templateMsg.URL = *req.Url
	}

	// 设置小程序跳转
	if req.MiniprogramAppid != nil && *req.MiniprogramAppid != "" {
		templateMsg.MiniProgram = &WechatMiniProgram{
			AppID:    *req.MiniprogramAppid,
			PagePath: req.GetMiniprogramPagepath(),
		}
	}

	// 3. 发送模板消息
	if err := s.sendTemplateMessageToWechat(ctx, accessToken, &templateMsg); err != nil {
		s.log.WithContext(ctx).Errorf("send template message failed: %v", err)
		return nil, consumerV1.ErrorWechatAPIFailed("发送模板消息失败: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// MiniProgramLogin 小程序登录
func (s *WechatService) MiniProgramLogin(ctx context.Context, req *consumerV1.MiniProgramLoginRequest) (*consumerV1.MiniProgramLoginResponse, error) {
	s.log.WithContext(ctx).Infof("MiniProgramLogin: code=%s", req.Code)

	// 1. 调用微信API获取session_key和openid
	sessionResp, err := s.getMiniProgramSession(ctx, req.Code)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get mini program session failed: %v", err)
		return nil, consumerV1.ErrorWechatAuthFailed("小程序登录失败: %v", err)
	}

	// 2. 返回登录结果
	return &consumerV1.MiniProgramLoginResponse{
		Openid:     sessionResp.OpenID,
		Unionid:    strPtrToPtr(sessionResp.UnionID),
		SessionKey: sessionResp.SessionKey,
	}, nil
}

// getPublicAccessToken 获取公众号access_token
func (s *WechatService) getPublicAccessToken(ctx context.Context) (string, error) {
	// 1. 尝试从缓存获取
	cacheKey := "wechat:public:access_token"
	accessToken, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && accessToken != "" {
		return accessToken, nil
	}

	// 2. 从微信API获取
	// https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=APPSECRET
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		s.appID,
		s.appSecret,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	var result WechatPublicAccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	// 3. 缓存access_token（提前60秒过期）
	ttl := time.Duration(result.ExpiresIn-60) * time.Second
	if ttl <= 0 {
		ttl = 7140 * time.Second
	}
	if err := s.redis.Set(ctx, cacheKey, result.AccessToken, ttl).Err(); err != nil {
		s.log.WithContext(ctx).Warnf("cache public access token failed: %v", err)
	}

	return result.AccessToken, nil
}

// sendTemplateMessageToWechat 发送模板消息到微信
func (s *WechatService) sendTemplateMessageToWechat(ctx context.Context, accessToken string, msg *WechatTemplateMessage) error {
	// 构建请求URL
	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", accessToken)

	// 序列化请求体
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	// 发送HTTP请求
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	// 解析响应
	var result WechatTemplateMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	// 发布事件
	if s.eventbus != nil {
		event := eventbus.WechatEventReceived{
			EventType: "template_message_sent",
			OpenID:    msg.ToUser,
			Timestamp: time.Now(),
		}
		if err := s.eventbus.Publish(ctx, event); err != nil {
			s.log.WithContext(ctx).Warnf("publish wechat event failed: %v", err)
		}
	}

	return nil
}

// getMiniProgramSession 获取小程序session
func (s *WechatService) getMiniProgramSession(ctx context.Context, code string) (*WechatMiniProgramSessionResponse, error) {
	// 构建请求URL
	// https://api.weixin.qq.com/sns/jscode2session?appid=APPID&secret=SECRET&js_code=JSCODE&grant_type=authorization_code
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.appID,
		s.appSecret,
		code,
	)

	// 发送HTTP请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 解析响应
	var result WechatMiniProgramSessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// WechatPublicAccessTokenResponse 公众号access_token响应
type WechatPublicAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WechatTemplateMessage 模板消息
type WechatTemplateMessage struct {
	ToUser      string                        `json:"touser"`
	TemplateID  string                        `json:"template_id"`
	URL         string                        `json:"url,omitempty"`
	MiniProgram *WechatMiniProgram            `json:"miniprogram,omitempty"`
	Data        map[string]WechatTemplateData `json:"data"`
}

// WechatTemplateData 模板数据
type WechatTemplateData struct {
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
}

// WechatMiniProgram 小程序信息
type WechatMiniProgram struct {
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath,omitempty"`
}

// WechatTemplateMessageResponse 模板消息响应
type WechatTemplateMessageResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	MsgID   int64  `json:"msgid"`
}

// WechatMiniProgramSessionResponse 小程序session响应
type WechatMiniProgramSessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}
