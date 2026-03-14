package sms

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

// TencentClient 腾讯云短信客户端
type TencentClient struct {
	client   *sms.Client
	config   *Config
	logger   *log.Helper
	signName string
	appID    string
}

// NewTencentClient 创建腾讯云短信客户端
func NewTencentClient(cfg *Config, logger log.Logger) (*TencentClient, error) {
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

	credential := common.NewCredential(cfg.AccessKeyID, cfg.AccessKeySecret)

	cpf := profile.NewClientProfile()
	if cfg.Timeout > 0 {
		cpf.HttpProfile.ReqTimeout = int(cfg.Timeout.Seconds())
	}
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	client, err := sms.NewClient(credential, "", cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create tencent sms client: %w", err)
	}

	l := log.NewHelper(log.With(logger, "module", "sms/tencent"))

	return &TencentClient{
		client:   client,
		config:   cfg,
		logger:   l,
		signName: cfg.SignName,
	}, nil
}

// Send 发送短信
func (c *TencentClient) Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error {
	if phone == "" {
		return fmt.Errorf("phone is required")
	}
	if templateCode == "" {
		return fmt.Errorf("template_code is required")
	}

	// 构建模板参数数组
	var templateParamSet []*string
	for _, v := range templateParams {
		templateParamSet = append(templateParamSet, common.StringPtr(v))
	}

	// 手机号需要加上国家码
	phoneNumber := phone
	if phone[0] != '+' {
		phoneNumber = "+86" + phone
	}

	request := sms.NewSendSmsRequest()
	request.PhoneNumberSet = common.StringPtrs([]string{phoneNumber})
	request.SignName = common.StringPtr(c.signName)
	request.TemplateId = common.StringPtr(templateCode)
	request.TemplateParamSet = templateParamSet

	response, err := c.client.SendSms(request)
	if err != nil {
		c.logger.Errorf("failed to send sms: %v", err)
		return fmt.Errorf("failed to send sms: %w", err)
	}

	if response.Response == nil || len(response.Response.SendStatusSet) == 0 {
		return fmt.Errorf("empty response")
	}

	// 检查发送状态
	status := response.Response.SendStatusSet[0]
	if status.Code == nil || *status.Code != "Ok" {
		code := ""
		if status.Code != nil {
			code = *status.Code
		}
		message := ""
		if status.Message != nil {
			message = *status.Message
		}
		c.logger.Errorf("sms send failed: code=%s, message=%s", code, message)
		return fmt.Errorf("sms send failed: %s", message)
	}

	requestID := ""
	if response.Response.RequestId != nil {
		requestID = *response.Response.RequestId
	}
	c.logger.Infof("sms sent successfully: phone=%s, template=%s, request_id=%s",
		phone, templateCode, requestID)

	return nil
}

// SendVerificationCode 发送验证码
func (c *TencentClient) SendVerificationCode(ctx context.Context, phone string, code string) error {
	if c.config.VerificationCodeTemplate == "" {
		return fmt.Errorf("verification_code_template is not configured")
	}

	templateParams := map[string]string{
		"code": code,
	}

	return c.Send(ctx, phone, c.config.VerificationCodeTemplate, templateParams)
}

// GetProvider 获取提供商类型
func (c *TencentClient) GetProvider() Provider {
	return ProviderTencent
}
