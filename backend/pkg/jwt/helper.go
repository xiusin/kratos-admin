package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	authn "github.com/tx7do/kratos-authn/engine"
)

// Helper JWT辅助工具
type Helper struct {
	secretKey string
	issuer    string
}

// NewHelper 创建JWT辅助工具
func NewHelper(secretKey, issuer string) *Helper {
	return &Helper{
		secretKey: secretKey,
		issuer:    issuer,
	}
}

// GenerateToken 生成JWT令牌
func (h *Helper) GenerateToken(userID uint32, duration time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(duration)

	claims := authn.AuthClaims{
		ClaimFieldUserID:                 userID,
		authn.ClaimFieldIssuedAt:         now.Unix(),
		authn.ClaimFieldExpirationTime:   expiresAt.Unix(),
		authn.ClaimFieldIssuer:           h.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	return token.SignedString([]byte(h.secretKey))
}

// ParseToken 解析JWT令牌
func (h *Helper) ParseToken(tokenString string) (*authn.AuthClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		authClaims := authn.AuthClaims(claims)
		return &authClaims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// ValidateToken 验证JWT令牌
func (h *Helper) ValidateToken(tokenString string) (uint32, error) {
	claims, err := h.ParseToken(tokenString)
	if err != nil {
		return 0, err
	}

	// 检查是否过期
	if IsTokenExpired(claims) {
		return 0, jwt.ErrTokenExpired
	}

	// 检查是否未生效
	if IsTokenNotValidYet(claims) {
		return 0, jwt.ErrTokenNotValidYet
	}

	userID, err := claims.GetUint32(ClaimFieldUserID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}
