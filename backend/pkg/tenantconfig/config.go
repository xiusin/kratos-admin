package tenantconfig

import (
	"context"
	"fmt"
	"sync"

	"go-wind-admin/pkg/oss"
)

// TenantConfig 租户配置
type TenantConfig struct {
	// TenantID 租户ID
	TenantID uint32

	// OSSConfig OSS配置
	OSSConfig *oss.Config

	// SMSConfig 短信配置
	SMSConfig *SMSConfig

	// PaymentConfig 支付配置
	PaymentConfig *PaymentConfig
}

// SMSConfig 短信配置
type SMSConfig struct {
	Provider        string `json:"provider"`         // aliyun/tencent
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SignName        string `json:"sign_name"`
	TemplateCode    string `json:"template_code"`
}

// PaymentConfig 支付配置
type PaymentConfig struct {
	WechatAppID     string `json:"wechat_app_id"`
	WechatMchID     string `json:"wechat_mch_id"`
	WechatAPIKey    string `json:"wechat_api_key"`
	AlipayAppID     string `json:"alipay_app_id"`
	AlipayPrivateKey string `json:"alipay_private_key"`
}

// Manager 租户配置管理器
type Manager interface {
	// GetConfig 获取租户配置
	GetConfig(ctx context.Context, tenantID uint32) (*TenantConfig, error)

	// GetOSSConfig 获取租户OSS配置
	GetOSSConfig(ctx context.Context, tenantID uint32) (*oss.Config, error)

	// SetConfig 设置租户配置
	SetConfig(ctx context.Context, config *TenantConfig) error

	// DeleteConfig 删除租户配置
	DeleteConfig(ctx context.Context, tenantID uint32) error
}

// memoryManager 内存配置管理器（用于开发和测试）
type memoryManager struct {
	mu      sync.RWMutex
	configs map[uint32]*TenantConfig
	
	// 默认配置
	defaultOSSConfig *oss.Config
}

// NewMemoryManager 创建内存配置管理器
func NewMemoryManager(defaultOSSConfig *oss.Config) Manager {
	return &memoryManager{
		configs:          make(map[uint32]*TenantConfig),
		defaultOSSConfig: defaultOSSConfig,
	}
}

// GetConfig 获取租户配置
func (m *memoryManager) GetConfig(ctx context.Context, tenantID uint32) (*TenantConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[tenantID]
	if !exists {
		// 返回默认配置
		return &TenantConfig{
			TenantID:  tenantID,
			OSSConfig: m.defaultOSSConfig,
		}, nil
	}

	return config, nil
}

// GetOSSConfig 获取租户OSS配置
func (m *memoryManager) GetOSSConfig(ctx context.Context, tenantID uint32) (*oss.Config, error) {
	config, err := m.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if config.OSSConfig == nil {
		return m.defaultOSSConfig, nil
	}

	return config.OSSConfig, nil
}

// SetConfig 设置租户配置
func (m *memoryManager) SetConfig(ctx context.Context, config *TenantConfig) error {
	if config == nil || config.TenantID == 0 {
		return fmt.Errorf("invalid config")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.configs[config.TenantID] = config
	return nil
}

// DeleteConfig 删除租户配置
func (m *memoryManager) DeleteConfig(ctx context.Context, tenantID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.configs, tenantID)
	return nil
}

// redisManager Redis配置管理器（生产环境）
// 实际项目中应该实现基于Redis的配置管理
// type redisManager struct {
//     redis *redis.Client
//     defaultOSSConfig *oss.Config
// }
//
// func NewRedisManager(redis *redis.Client, defaultOSSConfig *oss.Config) Manager {
//     return &redisManager{
//         redis: redis,
//         defaultOSSConfig: defaultOSSConfig,
//     }
// }
