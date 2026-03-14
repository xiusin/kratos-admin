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
	middlewares []middleware.Middleware,
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

	// TODO: 注册 Consumer Service
	// TODO: 注册 SMS Service
	// TODO: 注册 Payment Service
	// TODO: 注册 Finance Service
	// TODO: 注册 Wechat Service
	// TODO: 注册 Media Service
	// TODO: 注册 Logistics Service
	// TODO: 注册 Freight Service

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
