package tenantconfig

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestNewRedisManager(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	tests := []struct {
		name    string
		cfg     *RedisConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &RedisConfig{
				Addr: mr.Addr(),
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "invalid address",
			cfg: &RedisConfig{
				Addr: "invalid:99999",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewRedisManager(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, mgr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mgr)
				if mgr != nil {
					mgr.Close()
				}
			}
		})
	}
}

func TestRedisManager_SetAndGet(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
		TTL:  1 * time.Minute,
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 设置配置
	config := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider:  "aliyun",
			Endpoint:  "oss-cn-hangzhou.aliyuncs.com",
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
		},
	}

	err = mgr.SetConfig(ctx, config)
	assert.NoError(t, err)

	// 获取配置
	got, err := mgr.GetConfig(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, config.TenantID, got.TenantID)
	assert.Equal(t, config.OSSConfig.Provider, got.OSSConfig.Provider)
	assert.Equal(t, config.OSSConfig.Bucket, got.OSSConfig.Bucket)
}

func TestRedisManager_GetNotFound(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 获取不存在的配置
	_, err = mgr.GetConfig(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRedisManager_Delete(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 设置配置
	config := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider: "aliyun",
			Bucket:   "test-bucket",
		},
	}

	err = mgr.SetConfig(ctx, config)
	require.NoError(t, err)

	// 删除配置
	err = mgr.DeleteConfig(ctx, 1)
	assert.NoError(t, err)

	// 验证已删除
	_, err = mgr.GetConfig(ctx, 1)
	assert.Error(t, err)
}

func TestRedisManager_List(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 设置多个配置
	configs := []*TenantConfig{
		{
			TenantID: 1,
			OSSConfig: &oss.Config{
				Provider: "aliyun",
				Bucket:   "bucket1",
			},
		},
		{
			TenantID: 2,
			OSSConfig: &oss.Config{
				Provider: "tencent",
				Bucket:   "bucket2",
			},
		},
		{
			TenantID: 3,
			OSSConfig: &oss.Config{
				Provider: "aws",
				Bucket:   "bucket3",
			},
		},
	}

	for _, cfg := range configs {
		err := mgr.SetConfig(ctx, cfg)
		require.NoError(t, err)
	}

	// 列出所有配置
	list, err := mgr.(*redisManager).list(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestRedisManager_Refresh(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
		TTL:  1 * time.Second,
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 设置配置
	config := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider: "aliyun",
			Bucket:   "test-bucket",
		},
	}

	err = mgr.SetConfig(ctx, config)
	require.NoError(t, err)

	// 刷新TTL
	err = mgr.(*redisManager).refresh(ctx, 1)
	assert.NoError(t, err)

	// 验证配置仍然存在
	_, err = mgr.GetConfig(ctx, 1)
	assert.NoError(t, err)
}

func TestRedisManager_TTL(t *testing.T) {
	mr, _ := setupMiniRedis(t)
	defer mr.Close()

	mgr, err := NewRedisManager(&RedisConfig{
		Addr: mr.Addr(),
		TTL:  100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	// 设置配置
	config := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider: "aliyun",
			Bucket:   "test-bucket",
		},
	}

	err = mgr.SetConfig(ctx, config)
	require.NoError(t, err)

	// 立即获取应该成功
	_, err = mgr.GetConfig(ctx, 1)
	assert.NoError(t, err)

	// 快进时间
	mr.FastForward(200 * time.Millisecond)

	// 过期后获取应该失败
	_, err = mgr.GetConfig(ctx, 1)
	assert.Error(t, err)
}
