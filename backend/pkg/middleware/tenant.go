package middleware

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// TenantContextKey 租户上下文键
type TenantContextKey struct{}

// TenantInfo 租户信息
type TenantInfo struct {
	TenantID   uint32
	TenantName string
}

// Tenant 多租户中间件
// 从JWT或请求头中提取租户信息，并注入到上下文中
func Tenant(logger log.Logger) middleware.Middleware {
	l := log.NewHelper(log.With(logger, "module", "middleware/tenant"))

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从transport中获取请求信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				l.Error("missing transport in context")
				return nil, fmt.Errorf("missing transport in context")
			}

			// 尝试从请求头中获取租户ID
			tenantIDStr := tr.RequestHeader().Get("X-Tenant-ID")
			if tenantIDStr == "" {
				// 如果请求头中没有，尝试从JWT中获取（假设JWT已经被auth中间件处理）
				// 这里可以从上下文中获取JWT payload
				tenantIDStr = getTenantIDFromContext(ctx)
			}

			if tenantIDStr == "" {
				l.Error("missing tenant id")
				return nil, fmt.Errorf("missing tenant id")
			}

			// 解析租户ID
			tenantID, err := strconv.ParseUint(tenantIDStr, 10, 32)
			if err != nil {
				l.Errorf("invalid tenant id: %s", tenantIDStr)
				return nil, fmt.Errorf("invalid tenant id")
			}

			// 构建租户信息
			tenantInfo := &TenantInfo{
				TenantID: uint32(tenantID),
			}

			// 注入到上下文
			ctx = context.WithValue(ctx, TenantContextKey{}, tenantInfo)

			l.Debugf("tenant middleware: tenant_id=%d", tenantID)

			return handler(ctx, req)
		}
	}
}

// TenantFromContext 从上下文中获取租户信息
func TenantFromContext(ctx context.Context) (*TenantInfo, bool) {
	tenantInfo, ok := ctx.Value(TenantContextKey{}).(*TenantInfo)
	return tenantInfo, ok
}

// MustTenantFromContext 从上下文中获取租户信息（必须存在）
func MustTenantFromContext(ctx context.Context) *TenantInfo {
	tenantInfo, ok := TenantFromContext(ctx)
	if !ok {
		panic("missing tenant info in context")
	}
	return tenantInfo
}

// getTenantIDFromContext 从上下文中获取租户ID（从JWT payload）
func getTenantIDFromContext(ctx context.Context) string {
	// 这里需要根据实际的JWT payload结构来获取租户ID
	// 假设JWT payload已经被auth中间件注入到上下文中
	// 这里只是一个示例实现

	// 可以从auth中间件注入的上下文中获取
	// 例如：tokenPayload := auth.FromContext(ctx)
	// return strconv.FormatUint(tokenPayload.TenantID, 10)

	return ""
}

// ValidateTenantAccess 验证租户访问权限
// 确保用户只能访问自己租户的数据
func ValidateTenantAccess(ctx context.Context, resourceTenantID uint32) error {
	tenantInfo, ok := TenantFromContext(ctx)
	if !ok {
		return fmt.Errorf("missing tenant info in context")
	}

	if tenantInfo.TenantID != resourceTenantID {
		return fmt.Errorf("tenant access denied: user tenant_id=%d, resource tenant_id=%d",
			tenantInfo.TenantID, resourceTenantID)
	}

	return nil
}

// InjectTenantID 注入租户ID到请求对象
// 用于自动设置创建/更新操作的租户ID
func InjectTenantID(ctx context.Context, req interface{}) error {
	tenantInfo, ok := TenantFromContext(ctx)
	if !ok {
		return fmt.Errorf("missing tenant info in context")
	}

	// 使用反射设置租户ID字段
	// 这里需要根据实际的请求结构来实现
	// 例如：
	// if setter, ok := req.(TenantIDSetter); ok {
	//     setter.SetTenantID(tenantInfo.TenantID)
	// }

	_ = tenantInfo // 避免未使用变量警告

	return nil
}

// TenantIDSetter 租户ID设置器接口
type TenantIDSetter interface {
	SetTenantID(tenantID uint32)
}

// TenantIDGetter 租户ID获取器接口
type TenantIDGetter interface {
	GetTenantID() uint32
}
