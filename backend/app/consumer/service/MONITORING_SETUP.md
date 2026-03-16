# 监控和性能系统配置指南

本文档说明如何配置和部署 Consumer Service 的监控和性能系统。

---

## 1. Jaeger 链路追踪配置

### 1.1 安装 Jaeger

**使用 Docker 快速启动：**

```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

**访问 Jaeger UI：**
- URL: http://localhost:16686
- 可以查看所有追踪数据和服务依赖关系

### 1.2 修改 Jaeger 地址

编辑 `internal/service/tracing_service.go`：

```go
// 修改第 35 行左右
exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
    jaeger.WithEndpoint("http://your-jaeger-host:14268/api/traces"),  // 修改为实际地址
))
```

**配置选项：**
- 开发环境：`http://localhost:14268/api/traces`
- 生产环境：`http://jaeger.your-domain.com:14268/api/traces`
- Kubernetes：`http://jaeger-collector.monitoring.svc.cluster.local:14268/api/traces`

### 1.3 从配置文件读取（推荐）

修改 `tracing_service.go` 以支持配置文件：

```go
func NewTracingService(ctx *bootstrap.Context) (*TracingService, error) {
    logger := ctx.NewLoggerHelper("consumer/service/tracing-service")
    
    // 从配置文件读取 Jaeger 地址
    cfg := ctx.GetConfig()
    jaegerEndpoint := "http://localhost:14268/api/traces"  // 默认值
    if cfg != nil && cfg.Data != nil {
        if endpoint, ok := cfg.Data["jaeger_endpoint"].(string); ok {
            jaegerEndpoint = endpoint
        }
    }
    
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(jaegerEndpoint),
    ))
    // ... 其余代码
}
```

在 `configs/config.yaml` 中添加：

```yaml
data:
  jaeger_endpoint: "http://localhost:14268/api/traces"
```

---

## 2. 告警渠道配置

### 2.1 邮件告警配置

编辑 `internal/service/alert_service.go` 的 `sendEmail` 方法：

```go
func (s *AlertService) sendEmail(ctx context.Context, alert *Alert) error {
    // 使用 SMTP 发送邮件
    from := "alerts@your-domain.com"
    to := []string{"admin@your-domain.com", "ops@your-domain.com"}
    
    // 构建邮件内容
    subject := fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
    body := fmt.Sprintf(`
告警级别: %s
告警标题: %s
告警消息: %s
告警时间: %s

详细信息:
%v
`, alert.Level, alert.Title, alert.Message, alert.Timestamp.Format("2006-01-02 15:04:05"), alert.Metadata)
    
    // 使用 SMTP 库发送
    // 推荐使用: github.com/jordan-wright/email
    // 或: gopkg.in/gomail.v2
    
    s.log.Infof("Email sent to %v: %s", to, subject)
    return nil
}
```

**推荐的邮件库：**

```bash
go get github.com/jordan-wright/email
```

```go
import "github.com/jordan-wright/email"

func (s *AlertService) sendEmail(ctx context.Context, alert *Alert) error {
    e := email.NewEmail()
    e.From = "Consumer Service <alerts@your-domain.com>"
    e.To = []string{"admin@your-domain.com"}
    e.Subject = fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
    e.Text = []byte(alert.Message)
    
    // SMTP 配置
    return e.Send("smtp.gmail.com:587", smtp.PlainAuth(
        "",
        "your-email@gmail.com",
        "your-app-password",
        "smtp.gmail.com",
    ))
}
```

### 2.2 短信告警配置

编辑 `internal/service/alert_service.go` 的 `sendSMS` 方法：

```go
func (s *AlertService) sendSMS(ctx context.Context, alert *Alert) error {
    // 复用 SMSService 发送短信
    // 需要注入 SMSService 依赖
    
    // 构建短信内容（限制 70 字符）
    content := fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
    if len(content) > 70 {
        content = content[:67] + "..."
    }
    
    // 发送到管理员手机号
    phones := []string{"13800138000", "13900139000"}
    for _, phone := range phones {
        // 调用 SMSService.SendNotification
        s.log.Infof("SMS sent to %s: %s", phone, content)
    }
    
    return nil
}
```

**集成 SMSService：**

修改 `NewAlertService` 构造函数：

```go
func NewAlertService(
    ctx *bootstrap.Context,
    smsService *SMSService,  // 添加依赖
) *AlertService {
    return &AlertService{
        log:                   ctx.NewLoggerHelper("consumer/service/alert-service"),
        smsService:            smsService,
        responseTimeThreshold: 200 * time.Millisecond,
        // ...
    }
}
```

### 2.3 钉钉告警配置

编辑 `internal/service/alert_service.go` 的 `sendDingTalk` 方法：

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

func (s *AlertService) sendDingTalk(ctx context.Context, alert *Alert) error {
    // 钉钉机器人 Webhook URL
    webhookURL := "https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN"
    
    // 构建钉钉消息
    message := map[string]interface{}{
        "msgtype": "markdown",
        "markdown": map[string]string{
            "title": alert.Title,
            "text": fmt.Sprintf(`### %s

**级别**: %s  
**时间**: %s  
**消息**: %s

---
详细信息: %v
`, alert.Title, alert.Level, alert.Timestamp.Format("2006-01-02 15:04:05"), alert.Message, alert.Metadata),
        },
    }
    
    // 发送 HTTP 请求
    body, _ := json.Marshal(message)
    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    s.log.Infof("DingTalk alert sent: %s", alert.Title)
    return nil
}
```

**获取钉钉 Webhook URL：**
1. 打开钉钉群聊
2. 点击群设置 → 智能群助手 → 添加机器人
3. 选择"自定义"机器人
4. 复制 Webhook URL

**从配置文件读取（推荐）：**

在 `configs/config.yaml` 中添加：

```yaml
alert:
  email:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    from: "alerts@your-domain.com"
    to: ["admin@your-domain.com", "ops@your-domain.com"]
    username: "your-email@gmail.com"
    password: "your-app-password"
  
  sms:
    phones: ["13800138000", "13900139000"]
  
  dingtalk:
    webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
```

---

## 3. Prometheus 配置

### 3.1 安装 Prometheus

**使用 Docker 启动：**

```bash
# 创建配置文件
cat > prometheus.yml <<EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'consumer-service'
    static_configs:
      - targets: ['host.docker.internal:8000']  # Consumer Service 地址
        labels:
          service: 'consumer-service'
          environment: 'development'
EOF

# 启动 Prometheus
docker run -d --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus:latest
```

**访问 Prometheus UI：**
- URL: http://localhost:9090
- 可以查询指标和创建告警规则

### 3.2 验证指标采集

1. 启动 Consumer Service
2. 访问 http://localhost:8000/metrics（确认返回 Prometheus 格式指标）
3. 访问 http://localhost:9090/targets（确认 consumer-service 状态为 UP）

### 3.3 配置告警规则

创建 `prometheus-rules.yml`：

```yaml
groups:
  - name: consumer_service_alerts
    interval: 30s
    rules:
      # API 响应时间告警
      - alert: HighAPIResponseTime
        expr: api_response_time_p95 > 200
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API response time is high"
          description: "P95 response time is {{ $value }}ms (threshold: 200ms)"
      
      # 缓存命中率告警
      - alert: LowCacheHitRate
        expr: cache_hit_rate < 80
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Cache hit rate is low"
          description: "Cache hit rate is {{ $value }}% (threshold: 80%)"
      
      # 服务不可用告警
      - alert: ServiceDown
        expr: up{job="consumer-service"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Consumer service is down"
          description: "Consumer service has been down for more than 1 minute"
```

更新 `prometheus.yml`：

```yaml
global:
  scrape_interval: 15s

rule_files:
  - "prometheus-rules.yml"

scrape_configs:
  - job_name: 'consumer-service'
    static_configs:
      - targets: ['host.docker.internal:8000']
```

---

## 4. Grafana 配置

### 4.1 安装 Grafana

**使用 Docker 启动：**

```bash
docker run -d --name grafana \
  -p 3000:3000 \
  grafana/grafana:latest
```

**访问 Grafana UI：**
- URL: http://localhost:3000
- 默认用户名/密码: admin/admin

### 4.2 添加 Prometheus 数据源

1. 登录 Grafana
2. 点击左侧菜单 → Configuration → Data Sources
3. 点击 "Add data source"
4. 选择 "Prometheus"
5. 配置：
   - Name: Prometheus
   - URL: http://host.docker.internal:9090
   - Access: Server (default)
6. 点击 "Save & Test"

### 4.3 导入监控仪表板

创建 `grafana-dashboard.json`：

```json
{
  "dashboard": {
    "title": "Consumer Service Monitoring",
    "panels": [
      {
        "title": "API Response Time (P95)",
        "targets": [
          {
            "expr": "api_response_time_p95",
            "legendFormat": "P95"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "cache_hit_rate",
            "legendFormat": "Hit Rate %"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Database Pool",
        "targets": [
          {
            "expr": "database_pool_active",
            "legendFormat": "Active"
          },
          {
            "expr": "database_pool_idle",
            "legendFormat": "Idle"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

**导入仪表板：**
1. 点击左侧菜单 → Dashboards → Import
2. 上传 `grafana-dashboard.json` 或粘贴内容
3. 选择 Prometheus 数据源
4. 点击 "Import"

### 4.4 推荐的仪表板面板

**性能指标：**
- API 响应时间（P50/P95/P99）
- 请求吞吐量（QPS）
- 错误率

**资源指标：**
- 数据库连接池（Active/Idle/Max）
- 缓存命中率
- 内存使用率
- CPU 使用率

**业务指标：**
- 用户注册数
- 登录成功/失败次数
- 支付订单数
- 短信发送数

---

## 5. 生产环境部署建议

### 5.1 Kubernetes 部署

**Jaeger Operator：**

```bash
kubectl create namespace observability
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.42.0/jaeger-operator.yaml -n observability
```

**Prometheus Operator：**

```bash
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
```

**Grafana Helm Chart：**

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana grafana/grafana -n monitoring
```

### 5.2 告警通知集成

**Alertmanager 配置：**

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'

receivers:
  - name: 'default'
    email_configs:
      - to: 'alerts@your-domain.com'
        from: 'prometheus@your-domain.com'
        smarthost: 'smtp.gmail.com:587'
        auth_username: 'your-email@gmail.com'
        auth_password: 'your-app-password'
    
    webhook_configs:
      - url: 'https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN'
```

### 5.3 数据持久化

**Prometheus 数据保留：**

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

storage:
  tsdb:
    retention.time: 30d  # 保留 30 天数据
    retention.size: 50GB  # 最大 50GB
```

**Grafana 数据库：**

```yaml
# grafana.ini
[database]
type = mysql
host = mysql.your-domain.com:3306
name = grafana
user = grafana
password = your-password
```

---

## 6. 验证和测试

### 6.1 验证健康检查

```bash
# 健康检查
curl http://localhost:8000/health

# 预期输出：
{
  "status": "UP",
  "services": {
    "database": {"status": "UP"},
    "redis": {"status": "UP"},
    "kafka": {"status": "UP"}
  }
}
```

### 6.2 验证指标采集

```bash
# 获取 Prometheus 指标
curl http://localhost:8000/metrics

# 预期输出：
# HELP api_response_time_p50 API response time P50 in milliseconds
# TYPE api_response_time_p50 gauge
api_response_time_p50 45.23
# ...
```

### 6.3 验证链路追踪

1. 发送几个 API 请求
2. 访问 Jaeger UI: http://localhost:16686
3. 选择 Service: consumer-service
4. 点击 "Find Traces"
5. 查看追踪详情

### 6.4 验证告警

```bash
# 模拟高响应时间（修改代码添加延迟）
# 或者模拟数据库连接失败

# 检查告警是否触发
# - 查看日志
# - 检查邮件
# - 检查钉钉消息
```

---

## 7. 故障排查

### 7.1 Jaeger 连接失败

**症状：** 日志显示 "Failed to create Jaeger exporter"

**解决方案：**
1. 检查 Jaeger 是否运行：`docker ps | grep jaeger`
2. 检查端口是否开放：`telnet localhost 14268`
3. 检查配置的 Jaeger 地址是否正确

### 7.2 Prometheus 无法抓取指标

**症状：** Prometheus Targets 页面显示 consumer-service 为 DOWN

**解决方案：**
1. 检查 Consumer Service 是否运行
2. 访问 http://localhost:8000/metrics 确认指标可访问
3. 检查 Prometheus 配置中的 targets 地址
4. 检查防火墙规则

### 7.3 告警未发送

**症状：** 触发告警条件但未收到通知

**解决方案：**
1. 检查日志中是否有 "Sending alert" 消息
2. 检查邮件/短信/钉钉配置是否正确
3. 检查网络连接（SMTP、钉钉 Webhook）
4. 检查告警级别和渠道配置

---

## 8. 性能优化建议

### 8.1 指标采集优化

- 使用采样：不是所有请求都需要追踪
- 限制追踪数据大小：避免记录过大的请求/响应体
- 批量发送：减少网络开销

### 8.2 告警优化

- 设置合理的阈值：避免告警疲劳
- 使用告警分组：相同类型的告警合并发送
- 设置静默期：避免重复告警

### 8.3 存储优化

- 定期清理旧数据
- 使用数据压缩
- 考虑使用时序数据库（InfluxDB、TimescaleDB）

---

## 9. 参考资料

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [钉钉机器人文档](https://open.dingtalk.com/document/robots/custom-robot-access)

---

**最后更新：** 2026-03-16  
**维护者：** Consumer Service Team
