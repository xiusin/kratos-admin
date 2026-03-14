package middleware

import (
	"context"
	"net"
	"strings"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// TenantIDKey 租户ID键
	TenantIDKey ContextKey = "tenant_id"
	// UserIDKey 用户ID键
	UserIDKey ContextKey = "user_id"
	// IPAddressKey IP地址键
	IPAddressKey ContextKey = "ip_address"
	// UserAgentKey User Agent键
	UserAgentKey ContextKey = "user_agent"
)

// GetTenantID 从上下文获取租户ID
func GetTenantID(ctx context.Context) uint32 {
	if v := ctx.Value(TenantIDKey); v != nil {
		if tenantID, ok := v.(uint32); ok {
			return tenantID
		}
	}
	return 1 // 默认租户ID
}

// GetUserID 从上下文获取用户ID
func GetUserID(ctx context.Context) uint32 {
	if v := ctx.Value(UserIDKey); v != nil {
		if userID, ok := v.(uint32); ok {
			return userID
		}
	}
	return 0
}

// GetIPAddress 从上下文获取IP地址
func GetIPAddress(ctx context.Context) string {
	// 先尝试从上下文获取
	if v := ctx.Value(IPAddressKey); v != nil {
		if ip, ok := v.(string); ok && ip != "" {
			return ip
		}
	}

	// 从transport获取
	if tr, ok := transport.FromServerContext(ctx); ok {
		if ht, ok := tr.(http.Transporter); ok {
			req := ht.Request()

			// 尝试从X-Forwarded-For获取
			if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
				ips := strings.Split(xff, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}

			// 尝试从X-Real-IP获取
			if xri := req.Header.Get("X-Real-IP"); xri != "" {
				return xri
			}

			// 从RemoteAddr获取
			if req.RemoteAddr != "" {
				ip, _, err := net.SplitHostPort(req.RemoteAddr)
				if err == nil {
					return ip
				}
				return req.RemoteAddr
			}
		}
	}

	return "127.0.0.1"
}

// GetUserAgent 从上下文获取User Agent
func GetUserAgent(ctx context.Context) string {
	// 先尝试从上下文获取
	if v := ctx.Value(UserAgentKey); v != nil {
		if ua, ok := v.(string); ok && ua != "" {
			return ua
		}
	}

	// 从transport获取
	if tr, ok := transport.FromServerContext(ctx); ok {
		if ht, ok := tr.(http.Transporter); ok {
			return ht.Request().Header.Get("User-Agent")
		}
	}

	return "Unknown"
}

// GetDeviceType 从User Agent判断设备类型
func GetDeviceType(userAgent string) string {
	userAgentLower := strings.ToLower(userAgent)

	if strings.Contains(userAgentLower, "mobile") ||
		strings.Contains(userAgentLower, "android") ||
		strings.Contains(userAgentLower, "iphone") {
		return "mobile"
	} else if strings.Contains(userAgentLower, "tablet") ||
		strings.Contains(userAgentLower, "ipad") {
		return "tablet"
	}

	return "desktop"
}

// WithTenantID 设置租户ID到上下文
func WithTenantID(ctx context.Context, tenantID uint32) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// WithUserID 设置用户ID到上下文
func WithUserID(ctx context.Context, userID uint32) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithIPAddress 设置IP地址到上下文
func WithIPAddress(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, IPAddressKey, ip)
}

// WithUserAgent 设置User Agent到上下文
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, UserAgentKey, ua)
}
