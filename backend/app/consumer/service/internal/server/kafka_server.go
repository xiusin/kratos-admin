package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/kafka"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

const (
	// 事件主题定义
	TopicUserEvents      = "user-events"       // 用户事件
	TopicPaymentEvents   = "payment-events"    // 支付事件
	TopicLogisticsEvents = "logistics-events"  // 物流事件
)

// NewKafkaServer 创建 Kafka 服务器（事件总线）
func NewKafkaServer(
	ctx *bootstrap.Context,
) (*kafka.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Kafka == nil {
		return nil, nil
	}

	srv := kafka.NewServer(
		kafka.WithAddress(cfg.Server.Kafka.Addrs...),
		kafka.WithCodec(cfg.Server.Kafka.Codec),
		kafka.WithGlobalTracerProvider(),
		kafka.WithGlobalPropagator(),
	)

	// 注册事件订阅处理器
	registerEventHandlers(srv, ctx)

	return srv, nil
}

// registerEventHandlers 注册事件处理器
func registerEventHandlers(srv *kafka.Server, ctx *bootstrap.Context) {
	logger := log.NewHelper(log.With(ctx.GetLogger(), "module", "kafka"))

	// 订阅用户事件
	_ = srv.RegisterSubscriber(
		ctx.Context(),
		TopicUserEvents,
		handleUserEvent,
		broker.WithQueueName("consumer-service-user-events"),
	)

	// 订阅支付事件
	_ = srv.RegisterSubscriber(
		ctx.Context(),
		TopicPaymentEvents,
		handlePaymentEvent,
		broker.WithQueueName("consumer-service-payment-events"),
	)

	// 订阅物流事件
	_ = srv.RegisterSubscriber(
		ctx.Context(),
		TopicLogisticsEvents,
		handleLogisticsEvent,
		broker.WithQueueName("consumer-service-logistics-events"),
	)

	logger.Info("Kafka event handlers registered")
}

// handleUserEvent 处理用户事件
func handleUserEvent(ctx context.Context, topic string, headers broker.Headers, msg *broker.Message) error {
	logger := log.NewHelper(log.With(log.GetLogger(), "module", "kafka", "topic", topic))
	
	logger.Infof("Received user event: %s", string(msg.Body))
	
	// TODO: 实现用户事件处理逻辑
	// 例如：UserRegisteredEvent -> 创建财务账户
	
	return nil
}

// handlePaymentEvent 处理支付事件
func handlePaymentEvent(ctx context.Context, topic string, headers broker.Headers, msg *broker.Message) error {
	logger := log.NewHelper(log.With(log.GetLogger(), "module", "kafka", "topic", topic))
	
	logger.Infof("Received payment event: %s", string(msg.Body))
	
	// TODO: 实现支付事件处理逻辑
	// 例如：PaymentSuccessEvent -> 更新账户余额
	
	return nil
}

// handleLogisticsEvent 处理物流事件
func handleLogisticsEvent(ctx context.Context, topic string, headers broker.Headers, msg *broker.Message) error {
	logger := log.NewHelper(log.With(log.GetLogger(), "module", "kafka", "topic", topic))
	
	logger.Infof("Received logistics event: %s", string(msg.Body))
	
	// TODO: 实现物流事件处理逻辑
	// 例如：LogisticsStatusChangedEvent -> 通知用户
	
	return nil
}
