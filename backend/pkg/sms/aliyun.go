package sms

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

// AliyunClient 阿里云短信客户端
type AliyunClient struct {
	client   *dysmsapi.Client
	config   *Config
	logger   *log.Helper
	signName string
}

// NewAliyunClient 创建阿里云短信客户端
func NewAliyunClient(cfg *Config, logger log.Logger) (*AliyunClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("access_key_id is required")
	}
	if cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("access_key_secret is required")
	}
	if cfg.SignName == "" {
		return nil, fmt.Errorf("sign_name is required")
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
	}

	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun sms client: %w", err)
	}

	l := log.NewHelper(log.With(logger, "module", "sms/aliyun"))

	return &AliyunClient{
		client:   client,
		config:   cfg,
		logger:   l,
		signName: cfg.SignName,
	}, nil
}

// Send 发送短信
func (c *AliyunClient) Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error {
	if phone == "" {
		return fmt.Errorf("phone is required")
	}
	if templateCode == "" {
		return fmt.Errorf("template_code is required")
	}

	// 构建模板参数JSON字符串
	templateParamJSON := "{"
	first := true
	for k, v := range templateParams {
		if !first {
			templateParamJSON += ","
		}
		templateParamJSON += fmt.Sprintf(`"%s":"%s"`, k, v)
		first = false
	}
	templateParamJSON += "}"

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(c.signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(templateParamJSON),
	}

	runtime := &util.RuntimeOptions{}
	if c.config.Timeout > 0 {
		runtime.ReadTimeout = tea.Int(int(c.config.Timeout.Milliseconds()))
		runtime.ConnectTimeout = tea.Int(int(c.config.Timeout.Milliseconds()))
	}

	response, err := c.client.SendSmsWithOptions(request, runtime)
	if err != nil {
		c.logger.Errorf("failed to send sms: %v", err)
		return fmt.Errorf("failed to send sms: %w", err)
	}

	if response.Body == nil {
		return fmt.Errorf("empty response body")
	}

	// 检查返回码
	if tea.StringValue(response.Body.Code) != "OK" {
		c.logger.Errorf("sms send failed: code=%s, message=%s",
			tea.StringValue(response.Body.Code),
			tea.StringValue(response.Body.Message))
		return fmt.Errorf("sms send failed: %s", tea.StringValue(response.Body.Message))
	}

	c.logger.Infof("sms sent successfully: phone=%s, template=%s, request_id=%s",
		phone, templateCode, tea.StringValue(response.Body.RequestId))

	return nil
}

// SendVerificationCode 发送验证码
func (c *AliyunClient) SendVerificationCode(ctx context.Context, phone string, code string) error {
	if c.config.VerificationCodeTemplate == "" {
		return fmt.Errorf("verification_code_template is not configured")
	}

	templateParams := map[string]string{
		"code": code,
	}

	return c.Send(ctx, phone, c.config.VerificationCodeTemplate, templateParams)
}

// GetProvider 获取提供商类型
func (c *AliyunClient) GetProvider() Provider {
	return ProviderAliyun
}
