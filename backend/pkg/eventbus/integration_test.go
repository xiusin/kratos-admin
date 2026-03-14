package eventbus_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-wind-admin/pkg/eventbus"
)

// TestUserRegisteredEventFlow 测试用户注册事件流
// Requirements: 11.2, 11.6
func TestUserRegisteredEventFlow(t *testing.T) {
	// 创建事件总线
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于验证事件处理的通道
	eventReceived := make(chan bool, 1)
	var receivedEvent *eventbus.Event
	var mu sync.Mutex

	// 订阅用户注册事件
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
		eventReceived <- true
		return nil
	})

	err := bus.Subscribe(eventbus.EventUserCreated, handler)
	require.NoError(t, err, "should subscribe successfully")

	// 发布用户注册事件
	event := eventbus.NewEvent(eventbus.EventUserCreated, &eventbus.UserCreatedEvent{
		UserID:   123,
		Username: "testuser",
		Email:    "test@example.com",
	})

	ctx := context.Background()
	err = bus.Publish(ctx, event)
	require.NoError(t, err, "should publish event successfully")

	// 等待事件处理（异步）
	select {
	case <-eventReceived:
		// 事件已接收
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// 验证事件数据
	mu.Lock()
	defer mu.Unlock()
	assert.NotNil(t, receivedEvent, "event should be received")
	assert.Equal(t, eventbus.EventUserCreated, receivedEvent.Type)
}

// TestUserRegisteredEventAsyncProcessing 测试用户注册事件异步处理
// Requirements: 11.6
func TestUserRegisteredEventAsyncProcessing(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于验证异步处理的通道
	eventProcessed := make(chan bool, 1)
	processingStarted := make(chan bool, 1)

	// 订阅异步处理器
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		processingStarted <- true
		// 模拟耗时操作
		time.Sleep(100 * time.Millisecond)
		eventProcessed <- true
		return nil
	})

	err := bus.SubscribeAsync(eventbus.EventUserCreated, handler)
	require.NoError(t, err)

	// 发布事件
	event := eventbus.NewEvent(eventbus.EventUserCreated, &eventbus.UserCreatedEvent{
		UserID: 456,
	})

	ctx := context.Background()
	startTime := time.Now()
	err = bus.Publish(ctx, event)
	publishDuration := time.Since(startTime)

	require.NoError(t, err)

	// 验证发布操作是非阻塞的（应该立即返回）
	assert.Less(t, publishDuration, 50*time.Millisecond, "publish should be non-blocking")

	// 等待异步处理开始
	select {
	case <-processingStarted:
		// 处理已开始
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for processing to start")
	}

	// 等待异步处理完成
	select {
	case <-eventProcessed:
		// 处理已完成
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for processing to complete")
	}
}

// TestPaymentSuccessEventFlow 测试支付成功事件流
// Requirements: 11.3, 11.4
func TestPaymentSuccessEventFlow(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于验证事件处理的通道
	eventReceived := make(chan bool, 1)
	var receivedEvent *eventbus.Event
	var mu sync.Mutex

	// 订阅支付成功事件
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
		eventReceived <- true
		return nil
	})

	err := bus.Subscribe("payment.success", handler)
	require.NoError(t, err)

	// 发布支付成功事件
	event := eventbus.NewEvent("payment.success", map[string]interface{}{
		"order_no":       "ORDER123456",
		"consumer_id":    uint32(789),
		"amount":         "100.00",
		"payment_method": "wechat",
	})

	ctx := context.Background()
	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// 等待事件处理
	select {
	case <-eventReceived:
		// 事件已接收
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// 验证事件数据
	mu.Lock()
	defer mu.Unlock()
	assert.NotNil(t, receivedEvent)
	assert.Equal(t, "payment.success", receivedEvent.Type)
}

// TestPaymentSuccessEventRetry 测试支付成功事件重试机制
// Requirements: 11.7
func TestPaymentSuccessEventRetry(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于跟踪重试次数
	var attemptCount int32
	maxRetries := 3
	eventProcessed := make(chan bool, 1)

	// 创建会失败的处理器（前几次失败，最后一次成功）
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		currentAttempt := atomic.AddInt32(&attemptCount, 1)

		if currentAttempt < int32(maxRetries) {
			// 前几次失败
			return assert.AnError
		}

		// 最后一次成功
		eventProcessed <- true
		return nil
	})

	err := bus.Subscribe("payment.success", handler)
	require.NoError(t, err)

	// 发布事件并手动重试
	event := eventbus.NewEvent("payment.success", map[string]interface{}{
		"order_no": "ORDER_RETRY",
	})

	ctx := context.Background()

	// 模拟重试逻辑
	for i := 0; i < maxRetries; i++ {
		err = bus.Publish(ctx, event)
		require.NoError(t, err, "publish should not fail")

		if i == maxRetries-1 {
			// 最后一次应该成功
			select {
			case <-eventProcessed:
				// 成功处理
			case <-time.After(2 * time.Second):
				t.Fatal("timeout waiting for final retry")
			}
		} else {
			// 等待一小段时间再重试
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 验证重试次数
	assert.Equal(t, int32(maxRetries), atomic.LoadInt32(&attemptCount), "should retry exactly %d times", maxRetries)
}

// TestLogisticsStatusChangedEventFlow 测试物流状态变更事件流
// Requirements: 11.5, 11.6
func TestLogisticsStatusChangedEventFlow(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于验证事件处理的通道
	eventReceived := make(chan bool, 1)
	var receivedEvent *eventbus.Event
	var mu sync.Mutex

	// 订阅物流状态变更事件
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
		eventReceived <- true
		return nil
	})

	err := bus.Subscribe(eventbus.EventLogisticsStatusChanged, handler)
	require.NoError(t, err)

	// 发布物流状态变更事件
	event := eventbus.NewEvent(eventbus.EventLogisticsStatusChanged, &eventbus.LogisticsStatusChangedEvent{
		TrackingNo:     "SF1234567890",
		CourierCompany: "顺丰速运",
		OldStatus:      "in_transit",
		NewStatus:      "delivered",
		ChangedAt:      time.Now(),
	})

	ctx := context.Background()
	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// 等待事件处理
	select {
	case <-eventReceived:
		// 事件已接收
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// 验证事件数据
	mu.Lock()
	defer mu.Unlock()
	assert.NotNil(t, receivedEvent)
	assert.Equal(t, eventbus.EventLogisticsStatusChanged, receivedEvent.Type)
}

// TestEventFailureHandling 测试事件失败处理
// Requirements: 11.7, 11.8, 11.9
func TestEventFailureHandling(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于跟踪失败次数
	var failureCount int32
	maxRetries := 3
	allRetriesDone := make(chan bool, 1)

	// 创建总是失败的处理器
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		currentCount := atomic.AddInt32(&failureCount, 1)

		if currentCount >= int32(maxRetries) {
			allRetriesDone <- true
		}

		return assert.AnError // 总是返回错误
	})

	err := bus.Subscribe("test.failure", handler)
	require.NoError(t, err)

	// 发布事件并重试
	event := eventbus.NewEvent("test.failure", map[string]interface{}{
		"test": "failure_handling",
	})

	ctx := context.Background()

	// 模拟重试逻辑（最多3次）
	for i := 0; i < maxRetries; i++ {
		err = bus.Publish(ctx, event)
		// Publish 本身不应该失败，即使处理器失败
		require.NoError(t, err)

		if i == maxRetries-1 {
			// 等待所有重试完成
			select {
			case <-allRetriesDone:
				// 所有重试已完成
			case <-time.After(2 * time.Second):
				t.Fatal("timeout waiting for retries")
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 验证失败次数
	assert.Equal(t, int32(maxRetries), atomic.LoadInt32(&failureCount), "should fail exactly %d times", maxRetries)
}

// TestEventLogging 测试事件日志记录
// Requirements: 11.9
func TestEventLogging(t *testing.T) {
	bus := eventbus.NewEventBus(log.DefaultLogger)
	defer bus.Close()

	// 用于验证日志记录的通道
	eventLogged := make(chan bool, 1)

	// 订阅事件并记录日志
	handler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		// 验证事件包含必要的日志信息
		assert.NotEmpty(t, event.ID, "event should have ID")
		assert.NotEmpty(t, event.Type, "event should have type")
		assert.False(t, event.Timestamp.IsZero(), "event should have timestamp")

		eventLogged <- true
		return nil
	})

	err := bus.Subscribe("test.logging", handler)
	require.NoError(t, err)

	// 发布事件
	event := eventbus.NewEvent("test.logging", map[string]interface{}{
		"message": "test logging",
	})
	event.WithSource("test_service")
	event.WithMetadata("request_id", "req-123")

	ctx := context.Background()
	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// 等待事件处理
	select {
	case <-eventLogged:
		// 事件已记录
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event logging")
	}

	// 验证事件元数据
	assert.Equal(t, "test_service", event.Source)
	assert.Equal(t, "req-123", event.Metadata["request_id"])
}
