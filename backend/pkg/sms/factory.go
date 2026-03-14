package sms

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// NewClient 根据配置创建短信客户端
func NewClient(cfg *Config, logger log.Logger) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch cfg.Provider {
	case ProviderAliyun:
		return NewAliyunClient(cfg, logger)
	case ProviderTencent:
		return NewTencentClient(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported sms provider: %s", cfg.Provider)
	}
}

// NewManagerWithConfigs 根据配置创建短信管理器
// primaryCfg: 主通道配置
// secondaryCfg: 备用通道配置（可选）
func NewManagerWithConfigs(primaryCfg *Config, secondaryCfg *Config, logger log.Logger) (*Manager, error) {
	if primaryCfg == nil {
		return nil, fmt.Errorf("primary config is required")
	}

	primary, err := NewClient(primaryCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary sms client: %w", err)
	}

	var secondary Client
	if secondaryCfg != nil {
		secondary, err = NewClient(secondaryCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create secondary sms client: %w", err)
		}
	}

	return NewManager(primary, secondary, logger), nil
}
