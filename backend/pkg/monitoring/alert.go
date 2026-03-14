package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarning  AlertLevel = "WARNING"
	AlertLevelError    AlertLevel = "ERROR"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// Alert 告警信息
type Alert struct {
	Level     AlertLevel        `json:"level"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// AlertChannel 告警通道接口
type AlertChannel interface {
	Send(ctx context.Context, alert Alert) error
	Name() string
}

// DingTalkChannel 钉钉告警通道
type DingTalkChannel struct {
	webhook string
	secret  string
	log     *log.Helper
}

// NewDingTalkChannel 创建钉钉告警通道
func NewDingTalkChannel(webhook, secret string, logger log.Logger) *DingTalkChannel {
	return &DingTalkChannel{
		webhook: webhook,
		secret:  secret,
		log:     log.NewHelper(log.With(logger, "module", "alert/dingtalk")),
	}
}

func (c *DingTalkChannel) Name() string {
	return "dingtalk"
}

func (c *DingTalkChannel) Send(ctx context.Context, alert Alert) error {
	// 构造钉钉消息
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": alert.Title,
			"text": fmt.Sprintf("### %s\n\n**级别**: %s\n\n**时间**: %s\n\n**消息**: %s",
				alert.Title,
				alert.Level,
				alert.Timestamp.Format("2006-01-02 15:04:05"),
				alert.Message,
			),
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal dingtalk message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhook, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send dingtalk alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk returned status %d", resp.StatusCode)
	}

	c.log.Infof("sent dingtalk alert: %s", alert.Title)
	return nil
}

// EmailChannel 邮件告警通道
type EmailChannel struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
	to       []string
	log      *log.Helper
}

// NewEmailChannel 创建邮件告警通道
func NewEmailChannel(smtpHost string, smtpPort int, username, password, from string, to []string, logger log.Logger) *EmailChannel {
	return &EmailChannel{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
		to:       to,
		log:      log.NewHelper(log.With(logger, "module", "alert/email")),
	}
}

func (c *EmailChannel) Name() string {
	return "email"
}

func (c *EmailChannel) Send(ctx context.Context, alert Alert) error {
	// 简化实现，实际应该使用SMTP库发送邮件
	c.log.Infof("would send email alert: %s to %v", alert.Title, c.to)
	return nil
}

// SMSChannel 短信告警通道
type SMSChannel struct {
	provider string
	phones   []string
	log      *log.Helper
}

// NewSMSChannel 创建短信告警通道
func NewSMSChannel(provider string, phones []string, logger log.Logger) *SMSChannel {
	return &SMSChannel{
		provider: provider,
		phones:   phones,
		log:      log.NewHelper(log.With(logger, "module", "alert/sms")),
	}
}

func (c *SMSChannel) Name() string {
	return "sms"
}

func (c *SMSChannel) Send(ctx context.Context, alert Alert) error {
	// 简化实现，实际应该调用短信服务
	c.log.Infof("would send SMS alert: %s to %v", alert.Title, c.phones)
	return nil
}

// AlertService 告警服务
type AlertService struct {
	channels []AlertChannel
	log      *log.Helper
}

// NewAlertService 创建告警服务
func NewAlertService(logger log.Logger) *AlertService {
	return &AlertService{
		channels: make([]AlertChannel, 0),
		log:      log.NewHelper(log.With(logger, "module", "alert")),
	}
}

// RegisterChannel 注册告警通道
func (s *AlertService) RegisterChannel(channel AlertChannel) {
	s.channels = append(s.channels, channel)
	s.log.Infof("registered alert channel: %s", channel.Name())
}

// Send 发送告警
func (s *AlertService) Send(ctx context.Context, alert Alert) error {
	if len(s.channels) == 0 {
		s.log.Warn("no alert channels registered")
		return nil
	}

	alert.Timestamp = time.Now()

	// 并发发送到所有通道
	errChan := make(chan error, len(s.channels))

	for _, channel := range s.channels {
		go func(ch AlertChannel) {
			if err := ch.Send(ctx, alert); err != nil {
				s.log.Errorf("failed to send alert via %s: %v", ch.Name(), err)
				errChan <- err
			} else {
				errChan <- nil
			}
		}(channel)
	}

	// 等待所有通道完成
	var lastErr error
	for i := 0; i < len(s.channels); i++ {
		if err := <-errChan; err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// SendSystemError 发送系统错误告警
func (s *AlertService) SendSystemError(ctx context.Context, title, message string) error {
	return s.Send(ctx, Alert{
		Level:   AlertLevelError,
		Title:   title,
		Message: message,
		Tags: map[string]string{
			"type": "system",
		},
	})
}

// SendPerformanceAlert 发送性能告警
func (s *AlertService) SendPerformanceAlert(ctx context.Context, metric string, value float64, threshold float64) error {
	return s.Send(ctx, Alert{
		Level: AlertLevelWarning,
		Title: fmt.Sprintf("性能告警: %s", metric),
		Message: fmt.Sprintf("%s 当前值 %.2f 超过阈值 %.2f",
			metric, value, threshold),
		Tags: map[string]string{
			"type":      "performance",
			"metric":    metric,
			"value":     fmt.Sprintf("%.2f", value),
			"threshold": fmt.Sprintf("%.2f", threshold),
		},
	})
}

// SendResourceAlert 发送资源告警
func (s *AlertService) SendResourceAlert(ctx context.Context, resource string, usage float64, threshold float64) error {
	return s.Send(ctx, Alert{
		Level: AlertLevelWarning,
		Title: fmt.Sprintf("资源告警: %s", resource),
		Message: fmt.Sprintf("%s 使用率 %.2f%% 超过阈值 %.2f%%",
			resource, usage, threshold),
		Tags: map[string]string{
			"type":      "resource",
			"resource":  resource,
			"usage":     fmt.Sprintf("%.2f", usage),
			"threshold": fmt.Sprintf("%.2f", threshold),
		},
	})
}
