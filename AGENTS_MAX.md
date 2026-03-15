# AI 编程执行手册 (可执行版)

**Version:** 5.0.0  
**Last Updated:** 2026-03-15  
**核心：搜索→复制→验证，不假设任何东西**

---

## 🔥 三条铁律（违反立即停止）

### 铁律1: Pagination 必须搜索
```bash
# 每次使用 pagination 前执行
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1
# 复制结果，不要自己写
# ✅ 正确: paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
# ❌ 错误: paginationV1 "go-wind-admin/api/gen/go/pagination/v1"
```

### 铁律2: 增量开发+立即验证
```bash
# 一次只做一个最小功能
# 每次都编译
cd backend/app/consumer/service && go build ./...
# 有错立即修复，不继续
```

### 铁律3: 复制参考实现
```bash
# 找参考文件
REF="backend/app/consumer/service/internal/service/payment_service.go"
# 查看导入（复制，不改）
head -30 $REF
# 查看构造函数（复制，不改）
grep -A 10 "func NewPaymentService" $REF
```

---

## 📋 开发范式（复制粘贴执行）

### 范式1: 新 Service

**执行步骤：**
```bash
# 1. 找参考
REF="backend/app/consumer/service/internal/service/payment_service.go"

# 2. 复制导入（前30行）
head -30 $REF > /tmp/imports.txt
# 打开 /tmp/imports.txt，复制到新文件

# 3. 复制构造函数签名
grep -A 10 "func NewPaymentService" $REF
# 复制模式：func NewXxxService(ctx *bootstrap.Context, ...)

# 4. 实现一个方法
# 5. 立即编译
cd backend/app/consumer/service && go build ./internal/service/
```

**必须复制的内容：**
```go
// 导入（从参考文件复制）
import (
    "context"
    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/tx7do/kratos-bootstrap/bootstrap"
    paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"  // ← 必须这个
    consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
    "go-wind-admin/app/consumer/service/internal/data"
)

// 构造函数（从参考文件复制模式）
func NewXxxService(
    ctx *bootstrap.Context,  // ← 第一个参数必须是这个
    xxxRepo data.XxxRepo,
    // ... 其他依赖
) *XxxService {
    return &XxxService{
        xxxRepo: xxxRepo,
        log: ctx.NewLoggerHelper("consumer/service/xxx-service"),  // ← 这样获取 logger
    }
}
```

### 范式2: 新 Repository

**执行步骤：**
```bash
# 1. 找参考
REF="backend/app/consumer/service/internal/data/payment_order_repo.go"

# 2. 复制接口定义
grep -A 20 "type.*Repo interface" $REF

# 3. 复制构造函数
grep -A 5 "func New.*Repo" $REF

# 4. 立即编译
cd backend/app/consumer/service && go build ./internal/data/
```

**必须复制的内容：**
```go
// 构造函数（从参考文件复制）
func NewXxxRepo(
    ctx *bootstrap.Context,
    entClient *entCrud.EntClient[*ent.Client],  // ← 参数必须这样
) data.XxxRepo {
    return &xxxRepo{
        entClient: entClient,
        log: ctx.NewLoggerHelper("consumer/data/xxx-repo"),
    }
}
```

### 范式3: Wire 集成

**执行步骤（严格按顺序）：**
```bash
# 1. 添加到 ProviderSet
# 编辑 internal/data/providers/wire_set.go
data.NewXxxRepo,

# 编辑 internal/service/providers/wire_set.go
service.NewXxxService,

# 2. 删除旧的 wire_gen.go
rm cmd/server/wire_gen.go

# 3. 查看所有构造函数签名（重要！）
echo "=== Services ==="
grep -A 5 "func NewConsumerService" internal/service/consumer_service.go
grep -A 5 "func NewSMSService" internal/service/sms_service.go
grep -A 5 "func NewPaymentService" internal/service/payment_service.go
grep -A 5 "func NewFinanceService" internal/service/finance_service.go
grep -A 5 "func NewWechatService" internal/service/wechat_service.go
grep -A 5 "func NewMediaService" internal/service/media_service.go
grep -A 5 "func NewLogisticsService" internal/service/logistics_service.go
grep -A 5 "func NewFreightService" internal/service/freight_service.go
grep -A 5 "func NewConfigService" internal/service/config_service.go

echo "=== Repositories ==="
grep -A 3 "func NewConsumerRepo" internal/data/consumer_repo.go
grep -A 3 "func NewLoginLogRepo" internal/data/login_log_repo.go
grep -A 3 "func NewSMSLogRepo" internal/data/sms_log_repo.go
grep -A 3 "func NewPaymentOrderRepo" internal/data/payment_order_repo.go
grep -A 3 "func NewFinanceAccountRepo" internal/data/finance_account_repo.go
grep -A 3 "func NewFinanceTransactionRepo" internal/data/finance_transaction_repo.go
grep -A 3 "func NewMediaFileRepo" internal/data/media_file_repo.go
grep -A 3 "func NewLogisticsTrackingRepo" internal/data/logistics_tracking_repo.go
grep -A 3 "func NewFreightTemplateRepo" internal/data/freight_template_repo.go
grep -A 3 "func NewTenantConfigRepo" internal/data/tenant_config_repo.go

echo "=== Server ==="
grep -A 20 "func NewRestServer" internal/server/rest_server.go
grep -A 5 "func NewRestMiddleware" internal/server/rest_server.go
grep -A 3 "func NewKafkaServer" internal/server/kafka_server.go

echo "=== Data ==="
grep -A 5 "func NewData" internal/data/data.go

# 4. 重新生成
cd cmd/server && go generate

# 5. 立即编译
cd ../.. && go build ./...
```

**关键签名（必须记住）：**
```go
// NewData 返回 3 个值
func NewData(ctx *bootstrap.Context) (*Data, func(), error)

// Repository 构造函数
func NewXxxRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) data.XxxRepo

// Service 构造函数
func NewXxxService(ctx *bootstrap.Context, ...) *XxxService

// NewRestMiddleware 需要 redisClient
func NewRestMiddleware(ctx *bootstrap.Context, tokenChecker auth.AccessTokenChecker, redisClient *redis.Client) *server.RestMiddleware

// NewKafkaServer 只需要 context
func NewKafkaServer(context *bootstrap.Context) (*kafka.Server, error)
```

### 范式4: 修改 RestServer

**执行步骤：**
```bash
# 1. 查看当前签名
grep -A 20 "func NewRestServer" internal/server/rest_server.go

# 2. 添加新 Service 参数（在函数签名中）
# 3. 注册到路由（在函数体中）
consumerV1.RegisterXxxServiceHTTPServer(srv, xxxService)

# 4. 重新生成 Wire
rm cmd/server/wire_gen.go
cd cmd/server && go generate

# 5. 立即编译
cd ../.. && go build ./...
```

---

## 🚨 常见错误速查（复制粘贴修复）

### 错误1: Pagination 导入错误
```bash
# 症状：undefined: paginationV1
# 修复：搜索并复制
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1
# 复制结果到你的文件
```

### 错误2: 构造函数签名错误
```bash
# 症状：not enough arguments / too many arguments
# 修复：查看参考实现
grep -A 5 "func NewPaymentService" backend/app/consumer/service/internal/service/payment_service.go
# 复制签名模式
```

### 错误3: Wire 生成错误
```bash
# 症状：no provider found for xxx
# 修复：查看所有构造函数签名（见范式3步骤3）
# 严格按照实际签名创建 wire_gen.go
```

### 错误4: Ent 字段类型错误
```bash
# 症状：cannot use xxx (type *string) as type string
# 修复：查看生成的代码
cat internal/data/ent/mediafile.go | grep -A 30 "type MediaFile struct"
# 确认字段是指针还是值
```

### 错误5: 编译错误不匹配
```bash
# 症状：错误信息与代码不符
# 修复：清理缓存
go clean -cache
go build ./...
```

---

## ✅ 执行检查清单（每次生成代码前）

```bash
# 1. Pagination 检查
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1

# 2. 找参考实现
ls backend/app/consumer/service/internal/service/

# 3. 复制导入
head -30 <参考文件>

# 4. 复制构造函数签名
grep -A 10 "func New" <参考文件>

# 5. 只实现一个最小功能

# 6. 立即编译
cd backend/app/consumer/service && go build ./...

# 7. 有错立即修复，不继续
```

---

## 📝 快速命令（复制粘贴执行）

```bash
# 搜索 pagination 导入
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1

# 查看参考实现导入
head -30 backend/app/consumer/service/internal/service/payment_service.go

# 查看构造函数签名
grep -A 5 "func NewPaymentService" backend/app/consumer/service/internal/service/payment_service.go

# 查看 Ent 生成的字段类型
cat backend/app/consumer/service/internal/data/ent/mediafile.go | grep -A 30 "type MediaFile struct"

# 查看所有 Service 构造函数
grep -A 5 "func New.*Service" backend/app/consumer/service/internal/service/*.go

# 查看所有 Repository 构造函数
grep -A 3 "func New.*Repo" backend/app/consumer/service/internal/data/*_repo.go

# 编译验证
cd backend/app/consumer/service && go build ./...

# 清理缓存（错误不匹配时）
go clean -cache && go build ./...
```

---

## 🎯 核心原则（3句话）

1. **搜索 > 假设** - 任何导入、函数调用前先搜索
2. **复制 > 创造** - 从参考实现复制，不要自己写
3. **验证 > 继续** - 每次都编译，有错立即修复

---

## 🔧 实际执行流程（新 Service 完整示例）

```bash
# 步骤1: 搜索 pagination（铁律1）
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1
# 输出: paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

# 步骤2: 找参考实现（铁律3）
REF="backend/app/consumer/service/internal/service/payment_service.go"

# 步骤3: 复制导入
head -30 $REF > /tmp/imports.txt
# 打开 /tmp/imports.txt，复制到新文件 xxx_service.go

# 步骤4: 复制构造函数签名
grep -A 10 "func NewPaymentService" $REF
# 复制模式，修改为 NewXxxService

# 步骤5: 实现一个方法（最小功能）
# 只实现 GetXxx 或 ListXxx，不要一次实现所有方法

# 步骤6: 立即编译（铁律2）
cd backend/app/consumer/service && go build ./internal/service/

# 步骤7: 有错修复
# 如果有错，查看错误信息
# 不要猜测，查看参考实现或搜索

# 步骤8: 添加到 ProviderSet
echo "service.NewXxxService," >> internal/service/providers/wire_set.go

# 步骤9: 重新生成 Wire
rm cmd/server/wire_gen.go
cd cmd/server && go generate

# 步骤10: 最终编译
cd ../.. && go build ./...
```

---

**这就是全部！简单、可执行、有效。每一步都是可以复制粘贴执行的命令。**
