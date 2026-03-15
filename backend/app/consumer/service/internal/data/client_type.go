package data

import (
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// NewClientType 创建客户端类型（Consumer Service 使用 app 类型）
func NewClientType() authenticationV1.ClientType {
	return authenticationV1.ClientType_app
}
