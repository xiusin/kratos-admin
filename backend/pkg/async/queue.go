package async

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task 异步任务
type Task struct {
	// ID 任务ID
	ID string

	// Type 任务类型
	Type string

	// Payload 任务数据
	Payload interface{}

	// CreatedAt 创建时间
	CreatedAt time.Time

	// RetryCount 重试次数
	RetryCount int

	// MaxRetries 最大重试次数
	MaxRetries int
}

// TaskHandler 任务处理器
type TaskHandler func(ctx context.Context, task *Task) error

// Queue 异步任务队列
type Queue interface {
	// Enqueue 入队任务
	Enqueue(ctx context.Context, task *Task) error

	// Start 启动队列处理
	Start(ctx context.Context) error

	// Stop 停止队列处理
	Stop() error

	// RegisterHandler 注册任务处理器
	RegisterHandler(taskType string, handler TaskHandler)

	// GetStats 获取队列统计信息
	GetStats() *QueueStats
}

// QueueStats 队列统计信息
type QueueStats struct {
	// Pending 待处理任务数
	Pending int

	// Processing 处理中任务数
	Processing int

	// Completed 已完成任务数
	Completed int

	// Failed 失败任务数
	Failed int
}

// memoryQueue 内存队列实现
type memoryQueue struct {
	tasks    chan *Task
	handlers map[string]TaskHandler
	workers  int
	wg       sync.WaitGroup
	mu       sync.RWMutex
	stats    QueueStats
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config 队列配置
type Config struct {
	// Workers 工作协程数
	Workers int

	// BufferSize 队列缓冲大小
	BufferSize int
}

// NewMemoryQueue 创建内存队列
func NewMemoryQueue(cfg *Config) Queue {
	if cfg == nil {
		cfg = &Config{
			Workers:    10,
			BufferSize: 100,
		}
	}

	if cfg.Workers <= 0 {
		cfg.Workers = 10
	}

	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}

	return &memoryQueue{
		tasks:    make(chan *Task, cfg.BufferSize),
		handlers: make(map[string]TaskHandler),
		workers:  cfg.Workers,
	}
}

// Enqueue 入队任务
func (q *memoryQueue) Enqueue(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}

	if task.Type == "" {
		return fmt.Errorf("task type is required")
	}

	// 检查是否有处理器
	q.mu.RLock()
	_, exists := q.handlers[task.Type]
	q.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for task type: %s", task.Type)
	}

	// 设置默认值
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	// 入队
	select {
	case q.tasks <- task:
		q.mu.Lock()
		q.stats.Pending++
		q.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start 启动队列处理
func (q *memoryQueue) Start(ctx context.Context) error {
	q.ctx, q.cancel = context.WithCancel(ctx)

	// 启动工作协程
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}

	return nil
}

// Stop 停止队列处理
func (q *memoryQueue) Stop() error {
	if q.cancel != nil {
		q.cancel()
	}

	// 等待所有工作协程结束
	q.wg.Wait()

	// 关闭任务通道
	close(q.tasks)

	return nil
}

// RegisterHandler 注册任务处理器
func (q *memoryQueue) RegisterHandler(taskType string, handler TaskHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.handlers[taskType] = handler
}

// GetStats 获取队列统计信息
func (q *memoryQueue) GetStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return &QueueStats{
		Pending:    q.stats.Pending,
		Processing: q.stats.Processing,
		Completed:  q.stats.Completed,
		Failed:     q.stats.Failed,
	}
}

// worker 工作协程
func (q *memoryQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case task, ok := <-q.tasks:
			if !ok {
				return
			}

			q.processTask(task)

		case <-q.ctx.Done():
			return
		}
	}
}

// processTask 处理任务
func (q *memoryQueue) processTask(task *Task) {
	// 更新统计
	q.mu.Lock()
	q.stats.Pending--
	q.stats.Processing++
	q.mu.Unlock()

	// 获取处理器
	q.mu.RLock()
	handler, exists := q.handlers[task.Type]
	q.mu.RUnlock()

	if !exists {
		q.mu.Lock()
		q.stats.Processing--
		q.stats.Failed++
		q.mu.Unlock()
		return
	}

	// 执行任务
	err := handler(q.ctx, task)

	// 更新统计
	q.mu.Lock()
	q.stats.Processing--
	if err != nil {
		// 重试逻辑
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			q.stats.Pending++
			q.mu.Unlock()

			// 重新入队
			select {
			case q.tasks <- task:
			case <-q.ctx.Done():
			}
		} else {
			q.stats.Failed++
			q.mu.Unlock()
		}
	} else {
		q.stats.Completed++
		q.mu.Unlock()
	}
}
