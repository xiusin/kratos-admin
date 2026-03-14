package sms

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// Manager 短信管理器，支持主备通道故障转移
type Manager struct {
	primary   Client
	secondary Client
	logger    *log.Helper
}

// NewManager 创建短信管理器
// primary: 主通道客户端
// secondary: 备用通道客户端（可选）
func NewManager(primary Client, secondary Client, logger log.Logger) *Manager {
	l := log.NewHelper(log.With(logger, "module", "sms/manager"))

	return &Manager{
		primary:   primary,
		secondary: secondary,
		logger:    l,
	}
}

// Send 发送短信，支持故障转移
// 如果主通道发送失败，自动切换到备用通道
func (m *Manager) Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error {
	// 尝试主通道
	err := m.primary.Send(ctx, phone, templateCode, templateParams)
	if err == nil {
		return nil
	}

	m.logger.Warnf("primary sms channel (%s) failed: %v, switching to secondary channel",
		m.primary.GetProvider(), err)

	// 如果没有备用通道，直接返回错误
	if m.secondary == nil {
		return fmt.Errorf("primary sms channel failed and no secondary channel available: %w", err)
	}

	// 尝试备用通道
	err = m.secondary.Send(ctx, phone, templateCode, templateParams)
	if err != nil {
		m.logger.Errorf("secondary sms channel (%s) also failed: %v",
			m.secondary.GetProvider(), err)
		return fmt.Errorf("both sms channels failed: %w", err)
	}

	m.logger.Infof("sms sent successfully via secondary channel (%s)", m.secondary.GetProvider())
	return nil
}

// SendVerificationCode 发送验证码，支持故障转移
func (m *Manager) SendVerificationCode(ctx context.Context, phone string, code string) error {
	// 尝试主通道
	err := m.primary.SendVerificationCode(ctx, phone, code)
	if err == nil {
		return nil
	}

	m.logger.Warnf("primary sms channel (%s) failed: %v, switching to secondary channel",
		m.primary.GetProvider(), err)

	// 如果没有备用通道，直接返回错误
	if m.secondary == nil {
		return fmt.Errorf("primary sms channel failed and no secondary channel available: %w", err)
	}

	// 尝试备用通道
	err = m.secondary.SendVerificationCode(ctx, phone, code)
	if err != nil {
		m.logger.Errorf("secondary sms channel (%s) also failed: %v",
			m.secondary.GetProvider(), err)
		return fmt.Errorf("both sms channels failed: %w", err)
	}

	m.logger.Infof("verification code sent successfully via secondary channel (%s)", m.secondary.GetProvider())
	return nil
}

// GetPrimaryProvider 获取主通道提供商
func (m *Manager) GetPrimaryProvider() Provider {
	return m.primary.GetProvider()
}

// GetSecondaryProvider 获取备用通道提供商
func (m *Manager) GetSecondaryProvider() Provider {
	if m.secondary == nil {
		return ""
	}
	return m.secondary.GetProvider()
}
