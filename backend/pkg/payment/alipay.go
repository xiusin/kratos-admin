package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// AlipayClient 支付宝客户端
type AlipayClient struct {
	config     *Config
	logger     *log.Helper
	client     *http.Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewAlipayClient 创建支付宝客户端
func NewAlipayClient(cfg *Config, logger log.Logger) (*AlipayClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("private_key is required")
	}
	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("public_key is required")
	}

	l := log.NewHelper(log.With(logger, "module", "payment/alipay"))

	// 解析私钥
	privateKey, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 解析公钥
	publicKey, err := parsePublicKey(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &AlipayClient{
		config:     cfg,
		logger:     l,
		client:     &http.Client{Timeout: timeout},
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// CreateOrder 创建支付订单
func (c *AlipayClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
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
func (c *AlipayClient) createAppOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderNo,
		"total_amount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"subject":      req.Subject,
		"product_code": "QUICK_MSECURITY_PAY",
	}

	if req.Description != "" {
		bizContent["body"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		bizContent["timeout_express"] = fmt.Sprintf("%dm", int(time.Until(req.ExpireTime).Minutes()))
	}

	params := c.buildCommonParams("alipay.trade.app.pay")
	params["notify_url"] = c.config.NotifyURL

	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 签名
	sign, err := c.sign(params)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign

	// 构建APP支付参数字符串
	appPayData := c.buildOrderString(params)

	return &CreateOrderResponse{
		OrderNo:    req.OrderNo,
		AppPayData: map[string]string{"orderString": appPayData},
	}, nil
}

// createH5Order 创建H5支付订单
func (c *AlipayClient) createH5Order(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderNo,
		"total_amount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"subject":      req.Subject,
		"product_code": "QUICK_WAP_WAY",
	}

	if req.Description != "" {
		bizContent["body"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		bizContent["timeout_express"] = fmt.Sprintf("%dm", int(time.Until(req.ExpireTime).Minutes()))
	}

	params := c.buildCommonParams("alipay.trade.wap.pay")
	params["notify_url"] = c.config.NotifyURL
	params["return_url"] = req.Extra["return_url"]

	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 签名
	sign, err := c.sign(params)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign

	// 构建H5支付URL
	h5URL := "https://openapi.alipay.com/gateway.do?" + c.buildOrderString(params)

	return &CreateOrderResponse{
		OrderNo: req.OrderNo,
		H5URL:   h5URL,
	}, nil
}

// createQRCodeOrder 创建扫码支付订单
func (c *AlipayClient) createQRCodeOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderNo,
		"total_amount": fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"subject":      req.Subject,
	}

	if req.Description != "" {
		bizContent["body"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		bizContent["timeout_express"] = fmt.Sprintf("%dm", int(time.Until(req.ExpireTime).Minutes()))
	}

	params := c.buildCommonParams("alipay.trade.precreate")
	params["notify_url"] = c.config.NotifyURL

	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 调用API
	result, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}

	response := result["alipay_trade_precreate_response"].(map[string]interface{})
	if response["code"].(string) != "10000" {
		return nil, fmt.Errorf("alipay api error: %s", response["msg"])
	}

	return &CreateOrderResponse{
		OrderNo: req.OrderNo,
		CodeURL: response["qr_code"].(string),
	}, nil
}

// QueryOrder 查询支付订单
func (c *AlipayClient) QueryOrder(ctx context.Context, orderNo string) (*QueryOrderResponse, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("order_no is required")
	}

	bizContent := map[string]interface{}{
		"out_trade_no": orderNo,
	}

	params := c.buildCommonParams("alipay.trade.query")
	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 调用API
	result, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}

	response := result["alipay_trade_query_response"].(map[string]interface{})
	if response["code"].(string) != "10000" {
		return nil, fmt.Errorf("alipay api error: %s", response["msg"])
	}

	// 解析金额
	totalAmount := response["total_amount"].(string)
	var amount int64
	fmt.Sscanf(totalAmount, "%f", &amount)
	amount = int64(amount * 100)

	queryResp := &QueryOrderResponse{
		OrderNo:       response["out_trade_no"].(string),
		TransactionID: response["trade_no"].(string),
		Amount:        amount,
		Extra:         make(map[string]string),
	}

	// 解析交易状态
	tradeStatus := response["trade_status"].(string)
	switch tradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		queryResp.Status = OrderStatusSuccess
		if sendPayDate, ok := response["send_pay_date"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", sendPayDate); err == nil {
				queryResp.PaidAt = &t
			}
		}
	case "WAIT_BUYER_PAY":
		queryResp.Status = OrderStatusPending
	case "TRADE_CLOSED":
		queryResp.Status = OrderStatusClosed
	default:
		queryResp.Status = OrderStatusPending
	}

	return queryResp, nil
}

// CloseOrder 关闭支付订单
func (c *AlipayClient) CloseOrder(ctx context.Context, orderNo string) error {
	if orderNo == "" {
		return fmt.Errorf("order_no is required")
	}

	bizContent := map[string]interface{}{
		"out_trade_no": orderNo,
	}

	params := c.buildCommonParams("alipay.trade.close")
	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 调用API
	result, err := c.request(ctx, params)
	if err != nil {
		return err
	}

	response := result["alipay_trade_close_response"].(map[string]interface{})
	if response["code"].(string) != "10000" {
		return fmt.Errorf("alipay api error: %s", response["msg"])
	}

	return nil
}

// Refund 申请退款
func (c *AlipayClient) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
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

	bizContent := map[string]interface{}{
		"out_trade_no":   req.OrderNo,
		"out_request_no": req.RefundNo,
		"refund_amount":  fmt.Sprintf("%.2f", float64(req.Amount)/100),
	}

	if req.Reason != "" {
		bizContent["refund_reason"] = req.Reason
	}

	params := c.buildCommonParams("alipay.trade.refund")
	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 调用API
	result, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}

	response := result["alipay_trade_refund_response"].(map[string]interface{})
	if response["code"].(string) != "10000" {
		return nil, fmt.Errorf("alipay api error: %s", response["msg"])
	}

	return &RefundResponse{
		RefundNo: req.RefundNo,
		RefundID: response["trade_no"].(string),
		Status:   RefundStatusSuccess,
		Amount:   req.Amount,
	}, nil
}

// QueryRefund 查询退款
func (c *AlipayClient) QueryRefund(ctx context.Context, refundNo string) (*QueryRefundResponse, error) {
	if refundNo == "" {
		return nil, fmt.Errorf("refund_no is required")
	}

	bizContent := map[string]interface{}{
		"out_request_no": refundNo,
	}

	params := c.buildCommonParams("alipay.trade.fastpay.refund.query")
	bizContentJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizContentJSON)

	// 调用API
	result, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}

	response := result["alipay_trade_fastpay_refund_query_response"].(map[string]interface{})
	if response["code"].(string) != "10000" {
		return nil, fmt.Errorf("alipay api error: %s", response["msg"])
	}

	// 解析退款金额
	refundAmount := response["refund_amount"].(string)
	var amount int64
	fmt.Sscanf(refundAmount, "%f", &amount)
	amount = int64(amount * 100)

	queryResp := &QueryRefundResponse{
		RefundNo: refundNo,
		RefundID: response["trade_no"].(string),
		Amount:   amount,
		Status:   RefundStatusSuccess,
	}

	return queryResp, nil
}

// VerifyCallback 验证支付回调签名
func (c *AlipayClient) VerifyCallback(ctx context.Context, data map[string]string) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is required")
	}

	sign := data["sign"]
	signType := data["sign_type"]

	if sign == "" || signType != "RSA2" {
		return false, fmt.Errorf("invalid sign or sign_type")
	}

	// 移除sign和sign_type
	params := make(map[string]string)
	for k, v := range data {
		if k != "sign" && k != "sign_type" {
			params[k] = v
		}
	}

	// 验证签名
	return c.verifySign(params, sign)
}

// GetProvider 获取提供商类型
func (c *AlipayClient) GetProvider() Provider {
	return ProviderAlipay
}

// buildCommonParams 构建公共参数
func (c *AlipayClient) buildCommonParams(method string) map[string]string {
	return map[string]string{
		"app_id":    c.config.AppID,
		"method":    method,
		"format":    "JSON",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"version":   "1.0",
	}
}

// sign 签名
func (c *AlipayClient) sign(params map[string]string) (string, error) {
	// 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && params[k] != "" {
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

	// RSA2签名
	h := sha256.New()
	h.Write([]byte(sb.String()))
	hashed := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySign 验证签名
func (c *AlipayClient) verifySign(params map[string]string, sign string) (bool, error) {
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

	// 解码签名
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false, err
	}

	// 验证签名
	h := sha256.New()
	h.Write([]byte(sb.String()))
	hashed := h.Sum(nil)

	err = rsa.VerifyPKCS1v15(c.publicKey, crypto.SHA256, hashed, signBytes)
	return err == nil, nil
}

// buildOrderString 构建订单参数字符串
func (c *AlipayClient) buildOrderString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(url.QueryEscape(params[k]))
	}

	return sb.String()
}

// request 发送请求
func (c *AlipayClient) request(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	// 签名
	sign, err := c.sign(params)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign

	// 构建请求
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openapi.alipay.com/gateway.do", strings.NewReader(values.Encode()))
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

// parsePrivateKey 解析私钥
func parsePrivateKey(privateKeyStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsaPrivateKey, nil
}

// parsePublicKey 解析公钥
func parsePublicKey(publicKeyStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPublicKey, nil
}
