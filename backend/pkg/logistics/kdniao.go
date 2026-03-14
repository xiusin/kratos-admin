package logistics

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// KDNiaoClient 快递鸟API客户端
type KDNiaoClient struct {
	config *Config
	logger *log.Helper
	client *http.Client
}

// NewKDNiaoClient 创建快递鸟客户端
func NewKDNiaoClient(cfg *Config, logger log.Logger) (*KDNiaoClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	if cfg.AppKey == "" {
		return nil, fmt.Errorf("app_key is required")
	}

	l := log.NewHelper(log.With(logger, "module", "logistics/kdniao"))

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &KDNiaoClient{
		config: cfg,
		logger: l,
		client: &http.Client{Timeout: timeout},
	}, nil
}

// Query 查询物流信息
func (c *KDNiaoClient) Query(ctx context.Context, trackingNo string, courierCode string) (*TrackingInfo, error) {
	if trackingNo == "" {
		return nil, fmt.Errorf("tracking_no is required")
	}

	// 如果没有指定快递公司，自动识别
	if courierCode == "" {
		code, err := c.RecognizeCourier(ctx, trackingNo)
		if err != nil {
			return nil, err
		}
		courierCode = code
	}

	// 构建请求参数
	requestData := map[string]interface{}{
		"LogisticCode": trackingNo,
		"ShipperCode":  courierCode,
	}

	requestJSON, _ := json.Marshal(requestData)

	// 调用API
	result, err := c.request(ctx, "1002", string(requestJSON))
	if err != nil {
		return nil, err
	}

	// 解析响应
	if !result["Success"].(bool) {
		return nil, fmt.Errorf("kdniao api error: %s", result["Reason"])
	}

	// 构建物流信息
	trackingInfo := &TrackingInfo{
		TrackingNo:     trackingNo,
		CourierCode:    courierCode,
		CourierName:    c.getCourierName(courierCode),
		Status:         c.parseStatus(result["State"].(string)),
		LastUpdateTime: time.Now(),
		Traces:         make([]*TrackingTrace, 0),
	}

	// 解析物流轨迹
	if traces, ok := result["Traces"].([]interface{}); ok {
		for _, t := range traces {
			trace := t.(map[string]interface{})
			acceptTime, _ := time.Parse("2006-01-02 15:04:05", trace["AcceptTime"].(string))

			trackingInfo.Traces = append(trackingInfo.Traces, &TrackingTrace{
				Time:        acceptTime,
				Description: trace["AcceptStation"].(string),
				Location:    trace["Location"].(string),
			})
		}
	}

	return trackingInfo, nil
}

// RecognizeCourier 识别快递公司
func (c *KDNiaoClient) RecognizeCourier(ctx context.Context, trackingNo string) (string, error) {
	if trackingNo == "" {
		return "", fmt.Errorf("tracking_no is required")
	}

	// 构建请求参数
	requestData := map[string]interface{}{
		"LogisticCode": trackingNo,
	}

	requestJSON, _ := json.Marshal(requestData)

	// 调用API
	result, err := c.request(ctx, "2002", string(requestJSON))
	if err != nil {
		return "", err
	}

	// 解析响应
	if !result["Success"].(bool) {
		return "", fmt.Errorf("kdniao api error: %s", result["Reason"])
	}

	// 获取快递公司列表
	shippers, ok := result["Shippers"].([]interface{})
	if !ok || len(shippers) == 0 {
		return "", fmt.Errorf("no courier company found")
	}

	// 返回第一个匹配的快递公司
	shipper := shippers[0].(map[string]interface{})
	return shipper["ShipperCode"].(string), nil
}

// Subscribe 订阅物流状态
func (c *KDNiaoClient) Subscribe(ctx context.Context, trackingNo string, courierCode string) error {
	if trackingNo == "" {
		return fmt.Errorf("tracking_no is required")
	}
	if courierCode == "" {
		return fmt.Errorf("courier_code is required")
	}

	// 构建请求参数
	requestData := map[string]interface{}{
		"LogisticCode": trackingNo,
		"ShipperCode":  courierCode,
	}

	requestJSON, _ := json.Marshal(requestData)

	// 调用API
	result, err := c.request(ctx, "1008", string(requestJSON))
	if err != nil {
		return err
	}

	// 解析响应
	if !result["Success"].(bool) {
		return fmt.Errorf("kdniao api error: %s", result["Reason"])
	}

	c.logger.Infof("subscribed to logistics tracking: %s", trackingNo)
	return nil
}

// GetSupportedCouriers 获取支持的快递公司列表
func (c *KDNiaoClient) GetSupportedCouriers() []CourierInfo {
	return []CourierInfo{
		{Code: "SF", Name: "顺丰速运"},
		{Code: "HTKY", Name: "百世快递"},
		{Code: "ZTO", Name: "中通快递"},
		{Code: "STO", Name: "申通快递"},
		{Code: "YTO", Name: "圆通速递"},
		{Code: "YD", Name: "韵达速递"},
		{Code: "YZPY", Name: "邮政快递包裹"},
		{Code: "EMS", Name: "EMS"},
		{Code: "HHTT", Name: "天天快递"},
		{Code: "JD", Name: "京东快递"},
		{Code: "UC", Name: "优速快递"},
		{Code: "DBL", Name: "德邦快递"},
		{Code: "FAST", Name: "快捷快递"},
		{Code: "ZJS", Name: "宅急送"},
	}
}

// request 发送请求
func (c *KDNiaoClient) request(ctx context.Context, requestType string, requestData string) (map[string]interface{}, error) {
	// 生成签名
	dataSign := c.sign(requestData)

	// 构建请求参数
	params := url.Values{}
	params.Set("RequestData", url.QueryEscape(requestData))
	params.Set("EBusinessID", c.config.AppID)
	params.Set("RequestType", requestType)
	params.Set("DataSign", url.QueryEscape(dataSign))
	params.Set("DataType", "2") // JSON格式

	// 发送请求
	apiURL := "https://api.kdniao.com/Ebusiness/EbusinessOrderHandle.aspx"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// sign 生成签名
func (c *KDNiaoClient) sign(data string) string {
	// 拼接字符串
	signStr := data + c.config.AppKey

	// MD5加密
	hash := md5.Sum([]byte(signStr))
	hashStr := hex.EncodeToString(hash[:])

	// Base64编码
	return base64.StdEncoding.EncodeToString([]byte(hashStr))
}

// parseStatus 解析物流状态
func (c *KDNiaoClient) parseStatus(state string) TrackingStatus {
	switch state {
	case "0":
		return StatusPending
	case "1":
		return StatusPickedUp
	case "2":
		return StatusInTransit
	case "3":
		return StatusDelivering
	case "4":
		return StatusDelivered
	case "5":
		return StatusException
	case "6":
		return StatusReturning
	default:
		return StatusPending
	}
}

// getCourierName 获取快递公司名称
func (c *KDNiaoClient) getCourierName(code string) string {
	couriers := c.GetSupportedCouriers()
	for _, courier := range couriers {
		if courier.Code == code {
			return courier.Name
		}
	}
	return code
}
