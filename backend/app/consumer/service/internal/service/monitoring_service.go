package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/consumer/service/internal/data/ent"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusUp   HealthStatus = "UP"
	HealthStatusDown HealthStatus = "DOWN"
)

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status   HealthStatus             `json:"status"`
	Services map[string]ServiceHealth `json:"services"`
}

// MetricsData 指标数据
type MetricsData struct {
	// API 响应时间统计
	APIResponseTime struct {
		P50 float64 `json:"p50"`
		P95 float64 `json:"p95"`
		P99 float64 `json:"p99"`
	} `json:"api_response_time"`

	// 数据库连接池
	DatabasePool struct {
		Active int `json:"active"`
		Idle   int `json:"idle"`
		Max    int `json:"max"`
	} `json:"database_pool"`

	// 缓存命中率
	CacheHitRate struct {
		Hits   int64   `json:"hits"`
		Misses int64   `json:"misses"`
		Rate   float64 `json:"rate"`
	} `json:"cache_hit_rate"`
}

// MonitoringService 监控服务
type MonitoringService struct {
	entClient   *entCrud.EntClient[*ent.Client]
	redisClient *redis.Client
	alertService *AlertService
	log         *log.Helper

	// 响应时间记录
	responseTimeMu sync.RWMutex
	responseTimes  []float64

	// 缓存统计
	cacheStatsMu sync.RWMutex
	cacheHits    int64
	cacheMisses  int64
}

// NewMonitoringService 创建监控服务实例
func NewMonitoringService(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	redisClient *redis.Client,
	alertService *AlertService,
) *MonitoringService {
	return &MonitoringService{
		entClient:     entClient,
		redisClient:   redisClient,
		alertService:  alertService,
		log:           ctx.NewLoggerHelper("consumer/service/monitoring-service"),
		responseTimes: make([]float64, 0, 1000),
	}
}

// HealthCheck 健康检查
func (s *MonitoringService) HealthCheck(ctx context.Context) *HealthCheckResponse {
	response := &HealthCheckResponse{
		Status:   HealthStatusUp,
		Services: make(map[string]ServiceHealth),
	}

	// 检查数据库连接
	dbHealth := s.checkDatabase(ctx)
	response.Services["database"] = dbHealth
	if dbHealth.Status == HealthStatusDown {
		response.Status = HealthStatusDown
	}

	// 检查 Redis 连接
	redisHealth := s.checkRedis(ctx)
	response.Services["redis"] = redisHealth
	if redisHealth.Status == HealthStatusDown {
		response.Status = HealthStatusDown
	}

	// 检查 Kafka 连接（暂时标记为 UP，实际需要集成 Kafka 客户端）
	response.Services["kafka"] = ServiceHealth{
		Status:  HealthStatusUp,
		Message: "Kafka health check not implemented",
	}

	return response
}

// checkDatabase 检查数据库连接
func (s *MonitoringService) checkDatabase(ctx context.Context) ServiceHealth {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 执行简单查询
	err := s.entClient.Client().Consumer.Query().
		Limit(1).
		Select("id").
		Scan(ctx, &[]struct{ ID uint32 }{})

	if err != nil {
		s.log.Errorf("Database health check failed: %v", err)
		// 发送告警
		s.alertService.CheckDatabaseConnection(ctx, err)
		return ServiceHealth{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("Database connection failed: %v", err),
		}
	}

	return ServiceHealth{
		Status: HealthStatusUp,
	}
}

// checkRedis 检查 Redis 连接
func (s *MonitoringService) checkRedis(ctx context.Context) ServiceHealth {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 执行 PING 命令
	err := s.redisClient.Ping(ctx).Err()
	if err != nil {
		s.log.Errorf("Redis health check failed: %v", err)
		// 发送告警
		s.alertService.CheckRedisConnection(ctx, err)
		return ServiceHealth{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("Redis connection failed: %v", err),
		}
	}

	return ServiceHealth{
		Status: HealthStatusUp,
	}
}

// RecordResponseTime 记录 API 响应时间
func (s *MonitoringService) RecordResponseTime(duration time.Duration) {
	s.responseTimeMu.Lock()
	defer s.responseTimeMu.Unlock()

	// 转换为毫秒
	ms := float64(duration.Milliseconds())

	// 添加到记录中
	s.responseTimes = append(s.responseTimes, ms)

	// 保持最近 1000 条记录
	if len(s.responseTimes) > 1000 {
		s.responseTimes = s.responseTimes[len(s.responseTimes)-1000:]
	}
}

// RecordCacheHit 记录缓存命中
func (s *MonitoringService) RecordCacheHit() {
	s.cacheStatsMu.Lock()
	defer s.cacheStatsMu.Unlock()
	s.cacheHits++
}

// RecordCacheMiss 记录缓存未命中
func (s *MonitoringService) RecordCacheMiss() {
	s.cacheStatsMu.Lock()
	defer s.cacheStatsMu.Unlock()
	s.cacheMisses++
}

// GetMetrics 获取指标数据
func (s *MonitoringService) GetMetrics(ctx context.Context) *MetricsData {
	metrics := &MetricsData{}

	// 计算响应时间百分位
	s.responseTimeMu.RLock()
	if len(s.responseTimes) > 0 {
		metrics.APIResponseTime.P50 = s.calculatePercentile(s.responseTimes, 0.50)
		metrics.APIResponseTime.P95 = s.calculatePercentile(s.responseTimes, 0.95)
		metrics.APIResponseTime.P99 = s.calculatePercentile(s.responseTimes, 0.99)
	}
	s.responseTimeMu.RUnlock()

	// 获取数据库连接池统计
	// 注意：Ent 不直接暴露连接池统计，这里使用占位值
	// 实际应该从底层的 database/sql 获取
	metrics.DatabasePool.Active = 0
	metrics.DatabasePool.Idle = 0
	metrics.DatabasePool.Max = 100

	// 计算缓存命中率
	s.cacheStatsMu.RLock()
	metrics.CacheHitRate.Hits = s.cacheHits
	metrics.CacheHitRate.Misses = s.cacheMisses
	total := s.cacheHits + s.cacheMisses
	if total > 0 {
		metrics.CacheHitRate.Rate = float64(s.cacheHits) / float64(total) * 100
	}
	s.cacheStatsMu.RUnlock()

	return metrics
}

// calculatePercentile 计算百分位数
func (s *MonitoringService) calculatePercentile(data []float64, percentile float64) float64 {
	if len(data) == 0 {
		return 0
	}

	// 复制数据并排序
	sorted := make([]float64, len(data))
	copy(sorted, data)

	// 简单的冒泡排序（对于小数据集足够）
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// 计算百分位索引
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

// GetPrometheusMetrics 获取 Prometheus 格式的指标
func (s *MonitoringService) GetPrometheusMetrics(ctx context.Context) string {
	metrics := s.GetMetrics(ctx)

	// 构建 Prometheus 格式的指标
	var result string

	// API 响应时间
	result += fmt.Sprintf("# HELP api_response_time_p50 API response time P50 in milliseconds\n")
	result += fmt.Sprintf("# TYPE api_response_time_p50 gauge\n")
	result += fmt.Sprintf("api_response_time_p50 %.2f\n", metrics.APIResponseTime.P50)

	result += fmt.Sprintf("# HELP api_response_time_p95 API response time P95 in milliseconds\n")
	result += fmt.Sprintf("# TYPE api_response_time_p95 gauge\n")
	result += fmt.Sprintf("api_response_time_p95 %.2f\n", metrics.APIResponseTime.P95)

	result += fmt.Sprintf("# HELP api_response_time_p99 API response time P99 in milliseconds\n")
	result += fmt.Sprintf("# TYPE api_response_time_p99 gauge\n")
	result += fmt.Sprintf("api_response_time_p99 %.2f\n", metrics.APIResponseTime.P99)

	// 数据库连接池
	result += fmt.Sprintf("# HELP database_pool_active Active database connections\n")
	result += fmt.Sprintf("# TYPE database_pool_active gauge\n")
	result += fmt.Sprintf("database_pool_active %d\n", metrics.DatabasePool.Active)

	result += fmt.Sprintf("# HELP database_pool_idle Idle database connections\n")
	result += fmt.Sprintf("# TYPE database_pool_idle gauge\n")
	result += fmt.Sprintf("database_pool_idle %d\n", metrics.DatabasePool.Idle)

	result += fmt.Sprintf("# HELP database_pool_max Maximum database connections\n")
	result += fmt.Sprintf("# TYPE database_pool_max gauge\n")
	result += fmt.Sprintf("database_pool_max %d\n", metrics.DatabasePool.Max)

	// 缓存命中率
	result += fmt.Sprintf("# HELP cache_hits Total cache hits\n")
	result += fmt.Sprintf("# TYPE cache_hits counter\n")
	result += fmt.Sprintf("cache_hits %d\n", metrics.CacheHitRate.Hits)

	result += fmt.Sprintf("# HELP cache_misses Total cache misses\n")
	result += fmt.Sprintf("# TYPE cache_misses counter\n")
	result += fmt.Sprintf("cache_misses %d\n", metrics.CacheHitRate.Misses)

	result += fmt.Sprintf("# HELP cache_hit_rate Cache hit rate percentage\n")
	result += fmt.Sprintf("# TYPE cache_hit_rate gauge\n")
	result += fmt.Sprintf("cache_hit_rate %.2f\n", metrics.CacheHitRate.Rate)

	return result
}
