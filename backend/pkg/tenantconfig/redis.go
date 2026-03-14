package tenantconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go-wind-admin/pkg/oss"
)

// redisManager Redis配置管理器
type redisManager struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisConfig Redis配置
type RedisConfig struct {
	// Addr Redis地址
	Addr string

	// Password Redis密码
	Password string

	// DB Redis数据库
	DB int

	// TTL 缓存过期时间
	TTL time.Duration

	// Prefix 键前缀
	Prefix string
}

// NewRedisManager 创建Redis配置管理器
func NewRedisManager(cfg *RedisConfig) (Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}

	if cfg.TTL == 0 {
		cfg.TTL = 5 * time.Minute
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "tenant:config:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &redisManager{
		client: client,
		ttl:    cfg.TTL,
		prefix: cfg.Prefix,
	}, nil
}

// Get 获取租户配置（内部方法）
func (m *redisManager) get(ctx context.Context, tenantID int64) (*TenantConfig, error) {
	key := m.makeKey(tenantID)

	// 从Redis获取
	data, err := m.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("tenant config not found: %d", tenantID)
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// 反序列化
	var config TenantConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	return &config, nil
}

// Set 设置租户配置（内部方法）
func (m *redisManager) set(ctx context.Context, tenantID int64, config *TenantConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	key := m.makeKey(tenantID)

	// 序列化
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config failed: %w", err)
	}

	// 写入Redis
	if err := m.client.Set(ctx, key, data, m.ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Delete 删除租户配置（内部方法）
func (m *redisManager) delete(ctx context.Context, tenantID int64) error {
	key := m.makeKey(tenantID)

	if err := m.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}

	return nil
}

// List 列出所有租户配置（内部方法）
func (m *redisManager) list(ctx context.Context) ([]*TenantConfig, error) {
	// 扫描所有匹配的键
	pattern := m.prefix + "*"
	var configs []*TenantConfig

	iter := m.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		// 获取配置
		data, err := m.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var config TenantConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		configs = append(configs, &config)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("redis scan failed: %w", err)
	}

	return configs, nil
}

// Refresh 刷新租户配置（内部方法，重新加载）
func (m *redisManager) refresh(ctx context.Context, tenantID int64) error {
	// Redis管理器不需要刷新，因为数据直接从Redis读取
	// 这里只是重置TTL
	key := m.makeKey(tenantID)

	if err := m.client.Expire(ctx, key, m.ttl).Err(); err != nil {
		return fmt.Errorf("redis expire failed: %w", err)
	}

	return nil
}

// Close 关闭管理器
func (m *redisManager) Close() error {
	return m.client.Close()
}

// makeKey 生成Redis键
func (m *redisManager) makeKey(tenantID int64) string {
	return fmt.Sprintf("%s%d", m.prefix, tenantID)
}


// GetConfig 获取租户配置（实现 Manager 接口）
func (m *redisManager) GetConfig(ctx context.Context, tenantID uint32) (*TenantConfig, error) {
	return m.get(ctx, int64(tenantID))
}

// GetOSSConfig 获取租户OSS配置（实现 Manager 接口）
func (m *redisManager) GetOSSConfig(ctx context.Context, tenantID uint32) (*oss.Config, error) {
	config, err := m.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if config.OSSConfig == nil {
		return nil, fmt.Errorf("oss config not found for tenant: %d", tenantID)
	}

	return config.OSSConfig, nil
}

// SetConfig 设置租户配置（实现 Manager 接口）
func (m *redisManager) SetConfig(ctx context.Context, config *TenantConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	return m.set(ctx, int64(config.TenantID), config)
}

// DeleteConfig 删除租户配置（实现 Manager 接口）
func (m *redisManager) DeleteConfig(ctx context.Context, tenantID uint32) error {
	return m.delete(ctx, int64(tenantID))
}
