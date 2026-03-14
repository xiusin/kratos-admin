# 安全和限流中间件使用指南

本文档说明如何使用安全和限流相关的中间件。

## 目录

1. [JWT认证](#jwt认证)
2. [API限流](#api限流)
3. [安全防护](#安全防护)
4. [API日志](#api日志)
5. [完整示例](#完整示例)

---

## JWT认证

### 基本使用

```go
import (
    "go-wind-admin/pkg/auth"
    "github.com/redis/go-redis/v9"
)

// 创建Redis客户端
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// 创建JWT黑名单
blacklist := auth.NewTokenBlacklist(redisClient)

// 创建JWT管理器（带黑名单）
jwtManager := auth.NewJWTManagerWithBlacklist("your-secret-key", blacklist)

// 生成访问令牌
accessToken, expiresIn, err := jwtManager.GenerateAccessToken(userID, tenantID, phone)

// 生成刷新令牌
refreshToken, err := jwtManager.GenerateRefreshToken(userID, tenantID)

// 验证令牌（包含黑名单检查）
claims, err := jwtManager.ValidateTokenWithBlacklist(ctx, accessToken)

// 撤销令牌
err = jwtManager.RevokeToken(ctx, accessToken)

// 撤销用户的所有令牌
err = jwtManager.RevokeUserTokens(ctx, userID)
```

### 令牌刷新

```go
// 使用刷新令牌获取新的访问令牌
newAccessToken, expiresIn, err := jwtManager.RefreshAccessToken(refreshToken)
```

---

## API限流

### 配置限流中间件

```go
import (
    "go-wind-admin/pkg/middleware"
    "github.com/redis/go-redis/v9"
    "github.com/go-kratos/kratos/v2/log"
)

// 创建限流配置
rateLimitConfig := &middleware.RateLimitConfig{
    Redis:             redisClient,
    UserRatePerMinute: 60,   // 每用户每分钟60次
    IPRatePerMinute:   100,  // 每IP每分钟100次
    WindowSize:        60,   // 滑动窗口60秒
    EnableUserLimit:   true,
    EnableIPLimit:     true,
}

// 创建限流中间件
rateLimitMiddleware := middleware.RateLimit(rateLimitConfig, logger)
```

### 在Kratos服务中使用

```go
// 在HTTP服务器中使用
httpSrv := http.NewServer(
    http.Address(":8000"),
    http.Middleware(
        rateLimitMiddleware,
    ),
)

// 在gRPC服务器中使用
grpcSrv := grpc.NewServer(
    grpc.Address(":9000"),
    grpc.Middleware(
        rateLimitMiddleware,
    ),
)
```

### 自定义限流

```go
// 按自定义key限流
allowed, err := middleware.RateLimitByKey(ctx, rateLimitConfig, "custom:key", 10)
if !allowed {
    return errors.New(429, "RATE_LIMIT_EXCEEDED", "rate limit exceeded")
}

// 获取限流信息
current, remaining, err := middleware.GetRateLimitInfo(ctx, rateLimitConfig, "user:123", 60)
fmt.Printf("当前请求数: %d, 剩余配额: %d\n", current, remaining)
```

---

## 安全防护

### 配置安全中间件

```go
import "go-wind-admin/pkg/middleware"

// 创建安全配置
securityConfig := &middleware.SecurityConfig{
    Redis:                   redisClient,
    EnableSQLInjectionCheck: true,
    EnableXSSCheck:          true,
    EnableIPBlacklist:       true,
    EnforceHTTPS:            true,
}

// 创建安全中间件
securityMiddleware := middleware.Security(securityConfig, logger)
```

### IP黑名单管理

```go
// 添加IP到黑名单（永久）
err := middleware.AddIPToBlacklist(ctx, redisClient, "192.168.1.100", 0)

// 移除IP黑名单
err := middleware.RemoveIPFromBlacklist(ctx, redisClient, "192.168.1.100")
```

### 数据脱敏

```go
// 脱敏手机号
maskedPhone := middleware.MaskPhone("13812345678")
// 输出: 138****5678

// 脱敏邮箱
maskedEmail := middleware.MaskEmail("user@example.com")
// 输出: u***@example.com

// 脱敏身份证号
maskedIDCard := middleware.MaskIDCard("110101199001011234")
// 输出: 110101********1234

// 脱敏银行卡号
maskedBankCard := middleware.MaskBankCard("6222021234567890123")
// 输出: 622202*********0123

// 自动脱敏
masked := middleware.MaskSensitiveData("phone", "13812345678")
```

### 输入清理

```go
// 清理用户输入，移除危险字符
cleanInput := middleware.SanitizeString(userInput)
```

---

## API日志

### 配置API日志中间件

```go
import "go-wind-admin/pkg/middleware"

// 创建API日志配置
apiLogConfig := &middleware.APILogConfig{
    LogRequest:    true,
    LogResponse:   false,  // 生产环境建议关闭
    MaskSensitive: true,
    SensitiveFields: []string{
        "password", "token", "secret", "api_key",
    },
    SlowThreshold: 1000,  // 1秒
}

// 创建API日志中间件
apiLogMiddleware := middleware.APILog(apiLogConfig, logger)
```

### 记录敏感操作

```go
// 记录敏感操作（如删除、修改权限）
middleware.LogSensitiveOperation(ctx, logger, "delete_user", map[string]interface{}{
    "user_id": 123,
    "reason":  "user request",
})
```

### 记录安全事件

```go
// 记录安全事件（如登录失败、权限拒绝）
middleware.LogSecurityEvent(ctx, logger, "login_failed", map[string]interface{}{
    "username": "user@example.com",
    "reason":   "invalid password",
    "attempts": 3,
})
```

---

## 完整示例

### Consumer Service集成示例

```go
package main

import (
    "context"
    "log"

    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/redis/go-redis/v9"

    "go-wind-admin/pkg/auth"
    "go-wind-admin/pkg/middleware"
)

func main() {
    // 创建Redis客户端
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 创建JWT管理器
    blacklist := auth.NewTokenBlacklist(redisClient)
    jwtManager := auth.NewJWTManagerWithBlacklist("your-secret-key", blacklist)

    // 创建限流配置
    rateLimitConfig := &middleware.RateLimitConfig{
        Redis:             redisClient,
        UserRatePerMinute: 60,
        IPRatePerMinute:   100,
        WindowSize:        60,
        EnableUserLimit:   true,
        EnableIPLimit:     true,
    }

    // 创建安全配置
    securityConfig := &middleware.SecurityConfig{
        Redis:                   redisClient,
        EnableSQLInjectionCheck: true,
        EnableXSSCheck:          true,
        EnableIPBlacklist:       true,
    }

    // 创建API日志配置
    apiLogConfig := &middleware.APILogConfig{
        LogRequest:    true,
        LogResponse:   false,
        MaskSensitive: true,
        SlowThreshold: 1000,
    }

    // 创建HTTP服务器
    httpSrv := http.NewServer(
        http.Address(":8000"),
        http.Middleware(
            // 按顺序应用中间件
            middleware.APILog(apiLogConfig, logger),      // 1. 日志记录
            middleware.Security(securityConfig, logger),  // 2. 安全防护
            middleware.RateLimit(rateLimitConfig, logger), // 3. 限流
            // auth中间件（需要单独配置）
        ),
    )

    // 创建gRPC服务器
    grpcSrv := grpc.NewServer(
        grpc.Address(":9000"),
        grpc.Middleware(
            middleware.APILog(apiLogConfig, logger),
            middleware.Security(securityConfig, logger),
            middleware.RateLimit(rateLimitConfig, logger),
        ),
    )

    // 创建Kratos应用
    app := kratos.New(
        kratos.Name("consumer-service"),
        kratos.Server(httpSrv, grpcSrv),
    )

    // 启动应用
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 在Service层使用JWT

```go
package service

import (
    "context"
    
    pb "go-wind-admin/api/gen/go/consumer/service/v1"
    "go-wind-admin/pkg/auth"
)

type ConsumerService struct {
    pb.UnimplementedConsumerServiceServer
    
    jwtManager *auth.JWTManager
}

func (s *ConsumerService) LoginByPhone(ctx context.Context, req *pb.LoginByPhoneRequest) (*pb.LoginResponse, error) {
    // 验证用户密码...
    
    // 生成访问令牌
    accessToken, expiresIn, err := s.jwtManager.GenerateAccessToken(
        user.ID,
        user.TenantID,
        user.Phone,
    )
    if err != nil {
        return nil, err
    }
    
    // 生成刷新令牌
    refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, user.TenantID)
    if err != nil {
        return nil, err
    }
    
    return &pb.LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    expiresIn,
        User:         toProtoUser(user),
    }, nil
}

func (s *ConsumerService) Logout(ctx context.Context, req *pb.LogoutRequest) (*emptypb.Empty, error) {
    // 撤销访问令牌
    if err := s.jwtManager.RevokeToken(ctx, req.AccessToken); err != nil {
        return nil, err
    }
    
    // 撤销刷新令牌
    if err := s.jwtManager.RevokeToken(ctx, req.RefreshToken); err != nil {
        return nil, err
    }
    
    return &emptypb.Empty{}, nil
}
```

---

## 性能考虑

### Redis连接池

```go
redisClient := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,           // 连接池大小
    MinIdleConns: 10,            // 最小空闲连接
    MaxRetries:   3,             // 最大重试次数
})
```

### 限流窗口大小

- 窗口越小，限流越精确，但Redis压力越大
- 建议窗口大小：60秒（1分钟）
- 高并发场景可以适当增大窗口

### 日志级别

- 开发环境：LogRequest=true, LogResponse=true
- 生产环境：LogRequest=true, LogResponse=false
- 只记录慢请求和错误请求的详细信息

---

## 安全最佳实践

1. **JWT密钥管理**
   - 使用强随机密钥（至少32字节）
   - 定期轮换密钥
   - 不要硬编码密钥，使用环境变量或配置文件

2. **令牌有效期**
   - 访问令牌：2小时
   - 刷新令牌：7天
   - 敏感操作需要重新验证

3. **限流策略**
   - 用户级别：60次/分钟
   - IP级别：100次/分钟
   - 敏感接口（登录、注册）：更严格的限制

4. **IP黑名单**
   - 自动封禁：连续失败N次
   - 封禁时长：根据严重程度（1小时-永久）
   - 白名单：内部IP、可信IP

5. **数据脱敏**
   - 日志中脱敏所有敏感数据
   - API响应中脱敏（如手机号、邮箱）
   - 数据库查询结果脱敏

6. **输入验证**
   - 所有用户输入都要验证
   - 使用protoc-gen-validate进行Protobuf验证
   - 额外的SQL注入和XSS检测

---

## 监控和告警

### 关键指标

1. **限流指标**
   - 限流触发次数
   - 被限流的用户/IP
   - 限流触发的接口

2. **安全指标**
   - SQL注入检测次数
   - XSS攻击检测次数
   - IP黑名单命中次数

3. **性能指标**
   - API响应时间（P50/P95/P99）
   - 慢请求数量
   - 错误率

### 告警规则

```yaml
# 示例告警规则
alerts:
  - name: high_rate_limit_trigger
    condition: rate_limit_trigger_count > 100 in 5m
    severity: warning
    
  - name: sql_injection_detected
    condition: sql_injection_count > 0
    severity: critical
    
  - name: slow_api_requests
    condition: api_p95_duration > 2000ms
    severity: warning
```

---

## 故障排查

### 常见问题

1. **令牌验证失败**
   - 检查JWT密钥是否正确
   - 检查令牌是否过期
   - 检查令牌是否在黑名单中

2. **限流误触发**
   - 检查限流配置是否合理
   - 检查用户ID/IP提取是否正确
   - 检查Redis连接是否正常

3. **性能问题**
   - 检查Redis连接池配置
   - 检查日志级别配置
   - 检查中间件顺序

### 调试模式

```go
// 开启详细日志
logger := log.NewStdLogger(os.Stdout)
logger = log.With(logger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

// 禁用限流（仅用于调试）
rateLimitConfig.EnableUserLimit = false
rateLimitConfig.EnableIPLimit = false
```

---

## 参考资料

- [Kratos中间件文档](https://go-kratos.dev/docs/component/middleware/)
- [JWT最佳实践](https://tools.ietf.org/html/rfc8725)
- [OWASP安全指南](https://owasp.org/www-project-top-ten/)
