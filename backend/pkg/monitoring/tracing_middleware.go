package monitoring

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware 追踪中间件
func TracingMiddleware(tracer trace.Tracer) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从transport中获取操作信息
			var spanName string
			if info, ok := transport.FromServerContext(ctx); ok {
				spanName = info.Operation()
			} else {
				spanName = "unknown"
			}

			// 开始span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// 设置基本属性
			if info, ok := transport.FromServerContext(ctx); ok {
				span.SetAttributes(
					attribute.String("rpc.system", info.Kind().String()),
					attribute.String("rpc.service", info.Operation()),
				)

				// HTTP特定属性
				if info.Kind() == transport.KindHTTP {
					if header := info.RequestHeader(); header != nil {
						if method := header.Get("method"); method != "" {
							span.SetAttributes(attribute.String("http.method", method))
						}
						if path := header.Get("path"); path != "" {
							span.SetAttributes(attribute.String("http.path", path))
						}
					}
				}

				// gRPC特定属性
				if info.Kind() == transport.KindGRPC {
					span.SetAttributes(
						attribute.String("rpc.method", info.Operation()),
					)
				}
			}

			// 执行请求
			reply, err := handler(ctx, req)

			// 记录错误
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return reply, err
		}
	}
}

// TracingClientMiddleware 客户端追踪中间件
func TracingClientMiddleware(tracer trace.Tracer) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从transport中获取操作信息
			var spanName string
			if info, ok := transport.FromClientContext(ctx); ok {
				spanName = info.Operation()
			} else {
				spanName = "unknown"
			}

			// 开始span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindClient),
			)
			defer span.End()

			// 设置基本属性
			if info, ok := transport.FromClientContext(ctx); ok {
				span.SetAttributes(
					attribute.String("rpc.system", info.Kind().String()),
					attribute.String("rpc.service", info.Operation()),
				)
			}

			// 执行请求
			reply, err := handler(ctx, req)

			// 记录错误
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return reply, err
		}
	}
}

// TraceDB 追踪数据库操作的辅助函数
func TraceDB(ctx context.Context, tracer trace.Tracer, operation, query string, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("DB %s", operation),
		trace.WithAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.operation", operation),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TraceRedis 追踪Redis操作的辅助函数
func TraceRedis(ctx context.Context, tracer trace.Tracer, operation, key string, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("Redis %s", operation),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", operation),
			attribute.String("db.redis.key", key),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TraceKafka 追踪Kafka操作的辅助函数
func TraceKafka(ctx context.Context, tracer trace.Tracer, operation, topic string, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("Kafka %s", operation),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.operation", operation),
			attribute.String("messaging.destination", topic),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TraceHTTPClient 追踪HTTP客户端请求的辅助函数
func TraceHTTPClient(ctx context.Context, tracer trace.Tracer, method, url string, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("HTTP %s", method),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", url),
		),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
