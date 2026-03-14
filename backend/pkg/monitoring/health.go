package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusUP   HealthStatus = "UP"
	HealthStatusDOWN HealthStatus = "DOWN"
)

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status  HealthStatus      `json:"status"`
	Details map[string]string `json:"details,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	Check(ctx context.Context) ComponentHealth
	Name() string
}

// DatabaseHealthChecker 数据库健康检查器
type DatabaseHealthChecker struct {
	db  *sql.DB
	log *log.Helper
}

// NewDatabaseHealthChecker 创建数据库健康检查器
func NewDatabaseHealthChecker(db *sql.DB, logger log.Logger) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		db:  db,
		log: log.NewHelper(log.With(logger, "module", "health/database")),
	}
}

func (c *DatabaseHealthChecker) Name() string {
	return "database"
}

func (c *DatabaseHealthChecker) Check(ctx context.Context) ComponentHealth {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return ComponentHealth{
			Status: HealthStatusDOWN,
			Error:  err.Error(),
		}
	}

	stats := c.db.Stats()
	return ComponentHealth{
		Status: HealthStatusUP,
		Details: map[string]string{
			"open_connections": fmt.Sprintf("%d", stats.OpenConnections),
			"in_use":           fmt.Sprintf("%d", stats.InUse),
			"idle":             fmt.Sprintf("%d", stats.Idle),
		},
	}
}

// RedisHealthChecker Redis健康检查器
type RedisHealthChecker struct {
	client redis.UniversalClient
	log    *log.Helper
}

// NewRedisHealthChecker 创建Redis健康检查器
func NewRedisHealthChecker(client redis.UniversalClient, logger log.Logger) *RedisHealthChecker {
	return &RedisHealthChecker{
		client: client,
		log:    log.NewHelper(log.With(logger, "module", "health/redis")),
	}
}

func (c *RedisHealthChecker) Name() string {
	return "redis"
}

func (c *RedisHealthChecker) Check(ctx context.Context) ComponentHealth {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return ComponentHealth{
			Status: HealthStatusDOWN,
			Error:  err.Error(),
		}
	}

	// 获取Redis信息
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return ComponentHealth{
			Status: HealthStatusUP,
			Details: map[string]string{
				"ping": "ok",
			},
		}
	}

	return ComponentHealth{
		Status: HealthStatusUP,
		Details: map[string]string{
			"ping": "ok",
			"info": info,
		},
	}
}

// KafkaHealthChecker Kafka健康检查器
type KafkaHealthChecker struct {
	brokers []string
	log     *log.Helper
}

// NewKafkaHealthChecker 创建Kafka健康检查器
func NewKafkaHealthChecker(brokers []string, logger log.Logger) *KafkaHealthChecker {
	return &KafkaHealthChecker{
		brokers: brokers,
		log:     log.NewHelper(log.With(logger, "module", "health/kafka")),
	}
}

func (c *KafkaHealthChecker) Name() string {
	return "kafka"
}

func (c *KafkaHealthChecker) Check(ctx context.Context) ComponentHealth {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 尝试连接到Kafka broker
	conn, err := kafka.DialContext(ctx, "tcp", c.brokers[0])
	if err != nil {
		return ComponentHealth{
			Status: HealthStatusDOWN,
			Error:  err.Error(),
		}
	}
	defer conn.Close()

	// 获取broker信息
	brokers, err := conn.Brokers()
	if err != nil {
		return ComponentHealth{
			Status: HealthStatusUP,
			Details: map[string]string{
				"connected": "true",
			},
		}
	}

	return ComponentHealth{
		Status: HealthStatusUP,
		Details: map[string]string{
			"connected":    "true",
			"broker_count": fmt.Sprintf("%d", len(brokers)),
		},
	}
}

// HealthService 健康检查服务
type HealthService struct {
	checkers []HealthChecker
	log      *log.Helper
	mu       sync.RWMutex
}

// NewHealthService 创建健康检查服务
func NewHealthService(logger log.Logger) *HealthService {
	return &HealthService{
		checkers: make([]HealthChecker, 0),
		log:      log.NewHelper(log.With(logger, "module", "health")),
	}
}

// RegisterChecker 注册健康检查器
func (s *HealthService) RegisterChecker(checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers = append(s.checkers, checker)
}

// Check 执行健康检查
func (s *HealthService) Check(ctx context.Context) HealthResponse {
	s.mu.RLock()
	checkers := s.checkers
	s.mu.RUnlock()

	response := HealthResponse{
		Status:     HealthStatusUP,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	// 并发执行所有健康检查
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()

			health := c.Check(ctx)

			mu.Lock()
			response.Components[c.Name()] = health
			if health.Status == HealthStatusDOWN {
				response.Status = HealthStatusDOWN
			}
			mu.Unlock()
		}(checker)
	}

	wg.Wait()

	return response
}

// HealthHandler HTTP健康检查处理器
func (s *HealthService) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		response := s.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		statusCode := http.StatusOK
		if response.Status == HealthStatusDOWN {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.log.Errorf("failed to encode health response: %v", err)
		}
	}
}

// ReadyHandler HTTP就绪检查处理器（检查所有依赖服务）
func (s *HealthService) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		response := s.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		// 就绪检查要求所有组件都健康
		statusCode := http.StatusOK
		if response.Status == HealthStatusDOWN {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.log.Errorf("failed to encode ready response: %v", err)
		}
	}
}

// LiveHandler HTTP存活检查处理器（简单检查服务是否运行）
func (s *HealthService) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]string{
			"status":    "UP",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.log.Errorf("failed to encode live response: %v", err)
		}
	}
}
