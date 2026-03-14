package oss

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// NewClient 根据配置创建OSS客户端
func NewClient(cfg *Config, logger log.Logger) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch cfg.Provider {
	case ProviderAliyun:
		return NewAliyunOSSClient(cfg, logger)
	case ProviderTencent:
		return NewTencentCOSClient(cfg, logger)
	case ProviderMinio:
		// MinIO客户端已经存在，这里可以调用现有的实现
		return nil, fmt.Errorf("minio client should use existing implementation")
	default:
		return nil, fmt.Errorf("unsupported oss provider: %s", cfg.Provider)
	}
}
