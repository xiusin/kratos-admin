# AI 编程执行手册

**Version:** 5.0.0  
**Last Updated:** 2026-03-15  
**核心：搜索→复制→验证，绝不假设**

---

## 🔥 开始前必读（3条铁律）

- 交互使用中文
- 对话任务称呼我为“xiusin”


### 铁律1: Pagination 必须搜索（用户强调100次！）
```bash
# 🚨 每次使用 pagination 前必须执行这个命令
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1

# 输出示例：
# paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

# ✅ 复制这个导入路径
# ❌ 绝对不要写: "go-wind-admin/api/gen/go/pagination/v1"
```

### 铁律2: 增量开发+立即验证
```bash
# 一次只实现一个方法
# 每次都立即编译
cd backend/app/consumer/service && go build ./...

# 有错误？立即修复，不要继续
# 没错误？继续下一个方法
```

### 铁律3: 复制参考实现（不要创造）
```bash
# 1. 找到相似功能的参考文件
REF="backend/app/consumer/service/internal/service/payment_service.go"

# 2. 复制导入部分（前30行）
head -30 $REF

# 3. 复制构造函数签名
grep -A 10 "func NewPaymentService" $REF

# 4. 复制错误处理模式
grep "errors\." $REF | head -5
```

---

## 📋 范式1: 实现新 Service（最常用）

### 第1步: 搜索 Pagination（铁律1）
```bash
# 🚨 必须先执行这个
grep -r "paginationV1" backend/app/*/service/internal/service/*.go | head -1
# 记住输出结果，稍后要用
```

### 第2步: 找参考实现
```bash
# 找到相似功能的 Service
ls backend/app/consumer/service/internal/service/

# 选择参考文件（例如 payment_service.go）
REF="backend/app/consumer/service/internal/service/payment_service.go"
```

### 第3步: 复制导入部分
```bash
# 查看参考文件的导入
head -30 $REF

# 复制到新文件，特别注意 pagination 导入
# ✅ 必须是: paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
```

### 第4步: 复制构造函数签名
```bash
# 查看参考文件的构造函数
grep -A 10 "func NewPaymentService" $REF

# 复制模式，注意第一个参数必须是 ctx *bootstrap.Context
```

### 第5步: 实现一个方法（最小功能）
```go
// 只实现一个方法，例如 GetXxx 或 ListXxx
func (s *XxxService) GetXxx(ctx context.Context, req *consumerV1.GetXxxRequest) (*consumerV1.Xxx, error) {
    // 1. 参数验证
    // 2. 调用 Repository
    // 3. 返回结果
}
```

### 第6步: 立即编译验证
```bash
cd backend/app/consumer/service
go build ./internal/service/

# 有错误？查看错误信息，不要猜测
# 没错误？继续实现下一个方法
```

### 完整代码模板（复制粘贴）
```go
package service

import (
    "context"
    
    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/tx7do/kratos-bootstrap/bootstrap"
    "google.golang.org/protobuf/types/known/emptypb"
    "google.golang.org/protobuf/types/known/timestamppb"
    
    paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"  // 🚨 必须这个
    consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
    "go-wind-admin/app/consumer/service/internal/data"
    "go-wind-admin/app/consumer/service/internal/data/ent"
)

// XxxService 服务
type XxxService struct {
    consumerV1.UnimplementedXxxServiceServer
    
    xxxRepo data.XxxRepo
    log     *log.Helper
}

// NewXxxService 创建服务实例
func NewXxxService(
    ctx *bootstrap.Context,  // 🚨 第一个参数必须是这个
    xxxRepo data.XxxRepo,
) *XxxService {
    return &XxxService{
        xxxRepo: xxxRepo,
        log:     ctx.NewLoggerHelper("consumer/service/xxx-service"),  // 🚨 这样获取 logger
    }
}

// GetXxx 获取单个对象
func (s *XxxService) GetXxx(ctx context.Context, req *consumerV1.GetXxxRequest) (*consumerV1.Xxx, error) {
    // 从数据库获取
    obj, err := s.xxxRepo.Get(ctx, uint32(req.GetId()))
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, errors.NotFound("NOT_FOUND", "xxx not found")
        }
        return nil, errors.InternalServer("INTERNAL_ERROR", "failed to get xxx")
    }
    
    return s.toProto(obj), nil
}

// ListXxx 获取列表
func (s *XxxService) ListXxx(ctx context.Context, req *consumerV1.ListXxxRequest) (*consumerV1.ListXxxResponse, error) {
    // 分页参数
    page := int(req.GetPaging().GetPage())
    if page <= 0 {
        page = 1
    }
    pageSize := int(req.GetPaging().GetPageSize())
    if pageSize <= 0 {
        pageSize = 10
    }
    
    // 查询列表
    list, total, err := s.xxxRepo.List(ctx, page, pageSize)
    if err != nil {
        return nil, errors.InternalServer("INTERNAL_ERROR", "failed to list xxx")
    }
    
    // 转换为 Proto
    items := make([]*consumerV1.Xxx, 0, len(list))
    for _, item := range list {
        items = append(items, s.toProto(item))
    }
    
    return &consumerV1.ListXxxResponse{
        Items: items,
        Paging: &paginationV1.PagingResponse{
            Total: uint32(total),
            Page:  uint32(page),
            PageSize: uint32(pageSize),
        },
    }, nil
}

// toProto 转换为 Proto 格式
func (s *XxxService) toProto(obj *ent.Xxx) *consumerV1.Xxx {
    result := &consumerV1.Xxx{
        Id: func() *uint64 { v := uint64(obj.ID); return &v }(),
    }
    
    // 🚨 注意：Ent 生成的字段可能是指针类型
    // 使用前必须检查 nil
    if obj.CreatedAt != nil {
        result.CreatedAt = timestamppb.New(*obj.CreatedAt)
    }
    
    return result
}
```

---

## 📋 范式2: 实现新 Repository

### 第1步: 找参考实现
```bash
REF="backend/app/consumer/service/internal/data/payment_order_repo.go"
```

### 第2步: 复制接口定义
```bash
# 查看接口定义
grep -A 20 "type.*Repo interface" $REF
```

### 第3步: 复制构造函数
```bash
# 查看构造函数签名
grep -A 5 "func New.*Repo" $REF
```

### 第4步: 实现接口方法
```go
// 实现 Get、List、Create、Update、Delete 等方法
```

### 第5步: 立即编译验证
```bash
cd backend/app/consumer/service
go build ./internal/data/
```

### 完整代码模板（复制粘贴）
```go
package data

import (
    "context"
    
    "github.com/go-kratos/kratos/v2/log"
    "github.com/tx7do/kratos-bootstrap/bootstrap"
    entCrud "github.com/tx7do/kratos-bootstrap/gen/api/go/ent/v1"
    
    "go-wind-admin/app/consumer/service/internal/data/ent"
    "go-wind-admin/app/consumer/service/internal/data/ent/xxx"
)

// XxxRepo 接口
type XxxRepo interface {
    Get(ctx context.Context, id uint32) (*ent.Xxx, error)
    List(ctx context.Context, page, pageSize int) ([]*ent.Xxx, int, error)
    Create(ctx context.Context, obj *ent.Xxx) (*ent.Xxx, error)
    Update(ctx context.Context, obj *ent.Xxx) error
    Delete(ctx context.Context, id uint32) error
}

// xxxRepo 实现
type xxxRepo struct {
    entClient *entCrud.EntClient[*ent.Client]
    log       *log.Helper
}

// NewXxxRepo 创建 Repository 实例
func NewXxxRepo(
    ctx *bootstrap.Context,
    entClient *entCrud.EntClient[*ent.Client],  // 🚨 参数必须这样
) XxxRepo {
    return &xxxRepo{
        entClient: entClient,
        log:       ctx.NewLoggerHelper("consumer/data/xxx-repo"),
    }
}

// Get 获取单个对象
func (r *xxxRepo) Get(ctx context.Context, id uint32) (*ent.Xxx, error) {
    return r.entClient.Client.Xxx.Get(ctx, id)
}

// List 获取列表
func (r *xxxRepo) List(ctx context.Context, page, pageSize int) ([]*ent.Xxx, int, error) {
    query := r.entClient.Client.Xxx.Query()
    
    // 分页
    offset := (page - 1) * pageSize
    list, err := query.
        Offset(offset).
        Limit(pageSize).
        All(ctx)
    if err != nil {
        return nil, 0, err
    }
    
    // 总数
    total, err := query.Count(ctx)
    if err != nil {
        return nil, 0, err
    }
    
    return list, total, nil
}

// Create 创建对象
func (r *xxxRepo) Create(ctx context.Context, obj *ent.Xxx) (*ent.Xxx, error) {
    return r.entClient.Client.Xxx.Create().
        // 设置字段
        Save(ctx)
}

// Update 更新对象
func (r *xxxRepo) Update(ctx context.Context, obj *ent.Xxx) error {
    return r.entClient.Client.Xxx.UpdateOneID(obj.ID).
        // 设置字段
        Exec(ctx)
}

// Delete 删除对象
func (r *xxxRepo) Delete(ctx context.Context, id uint32) error {
    return r.entClient.Client.Xxx.DeleteOneID(id).Exec(ctx)
}
```

---

## 📋 范式3: Wire 集成（最容易出错）

### 🚨 重要：Wire 生成前必须查看所有构造函数签名

### 第1步: 添加到 ProviderSet
```bash
# 编辑 internal/data/providers/wire_set.go
# 添加一行
data.NewXxxRepo,

# 编辑 internal/service/providers/wire_set.go
# 添加一行
service.NewXxxService,
```

### 第2步: 删除旧的 wire_gen.go
```bash
rm cmd/server/wire_gen.go
```

### 第3步: 查看所有构造函数签名（🚨 最重要）
```bash
# 复制粘贴执行这个脚本
cat << 'EOF' > /tmp/check_signatures.sh
#!/bin/bash
cd backend/app/consumer/service

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

echo ""
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

echo ""
echo "=== Server ==="
grep -A 20 "func NewRestServer" internal/server/rest_server.go
grep -A 5 "func NewRestMiddleware" internal/server/rest_server.go
grep -A 3 "func NewKafkaServer" internal/server/kafka_server.go

echo ""
echo "=== Data ==="
grep -A 5 "func NewData" internal/data/data.go
EOF

chmod +x /tmp/check_signatures.sh
/tmp/check_signatures.sh > /tmp/signatures.txt

# 查看输出
cat /tmp/signatures.txt
```

### 第4步: 重新生成 Wire
```bash
cd backend/app/consumer/service/cmd/server
go generate

# 如果失败，查看错误信息
# 不要猜测，根据错误信息修复
```

### 第5步: 立即编译验证
```bash
cd ../..
go build ./...
```

### 关键签名速查（必须记住）

```go
// 🚨 NewData 返回 3 个值
func NewData(ctx *bootstrap.Context) (*Data, func(), error)

// 🚨 Repository 构造函数（2个参数）
func NewXxxRepo(
    ctx *bootstrap.Context,
    entClient *entCrud.EntClient[*ent.Client],
) XxxRepo

// 🚨 Service 构造函数（第一个参数必须是 ctx）
func NewXxxService(
    ctx *bootstrap.Context,
    xxxRepo data.XxxRepo,
    // ... 其他依赖
) *XxxService

// 🚨 NewRestMiddleware 需要 redisClient
func NewRestMiddleware(
    ctx *bootstrap.Context,
    tokenChecker auth.AccessTokenChecker,
    redisClient *redis.Client,
) *server.RestMiddleware

// 🚨 NewKafkaServer 只需要 context
func NewKafkaServer(context *bootstrap.Context) (*kafka.Server, error)

// 🚨 NewRestServer 参数顺序
func NewRestServer(
    ctx *bootstrap.Context,
    middleware *RestMiddleware,
    consumerService *service.ConsumerService,
    smsService *service.SMSService,
    paymentService *service.PaymentService,
    financeService *service.FinanceService,
    wechatService *service.WechatService,
    mediaService *service.MediaService,
    logisticsService *service.LogisticsService,
    freightService *service.FreightService,
    configService *service.ConfigService,
) (*http.Server, error)
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

### 错误6: initApp 函数未定义 ⚡ 自动修复
```bash
# 症状：cmd/server/main.go:XX:XX: undefined: initApp
# 原因：wire_gen.go 文件不存在或过期

# 🤖 自动修复流程（AI必须执行）：

# 步骤1: 检查 wire_gen.go 是否存在
if [ ! -f backend/app/consumer/service/cmd/server/wire_gen.go ]; then
    echo "wire_gen.go 不存在，需要生成"
fi

# 步骤2: 自动生成 Wire 代码
cd backend/app/consumer/service/cmd/server
go generate

# 步骤3: 验证生成结果
if [ -f wire_gen.go ]; then
    echo "✅ wire_gen.go 生成成功"
    ls -lh wire_gen.go
else
    echo "❌ wire_gen.go 生成失败"
    # 检查 Wire 工具是否安装
    go install github.com/google/wire/cmd/wire@latest
    go generate
fi

# 步骤4: 重新编译验证
cd ../..
go build ./...
```

**AI执行规则**：
- 当遇到 `undefined: initApp` 错误时，AI **必须自动执行**上述步骤
- **不要**询问用户是否执行
- **不要**等待用户手动操作
- 直接运行 `go generate` 命令
- 如果失败，自动安装 Wire 工具后重试
- 生成成功后立即编译验证

**为什么会出现这个错误**：
1. 新项目首次编译，wire_gen.go 还未生成
2. 修改了 wire.go 或 ProviderSet，但未重新生成
3. 删除了 wire_gen.go 文件
4. Git 忽略了 wire_gen.go（不应该忽略）

**预防措施**：
- wire_gen.go 应该提交到 Git（不要加入 .gitignore）
- 修改 ProviderSet 后立即运行 go generate
- CI/CD 流程中包含 go generate 步骤
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


---

## 💡 实战技巧

### 技巧1: 快速定位参考实现
```bash
# 按功能类型找参考
# - 需要调用第三方API → payment_service.go
# - 需要缓存 → sms_service.go, config_service.go
# - 需要事件发布 → consumer_service.go, payment_service.go
# - 简单CRUD → freight_service.go
# - 文件上传 → media_service.go
```

### 技巧2: 快速修复导入错误
```bash
# 如果编译报错 "cannot find package"
# 1. 先搜索项目中的用法
grep -r "包名" backend/app/*/service/internal/ | head -3

# 2. 复制正确的导入路径
# 3. 不要自己写
```

### 技巧3: 快速检查 Wire 依赖
```bash
# 如果 Wire 生成失败
# 1. 查看错误信息中的依赖链
# 2. 找到最底层缺失的依赖
# 3. 检查对应的构造函数签名
# 4. 修复签名，不要添加新的 Provider
```

### 技巧4: 快速验证 Ent 字段
```bash
# 如果遇到类型错误
# 1. 查看 Schema 定义
cat internal/data/ent/schema/xxx.go

# 2. 查看生成的代码
cat internal/data/ent/xxx.go | grep -A 30 "type Xxx struct"

# 3. 确认字段类型（指针或值）
# 4. 使用 Get*() 方法读取（自动处理 nil）
```

---

## 🎓 学习路径

### 第1天: 熟悉参考实现
```bash
# 阅读这3个文件
cat backend/app/consumer/service/internal/service/payment_service.go
cat backend/app/consumer/service/internal/data/payment_order_repo.go
cat backend/app/consumer/service/cmd/server/wire_gen.go

# 理解：
# 1. 导入部分（特别是 pagination）
# 2. 构造函数签名
# 3. 错误处理模式
# 4. Wire 依赖关系
```

### 第2天: 实践一个简单 Service
```bash
# 实现一个只有 Get 和 List 的 Service
# 严格按照范式1的步骤执行
# 每一步都编译验证
```

### 第3天: 实践 Wire 集成
```bash
# 添加新 Service 到 Wire
# 严格按照范式3的步骤执行
# 特别注意查看所有构造函数签名
```

---

## 📚 附录：常用命令速查表

| 目的 | 命令 |
|------|------|
| 搜索 pagination 导入 | `grep -r "paginationV1" backend/app/*/service/internal/service/*.go \| head -1` |
| 查看参考实现导入 | `head -30 backend/app/consumer/service/internal/service/payment_service.go` |
| 查看构造函数签名 | `grep -A 5 "func NewPaymentService" backend/app/consumer/service/internal/service/payment_service.go` |
| 查看 Ent 字段类型 | `cat backend/app/consumer/service/internal/data/ent/xxx.go \| grep -A 30 "type Xxx struct"` |
| 编译验证 | `cd backend/app/consumer/service && go build ./...` |
| 清理缓存 | `go clean -cache && go build ./...` |
| 查看所有 Service | `grep -A 5 "func New.*Service" backend/app/consumer/service/internal/service/*.go` |
| 查看所有 Repository | `grep -A 3 "func New.*Repo" backend/app/consumer/service/internal/data/*_repo.go` |
| 检查 Wire 签名 | `/tmp/check_signatures.sh > /tmp/signatures.txt && cat /tmp/signatures.txt` |

---




**这是血的教训，永不再犯！**

### 6.10 新增教训：2026-03-15 Checkpoint 10 验证

**🚨 本次错误总结：Checkpoint 验证中的三大失误**

#### 失误 1: 指针类型理解不深刻

**错误代码：**
```go
// account.Balance 已经是 *string 类型
BalanceBefore: &account.Balance  // ❌ 错误：**string
```

**根本原因：**
- 没有仔细查看 Protobuf 生成的 Go 类型定义
- 假设所有字段都需要取地址
- 没有理解 optional 字段在 Go 中的表示

**正确做法：**
```go
// 1. 先查看 proto 定义
cat backend/api/protos/consumer/service/v1/finance.proto | grep "balance"

// 2. 查看生成的 Go 类型
cat backend/api/gen/go/consumer/service/v1/finance.pb.go | grep "Balance"

// 3. 理解类型规则
// - optional string balance = 1;  → Go: Balance *string
// - string balance = 1;           → Go: Balance string

// 4. 正确使用
BalanceBefore: account.Balance  // ✅ 正确：*string
```

**新增铁律：**
```
铁律5: 类型先查，后使用（CHECK TYPE FIRST）

在使用任何 Protobuf 生成的字段前：
1. 查看 .proto 定义（是否 optional）
2. 查看生成的 .pb.go 类型（指针还是值）
3. 使用 Get*() 方法读取（自动处理 nil）
4. 直接使用字段写入（不要多余的 &）
```

#### 失误 2: 接口实现理解不完整

**错误代码：**
```go
// 直接传递函数，不满足 Handler 接口
s.eventBus.Subscribe(topic, func(ctx context.Context, event eventbus.Event) error {
    // ...
})  // ❌ 缺少 Handle 方法
```

**根本原因：**
- 没有查看 eventbus.Handler 接口定义
- 假设可以直接传递函数
- 没有查看参考实现或文档

**正确做法：**
```go
// 1. 先查看接口定义
cat backend/pkg/eventbus/handler.go | grep "type Handler interface" -A 5

// 2. 查看适配器实现
cat backend/pkg/eventbus/handler.go | grep "EventHandlerFunc" -A 10

// 3. 使用适配器包装
s.eventBus.Subscribe(topic, eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
    // ...
}))  // ✅ 正确：实现了 Handler 接口
```

**新增铁律：**
```
铁律6: 接口先查，后实现（CHECK INTERFACE FIRST）

在实现任何接口前：
1. 查看接口定义（方法签名）
2. 查看是否有适配器（如 HandlerFunc）
3. 查看参考实现（其他地方如何使用）
4. 使用正确的实现方式
```

#### 失误 3: Wire 生成文件理解不足

**错误现象：**
```go
// wire_gen.go (自动生成)
httpServer, err := server.NewRestServer(context, consumerService, smsService, paymentService)
// ❌ 缺少 financeService 参数
```

**根本原因：**
- 修改了 NewRestServer 函数签名，但没有重新生成 Wire 代码
- 不理解 wire_gen.go 是自动生成的文件
- 没有意识到需要运行 go generate

**正确做法：**
```bash
# 1. 理解 Wire 工作原理
# - wire.go: 定义依赖注入配置（手动编写）
# - wire_gen.go: 生成的依赖注入代码（自动生成）

# 2. 修改函数签名后必须重新生成
cd backend/app/consumer/service/cmd/server
go generate

# 3. 验证生成结果
grep "NewRestServer" wire_gen.go
```

**新增铁律：**
```
铁律7: Wire 文件必须重新生成（REGENERATE WIRE）

修改以下内容后必须运行 go generate：
1. 修改 Provider 函数签名
2. 添加新的 Service 或 Repository
3. 修改依赖注入关系
4. 更新 ProviderSet

验证方法：
- 检查 wire_gen.go 是否包含新的依赖
- 运行 go build 验证编译通过
```

#### 失误 4: 验证流程不完整

**本次验证流程的问题：**
1. ❌ 没有在修复后立即编译验证
2. ❌ 没有意识到 Wire 生成文件需要更新
3. ❌ 依赖用户手动执行编译命令

**改进的验证流程：**
```
标准验证流程（强制执行）：

1. 修复代码错误
   ↓
2. 检查是否需要重新生成（Wire、Protobuf、Ent）
   ↓
3. 如果需要，提示用户运行生成命令
   ↓
4. 等待用户反馈编译结果
   ↓
5. 根据结果继续修复或完成验证
```

**新增铁律：**
```
铁律8: 完整验证流程（COMPLETE VERIFICATION）

每次修复后必须：
1. 检查是否需要重新生成代码
2. 明确告知用户需要执行的命令
3. 等待用户反馈结果
4. 根据反馈继续修复
5. 不要假设修复成功

生成代码检查清单：
- [ ] Wire (修改 Provider 函数签名)
- [ ] Protobuf (修改 .proto 文件)
- [ ] Ent (修改 Schema)
- [ ] 其他代码生成工具
```

### 6.11 防幻觉检查清单（更新版）

**在生成任何代码前，必须完成：**

#### 基础验证（铁律 1-4）
- [ ] 验证所有函数是否存在
- [ ] 验证所有类型是否正确
- [ ] 查看参考实现
- [ ] 增量开发，立即验证

#### 类型验证（铁律 5）
- [ ] 查看 proto 定义（optional 关键字）
- [ ] 查看生成的 Go 类型（指针或值）
- [ ] 使用 Get*() 方法读取
- [ ] 直接使用字段写入（避免多余的 &）

#### 接口验证（铁律 6）
- [ ] 查看接口定义（方法签名）
- [ ] 查看是否有适配器
- [ ] 查看参考实现
- [ ] 使用正确的实现方式

#### 生成代码验证（铁律 7）
- [ ] 检查是否修改了 Provider 函数
- [ ] 检查是否需要重新生成 Wire
- [ ] 检查是否需要重新生成 Protobuf
- [ ] 检查是否需要重新生成 Ent

#### 完整验证流程（铁律 8）
- [ ] 修复后检查生成代码需求
- [ ] 明确告知用户执行命令
- [ ] 等待用户反馈结果
- [ ] 根据反馈继续修复
- [ ] 不假设修复成功

**持续改进措施：**
1. 每次错误后更新宪法
2. 建立错误模式库
3. 完善验证检查清单
4. 自动化验证流程

## 🚀 最后的话

记住这些铁律和教训：

1. **Pagination 必须搜索**（用户强调100次！）
2. **增量开发+立即验证**（一次一个方法，立即编译）
3. **复制参考实现**（不要创造，不要假设）

遵循这个手册，你的开发效率会提升5倍以上，错误会减少90%以上。

**老铁，加油！** 💪
