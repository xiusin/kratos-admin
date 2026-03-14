package logistics

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// NewClient 创建物流客户端
// 目前只支持快递鸟
func NewClient(cfg *Config, logger log.Logger) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 目前只支持快递鸟
	return NewKDNiaoClient(cfg, logger)
}
