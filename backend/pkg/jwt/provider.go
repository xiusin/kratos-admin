package jwt

import (
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// NewJWTHelper 创建JWT辅助工具
func NewJWTHelper(ctx *bootstrap.Context) *Helper {
	// 使用默认配置
	// TODO: 从配置文件读取 JWT 配置
	secret := "default-secret-key-change-in-production"
	issuer := "consumer-service"
	
	return NewHelper(secret, issuer)
}
