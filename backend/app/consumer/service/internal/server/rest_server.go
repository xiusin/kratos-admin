package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/redis/go-redis/v9"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/app/consumer/service/internal/service"
	pkgMiddleware "go-wind-admin/pkg/middleware"
	"go-wind-admin/pkg/middleware/auth"
	applogging "go-wind-admin/pkg/middleware/logging"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	accessTokenChecker auth.AccessTokenChecker,
	rdb *redis.Client,
) []middleware.Middleware {
	var ms []middleware.Middleware

	// 日志中间件（Kratos 基础日志）
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// 恢复中间件（panic 恢复）
	ms = append(ms, recovery.Recovery())

	// API 审计日志中间件（记录所有API调用）
	ms = append(ms, applogging.Server(
		applogging.WithWriteApiLogFunc(func(ctx context.Context, data *auditV1.ApiAuditLog) error {
			// Consumer Service 使用日志输出，不写入数据库
			// 生产环境可以配置日志收集系统（如 ELK、Loki）来收集和分析日志
			logger := ctx.Value("logger")
			if logger != nil {
				// TODO: 格式化输出审计日志
				// 包含：用户ID、租户ID、IP、操作、路径、状态码、响应时间等
			}
			return nil
		}),
		// Consumer Service 不需要登录日志（登录由 Admin Service 处理）
		applogging.WithWriteLoginLogFunc(nil),
	))

	// 验证中间件（Protobuf 字段验证）
	ms = append(ms, validate.Validator())

	// 安全防护中间件（XSS、SQL注入、IP黑名单、HTTPS重定向）
	securityCfg := pkgMiddleware.DefaultSecurityConfig()
	securityCfg.EnableHTTPSRedirect = false // 开发环境关闭，生产环境建议开启
	ms = append(ms, pkgMiddleware.Security(securityCfg, ctx.GetLogger()))

	// 限流中间件配置
	rateLimitCfg := &pkgMiddleware.RateLimitConfig{
		Redis:             rdb,
		UserRatePerMinute: 60,  // 每分钟60次
		IPRatePerMinute:   100, // 每分钟100次
		WindowSize:        60,  // 60秒窗口
		EnableUserLimit:   true,
		EnableIPLimit:     true,
	}

	// 添加白名单（不需要认证的接口）
	rpc.AddWhiteList(
		// 健康检查接口
		"/health",
		"/ready",
		// 微信回调接口
		"/api/wechat/callback",
		// TODO: 添加其他公开接口
	)

	// 认证和限流中间件（使用 selector 选择性应用）
	ms = append(ms, selector.Server(
		auth.Server(
			auth.WithAccessTokenChecker(accessTokenChecker),
			auth.WithInjectMetadata(true),
			auth.WithInjectTenantId(true),
			auth.WithInjectOperatorId(true),
			auth.WithInjectEnt(false),
			auth.WithEnableAuthority(false), // Consumer Service 不需要复杂的权限控制
		),
		pkgMiddleware.RateLimit(rateLimitCfg, ctx.GetLogger()),
	).
		Match(rpc.NewRestWhiteListMatcher()).
		Build(),
	)

	return ms
}

// NewRestServer 创建 REST 服务器
func NewRestServer(
	ctx *bootstrap.Context,
	middlewares []middleware.Middleware,
	consumerService *service.ConsumerService,
	smsService *service.SMSService,
	paymentService *service.PaymentService,
	financeService *service.FinanceService,
	wechatService *service.WechatService,
	mediaService *service.MediaService,
	logisticsService *service.LogisticsService,
	freightService *service.FreightService,
) (*khttp.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil, nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	// 注册健康检查接口
	registerHealthCheck(srv, ctx)

	// 注册 Consumer Service (gRPC-Gateway)
	// 由于Protobuf没有HTTP注解,这里暂时不注册HTTP服务
	// 可以通过gRPC-Gateway或者手动添加HTTP路由来实现REST API

	// TODO: 添加HTTP路由映射
	// TODO: 注册 SMS Service HTTP路由
	// TODO: 注册 Payment Service
	// TODO: 注册 Finance Service
	// TODO: 注册 Wechat Service (已添加到参数)
	// TODO: 注册 Media Service
	// TODO: 注册 Logistics Service
	// TODO: 注册 Freight Service

	// 暂时保留参数，避免编译错误
	_ = consumerService
	_ = smsService
	_ = paymentService
	_ = financeService
	_ = wechatService
	_ = mediaService
	_ = logisticsService
	_ = freightService

	// 注册微信事件回调接口
	registerWechatCallback(srv, wechatService, ctx)

	return srv, nil
}

// registerHealthCheck 注册健康检查接口
func registerHealthCheck(srv *khttp.Server, ctx *bootstrap.Context) {
	// /health - 基础健康检查
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := HealthResponse{
			Status: "UP",
			Services: map[string]string{
				"consumer-service": "UP",
			},
		}

		json.NewEncoder(w).Encode(resp)
	})

	// /ready - 就绪检查（检查依赖服务）
	srv.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// TODO: 检查数据库连接
		// TODO: 检查 Redis 连接
		// TODO: 检查 Kafka 连接

		// 暂时返回就绪状态
		w.WriteHeader(http.StatusOK)

		resp := HealthResponse{
			Status: "READY",
			Services: map[string]string{
				"database": "UP",
				"redis":    "UP",
				"kafka":    "UP",
			},
		}

		json.NewEncoder(w).Encode(resp)
	})
}

// registerWechatCallback 注册微信事件回调接口
func registerWechatCallback(srv *khttp.Server, wechatService *service.WechatService, ctx *bootstrap.Context) {
	logger := ctx.NewLoggerHelper("wechat/callback")

	// 微信事件回调接口（用于接收微信服务器推送的事件）
	srv.HandleFunc("/api/wechat/callback", func(w http.ResponseWriter, r *http.Request) {
		// 获取验证参数
		signature := r.URL.Query().Get("signature")
		timestamp := r.URL.Query().Get("timestamp")
		nonce := r.URL.Query().Get("nonce")
		echostr := r.URL.Query().Get("echostr")

		// GET 请求：微信服务器验证
		if r.Method == http.MethodGet {
			// 验证签名
			if !wechatService.VerifySignature(signature, timestamp, nonce) {
				logger.Errorf("Wechat signature verification failed: signature=%s, timestamp=%s, nonce=%s", signature, timestamp, nonce)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("signature verification failed"))
				return
			}

			logger.Infof("Wechat callback verification success: signature=%s, timestamp=%s, nonce=%s", signature, timestamp, nonce)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(echostr))
			return
		}

		// POST 请求：接收微信事件消息
		if r.Method == http.MethodPost {
			// 验证签名
			if !wechatService.VerifySignature(signature, timestamp, nonce) {
				logger.Errorf("Wechat signature verification failed: signature=%s, timestamp=%s, nonce=%s", signature, timestamp, nonce)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("signature verification failed"))
				return
			}

			// 读取请求体
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Errorf("read request body failed: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			// 解析 XML 消息
			logger.Infof("Received wechat event: %s", string(body))

			eventMsg, err := wechatService.ParseWechatEventXML(body)
			if err != nil {
				logger.Errorf("parse wechat event xml failed: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// 构建事件类型和数据
			eventType := eventMsg.MsgType
			if eventMsg.Event != "" {
				eventType = eventMsg.Event
			}

			eventData := map[string]interface{}{
				"to_user_name":   eventMsg.ToUserName,
				"from_user_name": eventMsg.FromUserName,
				"create_time":    eventMsg.CreateTime,
				"msg_type":       eventMsg.MsgType,
				"event":          eventMsg.Event,
				"event_key":      eventMsg.EventKey,
				"content":        eventMsg.Content,
				"raw_body":       string(body),
			}

			// 调用 WechatService 处理事件
			if err := wechatService.HandleWechatEvent(r.Context(), eventType, eventData); err != nil {
				logger.Errorf("handle wechat event failed: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// 返回成功响应
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
			return
		}

		// 其他请求方法不支持
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	logger.Info("Wechat callback registered: /api/wechat/callback")
}
