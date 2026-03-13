# Requirements Document

## Introduction

本文档定义了C端用户管理系统的功能需求。该系统为面向终端用户的综合服务平台，提供用户注册登录、支付、财务、媒体管理、物流等核心功能。系统采用多租户架构，支持租户级数据隔离，并通过事件驱动机制实现模块间解耦。

## Glossary

- **C_User_System**: C端用户管理系统，提供面向终端用户的服务
- **User_Service**: 用户服务，负责用户注册、登录、信息管理
- **SMS_Service**: 短信服务，负责发送验证码和通知短信
- **Payment_Service**: 支付服务，负责处理支付、退款等交易
- **Finance_Service**: 财务服务，负责账户余额、充值、提现管理
- **Wechat_Service**: 微信服务，负责OAuth登录、公众号、小程序集成
- **Media_Service**: 媒体服务，负责图片、视频上传和存储管理
- **Logistics_Service**: 物流服务，负责快递查询和物流跟踪
- **Freight_Calculator**: 运费计算器，负责计算订单运费
- **Tenant**: 租户，系统的独立业务实体，数据相互隔离
- **Risk_Score**: 风险评分，基于用户行为计算的安全风险指标
- **OSS**: 对象存储服务，用于存储媒体文件
- **OAuth**: 开放授权协议，用于第三方登录
- **Presigned_URL**: 预签名URL，用于前端直接上传文件到OSS

## Requirements

### Requirement 1: 用户注册与认证

**User Story:** 作为C端用户，我希望能够通过手机号或微信快速注册和登录，以便使用系统服务

#### Acceptance Criteria

1. WHEN 用户提供有效手机号和验证码, THE User_Service SHALL 创建新用户账户
2. WHEN 用户使用已注册手机号登录, THE User_Service SHALL 验证密码并返回访问令牌
3. WHEN 用户请求微信OAuth登录, THE Wechat_Service SHALL 重定向到微信授权页面
4. WHEN 微信授权成功, THE User_Service SHALL 创建或关联用户账户并返回访问令牌
5. IF 用户连续5次登录失败, THEN THE User_Service SHALL 锁定账户15分钟
6. WHILE 账户被锁定, THE User_Service SHALL 拒绝所有登录请求并返回锁定剩余时间
7. THE User_Service SHALL 记录每次登录尝试的时间、IP地址、设备信息到登录日志
8. THE User_Service SHALL 基于登录行为计算Risk_Score（0-100分）
9. IF Risk_Score超过80分, THEN THE User_Service SHALL 要求额外的安全验证
10. FOR ALL 用户密码, THE User_Service SHALL 使用bcrypt算法加密存储


### Requirement 2: 用户信息管理

**User Story:** 作为C端用户，我希望能够查看和修改个人信息，以便保持资料的准确性

#### Acceptance Criteria

1. THE User_Service SHALL 提供查询用户详细信息的接口
2. WHEN 用户更新个人信息, THE User_Service SHALL 验证数据格式并保存更改
3. WHEN 用户更新手机号, THE User_Service SHALL 要求验证新手机号的验证码
4. WHEN 用户更新邮箱, THE User_Service SHALL 发送验证邮件到新邮箱
5. THE User_Service SHALL 支持用户上传和更新头像
6. WHEN 用户注销账户, THE User_Service SHALL 标记账户为已注销状态并保留数据90天
7. THE User_Service SHALL 提供分页查询用户列表的接口（管理员权限）
8. WHERE 多租户环境, THE User_Service SHALL 仅返回当前Tenant的用户数据

### Requirement 3: 短信服务集成

**User Story:** 作为系统，我需要发送验证码和通知短信，以便完成用户验证和消息通知

#### Acceptance Criteria

1. THE SMS_Service SHALL 支持阿里云短信服务作为主要短信通道
2. THE SMS_Service SHALL 支持腾讯云短信服务作为备用短信通道
3. WHEN 发送验证码, THE SMS_Service SHALL 生成6位数字验证码并设置5分钟有效期
4. THE SMS_Service SHALL 限制同一手机号每分钟最多发送1条验证码
5. THE SMS_Service SHALL 限制同一手机号每天最多发送10条验证码
6. IF 主短信通道发送失败, THEN THE SMS_Service SHALL 自动切换到备用通道
7. THE SMS_Service SHALL 记录所有短信发送记录（手机号、内容、状态、时间）
8. THE SMS_Service SHALL 提供验证码校验接口，验证成功后立即失效
9. WHERE 多租户环境, THE SMS_Service SHALL 使用Tenant配置的短信账户
10. THE SMS_Service SHALL 支持短信模板管理和变量替换

### Requirement 4: 支付服务集成

**User Story:** 作为C端用户，我希望能够使用多种支付方式完成支付，以便灵活选择支付渠道

#### Acceptance Criteria

1. THE Payment_Service SHALL 支持微信支付（APP、H5、小程序、扫码）
2. THE Payment_Service SHALL 支持支付宝支付（APP、H5、扫码）
3. THE Payment_Service SHALL 支持易宝支付作为备用支付通道
4. WHEN 创建支付订单, THE Payment_Service SHALL 生成唯一订单号并设置30分钟超时
5. WHEN 支付成功, THE Payment_Service SHALL 发布PaymentSuccessEvent事件
6. WHEN 支付失败, THE Payment_Service SHALL 记录失败原因并允许重新支付
7. IF 支付超时未完成, THEN THE Payment_Service SHALL 自动关闭订单
8. THE Payment_Service SHALL 提供支付结果查询接口
9. THE Payment_Service SHALL 验证支付回调签名确保数据安全
10. THE Payment_Service SHALL 支持退款操作并记录退款流水
11. WHERE 多租户环境, THE Payment_Service SHALL 使用Tenant配置的支付商户号
12. THE Payment_Service SHALL 记录所有支付流水（订单号、金额、状态、时间、渠道）


### Requirement 5: 财务管理

**User Story:** 作为C端用户，我希望能够管理账户余额、充值和提现，以便进行资金操作

#### Acceptance Criteria

1. THE Finance_Service SHALL 为每个用户维护账户余额
2. WHEN 用户充值成功, THE Finance_Service SHALL 增加账户余额并记录充值流水
3. WHEN 用户申请提现, THE Finance_Service SHALL 验证余额充足并创建提现申请
4. THE Finance_Service SHALL 要求提现金额不低于10元且不超过单日限额5000元
5. WHEN 提现审核通过, THE Finance_Service SHALL 扣减账户余额并发起打款
6. IF 提现审核拒绝, THEN THE Finance_Service SHALL 返还冻结金额
7. THE Finance_Service SHALL 提供账户流水查询接口（充值、消费、提现、退款）
8. THE Finance_Service SHALL 支持按时间范围、交易类型筛选流水记录
9. THE Finance_Service SHALL 确保账户余额不能为负数
10. WHERE 多租户环境, THE Finance_Service SHALL 隔离不同Tenant的财务数据
11. THE Finance_Service SHALL 记录所有余额变动操作的操作人和操作时间
12. FOR ALL 金额计算, THE Finance_Service SHALL 使用decimal类型避免精度丢失

### Requirement 6: 微信集成服务

**User Story:** 作为C端用户，我希望能够使用微信登录和接收公众号消息，以便便捷使用服务

#### Acceptance Criteria

1. THE Wechat_Service SHALL 支持微信OAuth 2.0授权登录
2. WHEN 微信授权成功, THE Wechat_Service SHALL 获取用户openid和unionid
3. THE Wechat_Service SHALL 支持获取微信用户基本信息（昵称、头像）
4. THE Wechat_Service SHALL 支持微信公众号消息推送
5. THE Wechat_Service SHALL 支持微信小程序登录和用户信息获取
6. THE Wechat_Service SHALL 验证微信服务器回调签名确保安全
7. WHEN 接收微信事件消息, THE Wechat_Service SHALL 发布对应的系统事件
8. THE Wechat_Service SHALL 支持发送模板消息到微信用户
9. WHERE 多租户环境, THE Wechat_Service SHALL 使用Tenant配置的微信AppID和AppSecret
10. THE Wechat_Service SHALL 缓存access_token并在过期前自动刷新

### Requirement 7: 媒体管理服务

**User Story:** 作为C端用户，我希望能够上传图片和视频，以便分享内容

#### Acceptance Criteria

1. THE Media_Service SHALL 支持图片上传（JPEG、PNG、GIF格式）
2. THE Media_Service SHALL 支持视频上传（MP4、AVI、MOV格式）
3. THE Media_Service SHALL 限制单个图片文件不超过5MB
4. THE Media_Service SHALL 限制单个视频文件不超过100MB
5. THE Media_Service SHALL 将文件存储到OSS（阿里云OSS或腾讯云COS）
6. THE Media_Service SHALL 生成Presigned_URL供前端直接上传到OSS
7. WHEN 生成Presigned_URL, THE Media_Service SHALL 设置1小时有效期
8. THE Media_Service SHALL 为上传的图片生成缩略图（200x200）
9. THE Media_Service SHALL 记录媒体文件元数据（文件名、大小、格式、上传时间、上传者）
10. THE Media_Service SHALL 提供媒体文件列表查询接口
11. THE Media_Service SHALL 支持删除媒体文件（软删除）
12. WHERE 多租户环境, THE Media_Service SHALL 使用Tenant配置的OSS账户
13. FOR ALL 上传文件, THE Media_Service SHALL 扫描病毒和恶意内容


### Requirement 8: 物流管理服务

**User Story:** 作为C端用户，我希望能够查询快递物流信息，以便跟踪包裹状态

#### Acceptance Criteria

1. THE Logistics_Service SHALL 支持查询主流快递公司的物流信息（顺丰、圆通、中通、申通、韵达）
2. WHEN 查询物流信息, THE Logistics_Service SHALL 调用第三方物流API获取实时数据
3. THE Logistics_Service SHALL 缓存物流查询结果30分钟
4. THE Logistics_Service SHALL 提供物流轨迹查询接口（按运单号）
5. THE Logistics_Service SHALL 解析并格式化物流轨迹数据
6. IF 物流API调用失败, THEN THE Logistics_Service SHALL 返回缓存数据或错误信息
7. THE Logistics_Service SHALL 支持物流状态订阅（待揽收、运输中、派送中、已签收）
8. WHEN 物流状态变更, THE Logistics_Service SHALL 发布LogisticsStatusChangedEvent事件
9. THE Logistics_Service SHALL 记录所有物流查询记录
10. WHERE 多租户环境, THE Logistics_Service SHALL 使用Tenant配置的物流API密钥

### Requirement 9: 运费计算服务

**User Story:** 作为系统，我需要根据订单信息计算运费，以便准确收取物流费用

#### Acceptance Criteria

1. THE Freight_Calculator SHALL 支持按重量计算运费
2. THE Freight_Calculator SHALL 支持按距离计算运费
3. WHEN 按重量计算, THE Freight_Calculator SHALL 使用首重+续重的阶梯定价模式
4. WHEN 按距离计算, THE Freight_Calculator SHALL 根据发货地和收货地的省市区计算距离
5. THE Freight_Calculator SHALL 支持配置不同地区的运费模板
6. THE Freight_Calculator SHALL 支持包邮规则（满额包邮、指定地区包邮）
7. IF 订单满足包邮条件, THEN THE Freight_Calculator SHALL 返回运费为0
8. THE Freight_Calculator SHALL 支持偏远地区加价规则
9. WHERE 多租户环境, THE Freight_Calculator SHALL 使用Tenant配置的运费模板
10. THE Freight_Calculator SHALL 确保运费计算结果精确到分

### Requirement 10: 多租户数据隔离

**User Story:** 作为系统管理员，我需要确保不同租户的数据完全隔离，以便保障数据安全

#### Acceptance Criteria

1. THE C_User_System SHALL 在所有数据表中包含tenant_id字段
2. WHEN 执行数据查询, THE C_User_System SHALL 自动添加tenant_id过滤条件
3. WHEN 创建数据记录, THE C_User_System SHALL 自动设置当前Tenant的tenant_id
4. THE C_User_System SHALL 从请求上下文中获取当前Tenant信息
5. IF 请求缺少Tenant信息, THEN THE C_User_System SHALL 拒绝请求并返回401错误
6. THE C_User_System SHALL 禁止跨租户的数据访问
7. WHERE 管理员查询, THE C_User_System SHALL 支持查询所有Tenant的数据
8. THE C_User_System SHALL 在数据库层面创建tenant_id索引优化查询性能

### Requirement 11: 事件驱动模块解耦

**User Story:** 作为系统架构师，我需要通过事件机制解耦模块依赖，以便提高系统可维护性

#### Acceptance Criteria

1. THE C_User_System SHALL 使用eventbus实现模块间通信
2. WHEN 用户注册成功, THE User_Service SHALL 发布UserRegisteredEvent事件
3. WHEN 支付成功, THE Payment_Service SHALL 发布PaymentSuccessEvent事件
4. WHEN 充值成功, THE Finance_Service SHALL 订阅PaymentSuccessEvent并增加余额
5. WHEN 物流状态变更, THE Logistics_Service SHALL 发布LogisticsStatusChangedEvent事件
6. THE C_User_System SHALL 确保事件发布是异步非阻塞的
7. THE C_User_System SHALL 支持事件重试机制（最多3次）
8. IF 事件处理失败, THEN THE C_User_System SHALL 记录错误日志并进入死信队列
9. THE C_User_System SHALL 记录所有事件发布和消费日志


### Requirement 12: 登录日志与审计

**User Story:** 作为系统管理员，我需要查看用户登录日志，以便监控账户安全

#### Acceptance Criteria

1. THE User_Service SHALL 记录每次登录尝试（成功和失败）
2. THE User_Service SHALL 记录登录时间、IP地址、设备类型、浏览器信息
3. THE User_Service SHALL 提供登录日志查询接口（支持按用户、时间范围、状态筛选）
4. THE User_Service SHALL 支持分页查询登录日志
5. THE User_Service SHALL 保留登录日志至少90天
6. WHEN 检测到异常登录, THE User_Service SHALL 发送安全提醒通知
7. THE User_Service SHALL 支持导出登录日志为CSV格式
8. WHERE 多租户环境, THE User_Service SHALL 仅显示当前Tenant的登录日志

### Requirement 13: API安全与限流

**User Story:** 作为系统，我需要保护API免受恶意攻击，以便确保服务稳定性

#### Acceptance Criteria

1. THE C_User_System SHALL 对所有API接口实施JWT令牌认证
2. THE C_User_System SHALL 设置JWT令牌有效期为2小时
3. THE C_User_System SHALL 支持刷新令牌机制（refresh token有效期7天）
4. THE C_User_System SHALL 对敏感接口实施限流（每用户每分钟最多60次请求）
5. IF 超过限流阈值, THEN THE C_User_System SHALL 返回429错误
6. THE C_User_System SHALL 验证所有输入参数防止SQL注入和XSS攻击
7. THE C_User_System SHALL 对敏感数据（手机号、身份证号）进行脱敏显示
8. THE C_User_System SHALL 记录所有API调用日志（接口、参数、响应时间、状态码）
9. THE C_User_System SHALL 支持IP黑名单机制
10. WHERE 生产环境, THE C_User_System SHALL 启用HTTPS加密传输

### Requirement 14: 配置管理

**User Story:** 作为系统管理员，我需要灵活配置系统参数，以便适应不同业务场景

#### Acceptance Criteria

1. THE C_User_System SHALL 支持通过配置文件管理系统参数
2. THE C_User_System SHALL 支持热更新配置（无需重启服务）
3. WHERE 多租户环境, THE C_User_System SHALL 支持Tenant级别的配置覆盖
4. THE C_User_System SHALL 提供配置管理接口（查询、更新）
5. WHEN 更新配置, THE C_User_System SHALL 验证配置格式和有效性
6. THE C_User_System SHALL 记录配置变更历史（变更人、变更时间、变更内容）
7. THE C_User_System SHALL 支持配置回滚到历史版本
8. THE C_User_System SHALL 加密存储敏感配置（API密钥、数据库密码）

### Requirement 15: 性能与监控

**User Story:** 作为运维人员，我需要监控系统性能指标，以便及时发现和解决问题

#### Acceptance Criteria

1. THE C_User_System SHALL 提供健康检查接口（/health）
2. THE C_User_System SHALL 提供性能指标接口（/metrics）
3. THE C_User_System SHALL 记录API响应时间并计算P95、P99指标
4. THE C_User_System SHALL 监控数据库连接池使用率
5. THE C_User_System SHALL 监控缓存命中率
6. IF 系统异常, THEN THE C_User_System SHALL 发送告警通知
7. THE C_User_System SHALL 支持分布式链路追踪
8. THE C_User_System SHALL 确保API响应时间P95小于200ms
9. THE C_User_System SHALL 确保系统可用性达到99.9%

## 特殊要求说明

### 解析器和序列化器要求

本系统涉及多个数据格式的解析和序列化：

1. **JSON解析器**: 用于API请求响应的JSON数据处理
2. **Protobuf序列化器**: 用于gRPC服务间通信
3. **CSV导出器**: 用于导出登录日志和财务流水

对于所有解析器和序列化器，必须满足：
- 实现解析（parse）和格式化（format/print）功能
- 实现往返测试（round-trip property）：parse(format(x)) == x
- 处理边界情况和错误输入
- 提供清晰的错误信息

## 总结

本需求文档定义了C端用户管理系统的15个核心需求，涵盖用户管理、支付、财务、媒体、物流等8个主要功能模块。所有需求均遵循EARS模式和INCOSE质量规则，确保需求的清晰性、可测试性和完整性。系统采用多租户架构和事件驱动设计，支持灵活扩展和模块解耦。
