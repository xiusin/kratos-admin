package server

import (
	"github.com/tx7do/kratos-transport/transport/kafka"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// NewKafkaServer 创建 Kafka 服务器（事件总线）
// TODO: 实现完整的 Kafka 事件订阅逻辑
func NewKafkaServer(
	ctx *bootstrap.Context,
) (*kafka.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Kafka == nil {
		return nil, nil
	}

	// 创建 Kafka 服务器
	// 注意：使用默认地址，实际配置需要根据 bootstrap 配置结构调整
	srv := kafka.NewServer(
		kafka.WithCodec(cfg.Server.Kafka.Codec),
		kafka.WithGlobalTracerProvider(),
		kafka.WithGlobalPropagator(),
	)

	// TODO: 注册事件订阅处理器
	// registerEventHandlers(srv, ctx)

	return srv, nil
}
