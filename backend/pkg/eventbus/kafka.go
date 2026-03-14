package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// KafkaEventBus Kafka事件总线实现
type KafkaEventBus struct {
	mu         sync.RWMutex
	config     *KafkaConfig
	logger     *log.Helper
	writer     *kafka.Writer
	readers    map[string]*kafka.Reader
	handlers   map[string][]Handler
	closed     bool
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	// Brokers Kafka broker地址列表
	Brokers []string

	// Topic 主题名称
	Topic string

	// GroupID 消费者组ID
	GroupID string

	// MaxRetries 最大重试次数
	MaxRetries int

	// RetryBackoff 重试间隔
	RetryBackoff time.Duration

	// DeadLetterTopic 死信队列主题
	DeadLetterTopic string
}

// NewKafkaEventBus 创建Kafka事件总线
func NewKafkaEventBus(cfg *KafkaConfig, logger log.Logger) (EventBus, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("brokers is required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	// 设置默认值
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = time.Second
	}
	if cfg.DeadLetterTopic == "" {
		cfg.DeadLetterTopic = cfg.Topic + ".dlq"
	}

	l := log.NewHelper(log.With(logger, "module", "eventbus/kafka"))

	// 创建Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &KafkaEventBus{
		config:     cfg,
		logger:     l,
		writer:     writer,
		readers:    make(map[string]*kafka.Reader),
		handlers:   make(map[string][]Handler),
		cancelFunc: cancel,
	}

	// 启动消费者
	bus.wg.Add(1)
	go bus.consumeEvents(ctx)

	return bus, nil
}

// Subscribe 订阅事件
func (kb *KafkaEventBus) Subscribe(eventType string, handler Handler) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if kb.closed {
		return fmt.Errorf("event bus is closed")
	}

	kb.handlers[eventType] = append(kb.handlers[eventType], handler)
	kb.logger.Infof("subscribed to event type: %s", eventType)

	return nil
}

// SubscribeAsync 异步订阅事件
func (kb *KafkaEventBus) SubscribeAsync(eventType string, handler Handler) error {
	asyncHandler := NewAsyncHandler(handler)
	return kb.Subscribe(eventType, asyncHandler)
}

// SubscribeOnce 订阅一次性事件
func (kb *KafkaEventBus) SubscribeOnce(eventType string, handler Handler) error {
	// Kafka不支持一次性订阅，使用包装器实现
	onceHandler := &onceHandler{
		handler: handler,
		bus:     kb,
		eventType: eventType,
	}
	return kb.Subscribe(eventType, onceHandler)
}

// Unsubscribe 取消订阅
func (kb *KafkaEventBus) Unsubscribe(eventType string, handler Handler) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if handlers, exists := kb.handlers[eventType]; exists {
		for i, h := range handlers {
			if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
				kb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				kb.logger.Infof("unsubscribed from event type: %s", eventType)
				return nil
			}
		}
	}

	return fmt.Errorf("handler not found for event type: %s", eventType)
}

// Publish 发布事件
func (kb *KafkaEventBus) Publish(ctx context.Context, event *Event) error {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if kb.closed {
		return fmt.Errorf("event bus is closed")
	}

	// 序列化事件
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 发送到Kafka
	err = kb.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Type),
		Value: eventData,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "event_id", Value: []byte(event.ID)},
		},
	})

	if err != nil {
		kb.logger.Errorf("failed to publish event to kafka: %v", err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	kb.logger.Debugf("event published to kafka: type=%s, id=%s", event.Type, event.ID)
	return nil
}

// PublishAsync 异步发布事件
func (kb *KafkaEventBus) PublishAsync(ctx context.Context, event *Event) error {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := kb.Publish(bgCtx, event); err != nil {
			kb.logger.Errorf("async publish error for event %s: %v", event.Type, err)
		}
	}()
	return nil
}

// Close 关闭事件总线
func (kb *KafkaEventBus) Close() error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if kb.closed {
		return fmt.Errorf("event bus already closed")
	}

	kb.closed = true

	// 取消上下文
	if kb.cancelFunc != nil {
		kb.cancelFunc()
	}

	// 等待消费者退出
	kb.wg.Wait()

	// 关闭writer
	if err := kb.writer.Close(); err != nil {
		kb.logger.Errorf("failed to close kafka writer: %v", err)
	}

	// 关闭所有readers
	for _, reader := range kb.readers {
		if err := reader.Close(); err != nil {
			kb.logger.Errorf("failed to close kafka reader: %v", err)
		}
	}

	kb.handlers = make(map[string][]Handler)
	kb.logger.Info("kafka event bus closed")

	return nil
}

// consumeEvents 消费事件
func (kb *KafkaEventBus) consumeEvents(ctx context.Context) {
	defer kb.wg.Done()

	// 创建Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  kb.config.Brokers,
		Topic:    kb.config.Topic,
		GroupID:  kb.config.GroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	defer reader.Close()

	kb.logger.Info("kafka event consumer started")

	for {
		select {
		case <-ctx.Done():
			kb.logger.Info("kafka event consumer stopped")
			return
		default:
			// 读取消息
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				kb.logger.Errorf("failed to fetch message: %v", err)
				continue
			}

			// 处理消息
			if err := kb.handleMessage(ctx, msg); err != nil {
				kb.logger.Errorf("failed to handle message: %v", err)
				// 发送到死信队列
				kb.sendToDeadLetterQueue(ctx, msg, err)
			} else {
				// 提交消息
				if err := reader.CommitMessages(ctx, msg); err != nil {
					kb.logger.Errorf("failed to commit message: %v", err)
				}
			}
		}
	}
}

// handleMessage 处理消息
func (kb *KafkaEventBus) handleMessage(ctx context.Context, msg kafka.Message) error {
	// 反序列化事件
	var event Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	kb.mu.RLock()
	handlers := kb.handlers[event.Type]
	kb.mu.RUnlock()

	if len(handlers) == 0 {
		// 没有处理器，直接返回成功
		return nil
	}

	// 执行处理器（带重试）
	var lastErr error
	for i := 0; i <= kb.config.MaxRetries; i++ {
		if i > 0 {
			// 重试前等待
			time.Sleep(kb.config.RetryBackoff * time.Duration(i))
			kb.logger.Warnf("retrying event handler: type=%s, attempt=%d", event.Type, i)
		}

		// 执行所有处理器
		allSuccess := true
		for _, handler := range handlers {
			if err := handler.Handle(ctx, &event); err != nil {
				kb.logger.Errorf("handler error for event %s: %v", event.Type, err)
				lastErr = err
				allSuccess = false
			}
		}

		if allSuccess {
			return nil
		}
	}

	return fmt.Errorf("all retries failed: %w", lastErr)
}

// sendToDeadLetterQueue 发送到死信队列
func (kb *KafkaEventBus) sendToDeadLetterQueue(ctx context.Context, msg kafka.Message, err error) {
	dlqWriter := &kafka.Writer{
		Addr:     kafka.TCP(kb.config.Brokers...),
		Topic:    kb.config.DeadLetterTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer dlqWriter.Close()

	// 添加错误信息到headers
	headers := append(msg.Headers, kafka.Header{
		Key:   "error",
		Value: []byte(err.Error()),
	})

	dlqMsg := kafka.Message{
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
	}

	if err := dlqWriter.WriteMessages(ctx, dlqMsg); err != nil {
		kb.logger.Errorf("failed to send message to dead letter queue: %v", err)
	} else {
		kb.logger.Warnf("message sent to dead letter queue: topic=%s", kb.config.DeadLetterTopic)
	}
}

// onceHandler 一次性处理器包装器
type onceHandler struct {
	handler   Handler
	bus       *KafkaEventBus
	eventType string
	executed  bool
	mu        sync.Mutex
}

func (h *onceHandler) Handle(ctx context.Context, event *Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.executed {
		return nil
	}

	err := h.handler.Handle(ctx, event)
	if err == nil {
		h.executed = true
		// 取消订阅
		h.bus.Unsubscribe(h.eventType, h)
	}

	return err
}
