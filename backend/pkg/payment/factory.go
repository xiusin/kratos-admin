package payment

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// NewClient 根据配置创建支付客户端
func NewClient(cfg *Config, logger log.Logger) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch cfg.Provider {
	case ProviderWechat:
		return NewWechatClient(cfg, logger)
	case ProviderAlipay:
		return NewAlipayClient(cfg, logger)
	case ProviderYeepay:
		return NewYeepayClient(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported payment provider: %s", cfg.Provider)
	}
}
