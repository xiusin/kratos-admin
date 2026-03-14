package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist JWT令牌黑名单
// 用于实现令牌撤销功能
type TokenBlacklist struct {
	redis *redis.Client
}

// NewTokenBlacklist 创建令牌黑名单
func NewTokenBlacklist(redis *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		redis: redis,
	}
}

// AddToken 添加令牌到黑名单
// token: JWT令牌字符串
// expiration: 令牌过期时间（用于设置Redis过期时间）
func (b *TokenBlacklist) AddToken(ctx context.Context, token string, expiration time.Duration) error {
	key := fmt.Sprintf("jwt:blacklist:%s", token)
	return b.redis.Set(ctx, key, "1", expiration).Err()
}

// IsBlacklisted 检查令牌是否在黑名单中
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("jwt:blacklist:%s", token)
	exists, err := b.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// RemoveToken 从黑名单中移除令牌（通常不需要，因为会自动过期）
func (b *TokenBlacklist) RemoveToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("jwt:blacklist:%s", token)
	return b.redis.Del(ctx, key).Err()
}

// AddUserTokens 将用户的所有令牌加入黑名单
// 用于用户登出所有设备的场景
func (b *TokenBlacklist) AddUserTokens(ctx context.Context, userID uint32, expiration time.Duration) error {
	key := fmt.Sprintf("jwt:blacklist:user:%d", userID)
	return b.redis.Set(ctx, key, "1", expiration).Err()
}

// IsUserBlacklisted 检查用户的所有令牌是否被撤销
func (b *TokenBlacklist) IsUserBlacklisted(ctx context.Context, userID uint32) (bool, error) {
	key := fmt.Sprintf("jwt:blacklist:user:%d", userID)
	exists, err := b.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ClearUserBlacklist 清除用户的黑名单标记
func (b *TokenBlacklist) ClearUserBlacklist(ctx context.Context, userID uint32) error {
	key := fmt.Sprintf("jwt:blacklist:user:%d", userID)
	return b.redis.Del(ctx, key).Err()
}
