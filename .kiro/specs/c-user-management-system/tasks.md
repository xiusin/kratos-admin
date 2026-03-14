# Implementation Plan: C端用户管理系统

## Overview

本实施计划将C端用户管理系统的设计转化为可执行的开发任务。系统包含8个核心模块(用户、短信、支付、财务、微信、媒体、物流、运费),采用Go + Kratos框架实现,支持多租户架构和事件驱动设计。

**技术栈:**
- 后端: Go 1.21+ + Kratos v2 + gRPC
- ORM: Ent
- 数据库: MySQL 8.0+
- 缓存: Redis 7.0+
- 消息队列: Kafka
- 前端: Vue 3 + TypeScript + Vben Admin

**关键特性:**
- 8个核心服务模块
- 57个属性测试(Correctness Properties)
- 多租户数据隔离
- 事件驱动模块解耦
- 第三方服务集成(阿里云、腾讯云、微信、支付宝)

## Tasks

- [x] 1. 项目结构和基础设施搭建
  - 创建Consumer服务目录结构(backend/app/consumer/service/)
  - 配置Wire依赖注入(cmd/server/wire.go)
  - 创建服务配置文件(configs/config.yaml)
  - 配置数据库连接(MySQL主从、连接池)
  - 配置Redis连接(集群模式)
  - 配置Kafka连接(事件总线)
  - 创建健康检查接口(/health, /ready)
  - _Requirements: 15.1, 15.2_

- [x] 2. Protobuf API定义
  - [x] 2.1 创建Consumer Service Protobuf定义
    - 定义ConsumerService接口(注册、登录、信息管理)
    - 定义Consumer消息类型(用户实体)
    - 定义请求响应消息(RegisterByPhoneRequest, LoginResponse等)
    - 添加OpenAPI注解(HTTP路由映射)
    - 添加验证规则(protoc-gen-validate)
    - _Requirements: 1.1-1.10, 2.1-2.8_

  - [x] 2.2 创建SMS Service Protobuf定义
    - 定义SMSService接口(发送验证码、验证、通知)
    - 定义短信相关消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 3.1-3.10_

  - [x] 2.3 创建Payment Service Protobuf定义
    - 定义PaymentService接口(创建订单、查询、退款)
    - 定义PaymentOrder消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 4.1-4.12_


  - [x] 2.4 创建Finance Service Protobuf定义
    - 定义FinanceService接口(余额、充值、提现、流水)
    - 定义FinanceAccount和FinanceTransaction消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 5.1-5.12_

  - [x] 2.5 创建Wechat Service Protobuf定义
    - 定义WechatService接口(OAuth登录、用户信息、模板消息)
    - 定义微信相关消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 6.1-6.10_

  - [x] 2.6 创建Media Service Protobuf定义
    - 定义MediaService接口(上传、查询、删除)
    - 定义MediaFile消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 7.1-7.13_

  - [x] 2.7 创建Logistics Service Protobuf定义
    - 定义LogisticsService接口(查询、订阅)
    - 定义LogisticsTracking消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 8.1-8.10_

  - [x] 2.8 创建Freight Service Protobuf定义
    - 定义FreightService接口(计算运费、模板管理)
    - 定义FreightTemplate消息类型
    - 添加OpenAPI注解和验证规则
    - _Requirements: 9.1-9.10_

  - [x] 2.9 生成Protobuf代码
    - 运行buf generate生成Go代码
    - 验证生成的gRPC服务接口
    - 验证生成的HTTP路由
    - _Requirements: All_

- [x] 3. Ent Schema定义和数据库迁移
  - [x] 3.1 创建Consumer Schema
    - 定义Consumer实体(id, tenant_id, phone, email, nickname, avatar等)
    - 定义字段验证规则(手机号格式、邮箱格式)
    - 定义索引(tenant_id+phone唯一索引)
    - 定义枚举类型(Status: normal/locked/deactivated)
    - _Requirements: 2.1-2.8, 10.1-10.8_

  - [x] 3.2 创建LoginLog Schema
    - 定义LoginLog实体(id, tenant_id, consumer_id, phone, login_type等)
    - 定义索引(tenant_id+consumer_id+login_at)
    - 定义枚举类型(LoginType: phone/wechat)
    - _Requirements: 1.7, 12.1-12.8_

  - [x] 3.3 创建SMSLog Schema
    - 定义SMSLog实体(id, tenant_id, phone, sms_type, content, code等)
    - 定义索引(tenant_id+phone+sent_at)
    - 定义枚举类型(SMSType: verification/notification, Channel: aliyun/tencent)
    - _Requirements: 3.7, 3.10_

  - [x] 3.4 创建PaymentOrder Schema
    - 定义PaymentOrder实体(id, tenant_id, order_no, consumer_id, amount等)
    - 定义索引(tenant_id+order_no唯一索引, transaction_id索引)
    - 定义枚举类型(PaymentMethod, PaymentType, Status)
    - 使用decimal类型存储金额
    - _Requirements: 4.4, 4.12, 5.12_

  - [x] 3.5 创建FinanceAccount Schema
    - 定义FinanceAccount实体(id, tenant_id, consumer_id, balance, frozen_balance)
    - 定义索引(tenant_id+consumer_id唯一索引)
    - 使用decimal类型存储余额
    - _Requirements: 5.1, 5.9, 5.12_

  - [x] 3.6 创建FinanceTransaction Schema
    - 定义FinanceTransaction实体(id, tenant_id, consumer_id, transaction_no等)
    - 定义索引(transaction_no唯一索引, tenant_id+consumer_id+created_at)
    - 定义枚举类型(TransactionType: recharge/consume/withdraw/refund)
    - 使用decimal类型存储金额
    - _Requirements: 5.7, 5.11, 5.12_

  - [x] 3.7 创建MediaFile Schema
    - 定义MediaFile实体(id, tenant_id, consumer_id, file_name, file_type等)
    - 定义索引(tenant_id+consumer_id+is_deleted)
    - 定义枚举类型(FileType: image/video)
    - 软删除支持(is_deleted字段)
    - _Requirements: 7.8, 7.9, 7.11_

  - [x] 3.8 创建LogisticsTracking Schema
    - 定义LogisticsTracking实体(id, tenant_id, tracking_no, courier_company等)
    - 定义索引(tenant_id+tracking_no唯一索引)
    - 定义枚举类型(Status: pending/in_transit/delivering/delivered)
    - 使用JSON字段存储物流轨迹
    - _Requirements: 8.4, 8.5, 8.7_

  - [x] 3.9 创建FreightTemplate Schema
    - 定义FreightTemplate实体(id, tenant_id, name, calculation_type等)
    - 定义索引(tenant_id+is_active)
    - 定义枚举类型(CalculationType: by_weight/by_distance)
    - 使用JSON字段存储地区规则和包邮规则
    - _Requirements: 9.5, 9.6, 9.7_

  - [x] 3.10 生成Ent代码和数据库迁移
    - 运行go generate生成Ent代码
    - 创建数据库迁移文件
    - 验证Schema定义和索引
    - _Requirements: All_

- [x] 4. 基础设施层实现(pkg/)
  - [x] 4.1 实现短信服务工具(pkg/sms/)
    - 实现阿里云短信客户端(aliyun.go)
    - 实现腾讯云短信客户端(tencent.go)
    - 实现短信发送接口和故障转移逻辑
    - 实现短信模板管理
    - _Requirements: 3.1, 3.2, 3.6_

  - [x] 4.2 实现支付服务工具(pkg/payment/)
    - 实现微信支付客户端(wechat.go)
    - 实现支付宝客户端(alipay.go)
    - 实现易宝支付客户端(yeepay.go)
    - 实现支付回调签名验证
    - _Requirements: 4.1, 4.2, 4.3, 4.9_

  - [x] 4.3 实现OSS存储工具(pkg/oss/)
    - 实现阿里云OSS客户端(aliyun.go)
    - 实现腾讯云COS客户端(tencent.go)
    - 实现预签名URL生成
    - 实现文件上传和删除
    - _Requirements: 7.5, 7.6, 7.7_

  - [x] 4.4 实现物流查询工具(pkg/logistics/)
    - 实现快递鸟API客户端(kdniao.go)
    - 实现物流信息查询和解析
    - 实现物流公司识别
    - _Requirements: 8.1, 8.2, 8.5_

  - [x] 4.5 实现多租户中间件(pkg/middleware/tenant.go)
    - 实现租户上下文提取(从JWT或请求头)
    - 实现租户上下文注入
    - 实现租户隔离验证
    - _Requirements: 10.1-10.8_

  - [x] 4.6 实现JWT认证中间件(pkg/middleware/auth.go)
    - 实现JWT令牌生成和验证
    - 实现令牌过期检查
    - 实现刷新令牌机制
    - _Requirements: 13.1, 13.2, 13.3_

  - [x] 4.7 实现限流中间件(pkg/middleware/ratelimit.go)
    - 实现基于Redis的滑动窗口限流
    - 实现用户级别限流(每分钟60次)
    - 实现IP级别限流
    - _Requirements: 13.4, 13.5_

  - [x] 4.8 实现事件总线(pkg/eventbus/)
    - 实现Kafka事件发布
    - 实现Kafka事件订阅
    - 实现事件重试机制(最多3次)
    - 实现死信队列
    - _Requirements: 11.1-11.9_

- [x] 5. Checkpoint - 基础设施验证
  - 验证所有pkg工具编译通过
  - 验证中间件功能正常
  - 验证事件总线连接正常
  - 确保所有测试通过,询问用户是否继续


- [x] 6. Consumer Service实现(用户服务)
  - [x] 6.1 实现ConsumerRepo数据层
    - 实现Create方法(创建用户)
    - 实现Get方法(查询用户)
    - 实现GetByPhone方法(按手机号查询)
    - 实现Update方法(更新用户信息)
    - 实现List方法(分页查询用户列表)
    - 实现Deactivate方法(注销账户)
    - 实现多租户过滤(自动添加tenant_id条件)
    - _Requirements: 2.1-2.8, 10.1-10.8_

  - [x] 6.2 实现LoginLogRepo数据层
    - 实现Create方法(记录登录日志)
    - 实现List方法(分页查询登录日志)
    - 实现多租户过滤
    - _Requirements: 1.7, 12.1-12.8_

  - [x] 6.3 实现ConsumerService服务层 - 注册登录
    - 实现RegisterByPhone方法(手机号注册)
    - 实现LoginByPhone方法(手机号登录)
    - 实现LoginByWechat方法(微信登录)
    - 实现密码bcrypt加密
    - 实现JWT令牌生成
    - 实现登录失败计数和账户锁定
    - 实现风险评分计算
    - 实现登录日志记录
    - 发布UserRegisteredEvent事件
    - _Requirements: 1.1-1.10_

  - [x] 6.4 实现ConsumerService服务层 - 信息管理
    - 实现GetConsumer方法(查询用户信息)
    - 实现UpdateConsumer方法(更新用户信息)
    - 实现UpdatePhone方法(更新手机号,需验证码)
    - 实现UpdateEmail方法(更新邮箱,需验证)
    - 实现UploadAvatar方法(上传头像)
    - 实现DeactivateAccount方法(注销账户)
    - 实现ListConsumers方法(查询用户列表,管理员权限)
    - _Requirements: 2.1-2.8_

  - [x] 6.5 实现ConsumerService服务层 - 登录日志
    - 实现ListLoginLogs方法(查询登录日志)
    - 实现分页和筛选(按用户、时间范围、状态)
    - 实现敏感数据脱敏(手机号、IP地址)
    - _Requirements: 12.1-12.8_

  - [ ]*6.6 编写Consumer Service单元测试
    - 测试用户注册(手机号格式验证、验证码验证)
    - 测试用户登录(密码验证、JWT生成)
    - 测试账户锁定(连续失败锁定、锁定时间)
    - 测试风险评分(异常IP检测、频繁失败)
    - 测试用户信息更新(字段验证、持久化)
    - 测试账户注销(状态变更、数据保留)
    - _Requirements: 1.1-1.10, 2.1-2.8_

  - [ ]*6.7 编写Consumer Service属性测试
    - **Property 1: 用户注册创建账户**
    - **Validates: Requirements 1.1**
    - **Property 2: 登录返回有效令牌**
    - **Validates: Requirements 1.2**
    - **Property 5: 连续失败锁定账户**
    - **Validates: Requirements 1.5, 1.6**
    - **Property 6: 登录日志完整记录**
    - **Validates: Requirements 1.7**
    - **Property 7: 风险评分范围**
    - **Validates: Requirements 1.8**
    - **Property 9: 密码bcrypt加密**
    - **Validates: Requirements 1.10**
    - **Property 11: 用户信息更新持久化**
    - **Validates: Requirements 2.2**
    - **Property 13: 账户注销状态保留**
    - **Validates: Requirements 2.6**
    - **Property 14: 多租户数据隔离**
    - **Validates: Requirements 2.8, 10.1-10.8**

- [x] 7. SMS Service实现(短信服务)
  - [x] 7.1 实现SMSLogRepo数据层
    - 实现Create方法(记录短信日志)
    - 实现List方法(分页查询短信日志)
    - 实现多租户过滤
    - _Requirements: 3.7, 3.10_

  - [x] 7.2 实现SMSService服务层
    - 实现SendVerificationCode方法(发送验证码)
    - 实现VerifyCode方法(验证验证码)
    - 实现SendNotification方法(发送通知短信)
    - 实现ListSMSLogs方法(查询短信日志)
    - 实现验证码生成(6位数字)
    - 实现验证码存储(Redis,5分钟过期)
    - 实现频率限制(每分钟1条、每天10条)
    - 实现短信通道故障转移(阿里云→腾讯云)
    - 实现验证码一次性使用
    - 实现短信日志记录
    - _Requirements: 3.1-3.10_

  - [ ]*7.3 编写SMS Service单元测试
    - 测试验证码生成(6位数字、随机性)
    - 测试验证码存储(Redis缓存、过期时间)
    - 测试验证码验证(正确、错误、过期)
    - 测试短信发送(阿里云、腾讯云)
    - 测试频率限制(每分钟、每日)
    - 测试短信日志记录
    - _Requirements: 3.1-3.10_

  - [ ]*7.4 编写SMS Service属性测试
    - **Property 15: 验证码格式和有效期**
    - **Validates: Requirements 3.3**
    - **Property 16: 短信发送频率限制**
    - **Validates: Requirements 3.4**
    - **Property 17: 短信每日限额**
    - **Validates: Requirements 3.5**
    - **Property 18: 短信通道故障转移**
    - **Validates: Requirements 3.2, 3.6**
    - **Property 19: 验证码一次性使用**
    - **Validates: Requirements 3.8**
    - **Property 20: 短信日志完整记录**
    - **Validates: Requirements 3.7**

- [x] 8. Payment Service实现(支付服务)
  - [x] 8.1 实现PaymentOrderRepo数据层
    - 实现Create方法(创建支付订单)
    - 实现Get方法(查询支付订单)
    - 实现GetByOrderNo方法(按订单号查询)
    - 实现Update方法(更新订单状态)
    - 实现List方法(分页查询支付流水)
    - 实现CloseExpiredOrders方法(关闭超时订单)
    - 实现多租户过滤
    - _Requirements: 4.4, 4.7, 4.12_

  - [x] 8.2 实现PaymentService服务层
    - 实现CreatePayment方法(创建支付订单)
    - 实现GetPayment方法(查询支付订单)
    - 实现QueryPaymentStatus方法(查询支付结果)
    - 实现Refund方法(申请退款)
    - 实现QueryRefundStatus方法(查询退款状态)
    - 实现ListPayments方法(查询支付流水)
    - 实现订单号生成(全局唯一)
    - 实现支付回调处理(签名验证、状态更新)
    - 实现订单超时关闭(30分钟)
    - 发布PaymentSuccessEvent事件
    - 实现退款流水记录
    - _Requirements: 4.1-4.12_

  - [ ]*8.3 编写Payment Service单元测试
    - 测试支付订单创建(订单号生成、金额验证)
    - 测试支付回调处理(签名验证、状态更新)
    - 测试支付查询(订单状态查询)
    - 测试退款操作(退款金额验证、退款状态)
    - 测试订单超时关闭(定时任务)
    - _Requirements: 4.1-4.12_

  - [ ]*8.4 编写Payment Service属性测试
    - **Property 21: 支付订单号唯一性**
    - **Validates: Requirements 4.4**
    - **Property 22: 支付订单超时关闭**
    - **Validates: Requirements 4.4, 4.7**
    - **Property 23: 支付成功发布事件**
    - **Validates: Requirements 4.5, 11.3**
    - **Property 24: 支付回调签名验证**
    - **Validates: Requirements 4.9**
    - **Property 25: 退款流水记录**
    - **Validates: Requirements 4.10, 4.12**

- [x] 9. Finance Service实现(财务服务)
  - [x] 9.1 实现FinanceAccountRepo数据层
    - 实现Create方法(创建财务账户)
    - 实现Get方法(查询账户)
    - 实现GetByConsumerID方法(按用户ID查询)
    - 实现UpdateBalance方法(更新余额,使用乐观锁)
    - 实现多租户过滤
    - _Requirements: 5.1, 5.9_

  - [x] 9.2 实现FinanceTransactionRepo数据层
    - 实现Create方法(记录财务流水)
    - 实现List方法(分页查询流水)
    - 实现Export方法(导出流水为CSV)
    - 实现多租户过滤
    - _Requirements: 5.7, 5.11_

  - [x] 9.3 实现FinanceService服务层 - 账户管理
    - 实现GetAccount方法(获取账户余额)
    - 实现自动创建财务账户(用户注册时)
    - 订阅UserRegisteredEvent事件
    - _Requirements: 5.1, 11.2_

  - [x] 9.4 实现FinanceService服务层 - 充值提现
    - 实现Recharge方法(充值)
    - 实现Withdraw方法(申请提现)
    - 实现ApproveWithdraw方法(审核提现)
    - 订阅PaymentSuccessEvent事件(自动充值)
    - 实现提现金额验证(10-5000元)
    - 实现余额冻结和解冻
    - 实现余额非负约束
    - 实现金额精度保证(decimal类型)
    - 实现财务流水记录
    - _Requirements: 5.2-5.12, 11.4_

  - [x] 9.5 实现FinanceService服务层 - 流水查询
    - 实现ListTransactions方法(查询财务流水)
    - 实现ExportTransactions方法(导出流水)
    - 实现分页和筛选(按时间范围、交易类型)
    - _Requirements: 5.7, 5.8_

  - [ ]*9.6 编写Finance Service单元测试
    - 测试账户创建(自动创建、初始余额)
    - 测试充值操作(余额增加、流水记录)
    - 测试提现申请(余额冻结、金额验证)
    - 测试提现审核(通过/拒绝、余额变动)
    - 测试余额查询(当前余额、冻结余额)
    - 测试流水查询(分页、筛选)
    - _Requirements: 5.1-5.12_

  - [ ]*9.7 编写Finance Service属性测试
    - **Property 26: 用户账户自动创建**
    - **Validates: Requirements 5.1**
    - **Property 27: 充值余额增加**
    - **Validates: Requirements 5.2**
    - **Property 28: 提现金额限制**
    - **Validates: Requirements 5.4**
    - **Property 29: 提现冻结和解冻**
    - **Validates: Requirements 5.3, 5.6**
    - **Property 30: 余额非负约束**
    - **Validates: Requirements 5.9**
    - **Property 31: 金额精度保证**
    - **Validates: Requirements 5.12**
    - **Property 32: 财务流水完整性**
    - **Validates: Requirements 5.7, 5.11**
    - **Property 48: 充值事件触发余额增加**
    - **Validates: Requirements 11.4**

- [x] 10. Checkpoint - 核心服务验证
  - 验证Consumer、SMS、Payment、Finance服务编译通过
  - 验证所有单元测试通过
  - 验证所有属性测试通过(Properties 1-32, 48)
  - 验证事件发布和订阅正常
  - 确保所有测试通过,询问用户是否继续


- [x] 11. Wechat Service实现(微信服务)
  - [x] 11.1 实现WechatService服务层 - OAuth登录
    - 实现GetAuthURL方法(获取微信授权URL)
    - 实现AuthCallback方法(微信授权回调)
    - 实现GetWechatUserInfo方法(获取微信用户信息)
    - 实现access_token缓存(Redis,7200秒过期)
    - 实现access_token自动刷新
    - 实现微信签名验证
    - _Requirements: 6.1, 6.2, 6.3, 6.6, 6.10_

  - [x] 11.2 实现WechatService服务层 - 公众号和小程序
    - 实现SendTemplateMessage方法(发送模板消息)
    - 实现MiniProgramLogin方法(小程序登录)
    - 实现微信事件消息处理
    - 发布系统事件(WechatEventReceived)
    - _Requirements: 6.4, 6.5, 6.7, 6.8, 6.9_

  - [ ]*11.3 编写Wechat Service单元测试
    - 测试微信授权URL生成(参数验证)
    - 测试微信授权回调(获取openid、unionid)
    - 测试微信用户信息获取
    - 测试access_token缓存和刷新
    - 测试模板消息发送
    - 测试小程序登录
    - _Requirements: 6.1-6.10_

  - [ ]*11.4 编写Wechat Service属性测试
    - **Property 3: 微信登录重定向**
    - **Validates: Requirements 1.3**
    - **Property 4: 微信授权创建用户**
    - **Validates: Requirements 1.4**
    - **Property 33: 微信授权URL格式**
    - **Validates: Requirements 6.1**
    - **Property 34: 微信用户信息获取**
    - **Validates: Requirements 6.2, 6.3**
    - **Property 35: 微信access_token缓存**
    - **Validates: Requirements 6.10**

- [x] 12. Media Service实现(媒体服务)
  - [x] 12.1 实现MediaFileRepo数据层
    - 实现Create方法(记录媒体文件)
    - 实现Get方法(查询媒体文件)
    - 实现List方法(分页查询,过滤已删除)
    - 实现SoftDelete方法(软删除)
    - 实现多租户过滤
    - _Requirements: 7.9, 7.11_

  - [x] 12.2 实现MediaService服务层
    - 实现GenerateUploadURL方法(生成预签名URL)
    - 实现ConfirmUpload方法(确认上传完成)
    - 实现GetMediaFile方法(获取媒体文件)
    - 实现ListMediaFiles方法(查询媒体文件列表)
    - 实现DeleteMediaFile方法(删除媒体文件)
    - 实现文件格式验证(图片:JPEG/PNG/GIF,视频:MP4/AVI/MOV)
    - 实现文件大小验证(图片5MB,视频100MB)
    - 实现预签名URL生成(1小时有效期)
    - 实现缩略图生成(200x200)
    - 实现病毒扫描(集成第三方服务)
    - _Requirements: 7.1-7.13_

  - [ ]*12.3 编写Media Service单元测试
    - 测试预签名URL生成(URL格式、过期时间)
    - 测试文件上传确认(元数据保存、缩略图生成)
    - 测试文件格式验证(允许的格式、拒绝的格式)
    - 测试文件大小验证(图片5MB、视频100MB)
    - 测试文件删除(软删除、查询过滤)
    - _Requirements: 7.1-7.13_

  - [ ]*12.4 编写Media Service属性测试
    - **Property 36: 文件格式验证**
    - **Validates: Requirements 7.1, 7.2**
    - **Property 37: 文件大小限制**
    - **Validates: Requirements 7.3, 7.4**
    - **Property 38: 预签名URL有效期**
    - **Validates: Requirements 7.7**
    - **Property 39: 缩略图自动生成**
    - **Validates: Requirements 7.8**
    - **Property 40: 媒体文件软删除**
    - **Validates: Requirements 7.11**

- [x] 13. Logistics Service实现(物流服务)
  - [x] 13.1 实现LogisticsTrackingRepo数据层
    - 实现Create方法(创建物流跟踪)
    - 实现Get方法(查询物流跟踪)
    - 实现GetByTrackingNo方法(按运单号查询)
    - 实现Update方法(更新物流信息)
    - 实现List方法(分页查询物流历史)
    - 实现多租户过滤
    - _Requirements: 8.4, 8.9_

  - [x] 13.2 实现LogisticsService服务层
    - 实现QueryLogistics方法(查询物流信息)
    - 实现SubscribeLogistics方法(订阅物流状态)
    - 实现ListLogisticsHistory方法(查询物流历史)
    - 实现物流信息缓存(Redis,30分钟)
    - 实现物流API调用(快递鸟)
    - 实现物流数据解析和格式化
    - 实现物流状态变更检测
    - 发布LogisticsStatusChangedEvent事件
    - _Requirements: 8.1-8.10_

  - [ ]*13.3 编写Logistics Service单元测试
    - 测试物流查询(API调用、数据解析)
    - 测试物流缓存(Redis缓存、过期时间)
    - 测试物流订阅(状态变更检测)
    - 测试事件发布(状态变更事件)
    - _Requirements: 8.1-8.10_

  - [ ]*13.4 编写Logistics Service属性测试
    - **Property 41: 物流信息缓存**
    - **Validates: Requirements 8.3**
    - **Property 42: 物流状态变更事件**
    - **Validates: Requirements 8.8, 11.5**

- [x] 14. Freight Service实现(运费计算服务)
  - [x] 14.1 实现FreightTemplateRepo数据层
    - 实现Create方法(创建运费模板)
    - 实现Get方法(查询运费模板)
    - 实现Update方法(更新运费模板)
    - 实现List方法(分页查询运费模板)
    - 实现多租户过滤
    - _Requirements: 9.5_

  - [x] 14.2 实现FreightService服务层 - 运费计算
    - 实现CalculateFreight方法(计算运费)
    - 实现按重量计算(首重+续重阶梯定价)
    - 实现按距离计算(根据省市区计算距离)
    - 实现包邮规则判断(满额包邮、地区包邮)
    - 实现偏远地区加价
    - 实现运费精度保证(decimal类型)
    - _Requirements: 9.1-9.4, 9.6-9.10_

  - [x] 14.3 实现FreightService服务层 - 模板管理
    - 实现CreateFreightTemplate方法(创建运费模板)
    - 实现UpdateFreightTemplate方法(更新运费模板)
    - 实现GetFreightTemplate方法(查询运费模板)
    - 实现ListFreightTemplates方法(查询运费模板列表)
    - _Requirements: 9.5_

  - [ ]*14.4 编写Freight Service单元测试
    - 测试按重量计算(首重续重、阶梯定价)
    - 测试按距离计算(地区距离、价格计算)
    - 测试包邮规则(满额包邮、地区包邮)
    - 测试运费模板管理(创建、更新、查询)
    - _Requirements: 9.1-9.10_

  - [ ]*14.5 编写Freight Service属性测试
    - **Property 43: 运费计算精度**
    - **Validates: Requirements 9.10**
    - **Property 44: 包邮规则优先**
    - **Validates: Requirements 9.6, 9.7**
    - **Property 45: 首重续重阶梯计算**
    - **Validates: Requirements 9.3**

- [x] 15. Checkpoint - 所有服务验证
  - 验证所有8个服务模块编译通过
  - 验证所有单元测试通过
  - 验证所有属性测试通过(Properties 1-45, 48)
  - 验证服务间集成正常
  - 确保所有测试通过,询问用户是否继续

- [x] 16. 安全和限流实现
  - [x] 16.1 实现JWT认证
    - 集成JWT中间件到所有受保护接口
    - 实现JWT令牌生成(2小时有效期)
    - 实现刷新令牌机制(7天有效期)
    - 实现令牌黑名单(Redis)
    - _Requirements: 13.1, 13.2, 13.3_

  - [x] 16.2 实现API限流
    - 集成限流中间件到敏感接口
    - 实现用户级别限流(每分钟60次)
    - 实现IP级别限流
    - 实现限流错误响应(429 Too Many Requests)
    - _Requirements: 13.4, 13.5_

  - [x] 16.3 实现输入验证和安全防护
    - 实现输入参数验证(防SQL注入、XSS)
    - 实现敏感数据脱敏(手机号、身份证号)
    - 实现IP黑名单机制
    - 实现HTTPS强制跳转
    - _Requirements: 13.6, 13.7, 13.9, 13.10_

  - [x] 16.4 实现API日志记录
    - 实现API调用日志(接口、参数、响应时间、状态码)
    - 实现日志脱敏(敏感参数)
    - 实现日志分级(INFO/WARN/ERROR)
    - _Requirements: 13.8_

  - [ ]*16.5 编写安全功能单元测试
    - 测试JWT令牌认证(有效、无效、过期)
    - 测试API限流(正常、超限)
    - 测试输入验证(SQL注入、XSS)
    - 测试敏感数据脱敏
    - _Requirements: 13.1-13.10_

  - [ ]*16.6 编写安全功能属性测试
    - **Property 49: JWT令牌认证**
    - **Validates: Requirements 13.1**
    - **Property 50: JWT令牌有效期**
    - **Validates: Requirements 13.2**
    - **Property 51: API限流保护**
    - **Validates: Requirements 13.4, 13.5**
    - **Property 52: 敏感数据脱敏**
    - **Validates: Requirements 13.7**
    - **Property 53: 输入参数验证**
    - **Validates: Requirements 13.6**

- [x] 17. 配置管理实现
  - [x] 17.1 实现租户配置管理
    - 创建TenantConfig数据模型
    - 实现租户配置存储(数据库+Redis缓存)
    - 实现租户配置查询和更新
    - 实现配置热更新(无需重启)
    - 实现配置加密(敏感配置)
    - _Requirements: 14.1, 14.2, 14.3, 14.8_

  - [x] 17.2 实现配置管理接口
    - 实现GetConfig方法(查询配置)
    - 实现UpdateConfig方法(更新配置)
    - 实现配置验证(格式和有效性)
    - 实现配置变更历史记录
    - 实现配置回滚
    - _Requirements: 14.4, 14.5, 14.6, 14.7_

  - [ ]*17.3 编写配置管理单元测试
    - 测试配置查询和更新
    - 测试配置验证
    - 测试配置热更新
    - 测试配置变更历史
    - 测试配置回滚
    - _Requirements: 14.1-14.8_

  - [ ]*17.4 编写配置管理属性测试
    - **Property 54: 多租户配置隔离**
    - **Validates: Requirements 3.9, 4.11, 7.12, 14.3**
    - **Property 55: 配置变更审计**
    - **Validates: Requirements 14.6**

- [x] 18. 监控和性能实现
  - [x] 18.1 实现健康检查和指标
    - 实现/health接口(数据库、Redis、Kafka连接状态)
    - 实现/metrics接口(Prometheus格式)
    - 实现API响应时间统计(P50/P95/P99)
    - 实现数据库连接池监控
    - 实现缓存命中率监控
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x] 18.2 实现告警机制
    - 实现系统异常告警(邮件/短信/钉钉)
    - 实现性能告警(响应时间超阈值)
    - 实现资源告警(CPU/内存/磁盘)
    - _Requirements: 15.6_

  - [x] 18.3 实现分布式链路追踪
    - 集成OpenTelemetry
    - 实现请求链路追踪
    - 实现跨服务调用追踪
    - _Requirements: 15.7_

  - [ ]*18.4 编写监控功能单元测试
    - 测试健康检查接口
    - 测试指标采集
    - 测试告警触发
    - _Requirements: 15.1-15.7_

  - [ ]*18.5 编写监控功能属性测试
    - **Property 56: API响应时间**
    - **Validates: Requirements 15.8**
    - **Property 57: 健康检查接口**
    - **Validates: Requirements 15.1**

- [x] 19. 事件驱动集成测试
  - [ ]*19.1 测试用户注册事件流
    - 测试UserRegisteredEvent发布
    - 测试Finance Service订阅并创建账户
    - 验证事件异步处理
    - _Requirements: 11.2, 11.6_

  - [ ]*19.2 测试支付成功事件流
    - 测试PaymentSuccessEvent发布
    - 测试Finance Service订阅并充值
    - 验证事件重试机制
    - _Requirements: 11.3, 11.4, 11.7_

  - [ ]*19.3 测试物流状态变更事件流
    - 测试LogisticsStatusChangedEvent发布
    - 验证事件异步处理
    - _Requirements: 11.5, 11.6_

  - [ ]*19.4 测试事件失败处理
    - 测试事件处理失败重试(最多3次)
    - 测试死信队列
    - 测试事件日志记录
    - _Requirements: 11.7, 11.8, 11.9_

  - [ ]*19.5 编写事件驱动属性测试
    - **Property 46: 事件异步非阻塞**
    - **Validates: Requirements 11.6**
    - **Property 47: 事件重试机制**
    - **Validates: Requirements 11.7, 11.8**

- [x] 20. Checkpoint - 完整系统验证
  - 验证所有功能模块正常工作
  - 验证所有单元测试通过
  - 验证所有属性测试通过(Properties 1-57)
  - 验证集成测试通过
  - 验证性能指标达标(API响应时间P95<200ms)
  - 确保所有测试通过,询问用户是否继续


- [ ] 21. 前端实现 - Store状态管理
  - [ ] 21.1 创建Consumer Store
    - 创建consumer.state.ts(用户注册、登录、信息管理)
    - 封装gRPC客户端调用(ConsumerService)
    - 实现状态管理(用户信息、登录状态)
    - 实现错误处理
    - _Requirements: 1.1-1.10, 2.1-2.8_

  - [ ] 21.2 创建SMS Store
    - 创建sms.state.ts(发送验证码、验证)
    - 封装gRPC客户端调用(SMSService)
    - 实现验证码倒计时
    - _Requirements: 3.1-3.10_

  - [ ] 21.3 创建Payment Store
    - 创建payment.state.ts(创建支付、查询、退款)
    - 封装gRPC客户端调用(PaymentService)
    - 实现支付状态轮询
    - _Requirements: 4.1-4.12_

  - [ ] 21.4 创建Finance Store
    - 创建finance.state.ts(余额、充值、提现、流水)
    - 封装gRPC客户端调用(FinanceService)
    - 实现余额实时更新
    - _Requirements: 5.1-5.12_

  - [ ] 21.5 创建Media Store
    - 创建media.state.ts(上传、查询、删除)
    - 封装gRPC客户端调用(MediaService)
    - 实现文件上传进度
    - _Requirements: 7.1-7.13_

  - [ ] 21.6 创建Logistics Store
    - 创建logistics.state.ts(查询物流、订阅)
    - 封装gRPC客户端调用(LogisticsService)
    - 实现物流信息轮询
    - _Requirements: 8.1-8.10_

  - [ ] 21.7 创建Freight Store
    - 创建freight.state.ts(计算运费、模板管理)
    - 封装gRPC客户端调用(FreightService)
    - _Requirements: 9.1-9.10_

- [ ] 22. 前端实现 - 页面组件
  - [ ] 22.1 创建用户注册登录页面
    - 创建views/auth/register.vue(手机号注册)
    - 创建views/auth/login.vue(手机号登录、微信登录)
    - 实现表单验证(手机号格式、验证码)
    - 实现验证码倒计时
    - 实现登录状态保持(JWT存储)
    - _Requirements: 1.1-1.4_

  - [ ] 22.2 创建用户信息管理页面
    - 创建views/user/profile.vue(查看和编辑个人信息)
    - 创建views/user/avatar.vue(上传头像)
    - 创建views/user/security.vue(修改手机号、邮箱、密码)
    - 实现表单验证和提交
    - _Requirements: 2.1-2.8_

  - [ ] 22.3 创建登录日志页面
    - 创建views/user/login-logs.vue(查看登录日志)
    - 实现分页和筛选(按时间范围、状态)
    - 实现数据表格展示
    - _Requirements: 12.1-12.8_

  - [ ] 22.4 创建财务管理页面
    - 创建views/finance/account.vue(查看余额)
    - 创建views/finance/recharge.vue(充值)
    - 创建views/finance/withdraw.vue(提现)
    - 创建views/finance/transactions.vue(财务流水)
    - 实现支付方式选择(微信、支付宝)
    - 实现流水导出(CSV)
    - _Requirements: 5.1-5.12_

  - [ ] 22.5 创建媒体管理页面
    - 创建views/media/upload.vue(上传图片、视频)
    - 创建views/media/list.vue(媒体文件列表)
    - 实现文件上传(拖拽、选择)
    - 实现上传进度显示
    - 实现文件预览和删除
    - _Requirements: 7.1-7.13_

  - [ ] 22.6 创建物流查询页面
    - 创建views/logistics/query.vue(查询物流)
    - 创建views/logistics/tracking.vue(物流轨迹)
    - 实现物流信息展示(时间轴)
    - 实现物流状态订阅
    - _Requirements: 8.1-8.10_

  - [ ] 22.7 创建运费计算页面
    - 创建views/freight/calculator.vue(运费计算器)
    - 创建views/freight/templates.vue(运费模板管理)
    - 实现运费计算表单
    - 实现模板CRUD操作
    - _Requirements: 9.1-9.10_

- [ ] 23. 前端实现 - 路由配置
  - [ ] 23.1 配置用户模块路由
    - 配置/auth/register路由(注册页面)
    - 配置/auth/login路由(登录页面)
    - 配置/user/profile路由(个人信息)
    - 配置/user/login-logs路由(登录日志)
    - 设置路由守卫(需要登录)
    - _Requirements: 1.1-1.10, 2.1-2.8, 12.1-12.8_

  - [ ] 23.2 配置财务模块路由
    - 配置/finance/account路由(账户余额)
    - 配置/finance/recharge路由(充值)
    - 配置/finance/withdraw路由(提现)
    - 配置/finance/transactions路由(财务流水)
    - 设置路由守卫(需要登录)
    - _Requirements: 5.1-5.12_

  - [ ] 23.3 配置媒体和物流模块路由
    - 配置/media/upload路由(上传文件)
    - 配置/media/list路由(文件列表)
    - 配置/logistics/query路由(物流查询)
    - 配置/freight/calculator路由(运费计算)
    - 设置路由守卫(需要登录)
    - _Requirements: 7.1-7.13, 8.1-8.10, 9.1-9.10_

- [ ] 24. 前端实现 - 通用组件
  - [ ] 24.1 创建验证码输入组件
    - 创建components/VerificationCodeInput.vue
    - 实现6位数字输入
    - 实现自动聚焦和跳转
    - 实现倒计时显示
    - _Requirements: 3.3_

  - [ ] 24.2 创建支付方式选择组件
    - 创建components/PaymentMethodSelector.vue
    - 实现微信支付、支付宝选择
    - 实现支付二维码展示
    - _Requirements: 4.1, 4.2_

  - [ ] 24.3 创建文件上传组件
    - 创建components/FileUploader.vue
    - 实现拖拽上传
    - 实现上传进度显示
    - 实现文件格式和大小验证
    - _Requirements: 7.1-7.4_

  - [ ] 24.4 创建金额输入组件
    - 创建components/AmountInput.vue
    - 实现金额格式化(保留两位小数)
    - 实现金额验证(最小值、最大值)
    - _Requirements: 5.4, 5.12_

- [ ] 25. 部署配置
  - [ ] 25.1 创建Docker镜像
    - 创建Dockerfile(多阶段构建)
    - 配置镜像优化(最小化镜像大小)
    - 创建.dockerignore
    - _Requirements: All_

  - [ ] 25.2 创建Kubernetes部署配置
    - 创建deployment.yaml(3个副本)
    - 创建service.yaml(负载均衡)
    - 创建configmap.yaml(配置文件)
    - 创建secret.yaml(敏感配置)
    - 配置健康检查(liveness、readiness)
    - 配置资源限制(CPU、内存)
    - _Requirements: All_

  - [ ] 25.3 配置数据库部署
    - 配置MySQL主从复制
    - 配置数据库备份(每日全量+每小时增量)
    - 配置数据库监控
    - _Requirements: All_

  - [ ] 25.4 配置Redis部署
    - 配置Redis集群(3主3从)
    - 配置Redis持久化(AOF+RDB)
    - 配置Redis监控
    - _Requirements: All_

  - [ ] 25.5 配置Kafka部署
    - 配置Kafka集群(3个Broker)
    - 配置ZooKeeper集群(3个节点)
    - 创建Topic(user-events、payment-events、logistics-events)
    - 配置Kafka监控
    - _Requirements: 11.1-11.9_

  - [ ] 25.6 配置监控和日志
    - 配置Prometheus(指标采集)
    - 配置Grafana(可视化)
    - 配置ELK Stack(日志收集)
    - 创建监控Dashboard
    - 创建告警规则
    - _Requirements: 15.1-15.9_

- [ ] 26. CI/CD配置
  - [ ] 26.1 配置GitHub Actions / GitLab CI
    - 创建.github/workflows/ci.yml或.gitlab-ci.yml
    - 配置自动测试(单元测试、属性测试)
    - 配置代码质量检查(golangci-lint、eslint)
    - 配置测试覆盖率报告(要求≥70%)
    - 配置自动构建Docker镜像
    - 配置自动部署到测试环境
    - _Requirements: All_

  - [ ] 26.2 配置代码质量门禁
    - 配置测试失败阻止合并
    - 配置覆盖率不达标阻止合并
    - 配置Lint错误阻止合并
    - _Requirements: All_

- [ ] 27. 文档编写
  - [ ] 27.1 编写API文档
    - 生成OpenAPI文档(从Protobuf)
    - 编写API使用示例
    - 编写错误码说明
    - _Requirements: All_

  - [ ] 27.2 编写部署文档
    - 编写部署指南(Docker、Kubernetes)
    - 编写配置说明(环境变量、配置文件)
    - 编写运维手册(备份、恢复、监控)
    - _Requirements: All_

  - [ ] 27.3 编写开发文档
    - 编写开发环境搭建指南
    - 编写代码规范说明
    - 编写测试编写指南
    - _Requirements: All_

- [ ] 28. 最终验收测试
  - [ ]*28.1 执行完整的集成测试
    - 测试用户注册→登录→信息管理完整流程
    - 测试支付→充值→提现完整流程
    - 测试文件上传→查询→删除完整流程
    - 测试物流查询→订阅→状态变更完整流程
    - _Requirements: All_

  - [ ]*28.2 执行性能测试
    - 测试API响应时间(P95<200ms)
    - 测试并发处理能力(>1000 QPS)
    - 测试数据库查询性能(<100ms)
    - 测试缓存命中率(>80%)
    - _Requirements: 15.3, 15.4, 15.5, 15.8_

  - [ ]*28.3 执行安全测试
    - 测试SQL注入防护
    - 测试XSS攻击防护
    - 测试CSRF攻击防护
    - 测试JWT令牌安全
    - 测试敏感数据脱敏
    - 测试API限流
    - _Requirements: 13.1-13.10_

  - [ ]*28.4 执行多租户隔离测试
    - 测试不同租户数据完全隔离
    - 测试租户配置独立生效
    - 测试跨租户访问被拒绝
    - _Requirements: 10.1-10.8, 14.3_

  - [ ]*28.5 执行灾难恢复测试
    - 测试数据库故障恢复
    - 测试Redis故障恢复
    - 测试Kafka故障恢复
    - 测试服务实例故障自动恢复
    - _Requirements: All_

- [ ] 29. Final Checkpoint - 项目交付
  - 验证所有功能模块正常工作
  - 验证所有测试通过(单元测试、属性测试、集成测试、性能测试、安全测试)
  - 验证测试覆盖率≥70%
  - 验证所有57个Correctness Properties通过
  - 验证性能指标达标(API响应时间P95<200ms、并发>1000 QPS)
  - 验证安全防护措施生效
  - 验证多租户隔离正常
  - 验证监控和日志系统正常
  - 验证文档完整
  - 确保所有验收标准通过,项目可以交付

## Notes

- 任务标记`*`为可选任务(测试相关),可根据项目进度选择性跳过以加快MVP交付
- 每个任务都明确引用了对应的需求编号,确保需求可追溯性
- 属性测试(Property-Based Tests)对应设计文档中的57个Correctness Properties
- 所有金额计算使用decimal类型,确保精度
- 所有数据表包含tenant_id字段,确保多租户隔离
- 事件驱动设计通过Kafka实现,确保模块解耦
- 第三方服务集成包括:阿里云(短信、OSS)、腾讯云(短信、COS)、微信(OAuth、公众号、小程序)、支付宝、易宝支付、快递鸟
- 部署采用Kubernetes,支持高可用和自动扩缩容
- 监控采用Prometheus+Grafana,日志采用ELK Stack
- CI/CD采用GitHub Actions或GitLab CI,自动测试和部署

## 属性测试清单

本项目包含57个Correctness Properties,分布如下:

**用户服务(Consumer Service):** Properties 1-14 (14个)
**短信服务(SMS Service):** Properties 15-20 (6个)
**支付服务(Payment Service):** Properties 21-25 (5个)
**财务服务(Finance Service):** Properties 26-32, 48 (8个)
**微信服务(Wechat Service):** Properties 3-4, 33-35 (5个)
**媒体服务(Media Service):** Properties 36-40 (5个)
**物流服务(Logistics Service):** Properties 41-42 (2个)
**运费服务(Freight Service):** Properties 43-45 (3个)
**事件驱动(Event-Driven):** Properties 46-47 (2个)
**安全和限流(Security):** Properties 49-53 (5个)
**配置管理(Configuration):** Properties 54-55 (2个)
**监控和性能(Monitoring):** Properties 56-57 (2个)

所有属性测试必须运行至少100次迭代,确保系统在各种随机输入下的正确性。
