package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultJWTSecret 默认JWT密钥（生产环境应该从配置读取）
	DefaultJWTSecret = "your-secret-key-change-in-production"
	// AccessTokenExpire 访问令牌过期时间
	AccessTokenExpire = 2 * time.Hour
	// RefreshTokenExpire 刷新令牌过期时间
	RefreshTokenExpire = 7 * 24 * time.Hour
)

// Claims JWT声明
type Claims struct {
	UserID   uint32 `json:"user_id"`
	TenantID uint32 `json:"tenant_id"`
	Phone    string `json:"phone,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secret    []byte
	blacklist *TokenBlacklist
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secret string) *JWTManager {
	if secret == "" {
		secret = DefaultJWTSecret
	}
	return &JWTManager{
		secret: []byte(secret),
	}
}

// NewJWTManagerWithBlacklist 创建带黑名单的JWT管理器
func NewJWTManagerWithBlacklist(secret string, blacklist *TokenBlacklist) *JWTManager {
	if secret == "" {
		secret = DefaultJWTSecret
	}
	return &JWTManager{
		secret:    []byte(secret),
		blacklist: blacklist,
	}
}

// GenerateAccessToken 生成访问令牌
func (m *JWTManager) GenerateAccessToken(userID, tenantID uint32, phone string) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(AccessTokenExpire)

	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		Phone:    phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "consumer-service",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, int64(AccessTokenExpire.Seconds()), nil
}

// GenerateRefreshToken 生成刷新令牌
func (m *JWTManager) GenerateRefreshToken(userID, tenantID uint32) (string, error) {
	now := time.Now()
	expiresAt := now.Add(RefreshTokenExpire)

	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "consumer-service",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateToken 验证令牌
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshAccessToken 刷新访问令牌
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, int64, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", 0, err
	}

	return m.GenerateAccessToken(claims.UserID, claims.TenantID, claims.Phone)
}

// ValidateTokenWithBlacklist 验证令牌（包含黑名单检查）
func (m *JWTManager) ValidateTokenWithBlacklist(ctx context.Context, tokenString string) (*Claims, error) {
	// 先验证令牌格式和签名
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 如果配置了黑名单，检查令牌是否被撤销
	if m.blacklist != nil {
		// 检查令牌是否在黑名单中
		isBlacklisted, err := m.blacklist.IsBlacklisted(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to check blacklist: %w", err)
		}
		if isBlacklisted {
			return nil, fmt.Errorf("token has been revoked")
		}

		// 检查用户是否被全局撤销
		isUserBlacklisted, err := m.blacklist.IsUserBlacklisted(ctx, claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user blacklist: %w", err)
		}
		if isUserBlacklisted {
			return nil, fmt.Errorf("user tokens have been revoked")
		}
	}

	return claims, nil
}

// RevokeToken 撤销令牌（加入黑名单）
func (m *JWTManager) RevokeToken(ctx context.Context, tokenString string) error {
	if m.blacklist == nil {
		return fmt.Errorf("blacklist not configured")
	}

	// 解析令牌获取过期时间
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return err
	}

	// 计算剩余有效期
	expiration := time.Until(claims.ExpiresAt.Time)
	if expiration <= 0 {
		// 令牌已过期，无需加入黑名单
		return nil
	}

	// 加入黑名单
	return m.blacklist.AddToken(ctx, tokenString, expiration)
}

// RevokeUserTokens 撤销用户的所有令牌
func (m *JWTManager) RevokeUserTokens(ctx context.Context, userID uint32) error {
	if m.blacklist == nil {
		return fmt.Errorf("blacklist not configured")
	}

	// 使用最长的令牌有效期（刷新令牌的有效期）
	return m.blacklist.AddUserTokens(ctx, userID, RefreshTokenExpire)
}

// SetBlacklist 设置黑名单
func (m *JWTManager) SetBlacklist(blacklist *TokenBlacklist) {
	m.blacklist = blacklist
}
