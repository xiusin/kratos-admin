package payment

import (
	"context"
	"time"
)

// Provider 支付服务提供商类型
type Provider string

const (
	ProviderWechat Provider = "wechat" // 微信支付
	ProviderAlipay Provider = "alipay" // 支付宝
	ProviderYeepay Provider = "yeepay" // 易宝支付
)

// PaymentType 支付类型
type PaymentType string

const (
	PaymentTypeApp    PaymentType = "app"    // APP支付
	PaymentTypeH5     PaymentType = "h5"     // H5支付
	PaymentTypeMini   PaymentType = "mini"   // 小程序支付
	PaymentTypeQRCode PaymentType = "qrcode" // 扫码支付
	PaymentTypeJSAPI  PaymentType = "jsapi"  // JSAPI支付
	PaymentTypeNative PaymentType = "native" // Native支付
)

// Client 支付客户端接口
type Client interface {
	// CreateOrder 创建支付订单
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)

	// QueryOrder 查询支付订单
	QueryOrder(ctx context.Context, orderNo string) (*QueryOrderResponse, error)

	// CloseOrder 关闭支付订单
	CloseOrder(ctx context.Context, orderNo string) error

	// Refund 申请退款
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)

	// QueryRefund 查询退款
	QueryRefund(ctx context.Context, refundNo string) (*QueryRefundResponse, error)

	// VerifyCallback 验证支付回调签名
	VerifyCallback(ctx context.Context, data map[string]string) (bool, error)

	// GetProvider 获取提供商类型
	GetProvider() Provider
}

// Config 支付配置
type Config struct {
	// Provider 提供商 (wechat/alipay/yeepay)
	Provider Provider `json:"provider"`

	// AppID 应用ID
	AppID string `json:"app_id"`

	// MchID 商户号
	MchID string `json:"mch_id"`

	// APIKey API密钥
	APIKey string `json:"api_key"`

	// PrivateKey 私钥（用于签名）
	PrivateKey string `json:"private_key"`

	// PublicKey 公钥（用于验签）
	PublicKey string `json:"public_key"`

	// CertPath 证书路径
	CertPath string `json:"cert_path"`

	// NotifyURL 回调通知URL
	NotifyURL string `json:"notify_url"`

	// Timeout 超时时间
	Timeout time.Duration `json:"timeout"`
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	// OrderNo 订单号（商户订单号）
	OrderNo string

	// Amount 支付金额（单位：分）
	Amount int64

	// Currency 货币类型（默认：CNY）
	Currency string

	// Subject 订单标题
	Subject string

	// Description 订单描述
	Description string

	// PaymentType 支付类型
	PaymentType PaymentType

	// OpenID 用户OpenID（微信JSAPI/小程序支付必填）
	OpenID string

	// ExpireTime 订单过期时间
	ExpireTime time.Time

	// Extra 扩展参数
	Extra map[string]string
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	// OrderNo 订单号
	OrderNo string

	// PrepayID 预支付ID
	PrepayID string

	// CodeURL 二维码URL（扫码支付）
	CodeURL string

	// H5URL H5支付URL
	H5URL string

	// AppPayData APP支付数据
	AppPayData map[string]string

	// MiniPayData 小程序支付数据
	MiniPayData map[string]string

	// JSAPIPayData JSAPI支付数据
	JSAPIPayData map[string]string
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	// OrderNo 订单号
	OrderNo string

	// TransactionID 第三方交易号
	TransactionID string

	// Status 订单状态
	Status OrderStatus

	// Amount 支付金额（单位：分）
	Amount int64

	// PaidAt 支付时间
	PaidAt *time.Time

	// Extra 扩展信息
	Extra map[string]string
}

// RefundRequest 退款请求
type RefundRequest struct {
	// OrderNo 原订单号
	OrderNo string

	// RefundNo 退款单号
	RefundNo string

	// Amount 退款金额（单位：分）
	Amount int64

	// TotalAmount 订单总金额（单位：分）
	TotalAmount int64

	// Reason 退款原因
	Reason string
}

// RefundResponse 退款响应
type RefundResponse struct {
	// RefundNo 退款单号
	RefundNo string

	// RefundID 第三方退款ID
	RefundID string

	// Status 退款状态
	Status RefundStatus

	// Amount 退款金额（单位：分）
	Amount int64
}

// QueryRefundResponse 查询退款响应
type QueryRefundResponse struct {
	// RefundNo 退款单号
	RefundNo string

	// RefundID 第三方退款ID
	RefundID string

	// Status 退款状态
	Status RefundStatus

	// Amount 退款金额（单位：分）
	Amount int64

	// SuccessTime 退款成功时间
	SuccessTime *time.Time
}

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending" // 待支付
	OrderStatusSuccess OrderStatus = "success" // 支付成功
	OrderStatusFailed  OrderStatus = "failed"  // 支付失败
	OrderStatusClosed  OrderStatus = "closed"  // 已关闭
)

// RefundStatus 退款状态
type RefundStatus string

const (
	RefundStatusPending RefundStatus = "pending" // 退款中
	RefundStatusSuccess RefundStatus = "success" // 退款成功
	RefundStatusFailed  RefundStatus = "failed"  // 退款失败
)
