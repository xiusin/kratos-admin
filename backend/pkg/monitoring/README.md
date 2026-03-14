# Monitoring Package

监控和性能包，提供健康检查、指标收集、告警和分布式追踪功能。

## 功能特性

### 1. 健康检查 (Health Check)

提供多层次的健康检查：

- `/health` - 基础健康检查，检查服务是否运行
- `/ready` - 就绪检查，检查所有依赖服务（数据库、Redis、Kafka）
- `/live` - 存活检查，简单的心跳检查

支持的健康检查器：
- `DatabaseHealthChecker` - 数据库连接检查
- `RedisHealthChecker` - Redis连接检查
- `KafkaHealthChecker` - Kafka连接检查

### 2. 指标收集 (Metrics)

基于Prometheus的指标收集：

**HTTP指标：**
- `http_requests_total` - HTTP请求总数
- `http_request_duration_seconds` - HTTP请求持续时间
- `http_request_size_bytes` - HTTP请求大小
- `http_response_size_bytes` - HTTP响应大小

**数据库指标：**
- `db_connections_open` - 打开的数据库连接数
- `db_connections_in_use` - 使用中的连接数
- `db_connections_idle` - 空闲连接数
- `db_connections_wait_count` - 等待连接的总次数
- `db_connections_wait_duration_seconds` - 等待连接的总时间

**Redis指标：**
- `redis_cache_hits_total` - 缓存命中总数
- `redis_cache_misses_total` - 缓存未命中总数
- `redis_operation_duration_seconds` - Redis操作持续时间

**业务指标：**
- `business_operations_total` - 业务操作总数
- `business_operation_duration_seconds` - 业务操作持续时间

### 3. 告警机制 (Alert)

支持多种告警通道：

- **钉钉 (DingTalk)** - 通过Webhook发送告警
- **邮件 (Email)** - 通过SMTP发送告警邮件
- **短信 (SMS)** - 通过短信服务发送告警

告警级别：
- `INFO` - 信息
- `WARNING` - 警告
- `ERROR` - 错误
- `CRITICAL` - 严重

### 4. 分布式追踪 (Tracing)

基于OpenTelemetry的分布式追踪：

- 支持Jaeger导出
- 自动追踪HTTP请求
- 自动追踪数据库查询
- 自动追踪Redis操作
- 支持跨服务调用追踪

## 使用示例

### 健康检查

```go
import "go-wind-admin/pkg/monitoring"

// 创建健康检查服务
healthService := monitoring.NewHealthService(logger)

// 注册检查器
healthService.RegisterChecker(monitoring.NewDatabaseHealthChecker(db, logger))
healthService.RegisterChecker(monitoring.NewRedisHealthChecker(redis, logger))
healthService.RegisterChecker(monitoring.NewKafkaHealthChecker([]string{"kafka:9092"}, logger))

// 注册HTTP处理器
http.HandleFunc("/health", healthService.HealthHandler())
http.HandleFunc("/ready", healthService.ReadyHandler())
http.HandleFunc("/live", healthService.LiveHandler())
```

### 指标收集

```go
import "go-wind-admin/pkg/monitoring"

// 创建指标服务
metricsService := monitoring.NewMetricsService(db, redis, logger)

// 启动指标收集
metricsService.Start(ctx)

// 注册Prometheus端点
http.Handle("/metrics", metricsService.MetricsHandler())

// 使用HTTP指标中间件
http.Handle("/api/", monitoring.HTTPMetricsMiddleware(apiHandler))

// 记录业务操作
monitoring.RecordBusinessOperation("user.create", "success", duration)

// 记录缓存操作
monitoring.RecordCacheHit()
monitoring.RecordCacheMiss()
```

### 告警

```go
import "go-wind-admin/pkg/monitoring"

// 创建告警服务
alertService := monitoring.NewAlertService(logger)

// 注册告警通道
alertService.RegisterChannel(monitoring.NewDingTalkChannel(webhook, secret, logger))
alertService.RegisterChannel(monitoring.NewEmailChannel(host, port, user, pass, from, to, logger))

// 发送系统错误告警
alertService.SendSystemError(ctx, "数据库连接失败", "无法连接到MySQL主库")

// 发送性能告警
alertService.SendPerformanceAlert(ctx, "API响应时间", 0.5, 0.2)

// 发送资源告警
alertService.SendResourceAlert(ctx, "CPU", 85.0, 80.0)
```

### 分布式追踪

```go
import "go-wind-admin/pkg/monitoring"

// 创建追踪服务
tracingService, err := monitoring.NewTracingService(monitoring.TracingConfig{
    ServiceName:    "consumer-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    JaegerEndpoint: "http://jaeger:14268/api/traces",
    SamplingRate:   1.0,
}, logger)

// 启动追踪服务
tracingService.Start(ctx)

// 追踪HTTP请求
tracingService.TraceHTTPRequest(ctx, "POST", "/api/users", func(ctx context.Context) error {
    // 处理请求
    return nil
})

// 追踪数据库查询
tracingService.TraceDBQuery(ctx, "SELECT * FROM users", func(ctx context.Context) error {
    // 执行查询
    return nil
})

// 追踪Redis操作
tracingService.TraceRedisOperation(ctx, "GET", "user:123", func(ctx context.Context) error {
    // 执行Redis操作
    return nil
})
```

## 配置

在 `config.yaml` 中添加监控配置：

```yaml
monitoring:
  # 健康检查配置
  health:
    enabled: true
    check_interval: 30s
  
  # 指标配置
  metrics:
    enabled: true
    collect_interval: 15s
  
  # 告警配置
  alert:
    enabled: true
    dingtalk:
      webhook: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
      secret: "xxx"
    email:
      smtp_host: "smtp.example.com"
      smtp_port: 587
      username: "alert@example.com"
      password: "xxx"
      from: "alert@example.com"
      to:
        - "admin@example.com"
  
  # 追踪配置
  tracing:
    enabled: true
    service_name: "consumer-service"
    service_version: "1.0.0"
    environment: "production"
    jaeger_endpoint: "http://jaeger:14268/api/traces"
    sampling_rate: 1.0
```

## 性能指标

### API响应时间

- P50 < 50ms
- P95 < 150ms
- P99 < 200ms

### 数据库连接池

- 最大连接数: 100
- 空闲连接数: 25
- 连接超时: 10s

### 缓存命中率

- 目标命中率: > 80%

## 告警规则

### 性能告警

- API响应时间P95 > 200ms
- API响应时间P99 > 500ms
- 数据库查询时间 > 100ms

### 资源告警

- CPU使用率 > 80%
- 内存使用率 > 85%
- 磁盘使用率 > 90%

### 系统告警

- 数据库连接失败
- Redis连接失败
- Kafka连接失败
- 服务启动失败

## 依赖

```go
require (
    github.com/prometheus/client_golang v1.17.0
    go.opentelemetry.io/otel v1.19.0
    go.opentelemetry.io/otel/exporters/jaeger v1.17.0
    go.opentelemetry.io/otel/sdk v1.19.0
)
```

## 最佳实践

1. **健康检查**
   - 使用 `/health` 进行基础健康检查
   - 使用 `/ready` 进行Kubernetes就绪探针
   - 使用 `/live` 进行Kubernetes存活探针

2. **指标收集**
   - 定期收集数据库连接池指标
   - 记录所有HTTP请求的响应时间
   - 记录缓存命中率

3. **告警**
   - 根据告警级别选择合适的通道
   - 避免告警风暴（设置告警频率限制）
   - 提供清晰的告警信息和处理建议

4. **追踪**
   - 在生产环境使用采样（sampling_rate < 1.0）
   - 追踪关键业务操作
   - 追踪跨服务调用

## 故障排查

### 健康检查失败

1. 检查数据库连接
2. 检查Redis连接
3. 检查Kafka连接
4. 查看服务日志

### 指标收集异常

1. 检查Prometheus配置
2. 检查指标端点是否可访问
3. 检查指标格式是否正确

### 告警发送失败

1. 检查告警通道配置
2. 检查网络连接
3. 检查告警服务日志

### 追踪数据缺失

1. 检查Jaeger配置
2. 检查采样率设置
3. 检查追踪服务是否启动
