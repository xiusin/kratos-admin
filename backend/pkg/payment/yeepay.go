package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// YeepayClient 易宝支付客户端
type YeepayClient struct {
	config *Config
	logger *log.Helper
	client *http.Client
}

// NewYeepayClient 创建易宝支付客户端
func NewYeepayClient(cfg *Config, logger log.Logger) (*YeepayClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	if cfg.MchID == "" {
		return nil, fmt.Errorf("mch_id is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	l := log.NewHelper(log.With(logger, "module", "payment/yeepay"))

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &YeepayClient{
		config: cfg,
		logger: l,
		client: &http.Client{Timeout: timeout},
	}, nil
}

// CreateOrder 创建支付订单
func (c *YeepayClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.OrderNo == "" {
		return nil, fmt.Errorf("order_no is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// 根据支付类型选择不同的API
	switch req.PaymentType {
	case PaymentTypeApp:
		return c.createAppOrder(ctx, req)
	case PaymentTypeH5:
		return c.createH5Order(ctx, req)
	case PaymentTypeQRCode, PaymentTypeNative:
		return c.createQRCodeOrder(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}
}

// createAppOrder 创建APP支付订单
func (c *YeepayClient) createAppOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := map[string]string{
		"merchantNo":  c.config.MchID,
		"orderNo":     req.OrderNo,
		"orderAmount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"goodsName":   req.Subject,
		"notifyUrl":   c.config.NotifyURL,
		"payType":     "APP",
	}

	if req.Description != "" {
		params["goodsDesc"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		params["expireTime"] = req.ExpireTime.Format("2006-01-02 15:04:05")
	}

	// 调用API
	result, err := c.request(ctx, "/api/pay/create", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	return &CreateOrderResponse{
		OrderNo: req.OrderNo,
		AppPayData: map[string]string{
			"payInfo": data["payInfo"].(string),
		},
	}, nil
}

// createH5Order 创建H5支付订单
func (c *YeepayClient) createH5Order(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := map[string]string{
		"merchantNo":  c.config.MchID,
		"orderNo":     req.OrderNo,
		"orderAmount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"goodsName":   req.Subject,
		"notifyUrl":   c.config.NotifyURL,
		"payType":     "H5",
	}

	if req.Description != "" {
		params["goodsDesc"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		params["expireTime"] = req.ExpireTime.Format("2006-01-02 15:04:05")
	}

	if returnURL, ok := req.Extra["return_url"]; ok {
		params["returnUrl"] = returnURL
	}

	// 调用API
	result, err := c.request(ctx, "/api/pay/create", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	return &CreateOrderResponse{
		OrderNo: req.OrderNo,
		H5URL:   data["payUrl"].(string),
	}, nil
}

// createQRCodeOrder 创建扫码支付订单
func (c *YeepayClient) createQRCodeOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := map[string]string{
		"merchantNo":  c.config.MchID,
		"orderNo":     req.OrderNo,
		"orderAmount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"goodsName":   req.Subject,
		"notifyUrl":   c.config.NotifyURL,
		"payType":     "QRCODE",
	}

	if req.Description != "" {
		params["goodsDesc"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		params["expireTime"] = req.ExpireTime.Format("2006-01-02 15:04:05")
	}

	// 调用API
	result, err := c.request(ctx, "/api/pay/create", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	return &CreateOrderResponse{
		OrderNo: req.OrderNo,
		CodeURL: data["qrCode"].(string),
	}, nil
}

// QueryOrder 查询支付订单
func (c *YeepayClient) QueryOrder(ctx context.Context, orderNo string) (*QueryOrderResponse, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("order_no is required")
	}

	params := map[string]string{
		"merchantNo": c.config.MchID,
		"orderNo":    orderNo,
	}

	// 调用API
	result, err := c.request(ctx, "/api/pay/query", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	// 解析金额
	orderAmount := data["orderAmount"].(string)
	var amount int64
	fmt.Sscanf(orderAmount, "%f", &amount)
	amount = int64(amount * 100)

	queryResp := &QueryOrderResponse{
		OrderNo:       data["orderNo"].(string),
		TransactionID: data["tradeNo"].(string),
		Amount:        amount,
		Extra:         make(map[string]string),
	}

	// 解析订单状态
	orderStatus := data["orderStatus"].(string)
	switch orderStatus {
	case "SUCCESS":
		queryResp.Status = OrderStatusSuccess
		if payTime, ok := data["payTime"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", payTime); err == nil {
				queryResp.PaidAt = &t
			}
		}
	case "PAYING":
		queryResp.Status = OrderStatusPending
	case "CLOSED":
		queryResp.Status = OrderStatusClosed
	case "FAILED":
		queryResp.Status = OrderStatusFailed
	default:
		queryResp.Status = OrderStatusPending
	}

	return queryResp, nil
}

// CloseOrder 关闭支付订单
func (c *YeepayClient) CloseOrder(ctx context.Context, orderNo string) error {
	if orderNo == "" {
		return fmt.Errorf("order_no is required")
	}

	params := map[string]string{
		"merchantNo": c.config.MchID,
		"orderNo":    orderNo,
	}

	// 调用API
	result, err := c.request(ctx, "/api/pay/close", params)
	if err != nil {
		return err
	}

	if result["code"].(string) != "0000" {
		return fmt.Errorf("yeepay api error: %s", result["message"])
	}

	return nil
}

// Refund 申请退款
func (c *YeepayClient) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.OrderNo == "" {
		return nil, fmt.Errorf("order_no is required")
	}
	if req.RefundNo == "" {
		return nil, fmt.Errorf("refund_no is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	params := map[string]string{
		"merchantNo":   c.config.MchID,
		"orderNo":      req.OrderNo,
		"refundNo":     req.RefundNo,
		"refundAmount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
	}

	if req.Reason != "" {
		params["refundReason"] = req.Reason
	}

	// 调用API
	result, err := c.request(ctx, "/api/refund/create", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	return &RefundResponse{
		RefundNo: req.RefundNo,
		RefundID: data["refundId"].(string),
		Status:   RefundStatusSuccess,
		Amount:   req.Amount,
	}, nil
}

// QueryRefund 查询退款
func (c *YeepayClient) QueryRefund(ctx context.Context, refundNo string) (*QueryRefundResponse, error) {
	if refundNo == "" {
		return nil, fmt.Errorf("refund_no is required")
	}

	params := map[string]string{
		"merchantNo": c.config.MchID,
		"refundNo":   refundNo,
	}

	// 调用API
	result, err := c.request(ctx, "/api/refund/query", params)
	if err != nil {
		return nil, err
	}

	if result["code"].(string) != "0000" {
		return nil, fmt.Errorf("yeepay api error: %s", result["message"])
	}

	data := result["data"].(map[string]interface{})

	// 解析退款金额
	refundAmount := data["refundAmount"].(string)
	var amount int64
	fmt.Sscanf(refundAmount, "%f", &amount)
	amount = int64(amount * 100)

	queryResp := &QueryRefundResponse{
		RefundNo: refundNo,
		RefundID: data["refundId"].(string),
		Amount:   amount,
	}

	// 解析退款状态
	refundStatus := data["refundStatus"].(string)
	switch refundStatus {
	case "SUCCESS":
		queryResp.Status = RefundStatusSuccess
		if successTime, ok := data["successTime"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", successTime); err == nil {
				queryResp.SuccessTime = &t
			}
		}
	case "PROCESSING":
		queryResp.Status = RefundStatusPending
	case "FAILED":
		queryResp.Status = RefundStatusFailed
	default:
		queryResp.Status = RefundStatusPending
	}

	return queryResp, nil
}

// VerifyCallback 验证支付回调签名
func (c *YeepayClient) VerifyCallback(ctx context.Context, data map[string]string) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is required")
	}

	sign := data["sign"]
	if sign == "" {
		return false, fmt.Errorf("sign is required")
	}

	// 移除sign
	params := make(map[string]string)
	for k, v := range data {
		if k != "sign" {
			params[k] = v
		}
	}

	// 验证签名
	expectedSign := c.sign(params)
	return sign == expectedSign, nil
}

// GetProvider 获取提供商类型
func (c *YeepayClient) GetProvider() Provider {
	return ProviderYeepay
}

// sign 签名
func (c *YeepayClient) sign(params map[string]string) string {
	// 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 拼接字符串
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}

	// 添加key
	sb.WriteString("&key=")
	sb.WriteString(c.config.APIKey)

	// HMAC-SHA256签名
	h := hmac.New(sha256.New, []byte(c.config.APIKey))
	h.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// request 发送请求
func (c *YeepayClient) request(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error) {
	// 添加签名
	params["sign"] = c.sign(params)

	// 构建请求
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	apiURL := "https://api.yeepay.com" + path
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(values.Encode()))
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
