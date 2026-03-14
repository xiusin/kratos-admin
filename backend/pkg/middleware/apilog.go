package middleware

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// APILogConfig API日志配置
type APILogConfig struct {
	// 是否记录请求参数
	LogRequest bool

	// 是否记录响应数据
	LogResponse bool

	// 是否脱敏敏感参数
	MaskSensitive bool

	// 敏感字段列表（需要脱敏）
	SensitiveFields []string

	// 慢请求阈值（毫秒）
	SlowThreshold int64
}

// APILogEntry API日志条目
type APILogEntry struct {
	// 请求信息
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Query   string      `json:"query,omitempty"`
	Request interface{} `json:"request,omitempty"`

	// 响应信息
	Response   interface{} `json:"response,omitempty"`
	StatusCode int         `json:"status_code"`
	Error      string      `json:"error,omitempty"`

	// 性能信息
	Duration int64 `json:"duration_ms"`

	// 用户信息
	UserID    uint32 `json:"user_id,omitempty"`
	TenantID  uint32 `json:"tenant_id,omitempty"`
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent,omitempty"`

	// 时间信息
	Timestamp time.Time `json:"timestamp"`
}

// APILog API日志记录中间件
func APILog(cfg *APILogConfig, logger log.Logger) middleware.Middleware {
	l := log.NewHelper(log.With(logger, "module", "middleware/apilog"))

	// 设置默认值
	if cfg.SlowThreshold == 0 {
		cfg.SlowThreshold = 1000 // 默认1秒
	}
	if cfg.SensitiveFields == nil {
		cfg.SensitiveFields = []string{
			"password", "passwd", "pwd",
			"token", "access_token", "refresh_token",
			"secret", "api_key", "apikey",
			"credit_card", "card_number",
			"ssn", "id_card", "idcard",
		}
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startTime := time.Now()

			// 获取transport信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 构建日志条目
			entry := &APILogEntry{
				Timestamp: startTime,
				ClientIP:  GetIPAddress(ctx),
				UserAgent: GetUserAgent(ctx),
				UserID:    GetUserID(ctx),
				TenantID:  GetTenantID(ctx),
			}

			// 获取请求信息
			if tr.Kind() == transport.KindHTTP {
				if ht, ok := tr.(transport.Transporter); ok {
					entry.Method = ht.Operation()
					entry.Path = ht.Operation()
				}
			} else {
				entry.Method = tr.Operation()
			}

			// 记录请求参数
			if cfg.LogRequest {
				if cfg.MaskSensitive {
					entry.Request = maskSensitiveData(req, cfg.SensitiveFields)
				} else {
					entry.Request = req
				}
			}

			// 执行处理器
			reply, err := handler(ctx, req)

			// 计算耗时
			duration := time.Since(startTime)
			entry.Duration = duration.Milliseconds()

			// 记录响应
			if err != nil {
				entry.Error = err.Error()
				entry.StatusCode = 500
			} else {
				entry.StatusCode = 200
				if cfg.LogResponse {
					if cfg.MaskSensitive {
						entry.Response = maskSensitiveData(reply, cfg.SensitiveFields)
					} else {
						entry.Response = reply
					}
				}
			}

			// 根据日志级别记录
			logLevel := getLogLevel(entry, cfg)
			logMessage := formatLogMessage(entry)

			switch logLevel {
			case "ERROR":
				l.Errorw(
					"api_call",
					"method", entry.Method,
					"path", entry.Path,
					"status", entry.StatusCode,
					"duration_ms", entry.Duration,
					"error", entry.Error,
					"user_id", entry.UserID,
					"tenant_id", entry.TenantID,
					"client_ip", entry.ClientIP,
				)
			case "WARN":
				l.Warnw(
					"api_call",
					"method", entry.Method,
					"path", entry.Path,
					"status", entry.StatusCode,
					"duration_ms", entry.Duration,
					"user_id", entry.UserID,
					"tenant_id", entry.TenantID,
					"client_ip", entry.ClientIP,
				)
			default:
				l.Infow(
					"api_call",
					"method", entry.Method,
					"path", entry.Path,
					"status", entry.StatusCode,
					"duration_ms", entry.Duration,
					"user_id", entry.UserID,
					"tenant_id", entry.TenantID,
					"client_ip", entry.ClientIP,
				)
			}

			// 如果是慢请求，额外记录警告
			if entry.Duration > cfg.SlowThreshold {
				l.Warnf("slow request detected: %s", logMessage)
			}

			return reply, err
		}
	}
}

// getLogLevel 获取日志级别
func getLogLevel(entry *APILogEntry, cfg *APILogConfig) string {
	// 错误请求
	if entry.StatusCode >= 500 {
		return "ERROR"
	}

	// 慢请求或客户端错误
	if entry.Duration > cfg.SlowThreshold || entry.StatusCode >= 400 {
		return "WARN"
	}

	return "INFO"
}

// formatLogMessage 格式化日志消息
func formatLogMessage(entry *APILogEntry) string {
	return fmt.Sprintf(
		"[%s] %s %s - %d - %dms - user:%d tenant:%d ip:%s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.Duration,
		entry.UserID,
		entry.TenantID,
		entry.ClientIP,
	)
}

// maskSensitiveData 脱敏敏感数据
func maskSensitiveData(data interface{}, sensitiveFields []string) interface{} {
	if data == nil {
		return nil
	}

	// 将数据转换为字符串
	dataStr := fmt.Sprintf("%+v", data)

	// 对每个敏感字段进行脱敏
	for _, field := range sensitiveFields {
		// 匹配字段名和值的模式
		// 例如: password:"123456" -> password:"***"
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)%s[:\s]*["']?([^"'\s}]+)["']?`, field))
		dataStr = pattern.ReplaceAllString(dataStr, fmt.Sprintf(`%s:"***"`, field))
	}

	return dataStr
}

// LogSensitiveOperation 记录敏感操作
// 用于记录需要审计的敏感操作（如删除、修改权限等）
func LogSensitiveOperation(ctx context.Context, logger log.Logger, operation string, details map[string]interface{}) {
	l := log.NewHelper(log.With(logger, "module", "audit"))

	l.Infow(
		"sensitive_operation",
		"operation", operation,
		"user_id", GetUserID(ctx),
		"tenant_id", GetTenantID(ctx),
		"client_ip", GetIPAddress(ctx),
		"timestamp", time.Now().Format(time.RFC3339),
		"details", details,
	)
}

// LogSecurityEvent 记录安全事件
// 用于记录安全相关的事件（如登录失败、权限拒绝等）
func LogSecurityEvent(ctx context.Context, logger log.Logger, eventType string, details map[string]interface{}) {
	l := log.NewHelper(log.With(logger, "module", "security"))

	l.Warnw(
		"security_event",
		"event_type", eventType,
		"user_id", GetUserID(ctx),
		"tenant_id", GetTenantID(ctx),
		"client_ip", GetIPAddress(ctx),
		"timestamp", time.Now().Format(time.RFC3339),
		"details", details,
	)
}
