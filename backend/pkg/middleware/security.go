package middleware

import (
	"context"
	"net"
	"regexp"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// 是否启用XSS防护
	EnableXSSProtection bool
	// 是否启用SQL注入防护
	EnableSQLInjectionProtection bool
	// 是否启用HTTPS重定向
	EnableHTTPSRedirect bool
	// 是否启用IP黑名单
	EnableIPBlacklist bool
	// IP黑名单列表
	IPBlacklist []string
	// 敏感字段列表（用于脱敏）
	SensitiveFields []string
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableXSSProtection:          true,
		EnableSQLInjectionProtection: true,
		EnableHTTPSRedirect:          false, // 默认关闭，生产环境建议开启
		EnableIPBlacklist:            true,
		IPBlacklist:                  []string{},
		SensitiveFields: []string{
			"password", "passwd", "pwd",
			"token", "secret", "key",
			"phone", "mobile", "tel",
			"id_card", "idcard", "identity",
			"credit_card", "creditcard",
		},
	}
}

// XSS攻击特征正则表达式
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
	regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`), // onclick, onerror, etc.
	regexp.MustCompile(`(?i)<img[^>]*onerror`),
}

// SQL注入攻击特征正则表达式
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\bor\b|\band\b)\s+[\w\d]+\s*=\s*[\w\d]+`),
	regexp.MustCompile(`(?i)union\s+select`),
	regexp.MustCompile(`(?i)drop\s+table`),
	regexp.MustCompile(`(?i)delete\s+from`),
	regexp.MustCompile(`(?i)insert\s+into`),
	regexp.MustCompile(`(?i)update\s+\w+\s+set`),
	regexp.MustCompile(`(?i)exec\s*\(`),
	regexp.MustCompile(`(?i)execute\s*\(`),
	regexp.MustCompile(`(?i)--`),                    // SQL注释
	regexp.MustCompile(`(?i)/\*.*?\*/`),             // SQL注释
	regexp.MustCompile(`(?i);\s*(drop|delete|update|insert)`),
}

// Security 安全防护中间件
func Security(cfg *SecurityConfig, logger log.Logger) middleware.Middleware {
	if cfg == nil {
		cfg = DefaultSecurityConfig()
	}

	l := log.NewHelper(log.With(logger, "module", "middleware/security"))

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从transport中获取请求信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 只处理HTTP请求
			htr, ok := tr.(*http.Transport)
			if !ok {
				return handler(ctx, req)
			}

			// 1. IP黑名单检查
			if cfg.EnableIPBlacklist && len(cfg.IPBlacklist) > 0 {
				clientIP := getClientIP(htr)
				if isIPBlacklisted(clientIP, cfg.IPBlacklist) {
					l.Warnf("blocked request from blacklisted IP: %s", clientIP)
					return nil, errors.Forbidden("FORBIDDEN", "access denied")
				}
			}

			// 2. HTTPS重定向检查
			if cfg.EnableHTTPSRedirect {
				if !isHTTPS(htr) {
					l.Warnf("non-HTTPS request detected: %s", htr.Request().RequestURI)
					return nil, errors.BadRequest("HTTPS_REQUIRED", "HTTPS is required")
				}
			}

			// 3. XSS防护检查（检查请求参数）
			if cfg.EnableXSSProtection {
				if err := checkXSS(htr); err != nil {
					l.Warnf("XSS attack detected: %v", err)
					return nil, err
				}
			}

			// 4. SQL注入防护检查
			if cfg.EnableSQLInjectionProtection {
				if err := checkSQLInjection(htr); err != nil {
					l.Warnf("SQL injection attack detected: %v", err)
					return nil, err
				}
			}

			// 继续处理请求
			return handler(ctx, req)
		}
	}
}

// isIPBlacklisted 检查IP是否在黑名单中
func isIPBlacklisted(ip string, blacklist []string) bool {
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	for _, blackIP := range blacklist {
		// 支持CIDR格式
		if strings.Contains(blackIP, "/") {
			_, ipNet, err := net.ParseCIDR(blackIP)
			if err == nil && ipNet.Contains(clientIP) {
				return true
			}
		} else {
			// 精确匹配
			if ip == blackIP {
				return true
			}
		}
	}

	return false
}

// isHTTPS 检查是否是HTTPS请求
func isHTTPS(htr *http.Transport) bool {
	// 检查请求协议
	if htr.Request().TLS != nil {
		return true
	}

	// 检查X-Forwarded-Proto头（反向代理场景）
	proto := htr.RequestHeader().Get("X-Forwarded-Proto")
	if proto == "https" {
		return true
	}

	return false
}

// checkXSS 检查XSS攻击
func checkXSS(htr *http.Transport) error {
	// 检查URL参数
	query := htr.Request().URL.RawQuery
	for _, pattern := range xssPatterns {
		if pattern.MatchString(query) {
			return errors.BadRequest("XSS_DETECTED", "potential XSS attack detected in query parameters")
		}
	}

	// 检查请求头
	for key, values := range htr.Request().Header {
		for _, value := range values {
			for _, pattern := range xssPatterns {
				if pattern.MatchString(value) {
					return errors.BadRequest("XSS_DETECTED", "potential XSS attack detected in header: "+key)
				}
			}
		}
	}

	return nil
}

// checkSQLInjection 检查SQL注入攻击
func checkSQLInjection(htr *http.Transport) error {
	// 检查URL参数
	query := htr.Request().URL.RawQuery
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(query) {
			return errors.BadRequest("SQL_INJECTION_DETECTED", "potential SQL injection attack detected in query parameters")
		}
	}

	// 检查路径参数
	path := htr.Request().URL.Path
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(path) {
			return errors.BadRequest("SQL_INJECTION_DETECTED", "potential SQL injection attack detected in path")
		}
	}

	return nil
}

// MaskSensitiveData 脱敏敏感数据
// 用于日志记录时脱敏敏感字段
func MaskSensitiveData(data map[string]interface{}, sensitiveFields []string) map[string]interface{} {
	if data == nil {
		return nil
	}

	masked := make(map[string]interface{})
	for key, value := range data {
		if isSensitiveField(key, sensitiveFields) {
			masked[key] = maskValue(value)
		} else {
			masked[key] = value
		}
	}

	return masked
}

// isSensitiveField 判断是否是敏感字段
func isSensitiveField(field string, sensitiveFields []string) bool {
	lowerField := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerField, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// maskValue 脱敏值
func maskValue(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "***"
	}

	length := len(str)
	if length == 0 {
		return ""
	}

	// 手机号脱敏：保留前3位和后4位
	if length == 11 && isPhoneNumber(str) {
		return str[:3] + "****" + str[7:]
	}

	// 身份证号脱敏：保留前6位和后4位
	if length == 18 && isIDCard(str) {
		return str[:6] + "********" + str[14:]
	}

	// 其他敏感信息：只显示前2位
	if length <= 2 {
		return "***"
	}
	if length <= 4 {
		return str[:1] + "***"
	}
	return str[:2] + "***"
}

// isPhoneNumber 判断是否是手机号
func isPhoneNumber(str string) bool {
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, str)
	return matched
}

// isIDCard 判断是否是身份证号
func isIDCard(str string) bool {
	matched, _ := regexp.MatchString(`^\d{17}[\dXx]$`, str)
	return matched
}
