package monitoring

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracingConfig 追踪配置
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string // OTLP gRPC endpoint (e.g., "localhost:4317")
	SamplingRate   float64
}

// TracingService 追踪服务
type TracingService struct {
	tracer trace.Tracer
	tp     *tracesdk.TracerProvider
	log    *log.Helper
}

// NewTracingService 创建追踪服务
func NewTracingService(cfg TracingConfig, logger log.Logger) (*TracingService, error) {
	// 创建OTLP gRPC导出器
	ctx := context.Background()
	
	// 创建gRPC连接
	conn, err := grpc.DialContext(ctx, cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	// 创建OTLP trace导出器
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	// 创建资源
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建TracerProvider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// 设置全局TracerProvider
	otel.SetTracerProvider(tp)

	return &TracingService{
		tracer: tp.Tracer(cfg.ServiceName),
		tp:     tp,
		log:    log.NewHelper(log.With(logger, "module", "tracing")),
	}, nil
}

// Start 启动追踪服务
func (s *TracingService) Start(ctx context.Context) error {
	s.log.Info("tracing service started")
	return nil
}

// Stop 停止追踪服务
func (s *TracingService) Stop(ctx context.Context) error {
	s.log.Info("stopping tracing service")
	if err := s.tp.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

// GetTracer 获取Tracer
func (s *TracingService) GetTracer() trace.Tracer {
	return s.tracer
}

// StartSpan 开始一个新的span
func (s *TracingService) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, name, opts...)
}

// AddEvent 添加事件到当前span
func (s *TracingService) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes 设置span属性
func (s *TracingService) SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError 记录错误到span
func (s *TracingService) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// TraceOperation 追踪一个操作
func (s *TracingService) TraceOperation(ctx context.Context, operationName string, fn func(ctx context.Context) error) error {
	ctx, span := s.StartSpan(ctx, operationName)
	defer span.End()

	if err := fn(ctx); err != nil {
		s.RecordError(ctx, err)
		return err
	}

	return nil
}

// TraceHTTPRequest 追踪HTTP请求
func (s *TracingService) TraceHTTPRequest(ctx context.Context, method, path string, fn func(ctx context.Context) error) error {
	ctx, span := s.StartSpan(ctx, "HTTP "+method+" "+path,
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.path", path),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		s.RecordError(ctx, err)
		span.SetAttributes(attribute.String("http.status", "error"))
		return err
	}

	span.SetAttributes(attribute.String("http.status", "success"))
	return nil
}

// TraceDBQuery 追踪数据库查询
func (s *TracingService) TraceDBQuery(ctx context.Context, query string, fn func(ctx context.Context) error) error {
	ctx, span := s.StartSpan(ctx, "DB Query",
		trace.WithAttributes(
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		s.RecordError(ctx, err)
		return err
	}

	return nil
}

// TraceRedisOperation 追踪Redis操作
func (s *TracingService) TraceRedisOperation(ctx context.Context, operation, key string, fn func(ctx context.Context) error) error {
	ctx, span := s.StartSpan(ctx, "Redis "+operation,
		trace.WithAttributes(
			attribute.String("redis.operation", operation),
			attribute.String("redis.key", key),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		s.RecordError(ctx, err)
		return err
	}

	return nil
}
