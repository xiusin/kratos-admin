package tenantconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-wind-admin/pkg/oss"
)

func TestMemoryManager_GetConfig(t *testing.T) {
	defaultOSSConfig := &oss.Config{
		Provider:   oss.ProviderAliyun,
		BucketName: "default-bucket",
	}

	mgr := NewMemoryManager(defaultOSSConfig)
	ctx := context.Background()

	// 获取不存在的租户配置（应返回默认配置）
	config, err := mgr.GetConfig(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, uint32(1), config.TenantID)
	assert.Equal(t, defaultOSSConfig, config.OSSConfig)
}

func TestMemoryManager_SetAndGetConfig(t *testing.T) {
	defaultOSSConfig := &oss.Config{
		Provider:   oss.ProviderAliyun,
		BucketName: "default-bucket",
	}

	mgr := NewMemoryManager(defaultOSSConfig)
	ctx := context.Background()

	// 设置租户配置
	tenantConfig := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider:   oss.ProviderTencent,
			BucketName: "tenant-1-bucket",
		},
	}

	err := mgr.SetConfig(ctx, tenantConfig)
	require.NoError(t, err)

	// 获取租户配置
	config, err := mgr.GetConfig(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, uint32(1), config.TenantID)
	assert.Equal(t, oss.ProviderTencent, config.OSSConfig.Provider)
	assert.Equal(t, "tenant-1-bucket", config.OSSConfig.BucketName)
}

func TestMemoryManager_GetOSSConfig(t *testing.T) {
	defaultOSSConfig := &oss.Config{
		Provider:   oss.ProviderAliyun,
		BucketName: "default-bucket",
	}

	mgr := NewMemoryManager(defaultOSSConfig)
	ctx := context.Background()

	// 获取不存在的租户OSS配置（应返回默认配置）
	ossConfig, err := mgr.GetOSSConfig(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, ossConfig)
	assert.Equal(t, defaultOSSConfig, ossConfig)

	// 设置租户配置
	tenantConfig := &TenantConfig{
		TenantID: 2,
		OSSConfig: &oss.Config{
			Provider:   oss.ProviderTencent,
			BucketName: "tenant-2-bucket",
		},
	}

	err = mgr.SetConfig(ctx, tenantConfig)
	require.NoError(t, err)

	// 获取租户OSS配置
	ossConfig, err = mgr.GetOSSConfig(ctx, 2)
	require.NoError(t, err)
	assert.NotNil(t, ossConfig)
	assert.Equal(t, oss.ProviderTencent, ossConfig.Provider)
	assert.Equal(t, "tenant-2-bucket", ossConfig.BucketName)
}

func TestMemoryManager_DeleteConfig(t *testing.T) {
	defaultOSSConfig := &oss.Config{
		Provider:   oss.ProviderAliyun,
		BucketName: "default-bucket",
	}

	mgr := NewMemoryManager(defaultOSSConfig)
	ctx := context.Background()

	// 设置租户配置
	tenantConfig := &TenantConfig{
		TenantID: 1,
		OSSConfig: &oss.Config{
			Provider:   oss.ProviderTencent,
			BucketName: "tenant-1-bucket",
		},
	}

	err := mgr.SetConfig(ctx, tenantConfig)
	require.NoError(t, err)

	// 删除租户配置
	err = mgr.DeleteConfig(ctx, 1)
	require.NoError(t, err)

	// 获取租户配置（应返回默认配置）
	config, err := mgr.GetConfig(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, defaultOSSConfig, config.OSSConfig)
}

func TestMemoryManager_SetConfig_InvalidInput(t *testing.T) {
	mgr := NewMemoryManager(nil)
	ctx := context.Background()

	// nil配置
	err := mgr.SetConfig(ctx, nil)
	assert.Error(t, err)

	// 无效的租户ID
	err = mgr.SetConfig(ctx, &TenantConfig{TenantID: 0})
	assert.Error(t, err)
}
