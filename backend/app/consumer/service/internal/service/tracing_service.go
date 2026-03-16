package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingService 链路追踪服务
type TracingService struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	log      *log.Helper
}

// NewTracingService 创建链路追踪服务实例
func NewTracingService(ctx *bootstrap.Context) (*TracingService, error) {
	logger := ctx.NewLoggerHelper("consumer/service/tracing-service")

	// 创建 Jaeger exporter
	// TODO: 从配置文件读取 Jaeger 地址
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint("http://localhost:14268/api/traces"),
	))
	if err != nil {
		logger.Errorf("Failed to create Jaeger exporter: %v", err)
		// 不阻塞服务启动，只记录错误
		return &TracingService{
			log: logger,
		}, nil
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("consumer-service"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", "development"),
		)),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 创建 Tracer
	tracer := tp.Tracer("consumer-service")

	logger.Info("Tracing service initialized with Jaeger exporter")

	return &TracingService{
		tracer:   tracer,
		provider: tp,
		log:      logger,
	}, nil
}

// StartSpan 开始一个新的 Span
func (s *TracingService) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if s.tracer == nil {
		// 如果 tracer 未初始化，返回 noop span
		return ctx, trace.SpanFromContext(ctx)
	}
	return s.tracer.Start(ctx, spanName, opts...)
}

// AddEvent 添加事件到当前 Span
func (s *TracingService) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttributes 设置 Span 属性
func (s *TracingService) SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// RecordError 记录错误到 Span
func (s *TracingService) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && err != nil {
		span.RecordError(err)
	}
}

// Shutdown 关闭追踪服务
func (s *TracingService) Shutdown(ctx context.Context) error {
	if s.provider != nil {
		return s.provider.Shutdown(ctx)
	}
	return nil
}

// TraceConsumerOperation 追踪用户操作
func (s *TracingService) TraceConsumerOperation(ctx context.Context, operation string, consumerID uint32) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "consumer."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.Int64("consumer_id", int64(consumerID)),
	)
	return ctx, span
}

// TracePaymentOperation 追踪支付操作
func (s *TracingService) TracePaymentOperation(ctx context.Context, operation string, orderNo string) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "payment."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.String("order_no", orderNo),
	)
	return ctx, span
}

// TraceFinanceOperation 追踪财务操作
func (s *TracingService) TraceFinanceOperation(ctx context.Context, operation string, consumerID uint32, amount string) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "finance."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.Int64("consumer_id", int64(consumerID)),
		attribute.String("amount", amount),
	)
	return ctx, span
}

// TraceSMSOperation 追踪短信操作
func (s *TracingService) TraceSMSOperation(ctx context.Context, operation string, phone string) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "sms."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.String("phone", maskPhone(phone)),
	)
	return ctx, span
}

// TraceMediaOperation 追踪媒体操作
func (s *TracingService) TraceMediaOperation(ctx context.Context, operation string, fileType string) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "media."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.String("file_type", fileType),
	)
	return ctx, span
}

// TraceLogisticsOperation 追踪物流操作
func (s *TracingService) TraceLogisticsOperation(ctx context.Context, operation string, trackingNo string) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "logistics."+operation)
	s.SetAttributes(ctx,
		attribute.String("operation", operation),
		attribute.String("tracking_no", trackingNo),
	)
	return ctx, span
}

// maskPhone 脱敏手机号
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
