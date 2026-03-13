# C端用户管理模块开发计划

**项目：** GoWind Admin - Task 1.1 C端用户管理  
**创建时间：** 2026-03-13  
**状态：** 后端 85% 完成，前端未开始

---

## 一、总体目标

实现完整的C端用户管理系统，包括：
- 手机号注册/登录
- 微信OAuth登录
- 用户信息管理（CRUD）
- 登录日志查询
- 账户安全（失败锁定、风险评分）

---

## 二、已完成工作（85%）

### 2.1 数据库设计 ✅
- **文件：** `backend/app/admin/service/internal/data/ent/schema/customer.go`
- **表：** `c_customers`
- **字段：** mobile, nickname, avatar, wechat_openid, wechat_unionid, register_source, status, etc.

- **文件：** `backend/app/admin/service/internal/data/ent/schema/customer_login_log.go`
- **表：** `c_customer_login_logs`
- **字段：** customer_id, mobile, ip_address, login_type, status, risk_score, etc.

### 2.2 API 定义 ✅
- **文件：** `backend/api/protos/customer/service/v1/customer.proto`
- **服务：** CustomerService
- **方法：**
  - RegisterByMobile - 手机号注册
  - LoginByMobile - 手机号登录
  - LoginByWechat - 微信登录
  - GetCustomer - 获取用户信息
  - UpdateCustomer - 更新用户信息
  - ListCustomers - 用户列表查询
  - DeleteCustomer - 删除用户
  - ListLoginLogs - 登录日志查询

- **文件：** `backend/api/protos/customer/service/v1/customer_error.proto`
- **错误码：** CUSTOMER_NOT_FOUND, CUSTOMER_ALREADY_EXISTS, CUSTOMER_DISABLED, etc.

### 2.3 Protobuf 代码生成 ✅
- **目录：** `backend/api/gen/go/customer/service/v1/`
- **文件：** customer.pb.go, customer_grpc.pb.go, customer_http.pb.go, customer_error.pb.go

### 2.4 Repository 层 ✅
- **文件：** `backend/app/admin/service/internal/data/customer_repo.go`
- **接口：** CustomerRepo
- **实现：** customerRepo
- **方法：**
  - List - 用户列表（支持手机号、昵称、状态、注册来源等过滤）
  - Get - 获取单个用户
  - GetByMobile - 根据手机号查询
  - GetByWechatOpenID - 根据微信OpenID查询
  - Create - 创建用户（简化版）
  - Update - 更新用户信息
  - Delete - 删除用户
  - CreateLoginLog - 创建登录日志
  - ListLoginLogs - 登录日志列表

### 2.5 Service 层 ✅（简化版）
- **文件：** `backend/app/admin/service/internal/service/customer_service.go`
- **实现：** CustomerService
- **状态：** 基础框架完成，核心业务逻辑待实现

### 2.6 依赖注入 ✅
- **文件：** `backend/app/admin/service/internal/service/providers/wire_set.go`
- **添加：** service.NewCustomerService

- **文件：** `backend/app/admin/service/internal/data/providers/wire_set.go`
- **添加：** data.NewCustomerRepo

- **文件：** `backend/app/admin/service/internal/server/rest_server.go`
- **注册：** customerV1.RegisterCustomerServiceHTTPServer

### 2.7 代码生成 ✅
- Wire 依赖注入代码已生成
- Ent ORM 代码已生成（customer, customerloginlog）

---

## 三、待完成工作（15%）

### 3.1 后端核心业务逻辑（优先级：高）

#### 3.1.1 修复编译错误
**文件：** `backend/app/admin/service/internal/service/customer_service.go`

**问题：**
- 缺少 `pb.ErrorNotImplemented` 错误定义
- DeleteCustomer 返回类型不匹配

**解决方案：**
```bash
cd backend && go build ./app/admin/service/...
```
根据编译错误逐一修复。

#### 3.1.2 实现手机号注册
**文件：** `backend/app/admin/service/internal/service/customer_service.go`

**功能：**
1. 验证短信验证码（需集成短信服务）
2. 检查手机号是否已注册
3. 密码哈希（使用 bcrypt）
4. 创建用户记录
5. 生成 JWT token
6. 记录登录日志

**依赖：**
- 短信验证服务（Task 1.2）
- JWT token 生成工具
- 密码哈希工具：`github.com/tx7do/go-utils/crypto/password`

**参考代码：**
```go
import "github.com/tx7do/go-utils/crypto/password"

hashedPassword, err := password.HashPassword(req.Password)
```

#### 3.1.3 实现手机号登录
**文件：** `backend/app/admin/service/internal/service/customer_service.go`

**功能：**
1. 根据手机号查询用户
2. 验证密码
3. 检查账户状态（禁用、锁定）
4. 失败次数统计（5次失败锁定30分钟）
5. 生成 JWT token
6. 记录登录日志（成功/失败）
7. 风险评分（IP变化、设备变化）

**参考代码：**
```go
// 验证密码
if !password.CheckPasswordHash(req.Password, customer.PasswordHash) {
    // 记录失败日志
    // 增加失败次数
    // 检查是否需要锁定
    return nil, pb.ErrorInvalidPassword("invalid password")
}
```

#### 3.1.4 实现微信登录
**文件：** `backend/app/admin/service/internal/service/customer_service.go`

**功能：**
1. 验证微信 code
2. 获取 openid 和 unionid
3. 查询或创建用户
4. 生成 JWT token
5. 记录登录日志

**依赖：**
- 微信 OAuth SDK（Task 1.5）

#### 3.1.5 完善 customer_repo.go 的 Create 方法
**文件：** `backend/app/admin/service/internal/data/customer_repo.go`

**当前问题：**
- 只设置了 mobile 和 nickname
- 缺少 password_hash, register_source 等字段

**需要添加：**
```go
builder := r.entClient.Client().Customer.Create().
    SetMobile(data.Mobile).
    SetNickname(data.Nickname)

// 添加可选字段
if data.Avatar != "" {
    builder.SetAvatar(data.Avatar)
}
if data.Realname != "" {
    builder.SetRealname(data.Realname)
}
// ... 其他字段
```

**注意：**
- Protobuf 字段名和 Ent 字段名可能不同
- 使用 `SetNillable*` 方法需要传指针
- 枚举字段使用 `r.xxxConverter.ToEntity(&data.Xxx)`

### 3.2 前端实现（优先级：中）

#### 3.2.1 创建 Pinia Store
**文件：** `frontend/apps/admin/src/stores/customer.state.ts`

**内容：**
```typescript
import { defineStore } from 'pinia';
import { createCustomerServiceClient } from '#/generated/api/customer/service/v1';
import { requestClientRequestHandler } from '#/utils/request';

export const useCustomerListStore = defineStore('customer-list', () => {
  const service = createCustomerServiceClient(requestClientRequestHandler);

  async function listCustomers(params: {
    mobile?: string;
    nickname?: string;
    status?: number;
    page?: number;
    pageSize?: number;
  }) {
    return await service.listCustomers(params);
  }

  async function getCustomer(id: number) {
    return await service.getCustomer({ id });
  }

  async function updateCustomer(id: number, data: any) {
    return await service.updateCustomer({ id, ...data });
  }

  async function deleteCustomer(id: number) {
    return await service.deleteCustomer({ id });
  }

  return { listCustomers, getCustomer, updateCustomer, deleteCustomer };
});

export const useCustomerLoginLogStore = defineStore('customer-login-log', () => {
  const service = createCustomerServiceClient(requestClientRequestHandler);

  async function listLoginLogs(params: {
    customerId?: number;
    mobile?: string;
    page?: number;
    pageSize?: number;
  }) {
    return await service.listLoginLogs(params);
  }

  return { listLoginLogs };
});
```

**参考文件：**
- `frontend/apps/admin/src/stores/user.state.ts`

#### 3.2.2 创建用户列表页面
**文件：** `frontend/apps/admin/src/views/customer/user/index.vue`

**功能：**
- 用户列表表格
- 搜索过滤（手机号、昵称、状态）
- 分页
- 操作按钮（查看、编辑、删除）

**参考文件：**
- `frontend/apps/admin/src/views/system/user/index.vue`

**表格列：**
- ID
- 手机号
- 昵称
- 头像
- 注册来源
- 状态
- 注册时间
- 操作

#### 3.2.3 创建用户详情页面
**文件：** `frontend/apps/admin/src/views/customer/user/detail.vue`

**功能：**
- 用户信息展示/编辑表单
- 基本信息（手机号、昵称、头像、真实姓名）
- 扩展信息（邮箱、性别、生日、地址）
- 微信信息（openid、unionid）
- 保存按钮

**参考文件：**
- `frontend/apps/admin/src/views/system/user/detail.vue`

#### 3.2.4 创建登录日志页面
**文件：** `frontend/apps/admin/src/views/customer/login-log/index.vue`

**功能：**
- 登录日志列表表格
- 搜索过滤（用户ID、手机号、IP地址）
- 分页

**表格列：**
- 用户ID
- 手机号
- IP地址
- 地理位置
- 设备信息
- 登录方式
- 登录状态
- 风险等级
- 登录时间

#### 3.2.5 配置路由
**文件：** `frontend/apps/admin/src/router/routes/modules/customer.ts`

**内容：**
```typescript
import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/customer',
    name: 'Customer',
    component: () => import('#/layouts/basic.vue'),
    meta: {
      title: 'C端用户管理',
      icon: 'mdi:account-group',
    },
    children: [
      {
        path: 'user',
        name: 'CustomerUser',
        component: () => import('#/views/customer/user/index.vue'),
        meta: {
          title: '用户列表',
          permission: 'customer:user:list',
        },
      },
      {
        path: 'user/:id',
        name: 'CustomerUserDetail',
        component: () => import('#/views/customer/user/detail.vue'),
        meta: {
          title: '用户详情',
          permission: 'customer:user:view',
          hideInMenu: true,
        },
      },
      {
        path: 'login-log',
        name: 'CustomerLoginLog',
        component: () => import('#/views/customer/login-log/index.vue'),
        meta: {
          title: '登录日志',
          permission: 'customer:login-log:list',
        },
      },
    ],
  },
];

export default routes;
```

**注册路由：**
在 `frontend/apps/admin/src/router/routes/index.ts` 中导入并添加。

### 3.3 集成测试（优先级：中）

#### 3.3.1 后端编译验证
```bash
cd backend
go build ./app/admin/service/...
```

#### 3.3.2 启动后端服务
```bash
cd backend/app/admin/service
go run cmd/server/main.go
```

#### 3.3.3 测试 API
使用 Swagger UI 或 curl 测试：
- 访问：http://localhost:7788/docs/openapi.yaml
- 测试 ListCustomers
- 测试 GetCustomer
- 测试 UpdateCustomer
- 测试 ListLoginLogs

#### 3.3.4 前端编译验证
```bash
cd frontend
pnpm vue-tsc --noEmit
pnpm build
```

#### 3.3.5 启动前端服务
```bash
cd frontend
pnpm dev
```
访问：http://localhost:5666

#### 3.3.6 端到端测试
1. 登录管理后台
2. 访问 C端用户管理 → 用户列表
3. 测试搜索、分页
4. 测试查看用户详情
5. 测试编辑用户信息
6. 访问登录日志页面
7. 测试日志查询

### 3.4 数据库迁移（优先级：高）

#### 3.4.1 生成迁移文件
```bash
cd backend/app/admin/service
# 使用 Ent 的迁移工具或手动创建 SQL
```

#### 3.4.2 执行迁移
```sql
-- 创建 c_customers 表
CREATE TABLE c_customers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  tenant_id INT NOT NULL,
  mobile VARCHAR(20) NOT NULL,
  nickname VARCHAR(50),
  avatar VARCHAR(255),
  realname VARCHAR(50),
  email VARCHAR(100),
  gender ENUM('SECRET', 'MALE', 'FEMALE') DEFAULT 'SECRET',
  birthday DATE,
  province VARCHAR(50),
  city VARCHAR(50),
  address VARCHAR(255),
  wechat_openid VARCHAR(100),
  wechat_unionid VARCHAR(100),
  register_source ENUM('MOBILE', 'WECHAT', 'ADMIN') NOT NULL,
  referrer_id BIGINT,
  status ENUM('NORMAL', 'DISABLED', 'LOCKED', 'DELETED') DEFAULT 'NORMAL',
  failed_login_attempts INT DEFAULT 0,
  locked_until DATETIME,
  last_login_at DATETIME,
  last_login_ip VARCHAR(50),
  remark TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME,
  created_by BIGINT,
  updated_by BIGINT,
  UNIQUE KEY uk_tenant_mobile (tenant_id, mobile),
  UNIQUE KEY uk_tenant_openid (tenant_id, wechat_openid),
  INDEX idx_mobile (mobile),
  INDEX idx_nickname (nickname),
  INDEX idx_status (status)
);

-- 创建 c_customer_login_logs 表
CREATE TABLE c_customer_login_logs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  tenant_id INT NOT NULL,
  customer_id BIGINT NOT NULL,
  mobile VARCHAR(20),
  ip_address VARCHAR(50),
  geo_location VARCHAR(100),
  device_info VARCHAR(255),
  user_agent TEXT,
  login_type ENUM('MOBILE', 'WECHAT', 'SMS_CODE') NOT NULL,
  status ENUM('SUCCESS', 'FAILED', 'LOCKED') NOT NULL,
  failure_reason VARCHAR(255),
  risk_score INT DEFAULT 0,
  risk_level ENUM('LOW', 'MEDIUM', 'HIGH') DEFAULT 'LOW',
  created_at DATETIME NOT NULL,
  INDEX idx_customer_id (customer_id),
  INDEX idx_mobile (mobile),
  INDEX idx_ip (ip_address),
  INDEX idx_created_at (created_at)
);
```

---

## 四、技术难点与解决方案

### 4.1 枚举类型转换
**问题：** Protobuf 枚举和 Ent 枚举命名不一致

**解决方案：**
- Protobuf: `Customer_NORMAL` (int32)
- Ent: `StatusNormal` (string)
- 使用 `mapper.NewEnumTypeConverter` 自动转换
- 调用 `ToEntity(&pbEnum)` 转换为 Ent 枚举

### 4.2 分页参数
**问题：** Protobuf 分页参数是指针类型

**解决方案：**
```go
if req.Paging != nil {
    offset := int((*req.Paging.Page - 1) * *req.Paging.PageSize)
    query.Offset(offset).Limit(int(*req.Paging.PageSize))
}
```

### 4.3 密码哈希
**问题：** 需要安全的密码存储

**解决方案：**
```go
import "github.com/tx7do/go-utils/crypto/password"

// 注册时
hashedPassword, err := password.HashPassword(plainPassword)

// 登录时
isValid := password.CheckPasswordHash(plainPassword, hashedPassword)
```

### 4.4 JWT Token 生成
**问题：** 需要生成访问令牌和刷新令牌

**解决方案：**
参考现有的 `AuthenticationService` 实现：
- 文件：`backend/app/admin/service/internal/service/authentication_service.go`
- 使用 JWT 库生成 token
- 设置过期时间（access: 2h, refresh: 7d）

### 4.5 账户锁定逻辑
**问题：** 5次失败后锁定30分钟

**解决方案：**
```go
// 登录失败时
customer.FailedLoginAttempts++
if customer.FailedLoginAttempts >= 5 {
    customer.LockedUntil = time.Now().Add(30 * time.Minute)
    customer.Status = pb.Customer_LOCKED
}

// 登录成功时
customer.FailedLoginAttempts = 0
customer.LockedUntil = nil
```

---

## 五、开发顺序建议

### 阶段 1：修复编译错误（30分钟）
1. 修复 `customer_service.go` 编译错误
2. 完善 `customer_repo.go` 的 Create 方法
3. 验证后端编译通过

### 阶段 2：实现核心业务逻辑（2小时）
1. 实现手机号注册（不含短信验证）
2. 实现手机号登录（含密码验证、失败锁定）
3. 实现用户信息更新
4. 测试 API

### 阶段 3：前端基础页面（2小时）
1. 创建 Pinia Store
2. 创建用户列表页面
3. 创建用户详情页面
4. 配置路由

### 阶段 4：登录日志（1小时）
1. 完善登录日志记录逻辑
2. 创建登录日志页面
3. 测试日志查询

### 阶段 5：集成测试（1小时）
1. 端到端测试
2. 修复 bug
3. 优化用户体验

### 阶段 6：微信登录（预留，依赖 Task 1.5）
1. 集成微信 OAuth SDK
2. 实现微信登录逻辑
3. 测试微信登录流程

---

## 六、关键文件清单

### 后端文件
```
backend/
├── api/protos/customer/service/v1/
│   ├── customer.proto ✅
│   └── customer_error.proto ✅
├── api/gen/go/customer/service/v1/ ✅
├── app/admin/service/internal/
│   ├── data/
│   │   ├── ent/schema/
│   │   │   ├── customer.go ✅
│   │   │   └── customer_login_log.go ✅
│   │   ├── customer_repo.go ✅ (需完善)
│   │   └── providers/wire_set.go ✅
│   ├── service/
│   │   ├── customer_service.go ✅ (需实现)
│   │   └── providers/wire_set.go ✅
│   └── server/
│       └── rest_server.go ✅
```

### 前端文件
```
frontend/apps/admin/src/
├── stores/
│   └── customer.state.ts ❌ (待创建)
├── views/customer/
│   ├── user/
│   │   ├── index.vue ❌ (待创建)
│   │   └── detail.vue ❌ (待创建)
│   └── login-log/
│       └── index.vue ❌ (待创建)
└── router/routes/modules/
    └── customer.ts ❌ (待创建)
```

---

## 七、验证清单

### 后端验证
- [ ] `go build ./app/admin/service/...` 编译通过
- [ ] `go test ./app/admin/service/internal/service/` 测试通过
- [ ] Swagger UI 可访问
- [ ] ListCustomers API 返回正确数据
- [ ] GetCustomer API 返回正确数据
- [ ] UpdateCustomer API 更新成功
- [ ] ListLoginLogs API 返回正确数据

### 前端验证
- [ ] `pnpm vue-tsc --noEmit` 类型检查通过
- [ ] `pnpm eslint` 无错误
- [ ] `pnpm build` 构建成功
- [ ] 用户列表页面正常显示
- [ ] 搜索和分页功能正常
- [ ] 用户详情页面正常显示
- [ ] 编辑用户信息成功
- [ ] 登录日志页面正常显示

### 集成验证
- [ ] 后端服务启动成功
- [ ] 前端服务启动成功
- [ ] 前端可调用后端 API
- [ ] 数据正确展示
- [ ] 操作正确执行

---

## 八、注意事项

### 8.1 代码规范
- 遵循 Go 代码规范（gofmt, golangci-lint）
- 遵循 Vue 代码规范（eslint, prettier）
- 使用 TypeScript 类型定义
- 添加必要的注释

### 8.2 安全考虑
- 密码必须哈希存储（bcrypt）
- 敏感信息不能记录到日志
- API 需要认证授权
- 防止 SQL 注入（使用 Ent ORM）
- 防止 XSS 攻击（前端输入验证）

### 8.3 性能优化
- 数据库查询添加索引
- 分页查询避免全表扫描
- 使用缓存减少数据库压力
- 前端列表使用虚拟滚动（大数据量）

### 8.4 错误处理
- 所有错误必须正确处理
- 返回友好的错误信息
- 记录详细的错误日志
- 前端显示用户友好的提示

---

## 九、后续任务

完成 Task 1.1 后，继续以下任务：

实现完整的C端用户管理系统，包括：
  - 手机号注册/登录
  - 微信OAuth登录
  - 用户信息管理（CRUD）
  - 登录日志查询
  - 账户安全（失败锁定、风险评分）
  - 
- **Task 1.2:** 短信服务集成（阿里云/腾讯云）
- **Task 1.3:** 支付模块（微信支付/支付宝）
- **Task 1.4:** 财务管理（账户余额、充值、提现）
- **Task 1.5:** 微信集成（OAuth登录、公众号、小程序）
- **Task 1.6:** 媒体管理（图片、视频上传、OSS）
- **Task 1.7:** 物流管理（快递查询、物流跟踪）
- **Task 1.8:** 运费计算（按重量、按距离）
- **Task 1.9:** 域名管理（域名解析、SSL证书）

---

## 十、联系方式

如有问题，请联系：
- 微信：`yang_lin_bo`（备注：`go-wind-admin`）
- 掘金专栏：[go-wind-admin](https://juejin.cn/column/7541283508041826367)

---

**文档版本：** v1.0  
**最后更新：** 2026-03-13 21:02
