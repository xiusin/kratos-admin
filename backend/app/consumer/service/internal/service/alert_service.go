package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarning  AlertLevel = "WARNING"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// AlertChannel 告警渠道
type AlertChannel string

const (
	AlertChannelEmail  AlertChannel = "EMAIL"
	AlertChannelSMS    AlertChannel = "SMS"
	AlertChannelDingTalk AlertChannel = "DINGTALK"
)

// Alert 告警信息
type Alert struct {
	Level     AlertLevel
	Title     string
	Message   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// AlertService 告警服务
type AlertService struct {
	log *log.Helper

	// 告警阈值配置
	responseTimeThreshold time.Duration // 响应时间阈值
	cpuThreshold          float64       // CPU 使用率阈值
	memoryThreshold       float64       // 内存使用率阈值
	diskThreshold         float64       // 磁盘使用率阈值
}

// NewAlertService 创建告警服务实例
func NewAlertService(ctx *bootstrap.Context) *AlertService {
	return &AlertService{
		log:                   ctx.NewLoggerHelper("consumer/service/alert-service"),
		responseTimeThreshold: 200 * time.Millisecond, // 默认 200ms
		cpuThreshold:          80.0,                    // 默认 80%
		memoryThreshold:       80.0,                    // 默认 80%
		diskThreshold:         85.0,                    // 默认 85%
	}
}

// SendAlert 发送告警
func (s *AlertService) SendAlert(ctx context.Context, alert *Alert, channels ...AlertChannel) error {
	s.log.Infof("Sending alert: level=%s, title=%s, message=%s", alert.Level, alert.Title, alert.Message)

	// 如果没有指定渠道，根据告警级别选择默认渠道
	if len(channels) == 0 {
		switch alert.Level {
		case AlertLevelInfo:
			channels = []AlertChannel{AlertChannelEmail}
		case AlertLevelWarning:
			channels = []AlertChannel{AlertChannelEmail, AlertChannelDingTalk}
		case AlertLevelCritical:
			channels = []AlertChannel{AlertChannelEmail, AlertChannelSMS, AlertChannelDingTalk}
		}
	}

	// 发送到各个渠道
	for _, channel := range channels {
		if err := s.sendToChannel(ctx, alert, channel); err != nil {
			s.log.Errorf("Failed to send alert to %s: %v", channel, err)
		}
	}

	return nil
}

// sendToChannel 发送告警到指定渠道
func (s *AlertService) sendToChannel(ctx context.Context, alert *Alert, channel AlertChannel) error {
	switch channel {
	case AlertChannelEmail:
		return s.sendEmail(ctx, alert)
	case AlertChannelSMS:
		return s.sendSMS(ctx, alert)
	case AlertChannelDingTalk:
		return s.sendDingTalk(ctx, alert)
	default:
		return fmt.Errorf("unsupported alert channel: %s", channel)
	}
}

// sendEmail 发送邮件告警
func (s *AlertService) sendEmail(ctx context.Context, alert *Alert) error {
	// TODO: 集成邮件发送服务（如 SMTP、SendGrid、阿里云邮件推送）
	s.log.Infof("Sending email alert: %s - %s", alert.Title, alert.Message)
	return nil
}

// sendSMS 发送短信告警
func (s *AlertService) sendSMS(ctx context.Context, alert *Alert) error {
	// TODO: 集成短信服务（复用 SMSService）
	s.log.Infof("Sending SMS alert: %s - %s", alert.Title, alert.Message)
	return nil
}

// sendDingTalk 发送钉钉告警
func (s *AlertService) sendDingTalk(ctx context.Context, alert *Alert) error {
	// TODO: 集成钉钉机器人 Webhook
	s.log.Infof("Sending DingTalk alert: %s - %s", alert.Title, alert.Message)
	return nil
}

// CheckResponseTime 检查响应时间并告警
func (s *AlertService) CheckResponseTime(ctx context.Context, duration time.Duration) {
	if duration > s.responseTimeThreshold {
		alert := &Alert{
			Level:     AlertLevelWarning,
			Title:     "API Response Time Exceeded",
			Message:   fmt.Sprintf("API response time %v exceeded threshold %v", duration, s.responseTimeThreshold),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"duration":  duration.Milliseconds(),
				"threshold": s.responseTimeThreshold.Milliseconds(),
			},
		}
		s.SendAlert(ctx, alert)
	}
}

// CheckSystemResources 检查系统资源并告警
func (s *AlertService) CheckSystemResources(ctx context.Context, cpu, memory, disk float64) {
	// 检查 CPU 使用率
	if cpu > s.cpuThreshold {
		alert := &Alert{
			Level:     AlertLevelCritical,
			Title:     "High CPU Usage",
			Message:   fmt.Sprintf("CPU usage %.2f%% exceeded threshold %.2f%%", cpu, s.cpuThreshold),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"cpu":       cpu,
				"threshold": s.cpuThreshold,
			},
		}
		s.SendAlert(ctx, alert)
	}

	// 检查内存使用率
	if memory > s.memoryThreshold {
		alert := &Alert{
			Level:     AlertLevelCritical,
			Title:     "High Memory Usage",
			Message:   fmt.Sprintf("Memory usage %.2f%% exceeded threshold %.2f%%", memory, s.memoryThreshold),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"memory":    memory,
				"threshold": s.memoryThreshold,
			},
		}
		s.SendAlert(ctx, alert)
	}

	// 检查磁盘使用率
	if disk > s.diskThreshold {
		alert := &Alert{
			Level:     AlertLevelWarning,
			Title:     "High Disk Usage",
			Message:   fmt.Sprintf("Disk usage %.2f%% exceeded threshold %.2f%%", disk, s.diskThreshold),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"disk":      disk,
				"threshold": s.diskThreshold,
			},
		}
		s.SendAlert(ctx, alert)
	}
}

// CheckDatabaseConnection 检查数据库连接并告警
func (s *AlertService) CheckDatabaseConnection(ctx context.Context, err error) {
	if err != nil {
		alert := &Alert{
			Level:     AlertLevelCritical,
			Title:     "Database Connection Failed",
			Message:   fmt.Sprintf("Database connection check failed: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		}
		s.SendAlert(ctx, alert)
	}
}

// CheckRedisConnection 检查 Redis 连接并告警
func (s *AlertService) CheckRedisConnection(ctx context.Context, err error) {
	if err != nil {
		alert := &Alert{
			Level:     AlertLevelCritical,
			Title:     "Redis Connection Failed",
			Message:   fmt.Sprintf("Redis connection check failed: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		}
		s.SendAlert(ctx, alert)
	}
}

// CheckKafkaConnection 检查 Kafka 连接并告警
func (s *AlertService) CheckKafkaConnection(ctx context.Context, err error) {
	if err != nil {
		alert := &Alert{
			Level:     AlertLevelCritical,
			Title:     "Kafka Connection Failed",
			Message:   fmt.Sprintf("Kafka connection check failed: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		}
		s.SendAlert(ctx, alert)
	}
}
