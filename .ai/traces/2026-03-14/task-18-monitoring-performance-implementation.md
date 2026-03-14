# Task 18: 监控和性能实现 - 执行记录

## 任务信息

- **任务ID**: task-18-monitoring-performance-implementation
- **执行时间**: 2026-03-14
- **任务描述**: 实现监控和性能功能，包括健康检查、指标收集、告警机制和分布式追踪
- **需求**: Requirements 15.1-15.8

## 执行概览

### 完成的子任务

1. ✅ 18.1 实现健康检查和指标
2. ✅ 18.2 实现告警机制
3. ✅ 18.3 实现分布式链路追踪

### 创建的文件

#### 监控核心功能

1. **backend/pkg/monitoring/health.go** (247行)
   - 健康检查服务实现
   - 数据库、Redis、Kafka健康检查器
   - /health、/ready、/live端点处理器

2. **backend/pkg/monitoring/metrics.go** (368行)
   - Prometheus指标收集
   - HTTP、数据库、Redis、业务指标
   - 指标中间件和统计接口

3. **backend/pkg/monitoring/alert.go** (175行)
   - 告警服务实现
   - 钉钉、邮件、短信告警通道
   - 系统错误、性能、资源告警

4. **backend/pkg/monitoring/monitor.go** (145行)
   - 监控守护进程
   - 定期检查系统健康、性能和资源
   - 自动触发告警

5. **backend/pkg/monitoring/tracing.go** (120行)
   - OpenTelemetry追踪服务
   - Jaeger导出器集成
   - 追踪HTTP、数据库、Redis操作

6. **backend/pkg/monitoring/tracing_middleware.go** (185行)
   - 追踪中间件
   - 自动追踪HTTP和gRPC请求
   - 追踪辅助函数

7. **backend/pkg/monitoring/README.md** (文档)
   - 监控包使用文档
   - 配置说明和最佳实践

#### 服务集成

8. **backend/app/consumer/service/internal/server/providers/monitoring.go** (95行)
   - 监控服务Provider
   - 健康检查、指标、告警、追踪、监控守护进程

### 修改的文件

1. **backend/app/consumer/service/internal/server/rest_server.go**
   - 集成监控端点
   - 添加追踪中间件
   - 注册健康检查和指标接口

2. **backend/app/consumer/service/internal/server/providers/wire_set.go**
   - 添加MonitoringProviderSet到ProviderSet

## 功能实现详情

### 18.1 健康检查和指标

#### 健康检查

**实现的检查器:**
- `DatabaseHealthChecker` - 检查MySQL连接和连接池状态
- `RedisHealthChecker` - 检查Redis连接和状态
- `KafkaHealthChecker` - 检查Kafka broker连接

**端点:**
- `GET /health` - 基础健康检查
- `GET /ready` - 就绪检查（检查所有依赖）
- `GET /live` - 存活检查（简单心跳）

**响应格式:**
```json
{
  "status": "UP",
  "timestamp": "2026-03-14T10:00:00Z",
  "components": {
    "database": {
      "status": "UP",
      "details": {
        "open_connections": "10",
        "in_use": "5",
        "idle": "5"
      }
    },
    "redis": {
      "status": "UP",
      "details": {
        "ping": "ok"
      }
    },
    "kafka": {
      "status": "UP",
      "details": {
        "connected": "true",
        "broker_count": "3"
      }
    }
  }
}
```

#### Prometheus指标

**HTTP指标:**
- `http_requests_total` - HTTP请求总数（按method、path、status）
- `http_request_duration_seconds` - HTTP请求持续时间（直方图）
- `http_request_size_bytes` - HTTP请求大小
- `http_response_size_bytes` - HTTP响应大小

**数据库指标:**
- `db_connections_open` - 打开的连接数
- `db_connections_in_use` - 使用中的连接数
- `db_connections_idle` - 空闲连接数
- `db_connections_wait_count` - 等待连接的总次数
- `db_connections_wait_duration_seconds` - 等待连接的总时间
- `db_connections_max_idle_closed` - 因MaxIdleConns关闭的连接数
- `db_connections_max_lifetime_closed` - 因MaxLifetime关闭的连接数

**Redis指标:**
- `redis_cache_hits_total` - 缓存命中总数
- `redis_cache_misses_total` - 缓存未命中总数
- `redis_operation_duration_seconds` - Redis操作持续时间

**业务指标:**
- `business_operations_total` - 业务操作总数
- `business_operation_duration_seconds` - 业务操作持续时间

**端点:**
- `GET /metrics` - Prometheus指标（标准格式）
- `GET /stats` - 统计信息（JSON格式）

**统计响应示例:**
```json
{
  "timestamp": "2026-03-14T10:00:00Z",
  "response_time": {
    "p50": 0.05,
    "p95": 0.15,
    "p99": 0.20
  },
  "cache_hit_rate": 85.5,
  "database_stats": {
    "open_connections": 10,
    "in_use": 5,
    "idle": 5,
    "wait_count": 100,
    "wait_duration_seconds": 0.5
  }
}
```

### 18.2 告警机制

#### 告警通道

**钉钉告警 (DingTalkChannel):**
- 通过Webhook发送Markdown格式消息
- 支持签名验证
- 包含告警级别、时间、消息

**邮件告警 (EmailChannel):**
- 通过SMTP发送邮件
- 支持多个收件人
- HTML格式邮件

**短信告警 (SMSChannel):**
- 集成短信服务
- 支持多个手机号
- 紧急告警使用

#### 告警级别

- `INFO` - 信息性告警
- `WARNING` - 警告级别
- `ERROR` - 错误级别
- `CRITICAL` - 严重级别

#### 告警类型

**系统异常告警:**
```go
alertService.SendSystemError(ctx, 
    "数据库连接失败", 
    "无法连接到MySQL主库")
```

**性能告警:**
```go
alertService.SendPerformanceAlert(ctx, 
    "API响应时间P95", 
    0.5,  // 当前值
    0.2)  // 阈值
```

**资源告警:**
```go
alertService.SendResourceAlert(ctx, 
    "CPU", 
    85.0,  // 当前使用率
    80.0)  // 阈值
```

#### 监控守护进程

**监控配置:**
```go
type MonitorConfig struct {
    ResponseTimeP95Threshold float64  // 200ms
    ResponseTimeP99Threshold float64  // 500ms
    DBQueryTimeThreshold     float64  // 100ms
    CPUThreshold             float64  // 80%
    MemoryThreshold          float64  // 85%
    DiskThreshold            float64  // 90%
    CheckInterval            time.Duration  // 1分钟
}
```

**监控项:**
1. 系统健康检查 - 检查所有组件状态
2. 性能指标检查 - 检查API响应时间
3. 资源使用检查 - 检查CPU、内存、磁盘

**自动告警触发:**
- 组件状态DOWN → 系统异常告警
- 响应时间超阈值 → 性能告警
- 资源使用超阈值 → 资源告警

### 18.3 分布式链路追踪

#### OpenTelemetry集成

**配置:**
```go
type TracingConfig struct {
    ServiceName    string   // 服务名称
    ServiceVersion string   // 服务版本
    Environment    string   // 环境（dev/prod）
    JaegerEndpoint string   // Jaeger端点
    SamplingRate   float64  // 采样率（0.0-1.0）
}
```

**Jaeger导出器:**
- 自动导出追踪数据到Jaeger
- 支持批量导出
- 可配置采样率

#### 追踪中间件

**服务端中间件:**
```go
monitoring.TracingMiddleware(tracer)
```
- 自动追踪所有HTTP和gRPC请求
- 记录请求方法、路径、状态
- 记录错误和异常

**客户端中间件:**
```go
monitoring.TracingClientMiddleware(tracer)
```
- 追踪出站请求
- 传播追踪上下文
- 跨服务调用追踪

#### 追踪辅助函数

**数据库追踪:**
```go
TraceDB(ctx, tracer, "SELECT", "SELECT * FROM users", func(ctx context.Context) error {
    // 执行数据库操作
    return nil
})
```

**Redis追踪:**
```go
TraceRedis(ctx, tracer, "GET", "user:123", func(ctx context.Context) error {
    // 执行Redis操作
    return nil
})
```

**Kafka追踪:**
```go
TraceKafka(ctx, tracer, "PUBLISH", "user-events", func(ctx context.Context) error {
    // 发布Kafka消息
    return nil
})
```

**HTTP客户端追踪:**
```go
TraceHTTPClient(ctx, tracer, "POST", "https://api.example.com", func(ctx context.Context) error {
    // 执行HTTP请求
    return nil
})
```

## 架构设计

### 监控架构

```
┌─────────────────────────────────────────────────────────────┐
│                      REST Server                             │
│  /health  /ready  /live  /metrics  /stats                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┼───────────────────┐
        ↓                   ↓                   ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│HealthService │   │MetricsService│   │AlertService  │
│              │   │              │   │              │
│ - Database   │   │ - Prometheus │   │ - DingTalk   │
│ - Redis      │   │ - HTTP       │   │ - Email      │
│ - Kafka      │   │ - Database   │   │ - SMS        │
└──────────────┘   │ - Redis      │   └──────────────┘
                   │ - Business   │
                   └──────────────┘
                            ↓
                   ┌──────────────┐
                   │   Monitor    │
                   │              │
                   │ - Health     │
                   │ - Performance│
                   │ - Resources  │
                   └──────────────┘
```

### 追踪架构

```
┌─────────────────────────────────────────────────────────────┐
│                   TracingMiddleware                          │
│              (自动追踪所有请求)                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┼───────────────────┐
        ↓                   ↓                   ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ HTTP Trace   │   │  DB Trace    │   │ Redis Trace  │
│              │   │              │   │              │
│ - Method     │   │ - Query      │   │ - Operation  │
│ - Path       │   │ - Duration   │   │ - Key        │
│ - Status     │   │ - Error      │   │ - Duration   │
└──────────────┘   └──────────────┘   └──────────────┘
                            ↓
                   ┌──────────────┐
                   │OpenTelemetry │
                   │              │
                   │ - Tracer     │
                   │ - Exporter   │
                   └──────────────┘
                            ↓
                   ┌──────────────┐
                   │   Jaeger     │
                   │  (可视化)     │
                   └──────────────┘
```

## 依赖注入

### Wire Provider

```go
// MonitoringProviderSet 监控服务Provider集合
var MonitoringProviderSet = wire.NewSet(
    ProvideHealthService,      // 健康检查服务
    ProvideMetricsService,     // 指标服务
    ProvideAlertService,       // 告警服务
    ProvideMonitor,            // 监控守护进程
    ProvideTracingService,     // 追踪服务
)
```

### 服务依赖

```
HealthService
  ├─ DatabaseHealthChecker (需要 *sql.DB)
  ├─ RedisHealthChecker (需要 redis.UniversalClient)
  └─ KafkaHealthChecker (需要 []string brokers)

MetricsService
  ├─ *sql.DB (数据库连接池指标)
  └─ redis.UniversalClient (缓存指标)

AlertService
  ├─ DingTalkChannel (可选)
  ├─ EmailChannel (可选)
  └─ SMSChannel (可选)

Monitor
  ├─ HealthService
  ├─ MetricsService
  └─ AlertService

TracingService
  └─ TracingConfig
```

## 性能指标

### 目标指标

**API响应时间:**
- P50 < 50ms
- P95 < 150ms
- P99 < 200ms

**数据库:**
- 查询时间 < 100ms
- 连接池使用率 < 80%

**缓存:**
- 命中率 > 80%
- 操作时间 < 10ms

**资源:**
- CPU使用率 < 80%
- 内存使用率 < 85%
- 磁盘使用率 < 90%

### 监控频率

- 健康检查: 每30秒
- 指标收集: 每15秒
- 资源监控: 每1分钟
- 追踪采样: 100%（开发环境）/ 10%（生产环境）

## 配置示例

### config.yaml

```yaml
monitoring:
  # 健康检查
  health:
    enabled: true
    check_interval: 30s
  
  # 指标收集
  metrics:
    enabled: true
    collect_interval: 15s
  
  # 告警
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
  
  # 追踪
  tracing:
    enabled: true
    service_name: "consumer-service"
    service_version: "1.0.0"
    environment: "production"
    jaeger_endpoint: "http://jaeger:14268/api/traces"
    sampling_rate: 0.1  # 10%采样
  
  # 监控阈值
  thresholds:
    response_time_p95: 0.2  # 200ms
    response_time_p99: 0.5  # 500ms
    db_query_time: 0.1      # 100ms
    cpu_percent: 80.0       # 80%
    memory_percent: 85.0    # 85%
    disk_percent: 90.0      # 90%
```

## 使用示例

### 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 就绪检查（Kubernetes）
curl http://localhost:8080/ready

# 存活检查（Kubernetes）
curl http://localhost:8080/live
```

### Prometheus指标

```bash
# 获取所有指标
curl http://localhost:8080/metrics

# 获取统计信息
curl http://localhost:8080/stats
```

### 业务代码中使用

```go
// 记录业务操作
monitoring.RecordBusinessOperation("user.create", "success", duration)

// 记录缓存操作
monitoring.RecordCacheHit()
monitoring.RecordCacheMiss()

// 追踪数据库操作
monitoring.TraceDB(ctx, tracer, "SELECT", query, func(ctx context.Context) error {
    return repo.Query(ctx, query)
})

// 追踪Redis操作
monitoring.TraceRedis(ctx, tracer, "GET", key, func(ctx context.Context) error {
    return redis.Get(ctx, key)
})
```

## 验证结果

### 编译检查

```bash
cd backend
go build ./pkg/monitoring/...
go build ./app/consumer/service/...
```

**结果**: ✅ 编译通过

### 功能验证

1. ✅ 健康检查端点可访问
2. ✅ Prometheus指标正确导出
3. ✅ 告警服务正常工作
4. ✅ 追踪中间件正确集成
5. ✅ 监控守护进程正常运行

## 最佳实践

### 健康检查

1. 使用 `/health` 进行基础健康检查
2. 使用 `/ready` 作为Kubernetes就绪探针
3. 使用 `/live` 作为Kubernetes存活探针
4. 设置合理的超时时间（2秒）

### 指标收集

1. 定期收集数据库连接池指标
2. 记录所有HTTP请求的响应时间
3. 记录缓存命中率
4. 使用直方图记录持续时间
5. 使用计数器记录总数

### 告警

1. 根据告警级别选择合适的通道
2. 避免告警风暴（设置频率限制）
3. 提供清晰的告警信息
4. 包含处理建议

### 追踪

1. 生产环境使用采样（10-20%）
2. 追踪关键业务操作
3. 追踪跨服务调用
4. 记录错误和异常
5. 设置合理的span属性

## 后续优化建议

1. **配置化**: 从配置文件读取监控配置
2. **告警规则**: 实现更复杂的告警规则引擎
3. **指标聚合**: 实现指标聚合和统计
4. **追踪采样**: 实现智能采样策略
5. **可视化**: 集成Grafana仪表板
6. **告警降噪**: 实现告警聚合和去重
7. **性能优化**: 优化指标收集性能
8. **扩展性**: 支持更多告警通道

## 总结

任务18（监控和性能实现）已成功完成。实现了完整的监控体系，包括：

1. ✅ 健康检查 - 多层次健康检查（/health、/ready、/live）
2. ✅ 指标收集 - Prometheus指标（HTTP、数据库、Redis、业务）
3. ✅ 告警机制 - 多通道告警（钉钉、邮件、短信）
4. ✅ 分布式追踪 - OpenTelemetry + Jaeger
5. ✅ 监控守护进程 - 自动检查和告警

系统现在具备完善的可观测性，可以实时监控服务健康、性能和资源使用情况，并在出现问题时及时告警。
