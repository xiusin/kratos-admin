# C端用户管理系统 - 技术设计文档

## Overview

### 系统概述

C端用户管理系统（Consumer User Management System）是一个面向终端消费者的综合服务平台，提供用户注册登录、支付、财务、媒体管理、物流等核心功能。系统采用微服务架构，基于 Kratos 框架和 gRPC 通信协议，支持多租户数据隔离和事件驱动的模块解耦。

**关键特性：**
- 独立的 Consumer 服务模块，与管理后台用户系统（identity）完全隔离
- 多租户架构，支持租户级数据隔离和配置
- 事件驱动设计，通过 eventbus 实现模块间解耦
- 支持多种第三方服务集成（微信、支付宝、阿里云、腾讯云等）
- 完善的安全机制（JWT认证、限流、风险评分）

### 系统边界

**包含的功能：**
- 用户注册与认证（手机号、微信登录）
- 用户信息管理（个人资料、头像、联系方式）
- 短信服务（验证码、通知短信）
- 支付服务（微信支付、支付宝、易宝支付）
- 财务管理（余额、充值、提现、流水）
- 微信集成（OAuth登录、公众号、小程序）
- 媒体管理（图片、视频上传和存储）
- 物流管理（快递查询、物流跟踪）
- 运费计算（按重量、按距离）

**不包含的功能：**
- 商品管理（属于商品服务）
- 订单管理（属于订单服务）
- 库存管理（属于库存服务）
- 营销活动（属于营销服务）

### 技术栈

**后端：**
- 框架：Kratos v2
- 语言：Go 1.21+
- 通信：gRPC + HTTP/REST
- ORM：Ent
- 数据库：MySQL 8.0+
- 缓存：Redis 7.0+
- 消息队列：Kafka（事件总线）

**前端：**
- 框架：Vue 3 + TypeScript
- 状态管理：Pinia
- UI组件：Vben Admin
- 构建工具：Vite


## Architecture

### 系统架构

系统采用三层架构（API/App/Pkg）+ 微服务架构，Consumer 服务作为独立的微服务模块部署。

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                    (HTTP/gRPC 统一入口)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌───────▼────────┐   ┌───────▼────────┐
│  Admin Service │   │Consumer Service│   │  Other Services│
│  (管理后台)     │   │  (C端用户)     │   │  (其他服务)    │
└────────────────┘   └────────────────┘   └────────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │    Event Bus      │
                    │    (Kafka)        │
                    └───────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌───────▼────────┐   ┌───────▼────────┐
│   MySQL        │   │    Redis       │   │   OSS          │
│   (数据存储)    │   │   (缓存)       │   │  (文件存储)    │
└────────────────┘   └────────────────┘   └────────────────┘
```

### 三层架构

```
backend/
├── api/                          # API 定义层
│   └── protos/
│       └── consumer/             # Consumer 服务 Protobuf 定义
│           └── service/
│               └── v1/
│                   ├── consumer.proto      # C端用户服务
│                   ├── sms.proto           # 短信服务
│                   ├── payment.proto       # 支付服务
│                   ├── finance.proto       # 财务服务
│                   ├── wechat.proto        # 微信服务
│                   ├── media.proto         # 媒体服务
│                   ├── logistics.proto     # 物流服务
│                   └── freight.proto       # 运费计算服务
│
├── app/                          # 应用层
│   └── consumer/                 # Consumer 服务实现
│       └── service/
│           ├── cmd/              # 启动入口
│           │   └── server/
│           │       ├── main.go
│           │       └── wire.go
│           ├── configs/          # 配置文件
│           │   └── config.yaml
│           └── internal/         # 内部实现
│               ├── service/      # 服务层（业务逻辑）
│               │   ├── consumer_service.go
│               │   ├── sms_service.go
│               │   ├── payment_service.go
│               │   ├── finance_service.go
│               │   ├── wechat_service.go
│               │   ├── media_service.go
│               │   ├── logistics_service.go
│               │   └── freight_service.go
│               ├── data/         # 数据层（Repository）
│               │   ├── ent/      # Ent Schema
│               │   │   └── schema/
│               │   │       ├── consumer.go
│               │   │       ├── login_log.go
│               │   │       ├── sms_log.go
│               │   │       ├── payment_order.go
│               │   │       ├── finance_account.go
│               │   │       ├── finance_transaction.go
│               │   │       ├── media_file.go
│               │   │       └── logistics_tracking.go
│               │   ├── consumer_repo.go
│               │   ├── login_log_repo.go
│               │   ├── sms_repo.go
│               │   ├── payment_repo.go
│               │   ├── finance_repo.go
│               │   ├── media_repo.go
│               │   └── logistics_repo.go
│               └── server/       # 服务器层（HTTP/gRPC）
│                   ├── grpc.go
│                   └── http.go
│
└── pkg/                          # 基础设施层
    ├── sms/                      # 短信工具
    │   ├── aliyun.go
    │   └── tencent.go
    ├── payment/                  # 支付工具
    │   ├── wechat.go
    │   ├── alipay.go
    │   └── yeepay.go
    ├── oss/                      # 对象存储工具
    │   ├── aliyun.go
    │   └── tencent.go
    └── logistics/                # 物流工具
        └── kdniao.go
```


### 模块划分

系统按功能划分为 8 个核心模块：

1. **Consumer Service（用户服务）**
   - 用户注册、登录、信息管理
   - 登录日志、风险评分
   - 账户锁定、注销

2. **SMS Service（短信服务）**
   - 验证码发送和验证
   - 通知短信发送
   - 短信日志记录

3. **Payment Service（支付服务）**
   - 支付订单创建和查询
   - 支付回调处理
   - 退款操作

4. **Finance Service（财务服务）**
   - 账户余额管理
   - 充值、提现
   - 财务流水查询

5. **Wechat Service（微信服务）**
   - OAuth 登录
   - 公众号集成
   - 小程序集成

6. **Media Service（媒体服务）**
   - 图片、视频上传
   - OSS 存储管理
   - 媒体文件查询

7. **Logistics Service（物流服务）**
   - 快递查询
   - 物流跟踪
   - 物流状态订阅

8. **Freight Service（运费计算服务）**
   - 按重量计算运费
   - 按距离计算运费
   - 运费模板管理

### 依赖关系

```
Consumer Service (用户服务)
    ↓ 依赖
SMS Service (短信服务)
Wechat Service (微信服务)

Payment Service (支付服务)
    ↓ 发布事件
Finance Service (财务服务)
    ↓ 订阅事件
PaymentSuccessEvent

Logistics Service (物流服务)
    ↓ 发布事件
LogisticsStatusChangedEvent
```

**依赖规则：**
- 服务层可以依赖数据层和 pkg 层
- 数据层只能依赖 pkg 层
- pkg 层不能依赖 app 层
- 模块间通过事件总线通信，避免直接依赖


## Components and Interfaces

### Consumer Service（用户服务）

**职责：**
- 用户注册、登录、信息管理
- 登录日志记录和查询
- 风险评分计算
- 账户锁定和注销

**接口定义：**

```protobuf
service ConsumerService {
  // 手机号注册
  rpc RegisterByPhone(RegisterByPhoneRequest) returns (Consumer);
  
  // 手机号登录
  rpc LoginByPhone(LoginByPhoneRequest) returns (LoginResponse);
  
  // 微信登录
  rpc LoginByWechat(LoginByWechatRequest) returns (LoginResponse);
  
  // 获取用户信息
  rpc GetConsumer(GetConsumerRequest) returns (Consumer);
  
  // 更新用户信息
  rpc UpdateConsumer(UpdateConsumerRequest) returns (google.protobuf.Empty);
  
  // 更新手机号
  rpc UpdatePhone(UpdatePhoneRequest) returns (google.protobuf.Empty);
  
  // 更新邮箱
  rpc UpdateEmail(UpdateEmailRequest) returns (google.protobuf.Empty);
  
  // 上传头像
  rpc UploadAvatar(UploadAvatarRequest) returns (UploadAvatarResponse);
  
  // 注销账户
  rpc DeactivateAccount(DeactivateAccountRequest) returns (google.protobuf.Empty);
  
  // 查询登录日志
  rpc ListLoginLogs(pagination.PagingRequest) returns (ListLoginLogsResponse);
  
  // 查询用户列表（管理员）
  rpc ListConsumers(pagination.PagingRequest) returns (ListConsumersResponse);
}
```

**依赖：**
- SMSService（发送验证码）
- WechatService（微信登录）
- EventBus（发布 UserRegisteredEvent）

### SMS Service（短信服务）

**职责：**
- 发送验证码短信
- 发送通知短信
- 验证码校验
- 短信日志记录

**接口定义：**

```protobuf
service SMSService {
  // 发送验证码
  rpc SendVerificationCode(SendVerificationCodeRequest) returns (google.protobuf.Empty);
  
  // 验证验证码
  rpc VerifyCode(VerifyCodeRequest) returns (VerifyCodeResponse);
  
  // 发送通知短信
  rpc SendNotification(SendNotificationRequest) returns (google.protobuf.Empty);
  
  // 查询短信日志
  rpc ListSMSLogs(pagination.PagingRequest) returns (ListSMSLogsResponse);
}
```

**依赖：**
- pkg/sms（阿里云、腾讯云短信SDK）
- Redis（验证码缓存）

### Payment Service（支付服务）

**职责：**
- 创建支付订单
- 处理支付回调
- 查询支付结果
- 退款操作

**接口定义：**

```protobuf
service PaymentService {
  // 创建支付订单
  rpc CreatePayment(CreatePaymentRequest) returns (CreatePaymentResponse);
  
  // 查询支付订单
  rpc GetPayment(GetPaymentRequest) returns (PaymentOrder);
  
  // 查询支付结果
  rpc QueryPaymentStatus(QueryPaymentStatusRequest) returns (PaymentStatusResponse);
  
  // 申请退款
  rpc Refund(RefundRequest) returns (RefundResponse);
  
  // 查询退款状态
  rpc QueryRefundStatus(QueryRefundStatusRequest) returns (RefundStatusResponse);
  
  // 查询支付流水
  rpc ListPayments(pagination.PagingRequest) returns (ListPaymentsResponse);
}
```

**依赖：**
- pkg/payment（微信、支付宝、易宝SDK）
- EventBus（发布 PaymentSuccessEvent）


### Finance Service（财务服务）

**职责：**
- 账户余额管理
- 充值、提现操作
- 财务流水记录和查询
- 余额变动审计

**接口定义：**

```protobuf
service FinanceService {
  // 获取账户余额
  rpc GetAccount(GetAccountRequest) returns (FinanceAccount);
  
  // 充值
  rpc Recharge(RechargeRequest) returns (google.protobuf.Empty);
  
  // 申请提现
  rpc Withdraw(WithdrawRequest) returns (WithdrawResponse);
  
  // 审核提现
  rpc ApproveWithdraw(ApproveWithdrawRequest) returns (google.protobuf.Empty);
  
  // 查询财务流水
  rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
  
  // 导出财务流水
  rpc ExportTransactions(ExportTransactionsRequest) returns (ExportTransactionsResponse);
}
```

**依赖：**
- EventBus（订阅 PaymentSuccessEvent）
- PaymentService（发起打款）

### Wechat Service（微信服务）

**职责：**
- 微信 OAuth 登录
- 获取微信用户信息
- 公众号消息推送
- 小程序登录

**接口定义：**

```protobuf
service WechatService {
  // 获取微信授权URL
  rpc GetAuthURL(GetAuthURLRequest) returns (GetAuthURLResponse);
  
  // 微信授权回调
  rpc AuthCallback(AuthCallbackRequest) returns (AuthCallbackResponse);
  
  // 获取微信用户信息
  rpc GetWechatUserInfo(GetWechatUserInfoRequest) returns (WechatUserInfo);
  
  // 发送模板消息
  rpc SendTemplateMessage(SendTemplateMessageRequest) returns (google.protobuf.Empty);
  
  // 小程序登录
  rpc MiniProgramLogin(MiniProgramLoginRequest) returns (MiniProgramLoginResponse);
}
```

**依赖：**
- 微信 SDK
- Redis（access_token 缓存）

### Media Service（媒体服务）

**职责：**
- 图片、视频上传
- 生成预签名 URL
- 媒体文件管理
- 缩略图生成

**接口定义：**

```protobuf
service MediaService {
  // 生成上传预签名URL
  rpc GenerateUploadURL(GenerateUploadURLRequest) returns (GenerateUploadURLResponse);
  
  // 确认上传完成
  rpc ConfirmUpload(ConfirmUploadRequest) returns (MediaFile);
  
  // 获取媒体文件
  rpc GetMediaFile(GetMediaFileRequest) returns (MediaFile);
  
  // 查询媒体文件列表
  rpc ListMediaFiles(pagination.PagingRequest) returns (ListMediaFilesResponse);
  
  // 删除媒体文件
  rpc DeleteMediaFile(DeleteMediaFileRequest) returns (google.protobuf.Empty);
}
```

**依赖：**
- pkg/oss（阿里云 OSS、腾讯云 COS）
- 图片处理库（缩略图生成）

### Logistics Service（物流服务）

**职责：**
- 快递查询
- 物流轨迹跟踪
- 物流状态订阅
- 物流信息缓存

**接口定义：**

```protobuf
service LogisticsService {
  // 查询物流信息
  rpc QueryLogistics(QueryLogisticsRequest) returns (LogisticsInfo);
  
  // 订阅物流状态
  rpc SubscribeLogistics(SubscribeLogisticsRequest) returns (google.protobuf.Empty);
  
  // 查询物流历史
  rpc ListLogisticsHistory(ListLogisticsHistoryRequest) returns (ListLogisticsHistoryResponse);
}
```

**依赖：**
- pkg/logistics（快递鸟 API）
- Redis（物流信息缓存）
- EventBus（发布 LogisticsStatusChangedEvent）

### Freight Service（运费计算服务）

**职责：**
- 按重量计算运费
- 按距离计算运费
- 运费模板管理
- 包邮规则判断

**接口定义：**

```protobuf
service FreightService {
  // 计算运费
  rpc CalculateFreight(CalculateFreightRequest) returns (CalculateFreightResponse);
  
  // 创建运费模板
  rpc CreateFreightTemplate(CreateFreightTemplateRequest) returns (FreightTemplate);
  
  // 更新运费模板
  rpc UpdateFreightTemplate(UpdateFreightTemplateRequest) returns (google.protobuf.Empty);
  
  // 查询运费模板
  rpc GetFreightTemplate(GetFreightTemplateRequest) returns (FreightTemplate);
  
  // 查询运费模板列表
  rpc ListFreightTemplates(pagination.PagingRequest) returns (ListFreightTemplatesResponse);
}
```

**依赖：**
- 地理位置服务（计算距离）


## Data Models

### Consumer（C端用户）

```go
// Ent Schema
type Consumer struct {
    ent.Schema
}

func (Consumer) Fields() []ent.Field {
    return []ent.Field{
        field.Uint32("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.String("phone").MaxLen(20).Unique().Comment("手机号"),
        field.String("email").MaxLen(100).Optional().Comment("邮箱"),
        field.String("nickname").MaxLen(50).Optional().Comment("昵称"),
        field.String("avatar").MaxLen(500).Optional().Comment("头像URL"),
        field.String("password_hash").MaxLen(255).Comment("密码哈希"),
        field.String("wechat_openid").MaxLen(100).Optional().Comment("微信OpenID"),
        field.String("wechat_unionid").MaxLen(100).Optional().Comment("微信UnionID"),
        field.Enum("status").Values("normal", "locked", "deactivated").Default("normal").Comment("状态"),
        field.Int("risk_score").Default(0).Comment("风险评分 0-100"),
        field.Int("login_fail_count").Default(0).Comment("登录失败次数"),
        field.Time("locked_until").Optional().Nillable().Comment("锁定截止时间"),
        field.Time("last_login_at").Optional().Nillable().Comment("最后登录时间"),
        field.String("last_login_ip").MaxLen(50).Optional().Comment("最后登录IP"),
        field.Time("deactivated_at").Optional().Nillable().Comment("注销时间"),
        field.Uint32("created_by").Optional().Comment("创建者ID"),
        field.Uint32("updated_by").Optional().Comment("更新者ID"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
    }
}

func (Consumer) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "phone").Unique(),
        index.Fields("tenant_id", "wechat_openid"),
        index.Fields("tenant_id", "status"),
    }
}
```

**Protobuf 定义：**

```protobuf
message Consumer {
  enum Status {
    NORMAL = 0;       // 正常
    LOCKED = 1;       // 锁定
    DEACTIVATED = 2;  // 已注销
  }
  
  optional uint32 id = 1;
  optional uint32 tenant_id = 2;
  optional string phone = 3;
  optional string email = 4;
  optional string nickname = 5;
  optional string avatar = 6;
  optional string wechat_openid = 7;
  optional string wechat_unionid = 8;
  optional Status status = 9;
  optional int32 risk_score = 10;
  optional google.protobuf.Timestamp locked_until = 11;
  optional google.protobuf.Timestamp last_login_at = 12;
  optional string last_login_ip = 13;
  optional google.protobuf.Timestamp deactivated_at = 14;
  optional google.protobuf.Timestamp created_at = 15;
  optional google.protobuf.Timestamp updated_at = 16;
}
```

### LoginLog（登录日志）

```go
type LoginLog struct {
    ent.Schema
}

func (LoginLog) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.Uint32("consumer_id").Comment("用户ID"),
        field.String("phone").MaxLen(20).Comment("手机号"),
        field.Enum("login_type").Values("phone", "wechat").Comment("登录方式"),
        field.Bool("success").Comment("是否成功"),
        field.String("fail_reason").MaxLen(200).Optional().Comment("失败原因"),
        field.String("ip_address").MaxLen(50).Comment("IP地址"),
        field.String("user_agent").MaxLen(500).Optional().Comment("User Agent"),
        field.String("device_type").MaxLen(50).Optional().Comment("设备类型"),
        field.Time("login_at").Default(time.Now).Comment("登录时间"),
    }
}

func (LoginLog) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "consumer_id", "login_at"),
        index.Fields("tenant_id", "phone", "login_at"),
    }
}
```


### SMSLog（短信日志）

```go
type SMSLog struct {
    ent.Schema
}

func (SMSLog) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.String("phone").MaxLen(20).Comment("手机号"),
        field.Enum("sms_type").Values("verification", "notification").Comment("短信类型"),
        field.String("content").MaxLen(500).Comment("短信内容"),
        field.String("code").MaxLen(10).Optional().Comment("验证码"),
        field.Enum("channel").Values("aliyun", "tencent").Comment("短信通道"),
        field.Enum("status").Values("pending", "success", "failed").Comment("发送状态"),
        field.String("error_message").MaxLen(200).Optional().Comment("错误信息"),
        field.Time("sent_at").Default(time.Now).Comment("发送时间"),
        field.Time("expires_at").Optional().Nillable().Comment("过期时间"),
    }
}

func (SMSLog) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "phone", "sent_at"),
        index.Fields("tenant_id", "sms_type", "status"),
    }
}
```

### PaymentOrder（支付订单）

```go
type PaymentOrder struct {
    ent.Schema
}

func (PaymentOrder) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.String("order_no").MaxLen(64).Unique().Comment("订单号"),
        field.Uint32("consumer_id").Comment("用户ID"),
        field.Enum("payment_method").Values("wechat", "alipay", "yeepay").Comment("支付方式"),
        field.Enum("payment_type").Values("app", "h5", "mini", "qrcode").Comment("支付类型"),
        field.String("amount", field.TypeString).Comment("支付金额（decimal）"),
        field.Enum("status").Values("pending", "success", "failed", "closed", "refunded").Comment("订单状态"),
        field.String("transaction_id").MaxLen(100).Optional().Comment("第三方交易号"),
        field.String("callback_data").MaxLen(2000).Optional().Comment("回调数据"),
        field.Time("paid_at").Optional().Nillable().Comment("支付时间"),
        field.Time("closed_at").Optional().Nillable().Comment("关闭时间"),
        field.Time("expires_at").Comment("过期时间"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
    }
}

func (PaymentOrder) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "consumer_id", "status"),
        index.Fields("tenant_id", "order_no").Unique(),
        index.Fields("transaction_id"),
    }
}
```

### FinanceAccount（财务账户）

```go
type FinanceAccount struct {
    ent.Schema
}

func (FinanceAccount) Fields() []ent.Field {
    return []ent.Field{
        field.Uint32("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.Uint32("consumer_id").Unique().Comment("用户ID"),
        field.String("balance", field.TypeString).Default("0").Comment("账户余额（decimal）"),
        field.String("frozen_balance", field.TypeString).Default("0").Comment("冻结余额（decimal）"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
    }
}

func (FinanceAccount) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "consumer_id").Unique(),
    }
}
```

### FinanceTransaction（财务流水）

```go
type FinanceTransaction struct {
    ent.Schema
}

func (FinanceTransaction) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.Uint32("consumer_id").Comment("用户ID"),
        field.String("transaction_no").MaxLen(64).Unique().Comment("流水号"),
        field.Enum("transaction_type").Values("recharge", "consume", "withdraw", "refund").Comment("交易类型"),
        field.String("amount", field.TypeString).Comment("交易金额（decimal）"),
        field.String("balance_before", field.TypeString).Comment("交易前余额（decimal）"),
        field.String("balance_after", field.TypeString).Comment("交易后余额（decimal）"),
        field.String("description").MaxLen(200).Optional().Comment("交易描述"),
        field.String("related_order_no").MaxLen(64).Optional().Comment("关联订单号"),
        field.Uint32("operator_id").Optional().Comment("操作人ID"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
    }
}

func (FinanceTransaction) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "consumer_id", "created_at"),
        index.Fields("tenant_id", "transaction_type", "created_at"),
        index.Fields("transaction_no").Unique(),
    }
}
```


### MediaFile（媒体文件）

```go
type MediaFile struct {
    ent.Schema
}

func (MediaFile) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.Uint32("consumer_id").Comment("上传者ID"),
        field.String("file_name").MaxLen(255).Comment("文件名"),
        field.Enum("file_type").Values("image", "video").Comment("文件类型"),
        field.String("file_format").MaxLen(20).Comment("文件格式"),
        field.Uint64("file_size").Comment("文件大小（字节）"),
        field.String("file_url").MaxLen(500).Comment("文件URL"),
        field.String("thumbnail_url").MaxLen(500).Optional().Comment("缩略图URL"),
        field.String("oss_bucket").MaxLen(100).Comment("OSS Bucket"),
        field.String("oss_key").MaxLen(500).Comment("OSS Key"),
        field.Bool("is_deleted").Default(false).Comment("是否删除"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
        field.Time("deleted_at").Optional().Nillable().Comment("删除时间"),
    }
}

func (MediaFile) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "consumer_id", "is_deleted"),
        index.Fields("tenant_id", "file_type", "created_at"),
    }
}
```

### LogisticsTracking（物流跟踪）

```go
type LogisticsTracking struct {
    ent.Schema
}

func (LogisticsTracking) Fields() []ent.Field {
    return []ent.Field{
        field.Uint64("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.String("tracking_no").MaxLen(100).Comment("运单号"),
        field.String("courier_company").MaxLen(50).Comment("快递公司"),
        field.Enum("status").Values("pending", "in_transit", "delivering", "delivered").Comment("物流状态"),
        field.JSON("tracking_info", []map[string]interface{}{}).Comment("物流轨迹"),
        field.Time("last_updated_at").Comment("最后更新时间"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
    }
}

func (LogisticsTracking) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "tracking_no").Unique(),
        index.Fields("tenant_id", "status"),
    }
}
```

### FreightTemplate（运费模板）

```go
type FreightTemplate struct {
    ent.Schema
}

func (FreightTemplate) Fields() []ent.Field {
    return []ent.Field{
        field.Uint32("id"),
        field.Uint32("tenant_id").Comment("租户ID"),
        field.String("name").MaxLen(100).Comment("模板名称"),
        field.Enum("calculation_type").Values("by_weight", "by_distance").Comment("计算方式"),
        field.String("first_weight", field.TypeString).Optional().Comment("首重（kg）"),
        field.String("first_price", field.TypeString).Optional().Comment("首重价格"),
        field.String("additional_weight", field.TypeString).Optional().Comment("续重（kg）"),
        field.String("additional_price", field.TypeString).Optional().Comment("续重价格"),
        field.JSON("region_rules", []map[string]interface{}{}).Optional().Comment("地区规则"),
        field.JSON("free_shipping_rules", []map[string]interface{}{}).Optional().Comment("包邮规则"),
        field.Bool("is_active").Default(true).Comment("是否启用"),
        field.Uint32("created_by").Comment("创建者ID"),
        field.Uint32("updated_by").Optional().Comment("更新者ID"),
        field.Time("created_at").Default(time.Now).Comment("创建时间"),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
    }
}

func (FreightTemplate) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "is_active"),
    }
}
```

### 数据模型关系图

```
┌─────────────┐
│  Consumer   │
│  (C端用户)   │
└──────┬──────┘
       │
       ├──────────┐
       │          │
       ▼          ▼
┌─────────────┐ ┌──────────────┐
│  LoginLog   │ │FinanceAccount│
│  (登录日志)  │ │  (财务账户)   │
└─────────────┘ └──────┬───────┘
                       │
                       ▼
                ┌──────────────────┐
                │FinanceTransaction│
                │   (财务流水)      │
                └──────────────────┘

┌──────────────┐
│ PaymentOrder │
│  (支付订单)   │
└──────────────┘

┌──────────────┐
│  MediaFile   │
│  (媒体文件)   │
└──────────────┘

┌──────────────────┐
│LogisticsTracking │
│   (物流跟踪)      │
└──────────────────┘

┌──────────────────┐
│ FreightTemplate  │
│   (运费模板)      │
└──────────────────┘
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### 用户注册与认证 Properties

### Property 1: 用户注册创建账户

*For any* 有效的手机号和验证码，调用注册接口应该成功创建用户账户，并且可以通过该手机号查询到用户信息

**Validates: Requirements 1.1**

### Property 2: 登录返回有效令牌

*For any* 已注册用户，使用正确的手机号和密码登录应该返回有效的JWT令牌，并且该令牌可以用于后续API调用

**Validates: Requirements 1.2**

### Property 3: 微信登录重定向

*For any* 微信登录请求，返回的授权URL应该包含微信授权域名（open.weixin.qq.com）和必要的参数（appid, redirect_uri, response_type, scope）

**Validates: Requirements 1.3**

### Property 4: 微信授权创建用户

*For any* 有效的微信授权回调，系统应该创建或关联用户账户，并返回有效的访问令牌

**Validates: Requirements 1.4**

### Property 5: 连续失败锁定账户

*For any* 用户账户，连续5次登录失败后，账户应该被锁定15分钟，并且在锁定期间所有登录请求都应该被拒绝

**Validates: Requirements 1.5, 1.6**

### Property 6: 登录日志完整记录

*For any* 登录尝试（成功或失败），系统应该记录登录日志，包含时间、IP地址、设备信息、User Agent等字段

**Validates: Requirements 1.7**

### Property 7: 风险评分范围

*For any* 用户登录行为，计算的风险评分应该在0-100范围内，并且高风险行为（如异常IP、频繁失败）应该产生更高的评分

**Validates: Requirements 1.8**

### Property 8: 高风险要求额外验证

*For any* 风险评分超过80分的登录尝试，系统应该要求额外的安全验证（如短信验证码）

**Validates: Requirements 1.9**

### Property 9: 密码bcrypt加密

*For any* 用户密码，存储在数据库中的密码哈希应该以 $2a$ 或 $2b$ 开头（bcrypt格式），并且原始密码不应该以明文形式存储

**Validates: Requirements 1.10**

### 用户信息管理 Properties

### Property 10: 用户信息查询完整性

*For any* 已创建的用户，通过用户ID查询应该返回完整的用户信息，包括手机号、昵称、头像、状态等字段

**Validates: Requirements 2.1**

### Property 11: 用户信息更新持久化

*For any* 用户信息更新请求，更新后再次查询应该返回更新后的值

**Validates: Requirements 2.2**

### Property 12: 更新手机号需要验证码

*For any* 更新手机号请求，如果不提供有效的验证码，请求应该被拒绝

**Validates: Requirements 2.3**

### Property 13: 账户注销状态保留

*For any* 用户账户，注销后状态应该变为 DEACTIVATED，并且用户数据应该保留（不被物理删除）

**Validates: Requirements 2.6**

### Property 14: 多租户数据隔离

*For any* 租户，查询用户列表时应该只返回该租户的用户，不应该返回其他租户的用户数据

**Validates: Requirements 2.8, 10.1, 10.2, 10.3**


### 短信服务 Properties

### Property 15: 验证码格式和有效期

*For any* 验证码发送请求，生成的验证码应该是6位数字，并且在5分钟后过期失效

**Validates: Requirements 3.3**

### Property 16: 短信发送频率限制

*For any* 手机号，在1分钟内只能发送1条验证码，超过限制的请求应该被拒绝

**Validates: Requirements 3.4**

### Property 17: 短信每日限额

*For any* 手机号，每天最多发送10条验证码，超过限额的请求应该被拒绝

**Validates: Requirements 3.5**

### Property 18: 短信通道故障转移

*For any* 短信发送请求，如果主通道（阿里云）发送失败，系统应该自动切换到备用通道（腾讯云）重试

**Validates: Requirements 3.2, 3.6**

### Property 19: 验证码一次性使用

*For any* 验证码，验证成功后应该立即失效，再次使用相同验证码应该验证失败

**Validates: Requirements 3.8**

### Property 20: 短信日志完整记录

*For any* 短信发送操作，系统应该记录短信日志，包含手机号、内容、状态、时间、通道等字段

**Validates: Requirements 3.7**

### 支付服务 Properties

### Property 21: 支付订单号唯一性

*For any* 支付订单，生成的订单号应该是全局唯一的，不应该存在重复的订单号

**Validates: Requirements 4.4**

### Property 22: 支付订单超时关闭

*For any* 支付订单，如果在30分钟内未完成支付，订单状态应该自动变为 CLOSED

**Validates: Requirements 4.4, 4.7**

### Property 23: 支付成功发布事件

*For any* 支付成功的订单，系统应该发布 PaymentSuccessEvent 事件，并且事件应该包含订单号、金额、用户ID等信息

**Validates: Requirements 4.5, 11.3**

### Property 24: 支付回调签名验证

*For any* 支付回调请求，如果签名验证失败，请求应该被拒绝，订单状态不应该被更新

**Validates: Requirements 4.9**

### Property 25: 退款流水记录

*For any* 退款操作，系统应该记录退款流水，包含原订单号、退款金额、退款状态、退款时间等字段

**Validates: Requirements 4.10, 4.12**

### 财务管理 Properties

### Property 26: 用户账户自动创建

*For any* 新注册用户，系统应该自动创建财务账户，初始余额为0

**Validates: Requirements 5.1**

### Property 27: 充值余额增加

*For any* 充值操作，账户余额应该增加充值金额，并且充值前后的余额差应该等于充值金额

**Validates: Requirements 5.2**

### Property 28: 提现金额限制

*For any* 提现请求，如果提现金额小于10元或大于5000元，请求应该被拒绝

**Validates: Requirements 5.4**

### Property 29: 提现冻结和解冻

*For any* 提现申请，申请时应该冻结相应金额，审核拒绝时应该解冻返还到可用余额

**Validates: Requirements 5.3, 5.6**

### Property 30: 余额非负约束

*For any* 账户操作，账户余额不应该变为负数，如果操作导致余额为负，操作应该被拒绝

**Validates: Requirements 5.9**

### Property 31: 金额精度保证

*For any* 金额计算，使用decimal类型进行计算，结果应该精确到分，不应该出现浮点误差

**Validates: Requirements 5.12**

### Property 32: 财务流水完整性

*For any* 余额变动操作（充值、消费、提现、退款），系统应该记录财务流水，包含交易类型、金额、交易前后余额、操作人、操作时间等字段

**Validates: Requirements 5.7, 5.11**


### 微信服务 Properties

### Property 33: 微信授权URL格式

*For any* 微信授权请求，返回的URL应该包含必要的参数（appid, redirect_uri, response_type=code, scope），并且redirect_uri应该是URL编码的

**Validates: Requirements 6.1**

### Property 34: 微信用户信息获取

*For any* 有效的微信授权，系统应该能够获取微信用户的openid、unionid、昵称、头像等基本信息

**Validates: Requirements 6.2, 6.3**

### Property 35: 微信access_token缓存

*For any* 微信access_token，应该被缓存并在过期前（7200秒）自动刷新，避免频繁调用微信API

**Validates: Requirements 6.10**

### 媒体服务 Properties

### Property 36: 文件格式验证

*For any* 文件上传请求，如果文件格式不在允许列表中（图片：JPEG/PNG/GIF，视频：MP4/AVI/MOV），请求应该被拒绝

**Validates: Requirements 7.1, 7.2**

### Property 37: 文件大小限制

*For any* 文件上传请求，如果图片文件大于5MB或视频文件大于100MB，请求应该被拒绝

**Validates: Requirements 7.3, 7.4**

### Property 38: 预签名URL有效期

*For any* 生成的预签名URL，应该在1小时后过期失效，过期后不应该能够用于上传文件

**Validates: Requirements 7.7**

### Property 39: 缩略图自动生成

*For any* 上传的图片文件，系统应该自动生成200x200的缩略图，并且缩略图URL应该被保存到数据库

**Validates: Requirements 7.8**

### Property 40: 媒体文件软删除

*For any* 媒体文件删除操作，文件应该被标记为已删除（is_deleted=true），而不是物理删除，并且查询时不应该返回已删除的文件

**Validates: Requirements 7.11**

### 物流服务 Properties

### Property 41: 物流信息缓存

*For any* 物流查询请求，查询结果应该被缓存30分钟，30分钟内的重复查询应该返回缓存数据而不是调用第三方API

**Validates: Requirements 8.3**

### Property 42: 物流状态变更事件

*For any* 物流状态变更（从运输中变为派送中，或从派送中变为已签收），系统应该发布 LogisticsStatusChangedEvent 事件

**Validates: Requirements 8.8, 11.5**

### 运费计算 Properties

### Property 43: 运费计算精度

*For any* 运费计算，结果应该精确到分（两位小数），不应该出现精度丢失

**Validates: Requirements 9.10**

### Property 44: 包邮规则优先

*For any* 运费计算请求，如果订单满足包邮条件（满额包邮或指定地区包邮），返回的运费应该为0

**Validates: Requirements 9.6, 9.7**

### Property 45: 首重续重阶梯计算

*For any* 按重量计算运费的请求，如果重量超过首重，应该按照首重价格+续重价格*续重重量的方式计算

**Validates: Requirements 9.3**

### 事件驱动 Properties

### Property 46: 事件异步非阻塞

*For any* 事件发布操作，发布应该是异步的，不应该阻塞主业务流程，即使事件处理失败也不应该影响主流程

**Validates: Requirements 11.6**

### Property 47: 事件重试机制

*For any* 事件处理失败，系统应该自动重试最多3次，如果3次都失败，事件应该进入死信队列

**Validates: Requirements 11.7, 11.8**

### Property 48: 充值事件触发余额增加

*For any* PaymentSuccessEvent 事件（支付类型为充值），Finance Service 应该订阅该事件并自动增加用户账户余额

**Validates: Requirements 11.4**


### API安全与限流 Properties

### Property 49: JWT令牌认证

*For any* 受保护的API接口，如果请求不包含有效的JWT令牌，请求应该被拒绝并返回401错误

**Validates: Requirements 13.1**

### Property 50: JWT令牌有效期

*For any* JWT令牌，应该在2小时后过期，过期后不应该能够用于API调用

**Validates: Requirements 13.2**

### Property 51: API限流保护

*For any* 用户，在1分钟内对敏感接口的请求不应该超过60次，超过限制应该返回429错误

**Validates: Requirements 13.4, 13.5**

### Property 52: 敏感数据脱敏

*For any* 包含敏感数据（手机号、身份证号）的API响应，敏感字段应该被脱敏显示（如手机号显示为 138****1234）

**Validates: Requirements 13.7**

### Property 53: 输入参数验证

*For any* API请求，所有输入参数应该经过验证，防止SQL注入、XSS攻击等安全漏洞

**Validates: Requirements 13.6**

### 配置管理 Properties

### Property 54: 多租户配置隔离

*For any* 租户，应该能够配置独立的短信账户、支付商户号、OSS账户等，并且系统应该使用该租户的配置而不是其他租户的配置

**Validates: Requirements 3.9, 4.11, 7.12, 14.3**

### Property 55: 配置变更审计

*For any* 配置更新操作，系统应该记录配置变更历史，包含变更人、变更时间、变更内容（变更前后的值）

**Validates: Requirements 14.6**

### 性能与监控 Properties

### Property 56: API响应时间

*For any* API请求，P95响应时间应该小于200ms（在正常负载下）

**Validates: Requirements 15.8**

### Property 57: 健康检查接口

*For any* 时刻，/health 接口应该返回系统健康状态，包括数据库连接、Redis连接、第三方服务连接等状态

**Validates: Requirements 15.1**


## Error Handling

### 错误分类

系统采用分层错误处理机制，将错误分为以下几类：

1. **业务错误（Business Error）**
   - 用户输入错误（手机号格式错误、验证码错误）
   - 业务规则违反（余额不足、账户已锁定）
   - 资源不存在（用户不存在、订单不存在）
   - HTTP状态码：400 Bad Request, 404 Not Found

2. **认证授权错误（Auth Error）**
   - JWT令牌无效或过期
   - 权限不足
   - 租户隔离违反
   - HTTP状态码：401 Unauthorized, 403 Forbidden

3. **限流错误（Rate Limit Error）**
   - API调用频率超限
   - 短信发送频率超限
   - HTTP状态码：429 Too Many Requests

4. **第三方服务错误（External Service Error）**
   - 短信服务调用失败
   - 支付服务调用失败
   - OSS服务调用失败
   - 物流API调用失败
   - HTTP状态码：502 Bad Gateway, 503 Service Unavailable

5. **系统错误（System Error）**
   - 数据库连接失败
   - Redis连接失败
   - 内部逻辑错误
   - HTTP状态码：500 Internal Server Error

### 错误响应格式

所有错误响应遵循统一的格式：

```json
{
  "code": "ERROR_CODE",
  "message": "Human readable error message",
  "details": {
    "field": "error details",
    "reason": "specific reason"
  },
  "request_id": "unique-request-id",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 错误处理策略

#### 1. 业务错误处理

```go
// 示例：余额不足错误
if account.Balance < amount {
    return nil, consumerV1.ErrorInsufficientBalance(
        "账户余额不足，当前余额: %s, 需要: %s", 
        account.Balance, amount,
    )
}
```

**处理原则：**
- 返回清晰的错误信息，帮助用户理解问题
- 不暴露敏感的系统内部信息
- 记录错误日志（INFO级别）

#### 2. 第三方服务错误处理

```go
// 示例：短信发送失败自动切换通道
func (s *SMSService) SendVerificationCode(ctx context.Context, phone string) error {
    // 尝试主通道（阿里云）
    err := s.aliyunSMS.Send(ctx, phone, code)
    if err != nil {
        s.log.Warnf("aliyun sms failed: %v, switching to tencent", err)
        
        // 自动切换到备用通道（腾讯云）
        err = s.tencentSMS.Send(ctx, phone, code)
        if err != nil {
            s.log.Errorf("both sms channels failed: %v", err)
            return consumerV1.ErrorSMSServiceUnavailable("短信服务暂时不可用，请稍后重试")
        }
    }
    return nil
}
```

**处理原则：**
- 实现故障转移（Failover）机制
- 记录详细的错误日志（ERROR级别）
- 返回用户友好的错误信息
- 设置合理的超时时间
- 实现重试机制（指数退避）

#### 3. 数据库错误处理

```go
// 示例：数据库操作错误处理
func (r *ConsumerRepo) Create(ctx context.Context, consumer *Consumer) error {
    _, err := r.entClient.Client().Consumer.Create().
        SetPhone(consumer.Phone).
        SetPasswordHash(consumer.PasswordHash).
        Save(ctx)
    
    if err != nil {
        // 唯一约束冲突
        if ent.IsConstraintError(err) {
            return consumerV1.ErrorPhoneAlreadyExists("手机号已被注册")
        }
        
        // 其他数据库错误
        r.log.Errorf("create consumer failed: %v", err)
        return consumerV1.ErrorInternalServerError("创建用户失败")
    }
    
    return nil
}
```

**处理原则：**
- 区分不同类型的数据库错误
- 唯一约束冲突返回业务错误
- 其他数据库错误返回系统错误
- 记录详细的错误日志（ERROR级别）
- 不暴露数据库内部信息

#### 4. 事务错误处理

```go
// 示例：事务回滚
func (r *FinanceRepo) Withdraw(ctx context.Context, consumerID uint32, amount decimal.Decimal) error {
    tx, err := r.entClient.Client().Tx(ctx)
    if err != nil {
        return err
    }
    
    defer func() {
        if err != nil {
            if rollbackErr := tx.Rollback(); rollbackErr != nil {
                r.log.Errorf("transaction rollback failed: %v", rollbackErr)
            }
        }
    }()
    
    // 业务逻辑...
    
    if err = tx.Commit(); err != nil {
        r.log.Errorf("transaction commit failed: %v", err)
        return consumerV1.ErrorInternalServerError("提现失败")
    }
    
    return nil
}
```

**处理原则：**
- 使用defer确保事务回滚
- 记录回滚失败的错误
- 提交失败返回系统错误

#### 5. 并发错误处理

```go
// 示例：乐观锁处理
func (r *FinanceRepo) UpdateBalance(ctx context.Context, accountID uint32, amount decimal.Decimal) error {
    for retry := 0; retry < 3; retry++ {
        account, err := r.Get(ctx, accountID)
        if err != nil {
            return err
        }
        
        newBalance := account.Balance.Add(amount)
        
        // 使用版本号实现乐观锁
        affected, err := r.entClient.Client().FinanceAccount.Update().
            Where(
                financeaccount.IDEQ(accountID),
                financeaccount.VersionEQ(account.Version),
            ).
            SetBalance(newBalance).
            SetVersion(account.Version + 1).
            Save(ctx)
        
        if err != nil {
            return err
        }
        
        if affected == 0 {
            // 版本冲突，重试
            r.log.Warnf("balance update conflict, retry %d", retry)
            time.Sleep(time.Millisecond * 100)
            continue
        }
        
        return nil
    }
    
    return consumerV1.ErrorConcurrentUpdateFailed("余额更新失败，请重试")
}
```

**处理原则：**
- 使用乐观锁避免并发冲突
- 实现重试机制（最多3次）
- 重试失败返回业务错误

### 错误日志记录

所有错误都应该记录日志，包含以下信息：

```go
log.WithContext(ctx).
    WithField("request_id", requestID).
    WithField("user_id", userID).
    WithField("tenant_id", tenantID).
    WithField("error", err).
    Error("operation failed")
```

**日志级别：**
- INFO: 业务错误（用户输入错误、资源不存在）
- WARN: 可恢复的错误（第三方服务失败但有备用方案）
- ERROR: 系统错误（数据库错误、内部逻辑错误）
- FATAL: 致命错误（服务无法启动）


## Testing Strategy

### 测试方法论

系统采用双重测试策略：单元测试（Unit Tests）+ 属性测试（Property-Based Tests），确保代码的正确性和健壮性。

**单元测试（Unit Tests）：**
- 验证具体的业务场景和边界条件
- 测试特定的输入输出组合
- 测试错误处理逻辑
- 测试集成点和依赖关系

**属性测试（Property-Based Tests）：**
- 验证通用的系统属性和不变量
- 通过随机生成大量测试数据覆盖更多场景
- 每个属性测试运行至少100次迭代
- 对应设计文档中的 Correctness Properties

### 测试框架和工具

**后端测试：**
- 测试框架：Go testing + testify
- 属性测试库：gopter（Go Property Testing）
- Mock工具：gomock
- 数据库测试：testcontainers-go（Docker容器）
- HTTP测试：httptest

**前端测试：**
- 测试框架：Vitest
- 组件测试：@vue/test-utils
- E2E测试：Playwright

### 属性测试配置

每个属性测试必须：
1. 运行至少100次迭代
2. 使用注释标记对应的设计属性
3. 生成随机测试数据
4. 验证系统不变量

**标记格式：**
```go
// Feature: c-user-management-system, Property 1: 用户注册创建账户
// For any 有效的手机号和验证码，调用注册接口应该成功创建用户账户
func TestProperty_UserRegistration(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("user registration creates account", 
        prop.ForAll(
            func(phone string, code string) bool {
                // 测试逻辑
                return true
            },
            genValidPhone(),
            genValidCode(),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### 测试覆盖范围

#### 1. Consumer Service 测试

**单元测试：**
- 用户注册（手机号格式验证、验证码验证、密码加密）
- 用户登录（密码验证、JWT生成、登录失败计数）
- 账户锁定（连续失败锁定、锁定时间验证）
- 风险评分（异常IP检测、频繁失败检测）
- 用户信息更新（字段验证、数据持久化）
- 账户注销（状态变更、数据保留）

**属性测试：**
- Property 1: 用户注册创建账户
- Property 2: 登录返回有效令牌
- Property 5: 连续失败锁定账户
- Property 6: 登录日志完整记录
- Property 7: 风险评分范围
- Property 9: 密码bcrypt加密
- Property 11: 用户信息更新持久化
- Property 13: 账户注销状态保留
- Property 14: 多租户数据隔离

#### 2. SMS Service 测试

**单元测试：**
- 验证码生成（6位数字、随机性）
- 验证码存储（Redis缓存、过期时间）
- 验证码验证（正确验证、错误验证、过期验证）
- 短信发送（阿里云通道、腾讯云通道）
- 频率限制（每分钟限制、每日限制）
- 短信日志记录

**属性测试：**
- Property 15: 验证码格式和有效期
- Property 16: 短信发送频率限制
- Property 17: 短信每日限额
- Property 18: 短信通道故障转移
- Property 19: 验证码一次性使用
- Property 20: 短信日志完整记录

#### 3. Payment Service 测试

**单元测试：**
- 支付订单创建（订单号生成、金额验证）
- 支付回调处理（签名验证、状态更新）
- 支付查询（订单状态查询）
- 退款操作（退款金额验证、退款状态）
- 订单超时关闭（定时任务测试）

**属性测试：**
- Property 21: 支付订单号唯一性
- Property 22: 支付订单超时关闭
- Property 23: 支付成功发布事件
- Property 24: 支付回调签名验证
- Property 25: 退款流水记录

#### 4. Finance Service 测试

**单元测试：**
- 账户创建（自动创建、初始余额）
- 充值操作（余额增加、流水记录）
- 提现申请（余额冻结、金额验证）
- 提现审核（通过/拒绝、余额变动）
- 余额查询（当前余额、冻结余额）
- 流水查询（分页、筛选）

**属性测试：**
- Property 26: 用户账户自动创建
- Property 27: 充值余额增加
- Property 28: 提现金额限制
- Property 29: 提现冻结和解冻
- Property 30: 余额非负约束
- Property 31: 金额精度保证
- Property 32: 财务流水完整性

#### 5. Media Service 测试

**单元测试：**
- 预签名URL生成（URL格式、过期时间）
- 文件上传确认（元数据保存、缩略图生成）
- 文件格式验证（允许的格式、拒绝的格式）
- 文件大小验证（图片5MB、视频100MB）
- 文件删除（软删除、查询过滤）

**属性测试：**
- Property 36: 文件格式验证
- Property 37: 文件大小限制
- Property 38: 预签名URL有效期
- Property 39: 缩略图自动生成
- Property 40: 媒体文件软删除

#### 6. Logistics Service 测试

**单元测试：**
- 物流查询（API调用、数据解析）
- 物流缓存（Redis缓存、过期时间）
- 物流订阅（状态变更检测）
- 事件发布（状态变更事件）

**属性测试：**
- Property 41: 物流信息缓存
- Property 42: 物流状态变更事件

#### 7. Freight Service 测试

**单元测试：**
- 按重量计算（首重续重、阶梯定价）
- 按距离计算（地区距离、价格计算）
- 包邮规则（满额包邮、地区包邮）
- 运费模板管理（创建、更新、查询）

**属性测试：**
- Property 43: 运费计算精度
- Property 44: 包邮规则优先
- Property 45: 首重续重阶梯计算

### 集成测试

**测试场景：**
1. 用户注册 → 短信验证 → 账户创建 → 财务账户创建
2. 用户登录 → JWT生成 → API调用 → 权限验证
3. 支付成功 → 事件发布 → 财务充值 → 余额增加
4. 文件上传 → OSS存储 → 缩略图生成 → 元数据保存
5. 物流查询 → 缓存读取 → API调用 → 状态更新 → 事件发布

**测试工具：**
- testcontainers-go（启动MySQL、Redis、Kafka容器）
- httptest（模拟HTTP请求）
- gomock（模拟第三方服务）

### 性能测试

**测试指标：**
- API响应时间（P50、P95、P99）
- 数据库查询时间
- 缓存命中率
- 并发处理能力（QPS）
- 系统资源使用（CPU、内存）

**测试工具：**
- Go benchmark
- Apache JMeter
- Grafana K6

**性能目标：**
- API响应时间 P95 < 200ms
- 数据库查询 < 100ms
- 缓存命中率 > 80%
- 并发处理 > 1000 QPS

### 安全测试

**测试内容：**
- SQL注入测试
- XSS攻击测试
- CSRF攻击测试
- JWT令牌安全测试
- 敏感数据脱敏测试
- API限流测试
- 权限控制测试

**测试工具：**
- OWASP ZAP
- Burp Suite
- 自定义安全测试脚本

### 测试数据生成

**属性测试数据生成器：**

```go
// 生成有效手机号
func genValidPhone() gopter.Gen {
    return gen.RegexMatch("^1[3-9]\\d{9}$")
}

// 生成验证码
func genValidCode() gopter.Gen {
    return gen.RegexMatch("^\\d{6}$")
}

// 生成金额（0.01-10000.00）
func genAmount() gopter.Gen {
    return gen.Float64Range(0.01, 10000.00).
        Map(func(f float64) decimal.Decimal {
            return decimal.NewFromFloat(f).Round(2)
        })
}

// 生成租户ID
func genTenantID() gopter.Gen {
    return gen.UInt32Range(1, 1000)
}
```

### 测试环境

**开发环境：**
- 本地Docker容器（MySQL、Redis、Kafka）
- Mock第三方服务（短信、支付、OSS）
- 测试数据自动生成和清理

**CI/CD环境：**
- GitHub Actions / GitLab CI
- 自动运行单元测试和属性测试
- 测试覆盖率报告（要求 > 70%）
- 测试失败阻止合并

**测试环境：**
- 独立的测试环境（与生产隔离）
- 真实的第三方服务（测试账号）
- 完整的集成测试和E2E测试

### 测试覆盖率目标

- 单元测试覆盖率：≥ 70%
- 属性测试覆盖率：所有 Correctness Properties
- 集成测试覆盖率：所有关键业务流程
- E2E测试覆盖率：所有用户场景


## 多租户设计

### 租户隔离策略

系统采用**共享数据库、共享Schema、行级隔离**的多租户架构模式。

**隔离机制：**
1. 所有数据表包含 `tenant_id` 字段
2. 所有查询自动添加 `tenant_id` 过滤条件
3. 所有写入自动设置当前租户的 `tenant_id`
4. 数据库索引包含 `tenant_id` 字段优化查询性能

### 租户上下文传递

**请求流程：**
```
1. API Gateway 从请求头或JWT令牌中提取 tenant_id
   ↓
2. 将 tenant_id 注入到 Context 中
   ↓
3. Service 层从 Context 获取 tenant_id
   ↓
4. Data 层使用 tenant_id 过滤数据
```

**代码示例：**

```go
// 中间件：提取租户ID
func TenantMiddleware() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从JWT令牌中提取租户ID
            claims, ok := jwt.FromContext(ctx)
            if !ok {
                return nil, errors.Unauthorized("UNAUTHORIZED", "missing tenant info")
            }
            
            // 注入租户ID到Context
            ctx = tenant.NewContext(ctx, claims.TenantID)
            
            return handler(ctx, req)
        }
    }
}

// Service层：获取租户ID
func (s *ConsumerService) ListConsumers(ctx context.Context, req *pagination.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
    tenantID, err := tenant.FromContext(ctx)
    if err != nil {
        return nil, err
    }
    
    // 传递租户ID到数据层
    return s.consumerRepo.List(ctx, tenantID, req)
}

// Data层：使用租户ID过滤
func (r *ConsumerRepo) List(ctx context.Context, tenantID uint32, req *pagination.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
    builder := r.entClient.Client().Consumer.Query().
        Where(consumer.TenantIDEQ(tenantID))  // 自动添加租户过滤
    
    // 执行查询...
}
```

### 租户配置管理

每个租户可以独立配置：

**配置项：**
- 短信服务配置（阿里云/腾讯云账号、签名、模板）
- 支付服务配置（微信/支付宝商户号、密钥）
- OSS配置（阿里云/腾讯云账号、Bucket）
- 物流服务配置（快递鸟API密钥）
- 业务规则配置（提现限额、运费模板）

**配置存储：**
```go
type TenantConfig struct {
    TenantID uint32
    
    // 短信配置
    SMSProvider string  // "aliyun" or "tencent"
    SMSAccessKey string
    SMSSecretKey string
    SMSSignName string
    
    // 支付配置
    WechatAppID string
    WechatMchID string
    WechatAPIKey string
    AlipayAppID string
    AlipayPrivateKey string
    
    // OSS配置
    OSSProvider string  // "aliyun" or "tencent"
    OSSAccessKey string
    OSSSecretKey string
    OSSBucket string
    OSSRegion string
    
    // 业务规则
    WithdrawMinAmount decimal.Decimal
    WithdrawMaxAmount decimal.Decimal
    WithdrawDailyLimit decimal.Decimal
}
```

### 租户数据隔离验证

**自动化测试：**
```go
// Property 14: 多租户数据隔离
func TestProperty_TenantDataIsolation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("tenant data isolation", 
        prop.ForAll(
            func(tenant1ID, tenant2ID uint32) bool {
                // 创建租户1的用户
                ctx1 := tenant.NewContext(context.Background(), tenant1ID)
                user1, _ := service.CreateConsumer(ctx1, &CreateConsumerRequest{...})
                
                // 创建租户2的用户
                ctx2 := tenant.NewContext(context.Background(), tenant2ID)
                user2, _ := service.CreateConsumer(ctx2, &CreateConsumerRequest{...})
                
                // 租户1查询用户列表，不应该包含租户2的用户
                list1, _ := service.ListConsumers(ctx1, &pagination.PagingRequest{})
                for _, u := range list1.Items {
                    if u.TenantID == tenant2ID {
                        return false  // 隔离失败
                    }
                }
                
                // 租户2查询用户列表，不应该包含租户1的用户
                list2, _ := service.ListConsumers(ctx2, &pagination.PagingRequest{})
                for _, u := range list2.Items {
                    if u.TenantID == tenant1ID {
                        return false  // 隔离失败
                    }
                }
                
                return true
            },
            gen.UInt32Range(1, 1000),
            gen.UInt32Range(1, 1000),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

## 安全设计

### 认证机制

**JWT令牌结构：**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "user_id": 12345,
    "tenant_id": 1,
    "phone": "138****1234",
    "exp": 1704067200,
    "iat": 1704060000,
    "jti": "unique-token-id"
  },
  "signature": "..."
}
```

**令牌生成：**
```go
func (s *ConsumerService) generateJWT(consumer *Consumer) (string, error) {
    claims := jwt.MapClaims{
        "user_id":   consumer.ID,
        "tenant_id": consumer.TenantID,
        "phone":     maskPhone(consumer.Phone),
        "exp":       time.Now().Add(2 * time.Hour).Unix(),
        "iat":       time.Now().Unix(),
        "jti":       uuid.New().String(),
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}
```

**令牌验证：**
```go
func JWTMiddleware(secret string) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从请求头提取令牌
            token := extractToken(ctx)
            if token == "" {
                return nil, errors.Unauthorized("UNAUTHORIZED", "missing token")
            }
            
            // 验证令牌
            claims, err := validateToken(token, secret)
            if err != nil {
                return nil, errors.Unauthorized("UNAUTHORIZED", "invalid token")
            }
            
            // 检查令牌是否过期
            if time.Now().Unix() > claims.ExpiresAt {
                return nil, errors.Unauthorized("UNAUTHORIZED", "token expired")
            }
            
            // 注入用户信息到Context
            ctx = auth.NewContext(ctx, claims)
            
            return handler(ctx, req)
        }
    }
}
```

### 密码安全

**密码加密：**
```go
import "golang.org/x/crypto/bcrypt"

func hashPassword(password string) (string, error) {
    // 使用bcrypt加密，cost=12
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

func verifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**密码策略：**
- 最小长度：8位
- 必须包含：大写字母、小写字母、数字
- 禁止使用：常见弱密码（123456、password等）
- 密码历史：不能与最近3次使用的密码相同

### 限流保护

**限流策略：**
```go
// 基于Redis的滑动窗口限流
func RateLimitMiddleware(redis *redis.Client) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            userID := auth.FromContext(ctx).UserID
            key := fmt.Sprintf("rate_limit:%d", userID)
            
            // 滑动窗口限流：1分钟内最多60次请求
            count, err := redis.Incr(ctx, key).Result()
            if err != nil {
                return nil, err
            }
            
            if count == 1 {
                redis.Expire(ctx, key, time.Minute)
            }
            
            if count > 60 {
                return nil, errors.TooManyRequests("RATE_LIMIT_EXCEEDED", "请求过于频繁，请稍后重试")
            }
            
            return handler(ctx, req)
        }
    }
}
```

### 敏感数据保护

**数据脱敏：**
```go
// 手机号脱敏：138****1234
func maskPhone(phone string) string {
    if len(phone) != 11 {
        return phone
    }
    return phone[:3] + "****" + phone[7:]
}

// 身份证号脱敏：110101********1234
func maskIDCard(idCard string) string {
    if len(idCard) != 18 {
        return idCard
    }
    return idCard[:6] + "********" + idCard[14:]
}

// 邮箱脱敏：u***@example.com
func maskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return email
    }
    username := parts[0]
    if len(username) <= 1 {
        return email
    }
    return username[:1] + "***@" + parts[1]
}
```

**数据加密存储：**
- 密码：bcrypt加密
- 敏感配置：AES-256加密
- 通信：HTTPS/TLS 1.3

### 防护措施

**SQL注入防护：**
- 使用ORM（Ent）自动参数化查询
- 禁止拼接SQL字符串
- 输入参数验证和转义

**XSS防护：**
- 输出HTML转义
- Content-Security-Policy头
- 输入参数过滤

**CSRF防护：**
- CSRF Token验证
- SameSite Cookie
- Referer检查

**IP黑名单：**
```go
func IPBlacklistMiddleware(redis *redis.Client) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            ip := extractIP(ctx)
            
            // 检查IP是否在黑名单中
            exists, _ := redis.SIsMember(ctx, "ip_blacklist", ip).Result()
            if exists {
                return nil, errors.Forbidden("FORBIDDEN", "IP已被封禁")
            }
            
            return handler(ctx, req)
        }
    }
}
```


## 部署架构

### 服务部署

**容器化部署（Docker + Kubernetes）：**

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: consumer-service
  template:
    metadata:
      labels:
        app: consumer-service
    spec:
      containers:
      - name: consumer-service
        image: registry.example.com/consumer-service:v1.0.0
        ports:
        - containerPort: 8080  # HTTP
        - containerPort: 9090  # gRPC
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: consumer-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: consumer-secrets
              key: redis-url
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### 数据库部署

**MySQL主从架构：**
```
┌─────────────┐
│ MySQL Master│  (读写)
└──────┬──────┘
       │ 主从复制
       ├──────────┬──────────┐
       ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│MySQL Slave│ │MySQL Slave│ │MySQL Slave│  (只读)
└──────────┘ └──────────┘ └──────────┘
```

**读写分离：**
- 写操作：连接Master
- 读操作：连接Slave（负载均衡）
- 事务操作：连接Master

**数据库配置：**
```yaml
database:
  master:
    host: mysql-master.example.com
    port: 3306
    database: consumer_db
    max_connections: 100
    max_idle_connections: 10
    connection_lifetime: 3600s
  
  slaves:
    - host: mysql-slave-1.example.com
      port: 3306
    - host: mysql-slave-2.example.com
      port: 3306
    - host: mysql-slave-3.example.com
      port: 3306
```

### 缓存部署

**Redis集群：**
```
┌─────────────────────────────────────┐
│         Redis Cluster               │
│  ┌──────┐  ┌──────┐  ┌──────┐      │
│  │Master│  │Master│  │Master│      │
│  │ 0-5k │  │5k-10k│  │10k-16k│     │
│  └───┬──┘  └───┬──┘  └───┬──┘      │
│      │         │         │          │
│  ┌───▼──┐  ┌──▼───┐  ┌──▼───┐      │
│  │Slave │  │Slave │  │Slave │      │
│  └──────┘  └──────┘  └──────┘      │
└─────────────────────────────────────┘
```

**缓存策略：**
- 用户信息：TTL 30分钟
- 验证码：TTL 5分钟
- 物流信息：TTL 30分钟
- 配置信息：TTL 1小时
- 限流计数：TTL 1分钟

### 消息队列部署

**Kafka集群：**
```
┌─────────────────────────────────────┐
│         Kafka Cluster               │
│  ┌────────┐  ┌────────┐  ┌────────┐│
│  │Broker 1│  │Broker 2│  │Broker 3││
│  └────────┘  └────────┘  └────────┘│
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│         ZooKeeper Cluster           │
│  ┌────────┐  ┌────────┐  ┌────────┐│
│  │  ZK 1  │  │  ZK 2  │  │  ZK 3  ││
│  └────────┘  └────────┘  └────────┘│
└─────────────────────────────────────┘
```

**Topic配置：**
- user-events: 用户相关事件（注册、登录）
- payment-events: 支付相关事件（支付成功、退款）
- logistics-events: 物流相关事件（状态变更）
- 分区数：3
- 副本数：2
- 保留时间：7天

### 监控和日志

**监控系统（Prometheus + Grafana）：**
```
┌─────────────┐
│  Services   │
│  (Metrics)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Prometheus  │  (采集和存储)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Grafana    │  (可视化)
└─────────────┘
```

**监控指标：**
- 系统指标：CPU、内存、磁盘、网络
- 应用指标：QPS、响应时间、错误率
- 业务指标：注册量、登录量、支付量
- 数据库指标：连接数、查询时间、慢查询
- 缓存指标：命中率、内存使用、连接数

**日志系统（ELK Stack）：**
```
┌─────────────┐
│  Services   │
│   (Logs)    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Filebeat   │  (日志采集)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Logstash    │  (日志处理)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Elasticsearch│  (日志存储)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Kibana    │  (日志查询)
└─────────────┘
```

**日志级别：**
- DEBUG: 开发环境
- INFO: 测试环境
- WARN: 生产环境（可恢复的错误）
- ERROR: 生产环境（系统错误）
- FATAL: 生产环境（致命错误）

### 高可用设计

**服务高可用：**
- 多实例部署（至少3个实例）
- 负载均衡（Kubernetes Service）
- 健康检查（Liveness + Readiness）
- 自动重启（Kubernetes自动恢复）
- 滚动更新（零停机部署）

**数据库高可用：**
- 主从复制（Master-Slave）
- 自动故障转移（MHA或Orchestrator）
- 定期备份（每日全量 + 每小时增量）
- 备份保留（30天）

**缓存高可用：**
- Redis Cluster（分片 + 副本）
- 自动故障转移
- 持久化（AOF + RDB）

**消息队列高可用：**
- Kafka集群（多Broker）
- 副本机制（Replication Factor = 2）
- ZooKeeper集群（奇数节点）

### 灾难恢复

**备份策略：**
- 数据库：每日全量备份 + 每小时增量备份
- Redis：每小时RDB快照 + AOF持久化
- 配置文件：Git版本控制
- 日志：保留30天

**恢复流程：**
1. 评估故障范围和影响
2. 切换到备用系统（如果可用）
3. 从备份恢复数据
4. 验证数据完整性
5. 恢复服务
6. 监控系统状态
7. 事后分析和改进

**RTO/RPO目标：**
- RTO（恢复时间目标）：< 1小时
- RPO（恢复点目标）：< 1小时

## 总结

### 设计亮点

1. **独立的Consumer服务**：与管理后台用户系统（identity）完全隔离，使用独立的Protobuf package（consumer.service.v1）和数据表

2. **多租户架构**：支持租户级数据隔离和配置，所有数据表包含tenant_id字段，自动过滤和验证

3. **事件驱动设计**：通过eventbus实现模块间解耦，支持异步处理和重试机制

4. **完善的安全机制**：JWT认证、bcrypt密码加密、限流保护、风险评分、敏感数据脱敏

5. **故障转移机制**：短信服务主备通道自动切换，支付服务多通道支持

6. **属性测试覆盖**：57个Correctness Properties，确保系统正确性和健壮性

7. **金额精度保证**：使用decimal类型避免浮点误差，确保财务计算准确

8. **高可用部署**：多实例部署、主从复制、集群架构、自动故障转移

### 技术选型理由

- **Kratos框架**：成熟的微服务框架，支持gRPC和HTTP，内置中间件和依赖注入
- **Ent ORM**：类型安全的ORM，支持代码生成和Schema迁移
- **Redis**：高性能缓存，支持多种数据结构和持久化
- **Kafka**：高吞吐量消息队列，支持事件驱动架构
- **Protobuf**：高效的序列化协议，支持跨语言通信
- **gopter**：Go语言的属性测试库，支持随机测试数据生成

### 后续工作

1. **实现阶段**：
   - 创建Protobuf定义
   - 生成gRPC代码
   - 实现Service层和Data层
   - 编写单元测试和属性测试

2. **集成阶段**：
   - 集成第三方服务（短信、支付、OSS、物流）
   - 实现事件总线
   - 配置多租户支持

3. **测试阶段**：
   - 单元测试（覆盖率 ≥ 70%）
   - 属性测试（所有57个Properties）
   - 集成测试（关键业务流程）
   - 性能测试（API响应时间、并发能力）
   - 安全测试（SQL注入、XSS、CSRF）

4. **部署阶段**：
   - 容器化（Docker）
   - 编排（Kubernetes）
   - 监控（Prometheus + Grafana）
   - 日志（ELK Stack）
   - CI/CD（GitHub Actions / GitLab CI）

5. **运维阶段**：
   - 性能优化
   - 故障排查
   - 容量规划
   - 安全加固
   - 文档维护

