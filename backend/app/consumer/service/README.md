# Consumer Service - C端用户管理系统

## 概述

Consumer Service 是 C端用户管理系统的核心服务，提供面向终端消费者的综合服务平台功能。

## 功能模块

### 1. 用户服务 (Consumer Service)
- 用户注册与认证（手机号、微信登录）
- 用户信息管理
- 登录日志记录
- 风险评分计算
- 账户锁定和注销

### 2. 短信服务 (SMS Service)
- 验证码发送和验证
- 通知短信发送
- 短信日志记录
- 多通道故障转移（阿里云、腾讯云）

### 3. 支付服务 (Payment Service)
- 支付订单创建和查询
- 支付回调处理
- 退款操作
- 支持微信支付、支付宝、易宝支付

### 4. 财务服务 (Finance Service)
- 账户余额管理
- 充值、提现操作
- 财务流水查询
- 余额变动审计

### 5. 微信服务 (Wechat Service)
- OAuth 登录
- 公众号集成
- 小程序集成
- 模板消息推送

### 6. 媒体服务 (Media Service)
- 图片、视频上传
- OSS 存储管理
- 媒体文件查询
- 缩略图生成

### 7. 物流服务 (Logistics Service)
- 快递查询
- 物流跟踪
- 物流状态订阅
- 物流信息缓存

### 8. 运费计算服务 (Freight Service)
- 按重量计算运费
- 按距离计算运费
- 运费模板管理
- 包邮规则判断

## 技术栈

- **框架**: Kratos v2
- **语言**: Go 1.21+
- **通信**: gRPC + HTTP/REST
- **ORM**: Ent
- **数据库**: MySQL 8.0+ (主从复制)
- **缓存**: Redis 7.0+ (集群模式)
- **消息队列**: Kafka (事件总线)

## 架构特性

- **多租户架构**: 支持租户级数据隔离和配置
- **事件驱动设计**: 通过 Kafka 实现模块间解耦
- **第三方服务集成**: 微信、支付宝、阿里云、腾讯云等
- **完善的安全机制**: JWT认证、限流、风险评分

## 目录结构

```
consumer/service/
├── cmd/
│   └── server/
│       ├── main.go           # 服务启动入口
│       └── wire.go           # Wire 依赖注入配置
├── configs/
│   └── config.yaml           # 服务配置文件
├── internal/
│   ├── data/                 # 数据层（Repository）
│   │   ├── data.go           # 数据层初始化
│   │   ├── ent/              # Ent Schema 定义
│   │   └── providers/        # Wire Providers
│   ├── server/               # 服务器层（HTTP/Kafka）
│   │   ├── rest_server.go    # REST 服务器
│   │   ├── kafka_server.go   # Kafka 事件总线
│   │   └── providers/        # Wire Providers
│   └── service/              # 服务层（业务逻辑）
│       └── providers/        # Wire Providers
├── Makefile                  # 构建脚本
└── README.md                 # 本文件
```

## 快速开始

### 1. 安装依赖

```bash
cd backend
go mod download
```

### 2. 配置服务

编辑 `configs/config.yaml` 文件，配置数据库、Redis、Kafka 等连接信息。

### 3. 生成 Wire 代码

```bash
cd app/consumer/service
go generate ./...
```

### 4. 运行服务

```bash
# 开发模式
make run

# 或直接运行
go run cmd/server/main.go
```

### 5. 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready
```

## API 文档

服务启动后，访问 Swagger UI 查看 API 文档：

```
http://localhost:8080/swagger/
```

## 开发指南

### 添加新的服务模块

1. 在 `api/protos/consumer/service/v1/` 定义 Protobuf
2. 运行 `buf generate` 生成代码
3. 在 `internal/data/` 实现 Repository
4. 在 `internal/service/` 实现 Service
5. 在 `internal/server/rest_server.go` 注册服务
6. 更新 Wire Providers

### 数据库迁移

```bash
# 生成迁移文件
make migrate-create name=create_users_table

# 运行迁移
make migrate-up

# 回滚迁移
make migrate-down
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 查看测试覆盖率
make test-coverage
```

## 配置说明

### 数据库配置

```yaml
data:
  database:
    driver: "mysql"
    source: "root:password@tcp(mysql-master:3306)/consumer_db"
    read_source: "root:password@tcp(mysql-slave:3306)/consumer_db"
    max_idle_connections: 25
    max_open_connections: 100
```

### Redis 集群配置

```yaml
data:
  redis:
    addrs:
      - "redis-cluster-0:6379"
      - "redis-cluster-1:6379"
      - "redis-cluster-2:6379"
    password: "your-password"
    pool_size: 100
```

### Kafka 配置

```yaml
server:
  kafka:
    addrs:
      - "kafka:9092"
    codec: "json"
```

## 事件总线

### 发布事件

```go
// 发布用户注册事件
event := &UserRegisteredEvent{
    UserID: user.ID,
    Phone:  user.Phone,
}
eventbus.Publish(ctx, TopicUserEvents, event)
```

### 订阅事件

事件订阅在 `internal/server/kafka_server.go` 中配置。

## 监控和日志

### 日志

日志输出到 stdout 和文件，格式为 JSON。

### 指标

访问 `/metrics` 端点查看 Prometheus 指标。

### 链路追踪

集成 OpenTelemetry，支持分布式链路追踪。

## 部署

### Docker

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

### Kubernetes

```bash
# 部署到 K8s
kubectl apply -f deploy/k8s/
```

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查数据库配置
   - 确认数据库服务运行正常
   - 检查网络连接

2. **Redis 连接失败**
   - 检查 Redis 集群配置
   - 确认 Redis 服务运行正常
   - 检查密码是否正确

3. **Kafka 连接失败**
   - 检查 Kafka 地址配置
   - 确认 Kafka 服务运行正常
   - 检查 Topic 是否已创建

## 贡献指南

请参考项目根目录的 CONTRIBUTING.md 文件。

## 许可证

请参考项目根目录的 LICENSE 文件。
