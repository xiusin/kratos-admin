package sms

import (
	"context"
	"time"
)

// Provider 短信服务提供商类型
type Provider string

const (
	ProviderAliyun  Provider = "aliyun"  // 阿里云短信
	ProviderTencent Provider = "tencent" // 腾讯云短信
)

// Client 短信客户端接口
type Client interface {
	// Send 发送短信
	// phone: 手机号
	// templateCode: 模板代码
	// templateParams: 模板参数
	Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error

	// SendVerificationCode 发送验证码
	// phone: 手机号
	// code: 验证码
	SendVerificationCode(ctx context.Context, phone string, code string) error

	// GetProvider 获取提供商类型
	GetProvider() Provider
}

// Config 短信配置
type Config struct {
	// Provider 提供商 (aliyun/tencent)
	Provider Provider `json:"provider"`

	// AccessKeyID 访问密钥ID
	AccessKeyID string `json:"access_key_id"`

	// AccessKeySecret 访问密钥Secret
	AccessKeySecret string `json:"access_key_secret"`

	// SignName 短信签名
	SignName string `json:"sign_name"`

	// VerificationCodeTemplate 验证码模板代码
	VerificationCodeTemplate string `json:"verification_code_template"`

	// NotificationTemplate 通知短信模板代码
	NotificationTemplate string `json:"notification_template"`

	// Timeout 超时时间
	Timeout time.Duration `json:"timeout"`
}

// Template 短信模板
type Template struct {
	// Code 模板代码
	Code string

	// Params 模板参数
	Params map[string]string
}

// SendResult 发送结果
type SendResult struct {
	// Success 是否成功
	Success bool

	// RequestID 请求ID
	RequestID string

	// Message 消息
	Message string

	// Provider 提供商
	Provider Provider
}
