package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-admin/app/consumer/service/internal/service"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
) []middleware.Middleware {
	var ms []middleware.Middleware

	// 日志中间件
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// 恢复中间件
	ms = append(ms, recovery.Recovery())

	// 验证中间件
	ms = append(ms, validate.Validator())

	// TODO: 添加认证中间件
	// TODO: 添加限流中间件
	// TODO: 添加租户中间件

	return ms
}

// NewRestServer 创建 REST 服务器
func NewRestServer(
	ctx *bootstrap.Context,
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

	middlewares := NewRestMiddleware(ctx)
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
