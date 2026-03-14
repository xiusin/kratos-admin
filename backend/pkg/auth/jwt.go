package auth

import (
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
	secret []byte
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
