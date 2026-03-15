package service

import "github.com/google/wire"

// ProviderSet 服务层的依赖注入集合
var ProviderSet = wire.NewSet(
	NewConsumerService,
	NewSMSService,
	NewPaymentService,
	NewFinanceService,
	NewWechatService,
)
