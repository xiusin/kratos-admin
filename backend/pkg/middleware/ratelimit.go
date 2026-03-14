package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// Redis客户端
	Redis *redis.Client

	// 用户级别限流：每分钟最大请求数
	UserRatePerMinute int

	// IP级别限流：每分钟最大请求数
	IPRatePerMinute int

	// 滑动窗口大小（秒）
	WindowSize int

	// 是否启用用户级别限流
	EnableUserLimit bool

	// 是否启用IP级别限流
	EnableIPLimit bool
}

// RateLimit 限流中间件
// 基于Redis的滑动窗口限流算法
func RateLimit(cfg *RateLimitConfig, logger log.Logger) middleware.Middleware {
	l := log.NewHelper(log.With(logger, "module", "middleware/ratelimit"))

	// 设置默认值
	if cfg.UserRatePerMinute == 0 {
		cfg.UserRatePerMinute = 60
	}
	if cfg.IPRatePerMinute == 0 {
		cfg.IPRatePerMinute = 100
	}
	if cfg.WindowSize == 0 {
		cfg.WindowSize = 60
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从transport中获取请求信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				l.Error("missing transport in context")
				return handler(ctx, req)
			}

			// 用户级别限流
			if cfg.EnableUserLimit {
				userID := getUserIDFromContext(ctx)
				if userID != "" {
					allowed, err := checkRateLimit(ctx, cfg, "user:"+userID, cfg.UserRatePerMinute)
					if err != nil {
						l.Errorf("rate limit check failed: %v", err)
						return nil, fmt.Errorf("rate limit check failed")
					}
					if !allowed {
						l.Warnf("user rate limit exceeded: user_id=%s", userID)
						return nil, fmt.Errorf("rate limit exceeded: too many requests")
					}
				}
			}

			// IP级别限流
			if cfg.EnableIPLimit {
				clientIP := getClientIP(tr)
				if clientIP != "" {
					allowed, err := checkRateLimit(ctx, cfg, "ip:"+clientIP, cfg.IPRatePerMinute)
					if err != nil {
						l.Errorf("rate limit check failed: %v", err)
						return nil, fmt.Errorf("rate limit check failed")
					}
					if !allowed {
						l.Warnf("ip rate limit exceeded: ip=%s", clientIP)
						return nil, fmt.Errorf("rate limit exceeded: too many requests")
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// checkRateLimit 检查限流
// 使用滑动窗口算法
func checkRateLimit(ctx context.Context, cfg *RateLimitConfig, key string, maxRequests int) (bool, error) {
	now := time.Now().Unix()
	windowStart := now - int64(cfg.WindowSize)

	// Redis key
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// 使用Redis的ZSET实现滑动窗口
	pipe := cfg.Redis.Pipeline()

	// 1. 删除窗口外的记录
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart, 10))

	// 2. 统计当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, redisKey)

	// 3. 添加当前请求
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})

	// 4. 设置过期时间
	pipe.Expire(ctx, redisKey, time.Duration(cfg.WindowSize)*time.Second)

	// 执行管道
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 获取当前请求数
	count := countCmd.Val()

	// 判断是否超过限制
	return count < int64(maxRequests), nil
}

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(ctx context.Context) string {
	// 这里需要根据实际的认证中间件实现来获取用户ID
	// 例如：
	// tokenPayload := auth.FromContext(ctx)
	// return strconv.FormatUint(tokenPayload.UserID, 10)
	
	return ""
}

// getClientIP 获取客户端IP
func getClientIP(tr transport.Transporter) string {
	// 尝试从X-Forwarded-For获取
	if ip := tr.RequestHeader().Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	// 尝试从X-Real-IP获取
	if ip := tr.RequestHeader().Get("X-Real-IP"); ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	// 注意：这里需要根据实际的transport实现来获取
	// 例如：
	// if httpTr, ok := tr.(*http.Transport); ok {
	//     return httpTr.Request().RemoteAddr
	// }

	return ""
}

// RateLimitByKey 按指定key限流
// 可用于自定义限流场景
func RateLimitByKey(ctx context.Context, cfg *RateLimitConfig, key string, maxRequests int) (bool, error) {
	return checkRateLimit(ctx, cfg, key, maxRequests)
}

// GetRateLimitInfo 获取限流信息
// 返回当前窗口内的请求数和剩余配额
func GetRateLimitInfo(ctx context.Context, cfg *RateLimitConfig, key string, maxRequests int) (current int64, remaining int64, err error) {
	now := time.Now().Unix()
	windowStart := now - int64(cfg.WindowSize)

	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// 删除窗口外的记录
	err = cfg.Redis.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart, 10)).Err()
	if err != nil {
		return 0, 0, err
	}

	// 统计当前窗口内的请求数
	current, err = cfg.Redis.ZCard(ctx, redisKey).Result()
	if err != nil {
		return 0, 0, err
	}

	remaining = int64(maxRequests) - current
	if remaining < 0 {
		remaining = 0
	}

	return current, remaining, nil
}
