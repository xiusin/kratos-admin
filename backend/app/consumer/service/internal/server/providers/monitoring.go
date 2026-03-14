package providers

import (
	"context"
	"database/sql"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/pkg/monitoring"
)

// ProvideHealthService 提供健康检查服务
func ProvideHealthService(
	ctx *bootstrap.Context,
	db *sql.DB,
	redis redis.UniversalClient,
) *monitoring.HealthService {
	logger := ctx.GetLogger()
	healthService := monitoring.NewHealthService(logger)

	// 注册健康检查器
	healthService.RegisterChecker(monitoring.NewDatabaseHealthChecker(db, logger))
	healthService.RegisterChecker(monitoring.NewRedisHealthChecker(redis, logger))

	// 从配置获取Kafka地址
	// TODO: 修复 Kafka 配置字段名
	// cfg := ctx.GetConfig()
	// if cfg != nil && cfg.Server != nil && cfg.Server.Kafka != nil && len(cfg.Server.Kafka.Brokers) > 0 {
	// 	healthService.RegisterChecker(monitoring.NewKafkaHealthChecker(cfg.Server.Kafka.Brokers, logger))
	// }

	return healthService
}

// ProvideMetricsService 提供指标服务
func ProvideMetricsService(
	ctx *bootstrap.Context,
	db *sql.DB,
	redis redis.UniversalClient,
) *monitoring.MetricsService {
	logger := ctx.GetLogger()
	metricsService := monitoring.NewMetricsService(db, redis, logger)

	// 启动指标收集
	go func() {
		if err := metricsService.Start(context.Background()); err != nil {
			log.Errorf("failed to start metrics service: %v", err)
		}
	}()

	return metricsService
}

// ProvideAlertService 提供告警服务
func ProvideAlertService(ctx *bootstrap.Context) *monitoring.AlertService {
	logger := ctx.GetLogger()
	alertService := monitoring.NewAlertService(logger)

	// TODO: 从配置读取告警通道配置并注册
	// cfg := ctx.GetConfig()
	// if cfg.Monitoring.Alert.DingTalk.Enabled {
	//     alertService.RegisterChannel(monitoring.NewDingTalkChannel(...))
	// }

	return alertService
}

// ProvideMonitor 提供监控守护进程
func ProvideMonitor(
	ctx *bootstrap.Context,
	healthService *monitoring.HealthService,
	metricsService *monitoring.MetricsService,
	alertService *monitoring.AlertService,
) *monitoring.Monitor {
	logger := ctx.GetLogger()
	config := monitoring.DefaultMonitorConfig()

	monitor := monitoring.NewMonitor(config, healthService, metricsService, alertService, logger)

	// 启动监控
	go func() {
		if err := monitor.Start(context.Background()); err != nil {
			log.Errorf("failed to start monitor: %v", err)
		}
	}()

	return monitor
}

// ProvideTracingService 提供追踪服务
func ProvideTracingService(ctx *bootstrap.Context) (*monitoring.TracingService, error) {
	logger := ctx.GetLogger()

	// TODO: 从配置读取追踪配置
	cfg := monitoring.TracingConfig{
		ServiceName:    "consumer-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4317",
		SamplingRate:   1.0,
	}

	tracingService, err := monitoring.NewTracingService(cfg, logger)
	if err != nil {
		return nil, err
	}

	// 启动追踪服务
	go func() {
		if err := tracingService.Start(context.Background()); err != nil {
			log.Errorf("failed to start tracing service: %v", err)
		}
	}()

	return tracingService, nil
}

// MonitoringProviderSet 监控服务Provider集合
var MonitoringProviderSet = wire.NewSet(
	ProvideHealthService,
	ProvideMetricsService,
	ProvideAlertService,
	ProvideMonitor,
	ProvideTracingService,
)
