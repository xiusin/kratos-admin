package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	dto "github.com/prometheus/client_model/go"
)

var (
	// HTTP请求总数
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP请求持续时间
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"method", "path", "status"},
	)

	// HTTP请求大小
	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// HTTP响应大小
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// 数据库连接池指标
	dbConnectionsOpen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		},
	)

	dbConnectionsInUse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	dbConnectionsWaitCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_wait_count",
			Help: "Total number of connections waited for",
		},
	)

	dbConnectionsWaitDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_wait_duration_seconds",
			Help: "Total time blocked waiting for a new connection",
		},
	)

	dbConnectionsMaxIdleClosed = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_max_idle_closed",
			Help: "Total number of connections closed due to SetMaxIdleConns",
		},
	)

	dbConnectionsMaxLifetimeClosed = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_max_lifetime_closed",
			Help: "Total number of connections closed due to SetConnMaxLifetime",
		},
	)

	// Redis缓存指标
	redisCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hits_total",
			Help: "Total number of Redis cache hits",
		},
	)

	redisCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_misses_total",
			Help: "Total number of Redis cache misses",
		},
	)

	redisOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Redis operation duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"operation"},
	)

	// 业务指标
	businessOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_operations_total",
			Help: "Total number of business operations",
		},
		[]string{"operation", "status"},
	)

	businessOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "business_operation_duration_seconds",
			Help:    "Business operation duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
		[]string{"operation"},
	)
)

func init() {
	// 注册所有指标
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		dbConnectionsOpen,
		dbConnectionsInUse,
		dbConnectionsIdle,
		dbConnectionsWaitCount,
		dbConnectionsWaitDuration,
		dbConnectionsMaxIdleClosed,
		dbConnectionsMaxLifetimeClosed,
		redisCacheHits,
		redisCacheMisses,
		redisOperationDuration,
		businessOperationsTotal,
		businessOperationDuration,
	)
}

// MetricsService 指标服务
type MetricsService struct {
	db     *sql.DB
	redis  redis.UniversalClient
	log    *log.Helper
	ticker *time.Ticker
	done   chan struct{}
}

// NewMetricsService 创建指标服务
func NewMetricsService(db *sql.DB, redis redis.UniversalClient, logger log.Logger) *MetricsService {
	return &MetricsService{
		db:     db,
		redis:  redis,
		log:    log.NewHelper(log.With(logger, "module", "metrics")),
		ticker: time.NewTicker(15 * time.Second),
		done:   make(chan struct{}),
	}
}

// Start 启动指标收集
func (s *MetricsService) Start(ctx context.Context) error {
	s.log.Info("starting metrics collection")

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.collectDatabaseMetrics()
			case <-s.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop 停止指标收集
func (s *MetricsService) Stop(ctx context.Context) error {
	s.log.Info("stopping metrics collection")
	s.ticker.Stop()
	close(s.done)
	return nil
}

// collectDatabaseMetrics 收集数据库指标
func (s *MetricsService) collectDatabaseMetrics() {
	if s.db == nil {
		return
	}

	stats := s.db.Stats()

	dbConnectionsOpen.Set(float64(stats.OpenConnections))
	dbConnectionsInUse.Set(float64(stats.InUse))
	dbConnectionsIdle.Set(float64(stats.Idle))
	dbConnectionsWaitCount.Set(float64(stats.WaitCount))
	dbConnectionsWaitDuration.Set(stats.WaitDuration.Seconds())
	dbConnectionsMaxIdleClosed.Set(float64(stats.MaxIdleClosed))
	dbConnectionsMaxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))
}

// MetricsHandler 返回Prometheus指标处理器
func (s *MetricsService) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware HTTP指标中间件
func MetricsMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startTime := time.Now()

			// 执行请求
			reply, err := handler(ctx, req)

			// 记录指标
			duration := time.Since(startTime).Seconds()

			// 从上下文中获取HTTP信息
			if operation, ok := ctx.Value("operation").(string); ok {
				status := "success"
				if err != nil {
					status = "error"
				}

				httpRequestsTotal.WithLabelValues("POST", operation, status).Inc()
				httpRequestDuration.WithLabelValues("POST", operation, status).Observe(duration)
			}

			return reply, err
		}
	}
}

// RecordCacheHit 记录缓存命中
func RecordCacheHit() {
	redisCacheHits.Inc()
}

// RecordCacheMiss 记录缓存未命中
func RecordCacheMiss() {
	redisCacheMisses.Inc()
}

// RecordRedisOperation 记录Redis操作
func RecordRedisOperation(operation string, duration time.Duration) {
	redisOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordBusinessOperation 记录业务操作
func RecordBusinessOperation(operation, status string, duration time.Duration) {
	businessOperationsTotal.WithLabelValues(operation, status).Inc()
	businessOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// GetCacheHitRate 获取缓存命中率
func GetCacheHitRate() float64 {
	hits := getCounterValue(redisCacheHits)
	misses := getCounterValue(redisCacheMisses)

	total := hits + misses
	if total == 0 {
		return 0
	}

	return hits / total * 100
}

// getCounterValue 获取Counter的当前值
func getCounterValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	return metric.GetCounter().GetValue()
}

// ResponseTimeStats 响应时间统计
type ResponseTimeStats struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// GetResponseTimeStats 获取响应时间统计（P50/P95/P99）
func GetResponseTimeStats(method, path string) ResponseTimeStats {
	// 这里需要从Prometheus查询API获取分位数
	// 简化实现，实际应该查询Prometheus
	return ResponseTimeStats{
		P50: 0.05,  // 50ms
		P95: 0.15,  // 150ms
		P99: 0.20,  // 200ms
	}
}

// StatsResponse 统计响应
type StatsResponse struct {
	Timestamp       time.Time         `json:"timestamp"`
	ResponseTime    ResponseTimeStats `json:"response_time"`
	CacheHitRate    float64           `json:"cache_hit_rate"`
	DatabaseStats   DatabaseStats     `json:"database_stats"`
	RequestsPerMin  int64             `json:"requests_per_minute"`
}

// DatabaseStats 数据库统计
type DatabaseStats struct {
	OpenConnections      int     `json:"open_connections"`
	InUse                int     `json:"in_use"`
	Idle                 int     `json:"idle"`
	WaitCount            int64   `json:"wait_count"`
	WaitDuration         float64 `json:"wait_duration_seconds"`
	MaxIdleClosed        int64   `json:"max_idle_closed"`
	MaxLifetimeClosed    int64   `json:"max_lifetime_closed"`
}

// StatsHandler 统计信息处理器
func (s *MetricsService) StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := s.db.Stats()

		response := StatsResponse{
			Timestamp: time.Now(),
			ResponseTime: ResponseTimeStats{
				P50: 0.05,
				P95: 0.15,
				P99: 0.20,
			},
			CacheHitRate: GetCacheHitRate(),
			DatabaseStats: DatabaseStats{
				OpenConnections:   stats.OpenConnections,
				InUse:             stats.InUse,
				Idle:              stats.Idle,
				WaitCount:         stats.WaitCount,
				WaitDuration:      stats.WaitDuration.Seconds(),
				MaxIdleClosed:     stats.MaxIdleClosed,
				MaxLifetimeClosed: stats.MaxLifetimeClosed,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.log.Errorf("failed to encode stats response: %v", err)
		}
	}
}

// HTTPMetricsMiddleware HTTP指标中间件（用于HTTP服务器）
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 包装ResponseWriter以捕获状态码和响应大小
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 记录请求大小
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// 执行请求
		next.ServeHTTP(rw, r)

		// 记录指标
		duration := time.Since(startTime).Seconds()
		status := strconv.Itoa(rw.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(rw.size))
	})
}

// responseWriter 包装http.ResponseWriter以捕获状态码和响应大小
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
