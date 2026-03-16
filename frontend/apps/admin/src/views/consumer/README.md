# C端用户管理系统 - 前端页面组件

## 已完成的页面

### 1. 用户认证模块 (auth/)
- ✅ `register.vue` - 用户注册页面
  - 手机号注册
  - 验证码验证
  - 密码设置
  - 昵称设置（可选）
  
- ✅ `login.vue` - 用户登录页面
  - 手机号登录
  - 微信登录
  - 登录状态保持（JWT存储）

### 2. 用户信息管理模块 (user/)
- ✅ `profile.vue` - 个人信息页面
  - 查看用户信息
  - 编辑昵称、邮箱
  - 显示账户状态和风险评分
  
- ✅ `avatar.vue` - 头像上传页面
  - 图片上传（支持拖拽）
  - 文件格式验证（JPEG/PNG/GIF）
  - 文件大小限制（5MB）
  - 上传进度显示
  
- ✅ `security.vue` - 安全设置页面
  - 修改手机号（需验证码）
  - 修改邮箱（需验证码）
  - 修改密码
  
- ✅ `login-logs.vue` - 登录日志页面
  - 分页查询登录日志
  - 按手机号、登录方式、状态筛选
  - 按时间范围筛选
  - 数据表格展示

### 3. 财务管理模块 (finance/)
- ✅ `account.vue` - 账户余额页面
  - 显示总余额、可用余额、冻结余额
  - 账户信息展示
  - 快捷操作（充值、提现、查看流水）
  
- ✅ `recharge.vue` - 充值页面
  - 充值金额输入
  - 支付方式选择（微信、支付宝）
  - 支付类型选择（APP、H5、扫码）
  - 二维码支付弹窗
  - 支付状态轮询
  
- ✅ `withdraw.vue` - 提现页面
  - 提现金额输入（10-5000元）
  - 银行账户信息
  - 余额验证
  - 提现说明
  
- ✅ `transactions.vue` - 财务流水页面
  - 分页查询流水
  - 按交易类型筛选
  - 按时间范围筛选
  - 导出CSV功能

## 待完成的页面

### 4. 媒体管理模块 (media/)
- ⏳ `upload.vue` - 媒体上传页面
- ⏳ `list.vue` - 媒体文件列表页面

### 5. 物流查询模块 (logistics/)
- ⏳ `query.vue` - 物流查询页面
- ⏳ `tracking.vue` - 物流轨迹页面

### 6. 运费计算模块 (freight/)
- ⏳ `calculator.vue` - 运费计算器页面
- ⏳ `templates.vue` - 运费模板管理页面

## 技术栈

- **框架**: Vue 3 + TypeScript
- **UI组件**: Vben Admin + Ant Design Vue
- **状态管理**: Pinia
- **表单验证**: Zod
- **表格组件**: VxeTable
- **日期处理**: Day.js

## Store 引用

所有 Store 都从 `@vben/stores` 包导出，在 `frontend/apps/admin/src/stores/` 目录下创建了引用文件：

- `consumer.state.ts` - C端用户状态管理
- `sms.state.ts` - 短信服务状态管理
- `finance.state.ts` - 财务服务状态管理
- `payment.state.ts` - 支付服务状态管理
- `media.state.ts` - 媒体服务状态管理

## 路由配置

需要在 `frontend/apps/admin/src/router/routes/` 目录下创建路由配置文件，将这些页面注册到路由系统中。

示例路由结构：
```
/consumer
  /auth
    /register - 注册
    /login - 登录
  /user
    /profile - 个人信息
    /avatar - 上传头像
    /security - 安全设置
    /login-logs - 登录日志
  /finance
    /account - 账户余额
    /recharge - 充值
    /withdraw - 提现
    /transactions - 财务流水
  /media
    /upload - 上传媒体
    /list - 媒体列表
  /logistics
    /query - 物流查询
    /tracking - 物流轨迹
  /freight
    /calculator - 运费计算
    /templates - 运费模板
```

## 下一步工作

1. 完成剩余的媒体管理、物流查询、运费计算页面
2. 创建路由配置文件
3. 集成 gRPC 客户端，替换 Store 中的模拟数据
4. 添加国际化支持
5. 编写单元测试和集成测试
6. 优化用户体验和界面设计

## 注意事项

1. 所有页面都使用了 Vben Admin 的 `Page` 组件作为容器
2. 表单验证使用 Zod schema
3. 所有 API 调用都通过 Store 进行，便于后续集成真实的 gRPC 客户端
4. 敏感数据（如手机号）需要脱敏显示
5. 所有金额计算使用字符串类型，避免精度丢失
6. 文件上传使用预签名 URL 方式，前端直传 OSS
7. 支付状态使用轮询方式查询，避免长时间等待
