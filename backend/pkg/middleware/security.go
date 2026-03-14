package middleware

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/redis/go-redis/v9"
)

// 安全错误定义
var (
	ErrSQLInjectionDetected = errors.BadRequest("SQL_INJECTION_DETECTED", "potential SQL injection detected")
	ErrXSSDetected          = errors.BadRequest("XSS_DETECTED", "potential XSS attack detected")
	ErrIPBlacklisted        = errors.Forbidden("IP_BLACKLISTED", "your IP address has been blacklisted")
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// Redis客户端（用于IP黑名单）
	Redis *redis.Client

	// 是否启用SQL注入检测
	EnableSQLInjectionCheck bool

	// 是否启用XSS检测
	EnableXSSCheck bool

	// 是否启用IP黑名单
	EnableIPBlacklist bool

	// 是否强制HTTPS
	EnforceHTTPS bool
}

// SQL注入检测正则表达式
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union.*select|select.*from|insert.*into|delete.*from|drop.*table|update.*set)`),
	regexp.MustCompile(`(?i)(exec|execute|script|javascript|<script|</script>)`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1|and\s+1\s*=\s*1)`),
	regexp.MustCompile(`(?i)(--|;|\/\*|\*\/)`),
}

// XSS检测正则表达式
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`), // onclick, onload等
	regexp.MustCompile(`(?i)<iframe[^>]*>`),
	regexp.MustCompile(`(?i)<object[^>]*>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
}

// Security 安全防护中间件
func Security(cfg *SecurityConfig, logger log.Logger) middleware.Middleware {
	l := log.NewHelper(log.With(logger, "module", "middleware/security"))

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// IP黑名单检查
			if cfg.EnableIPBlacklist && cfg.Redis != nil {
				clientIP := GetIPAddress(ctx)
				isBlacklisted, err := checkIPBlacklist(ctx, cfg.Redis, clientIP)
				if err != nil {
					l.Errorf("failed to check IP blacklist: %v", err)
				} else if isBlacklisted {
					l.Warnf("blocked blacklisted IP: %s", clientIP)
					return nil, ErrIPBlacklisted
				}
			}

			// 输入验证
			if cfg.EnableSQLInjectionCheck || cfg.EnableXSSCheck {
				if err := validateInput(req, cfg); err != nil {
					l.Warnf("input validation failed: %v", err)
					return nil, err
				}
			}

			return handler(ctx, req)
		}
	}
}

// validateInput 验证输入参数
func validateInput(req interface{}, cfg *SecurityConfig) error {
	// 将请求转换为字符串进行检查
	reqStr := fmt.Sprintf("%+v", req)

	// SQL注入检测
	if cfg.EnableSQLInjectionCheck {
		for _, pattern := range sqlInjectionPatterns {
			if pattern.MatchString(reqStr) {
				return ErrSQLInjectionDetected
			}
		}
	}

	// XSS检测
	if cfg.EnableXSSCheck {
		for _, pattern := range xssPatterns {
			if pattern.MatchString(reqStr) {
				return ErrXSSDetected
			}
		}
	}

	return nil
}

// checkIPBlacklist 检查IP是否在黑名单中
func checkIPBlacklist(ctx context.Context, redis *redis.Client, ip string) (bool, error) {
	key := fmt.Sprintf("security:ip:blacklist:%s", ip)
	exists, err := redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// AddIPToBlacklist 添加IP到黑名单
func AddIPToBlacklist(ctx context.Context, redis *redis.Client, ip string, duration int64) error {
	key := fmt.Sprintf("security:ip:blacklist:%s", ip)
	return redis.Set(ctx, key, "1", 0).Err()
}

// RemoveIPFromBlacklist 从黑名单移除IP
func RemoveIPFromBlacklist(ctx context.Context, redis *redis.Client, ip string) error {
	key := fmt.Sprintf("security:ip:blacklist:%s", ip)
	return redis.Del(ctx, key).Err()
}

// SanitizeString 清理字符串，移除潜在的危险字符
func SanitizeString(input string) string {
	// 移除HTML标签
	input = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(input, "")

	// 移除JavaScript
	input = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(input, "")

	// 移除SQL关键字
	input = regexp.MustCompile(`(?i)(union|select|insert|delete|drop|update|exec|execute)`).ReplaceAllString(input, "")

	return strings.TrimSpace(input)
}

// MaskPhone 脱敏手机号
// 示例: 13812345678 -> 138****5678
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskEmail 脱敏邮箱
// 示例: user@example.com -> u***@example.com
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	if len(username) <= 1 {
		return email
	}

	masked := string(username[0]) + "***"
	return masked + "@" + parts[1]
}

// MaskIDCard 脱敏身份证号
// 示例: 110101199001011234 -> 110101********1234
func MaskIDCard(idCard string) string {
	if len(idCard) != 18 {
		return idCard
	}
	return idCard[:6] + "********" + idCard[14:]
}

// MaskBankCard 脱敏银行卡号
// 示例: 6222021234567890123 -> 622202*********0123
func MaskBankCard(bankCard string) string {
	if len(bankCard) < 8 {
		return bankCard
	}
	return bankCard[:6] + strings.Repeat("*", len(bankCard)-10) + bankCard[len(bankCard)-4:]
}

// MaskSensitiveData 脱敏敏感数据
// 根据数据类型自动选择脱敏方式
func MaskSensitiveData(dataType string, data string) string {
	switch dataType {
	case "phone":
		return MaskPhone(data)
	case "email":
		return MaskEmail(data)
	case "idcard":
		return MaskIDCard(data)
	case "bankcard":
		return MaskBankCard(data)
	default:
		// 默认脱敏：只显示前后各2个字符
		if len(data) <= 4 {
			return "***"
		}
		return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
	}
}
