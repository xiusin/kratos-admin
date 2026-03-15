package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/consumer/service/internal/data/ent"
)

const (
	// 配置缓存键前缀
	tenantConfigCachePrefix = "tenant:config:"
	// 配置缓存过期时间（1小时）
	tenantConfigCacheTTL = 1 * time.Hour
)

// TenantConfigCache 租户配置缓存接口
type TenantConfigCache interface {
	// Get 获取配置缓存
	Get(ctx context.Context, tenantID uint32, configKey string) (*ent.TenantConfig, error)
	// Set 设置配置缓存
	Set(ctx context.Context, config *ent.TenantConfig) error
	// Delete 删除配置缓存
	Delete(ctx context.Context, tenantID uint32, configKey string) error
	// DeleteByTenant 删除租户所有配置缓存
	DeleteByTenant(ctx context.Context, tenantID uint32) error
}

type tenantConfigCache struct {
	redis *redis.Client
	log   *log.Helper
}

// NewTenantConfigCache 创建租户配置缓存实例
func NewTenantConfigCache(ctx *bootstrap.Context, redis *redis.Client) TenantConfigCache {
	return &tenantConfigCache{
		redis: redis,
		log:   ctx.NewLoggerHelper("consumer/data/tenant-config-cache"),
	}
}

// buildCacheKey 构建缓存键
func (c *tenantConfigCache) buildCacheKey(tenantID uint32, configKey string) string {
	return fmt.Sprintf("%s%d:%s", tenantConfigCachePrefix, tenantID, configKey)
}

// Get 获取配置缓存
func (c *tenantConfigCache) Get(ctx context.Context, tenantID uint32, configKey string) (*ent.TenantConfig, error) {
	key := c.buildCacheKey(tenantID, configKey)
	c.log.Debugf("Get config cache: key=%s", key)

	data, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存不存在
		}
		return nil, err
	}

	var config ent.TenantConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Set 设置配置缓存
func (c *tenantConfigCache) Set(ctx context.Context, config *ent.TenantConfig) error {
	if config.TenantID == nil {
		return fmt.Errorf("tenant_id is required")
	}

	key := c.buildCacheKey(*config.TenantID, config.ConfigKey)
	c.log.Debugf("Set config cache: key=%s", key)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, tenantConfigCacheTTL).Err()
}

// Delete 删除配置缓存
func (c *tenantConfigCache) Delete(ctx context.Context, tenantID uint32, configKey string) error {
	key := c.buildCacheKey(tenantID, configKey)
	c.log.Debugf("Delete config cache: key=%s", key)

	return c.redis.Del(ctx, key).Err()
}

// DeleteByTenant 删除租户所有配置缓存
func (c *tenantConfigCache) DeleteByTenant(ctx context.Context, tenantID uint32) error {
	pattern := fmt.Sprintf("%s%d:*", tenantConfigCachePrefix, tenantID)
	c.log.Debugf("Delete tenant config cache: pattern=%s", pattern)

	// 使用 SCAN 命令查找所有匹配的键
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// 批量删除
	if len(keys) > 0 {
		return c.redis.Del(ctx, keys...).Err()
	}

	return nil
}
