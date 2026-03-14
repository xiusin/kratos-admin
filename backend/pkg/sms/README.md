# SMS Package

短信服务工具包，支持阿里云和腾讯云短信服务，提供故障转移机制和模板管理功能。

## 功能特性

- ✅ 支持阿里云短信服务
- ✅ 支持腾讯云短信服务
- ✅ 主备通道自动故障转移
- ✅ 短信模板管理
- ✅ 验证码发送
- ✅ 通知短信发送
- ✅ 超时控制

## 快速开始

### 1. 创建单个短信客户端

```go
import (
    "github.com/go-kratos/kratos/v2/log"
    "go-wind-admin/pkg/sms"
)

// 阿里云短信客户端
aliyunCfg := &sms.Config{
    Provider:                     sms.ProviderAliyun,
    AccessKeyID:                  "your-access-key-id",
    AccessKeySecret:              "your-access-key-secret",
    SignName:                     "your-sign-name",
    VerificationCodeTemplate:     "SMS_123456789",
    NotificationTemplate:         "SMS_987654321",
    Timeout:                      time.Second * 5,
}

client, err := sms.NewAliyunClient(aliyunCfg, log.DefaultLogger)
if err != nil {
    log.Fatal(err)
}

// 发送验证码
err = client.SendVerificationCode(context.Background(), "13800138000", "123456")
if err != nil {
    log.Error(err)
}
```

### 2. 使用短信管理器（推荐）

```go
// 主通道：阿里云
primaryCfg := &sms.Config{
    Provider:                     sms.ProviderAliyun,
    AccessKeyID:                  "aliyun-key-id",
    AccessKeySecret:              "aliyun-key-secret",
    SignName:                     "aliyun-sign",
    VerificationCodeTemplate:     "SMS_123456789",
    Timeout:                      time.Second * 5,
}

// 备用通道：腾讯云
secondaryCfg := &sms.Config{
    Provider:                     sms.ProviderTencent,
    AccessKeyID:                  "tencent-key-id",
    AccessKeySecret:              "tencent-key-secret",
    SignName:                     "tencent-sign",
    VerificationCodeTemplate:     "123456",
    Timeout:                      time.Second * 5,
}

// 创建管理器
manager, err := sms.NewManagerWithConfigs(primaryCfg, secondaryCfg, log.DefaultLogger)
if err != nil {
    log.Fatal(err)
}

// 发送验证码（自动故障转移）
err = manager.SendVerificationCode(context.Background(), "13800138000", "123456")
if err != nil {
    log.Error(err)
}
```

### 3. 使用模板管理器

```go
// 创建模板管理器
tm := sms.NewTemplateManager()

// 注册模板
err := tm.Register("login_code", &sms.Template{
    Code: "SMS_123456789",
    Params: map[string]string{
        "code": "",
    },
})

// 获取模板
template, err := tm.Get("login_code")
if err != nil {
    log.Error(err)
}

// 使用模板发送短信
templateParams := map[string]string{
    "code": "123456",
}
err = client.Send(context.Background(), "13800138000", template.Code, templateParams)
```

## 故障转移机制

短信管理器支持主备通道自动故障转移：

1. 首先尝试主通道发送短信
2. 如果主通道失败，自动切换到备用通道
3. 如果备用通道也失败，返回错误

```
主通道（阿里云）
    ↓ 发送失败
备用通道（腾讯云）
    ↓ 发送成功
返回成功
```

## 配置说明

### Config 结构

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Provider | Provider | 是 | 提供商类型（aliyun/tencent） |
| AccessKeyID | string | 是 | 访问密钥ID |
| AccessKeySecret | string | 是 | 访问密钥Secret |
| SignName | string | 是 | 短信签名 |
| VerificationCodeTemplate | string | 否 | 验证码模板代码 |
| NotificationTemplate | string | 否 | 通知短信模板代码 |
| Timeout | time.Duration | 否 | 超时时间（默认5秒） |

### 阿里云短信配置

- Endpoint: `dysmsapi.aliyuncs.com`
- 模板参数格式: JSON字符串 `{"code":"123456"}`
- 模板代码示例: `SMS_123456789`

### 腾讯云短信配置

- Endpoint: `sms.tencentcloudapi.com`
- 模板参数格式: 字符串数组 `["123456"]`
- 模板代码示例: `123456`（纯数字）
- 手机号格式: 自动添加国家码 `+86`

## 接口说明

### Client 接口

```go
type Client interface {
    // 发送短信
    Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error
    
    // 发送验证码
    SendVerificationCode(ctx context.Context, phone string, code string) error
    
    // 获取提供商类型
    GetProvider() Provider
}
```

### Manager 方法

```go
// 发送短信（支持故障转移）
func (m *Manager) Send(ctx context.Context, phone string, templateCode string, templateParams map[string]string) error

// 发送验证码（支持故障转移）
func (m *Manager) SendVerificationCode(ctx context.Context, phone string, code string) error

// 获取主通道提供商
func (m *Manager) GetPrimaryProvider() Provider

// 获取备用通道提供商
func (m *Manager) GetSecondaryProvider() Provider
```

### TemplateManager 方法

```go
// 注册模板
func (tm *TemplateManager) Register(name string, template *Template) error

// 获取模板
func (tm *TemplateManager) Get(name string) (*Template, error)

// 获取模板代码
func (tm *TemplateManager) GetCode(name string) (string, error)

// 列出所有模板
func (tm *TemplateManager) List() map[string]*Template

// 删除模板
func (tm *TemplateManager) Delete(name string) error

// 获取模板数量
func (tm *TemplateManager) Count() int
```

## 错误处理

所有方法都返回 `error` 类型，常见错误：

- 配置错误：缺少必填字段
- 网络错误：连接超时、网络不可达
- 认证错误：AccessKey 错误
- 业务错误：手机号格式错误、模板不存在、余额不足

建议使用日志记录错误详情，并根据错误类型进行重试或降级处理。

## 最佳实践

1. **使用短信管理器**：推荐使用 `Manager` 而不是直接使用 `Client`，以获得故障转移能力
2. **配置备用通道**：配置不同提供商的备用通道，提高可用性
3. **设置合理超时**：根据业务需求设置超时时间，避免长时间等待
4. **使用模板管理器**：集中管理短信模板，便于维护和更新
5. **记录日志**：记录发送成功和失败的日志，便于排查问题
6. **限流保护**：在业务层实现限流，避免短信轰炸

## 依赖

- 阿里云 SDK: `github.com/alibabacloud-go/dysmsapi-20170525/v3`
- 腾讯云 SDK: `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111`
- Kratos 日志: `github.com/go-kratos/kratos/v2/log`

## Requirements

- Requirements 3.1: 支持阿里云短信服务作为主要短信通道
- Requirements 3.2: 支持腾讯云短信服务作为备用短信通道
- Requirements 3.6: 主短信通道发送失败时自动切换到备用通道
- Requirements 3.10: 支持短信模板管理和变量替换
