package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/service"
	"go-wind-admin/pkg/monitoring"
)

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	tracingService *monitoring.TracingService,
) []middleware.Middleware {
	var ms []middleware.Middleware

	// 日志中间件
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// 恢复中间件
	ms = append(ms, recovery.Recovery())

	// 追踪中间件
	if tracingService != nil {
		ms = append(ms, monitoring.TracingMiddleware(tracingService.GetTracer()))
	}

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
	middlewares []middleware.Middleware,
	consumerService *service.ConsumerService,
	smsService *service.SMSService,
	paymentService *service.PaymentService,
	financeService *service.FinanceService,
	wechatService *service.WechatService,
	mediaService *service.MediaService,
	logisticsService *service.LogisticsService,
	freightService *service.FreightService,
	healthService *monitoring.HealthService,
	metricsService *monitoring.MetricsService,
) (*khttp.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil, nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	// 注册监控端点
	registerMonitoringEndpoints(srv, ctx, healthService, metricsService)

	// 注册服务
	consumerV1.RegisterConsumerServiceHTTPServer(srv, consumerService)
	consumerV1.RegisterSMSServiceHTTPServer(srv, smsService)
	consumerV1.RegisterPaymentServiceHTTPServer(srv, paymentService)
	consumerV1.RegisterFinanceServiceHTTPServer(srv, financeService)
	consumerV1.RegisterWechatServiceHTTPServer(srv, wechatService)
	consumerV1.RegisterMediaServiceHTTPServer(srv, mediaService)
	consumerV1.RegisterLogisticsServiceHTTPServer(srv, logisticsService)
	consumerV1.RegisterFreightServiceHTTPServer(srv, freightService)

	return srv, nil
}

// registerMonitoringEndpoints 注册监控端点
func registerMonitoringEndpoints(srv *khttp.Server, ctx *bootstrap.Context, healthService *monitoring.HealthService, metricsService *monitoring.MetricsService) {
	// 健康检查端点
	srv.HandleFunc("/health", healthService.HealthHandler())
	srv.HandleFunc("/ready", healthService.ReadyHandler())
	srv.HandleFunc("/live", healthService.LiveHandler())

	// Prometheus指标端点
	srv.Handle("/metrics", metricsService.MetricsHandler())

	// 统计信息端点
	srv.HandleFunc("/stats", metricsService.StatsHandler())
}
