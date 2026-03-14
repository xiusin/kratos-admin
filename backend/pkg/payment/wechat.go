package payment

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// WechatClient 微信支付客户端
type WechatClient struct {
	config *Config
	logger *log.Helper
	client *http.Client
}

// NewWechatClient 创建微信支付客户端
func NewWechatClient(cfg *Config, logger log.Logger) (*WechatClient, error) {
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

	l := log.NewHelper(log.With(logger, "module", "payment/wechat"))

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &WechatClient{
		config: cfg,
		logger: l,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// CreateOrder 创建支付订单
func (c *WechatClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
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
	case PaymentTypeMini, PaymentTypeJSAPI:
		return c.createJSAPIOrder(ctx, req)
	case PaymentTypeNative, PaymentTypeQRCode:
		return c.createNativeOrder(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}
}

// createAppOrder 创建APP支付订单
func (c *WechatClient) createAppOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := c.buildUnifiedOrderParams(req, "APP")

	// 调用统一下单API
	result, err := c.unifiedOrder(ctx, params)
	if err != nil {
		return nil, err
	}

	// 构建APP支付参数
	appPayData := c.buildAppPayData(result["prepay_id"])

	return &CreateOrderResponse{
		OrderNo:    req.OrderNo,
		PrepayID:   result["prepay_id"],
		AppPayData: appPayData,
	}, nil
}

// createH5Order 创建H5支付订单
func (c *WechatClient) createH5Order(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := c.buildUnifiedOrderParams(req, "MWEB")

	// 调用统一下单API
	result, err := c.unifiedOrder(ctx, params)
	if err != nil {
		return nil, err
	}

	return &CreateOrderResponse{
		OrderNo:  req.OrderNo,
		PrepayID: result["prepay_id"],
		H5URL:    result["mweb_url"],
	}, nil
}

// createJSAPIOrder 创建JSAPI/小程序支付订单
func (c *WechatClient) createJSAPIOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if req.OpenID == "" {
		return nil, fmt.Errorf("openid is required for JSAPI payment")
	}

	params := c.buildUnifiedOrderParams(req, "JSAPI")
	params["openid"] = req.OpenID

	// 调用统一下单API
	result, err := c.unifiedOrder(ctx, params)
	if err != nil {
		return nil, err
	}

	// 构建JSAPI支付参数
	jsapiPayData := c.buildJSAPIPayData(result["prepay_id"])

	return &CreateOrderResponse{
		OrderNo:      req.OrderNo,
		PrepayID:     result["prepay_id"],
		JSAPIPayData: jsapiPayData,
		MiniPayData:  jsapiPayData, // 小程序支付参数与JSAPI相同
	}, nil
}

// createNativeOrder 创建Native扫码支付订单
func (c *WechatClient) createNativeOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := c.buildUnifiedOrderParams(req, "NATIVE")

	// 调用统一下单API
	result, err := c.unifiedOrder(ctx, params)
	if err != nil {
		return nil, err
	}

	return &CreateOrderResponse{
		OrderNo:  req.OrderNo,
		PrepayID: result["prepay_id"],
		CodeURL:  result["code_url"],
	}, nil
}

// buildUnifiedOrderParams 构建统一下单参数
func (c *WechatClient) buildUnifiedOrderParams(req *CreateOrderRequest, tradeType string) map[string]string {
	params := map[string]string{
		"appid":            c.config.AppID,
		"mch_id":           c.config.MchID,
		"nonce_str":        c.generateNonceStr(),
		"body":             req.Subject,
		"out_trade_no":     req.OrderNo,
		"total_fee":        fmt.Sprintf("%d", req.Amount),
		"spbill_create_ip": "127.0.0.1",
		"notify_url":       c.config.NotifyURL,
		"trade_type":       tradeType,
	}

	if req.Description != "" {
		params["detail"] = req.Description
	}

	if !req.ExpireTime.IsZero() {
		params["time_expire"] = req.ExpireTime.Format("20060102150405")
	}

	return params
}

// unifiedOrder 统一下单
func (c *WechatClient) unifiedOrder(ctx context.Context, params map[string]string) (map[string]string, error) {
	// 添加签名
	params["sign"] = c.sign(params)

	// 构建XML请求
	xmlData := c.mapToXML(params)

	// 发送请求
	resp, err := c.postXML(ctx, "https://api.mch.weixin.qq.com/pay/unifiedorder", xmlData)
	if err != nil {
		return nil, err
	}

	// 解析响应
	result, err := c.xmlToMap(resp)
	if err != nil {
		return nil, err
	}

	// 检查返回码
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat api error: %s", result["return_msg"])
	}

	if result["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat business error: %s", result["err_code_des"])
	}

	// 验证签名
	if !c.verifySign(result) {
		return nil, fmt.Errorf("invalid signature")
	}

	return result, nil
}

// QueryOrder 查询支付订单
func (c *WechatClient) QueryOrder(ctx context.Context, orderNo string) (*QueryOrderResponse, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("order_no is required")
	}

	params := map[string]string{
		"appid":         c.config.AppID,
		"mch_id":        c.config.MchID,
		"out_trade_no":  orderNo,
		"nonce_str":     c.generateNonceStr(),
	}

	// 添加签名
	params["sign"] = c.sign(params)

	// 构建XML请求
	xmlData := c.mapToXML(params)

	// 发送请求
	resp, err := c.postXML(ctx, "https://api.mch.weixin.qq.com/pay/orderquery", xmlData)
	if err != nil {
		return nil, err
	}

	// 解析响应
	result, err := c.xmlToMap(resp)
	if err != nil {
		return nil, err
	}

	// 检查返回码
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat api error: %s", result["return_msg"])
	}

	// 验证签名
	if !c.verifySign(result) {
		return nil, fmt.Errorf("invalid signature")
	}

	// 构建响应
	response := &QueryOrderResponse{
		OrderNo:       result["out_trade_no"],
		TransactionID: result["transaction_id"],
		Amount:        c.parseAmount(result["total_fee"]),
		Extra:         result,
	}

	// 解析订单状态
	switch result["trade_state"] {
	case "SUCCESS":
		response.Status = OrderStatusSuccess
		if timeEnd := result["time_end"]; timeEnd != "" {
			if t, err := time.Parse("20060102150405", timeEnd); err == nil {
				response.PaidAt = &t
			}
		}
	case "NOTPAY":
		response.Status = OrderStatusPending
	case "CLOSED":
		response.Status = OrderStatusClosed
	case "PAYERROR":
		response.Status = OrderStatusFailed
	default:
		response.Status = OrderStatusPending
	}

	return response, nil
}

// CloseOrder 关闭支付订单
func (c *WechatClient) CloseOrder(ctx context.Context, orderNo string) error {
	if orderNo == "" {
		return fmt.Errorf("order_no is required")
	}

	params := map[string]string{
		"appid":        c.config.AppID,
		"mch_id":       c.config.MchID,
		"out_trade_no": orderNo,
		"nonce_str":    c.generateNonceStr(),
	}

	// 添加签名
	params["sign"] = c.sign(params)

	// 构建XML请求
	xmlData := c.mapToXML(params)

	// 发送请求
	resp, err := c.postXML(ctx, "https://api.mch.weixin.qq.com/pay/closeorder", xmlData)
	if err != nil {
		return err
	}

	// 解析响应
	result, err := c.xmlToMap(resp)
	if err != nil {
		return err
	}

	// 检查返回码
	if result["return_code"] != "SUCCESS" {
		return fmt.Errorf("wechat api error: %s", result["return_msg"])
	}

	if result["result_code"] != "SUCCESS" {
		return fmt.Errorf("wechat business error: %s", result["err_code_des"])
	}

	return nil
}

// Refund 申请退款
func (c *WechatClient) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
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
		"appid":         c.config.AppID,
		"mch_id":        c.config.MchID,
		"nonce_str":     c.generateNonceStr(),
		"out_trade_no":  req.OrderNo,
		"out_refund_no": req.RefundNo,
		"total_fee":     fmt.Sprintf("%d", req.TotalAmount),
		"refund_fee":    fmt.Sprintf("%d", req.Amount),
	}

	if req.Reason != "" {
		params["refund_desc"] = req.Reason
	}

	// 添加签名
	params["sign"] = c.sign(params)

	// 构建XML请求
	xmlData := c.mapToXML(params)

	// 发送请求（退款需要使用证书）
	resp, err := c.postXMLWithCert(ctx, "https://api.mch.weixin.qq.com/secapi/pay/refund", xmlData)
	if err != nil {
		return nil, err
	}

	// 解析响应
	result, err := c.xmlToMap(resp)
	if err != nil {
		return nil, err
	}

	// 检查返回码
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat api error: %s", result["return_msg"])
	}

	if result["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat business error: %s", result["err_code_des"])
	}

	// 验证签名
	if !c.verifySign(result) {
		return nil, fmt.Errorf("invalid signature")
	}

	return &RefundResponse{
		RefundNo: result["out_refund_no"],
		RefundID: result["refund_id"],
		Status:   RefundStatusSuccess,
		Amount:   c.parseAmount(result["refund_fee"]),
	}, nil
}

// QueryRefund 查询退款
func (c *WechatClient) QueryRefund(ctx context.Context, refundNo string) (*QueryRefundResponse, error) {
	if refundNo == "" {
		return nil, fmt.Errorf("refund_no is required")
	}

	params := map[string]string{
		"appid":         c.config.AppID,
		"mch_id":        c.config.MchID,
		"out_refund_no": refundNo,
		"nonce_str":     c.generateNonceStr(),
	}

	// 添加签名
	params["sign"] = c.sign(params)

	// 构建XML请求
	xmlData := c.mapToXML(params)

	// 发送请求
	resp, err := c.postXML(ctx, "https://api.mch.weixin.qq.com/pay/refundquery", xmlData)
	if err != nil {
		return nil, err
	}

	// 解析响应
	result, err := c.xmlToMap(resp)
	if err != nil {
		return nil, err
	}

	// 检查返回码
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat api error: %s", result["return_msg"])
	}

	if result["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat business error: %s", result["err_code_des"])
	}

	// 验证签名
	if !c.verifySign(result) {
		return nil, fmt.Errorf("invalid signature")
	}

	response := &QueryRefundResponse{
		RefundNo: result["out_refund_no"],
		RefundID: result["refund_id"],
		Amount:   c.parseAmount(result["refund_fee_0"]),
	}

	// 解析退款状态
	switch result["refund_status_0"] {
	case "SUCCESS":
		response.Status = RefundStatusSuccess
		if successTime := result["refund_success_time_0"]; successTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", successTime); err == nil {
				response.SuccessTime = &t
			}
		}
	case "PROCESSING":
		response.Status = RefundStatusPending
	case "CHANGE", "REFUNDCLOSE":
		response.Status = RefundStatusFailed
	default:
		response.Status = RefundStatusPending
	}

	return response, nil
}

// VerifyCallback 验证支付回调签名
func (c *WechatClient) VerifyCallback(ctx context.Context, data map[string]string) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is required")
	}

	// 检查返回码
	if data["return_code"] != "SUCCESS" {
		return false, fmt.Errorf("callback return_code is not SUCCESS")
	}

	// 验证签名
	return c.verifySign(data), nil
}

// GetProvider 获取提供商类型
func (c *WechatClient) GetProvider() Provider {
	return ProviderWechat
}

// sign 签名
func (c *WechatClient) sign(params map[string]string) string {
	// 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 拼接字符串
	var sb strings.Builder
	for _, k := range keys {
		if params[k] != "" {
			if sb.Len() > 0 {
				sb.WriteString("&")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(params[k])
		}
	}

	// 添加key
	sb.WriteString("&key=")
	sb.WriteString(c.config.APIKey)

	// MD5签名
	hash := md5.Sum([]byte(sb.String()))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// verifySign 验证签名
func (c *WechatClient) verifySign(params map[string]string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}

	expectedSign := c.sign(params)
	return sign == expectedSign
}

// generateNonceStr 生成随机字符串
func (c *WechatClient) generateNonceStr() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// buildAppPayData 构建APP支付参数
func (c *WechatClient) buildAppPayData(prepayID string) map[string]string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := c.generateNonceStr()

	params := map[string]string{
		"appid":     c.config.AppID,
		"partnerid": c.config.MchID,
		"prepayid":  prepayID,
		"package":   "Sign=WXPay",
		"noncestr":  nonceStr,
		"timestamp": timestamp,
	}

	params["sign"] = c.sign(params)

	return params
}

// buildJSAPIPayData 构建JSAPI支付参数
func (c *WechatClient) buildJSAPIPayData(prepayID string) map[string]string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := c.generateNonceStr()

	params := map[string]string{
		"appId":     c.config.AppID,
		"timeStamp": timestamp,
		"nonceStr":  nonceStr,
		"package":   "prepay_id=" + prepayID,
		"signType":  "MD5",
	}

	params["paySign"] = c.sign(params)

	return params
}

// mapToXML 将map转换为XML
func (c *WechatClient) mapToXML(params map[string]string) []byte {
	var sb strings.Builder
	sb.WriteString("<xml>")
	for k, v := range params {
		sb.WriteString("<")
		sb.WriteString(k)
		sb.WriteString("><![CDATA[")
		sb.WriteString(v)
		sb.WriteString("]]></")
		sb.WriteString(k)
		sb.WriteString(">")
	}
	sb.WriteString("</xml>")
	return []byte(sb.String())
}

// xmlToMap 将XML转换为map
func (c *WechatClient) xmlToMap(data []byte) (map[string]string, error) {
	result := make(map[string]string)

	type xmlMap struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}

	type xmlRoot struct {
		Items []xmlMap `xml:",any"`
	}

	var root xmlRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	for _, item := range root.Items {
		result[item.XMLName.Local] = item.Value
	}

	return result, nil
}

// postXML 发送XML请求
func (c *WechatClient) postXML(ctx context.Context, url string, data []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// postXMLWithCert 发送带证书的XML请求
func (c *WechatClient) postXMLWithCert(ctx context.Context, url string, data []byte) ([]byte, error) {
	// TODO: 实现证书加载和HTTPS请求
	// 这里简化处理，实际需要加载证书文件
	return c.postXML(ctx, url, data)
}

// parseAmount 解析金额
func (c *WechatClient) parseAmount(amountStr string) int64 {
	var amount int64
	fmt.Sscanf(amountStr, "%d", &amount)
	return amount
}
