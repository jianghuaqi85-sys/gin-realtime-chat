package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func OtelMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer("gin-server")
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// 1. 从 Header 提取上游链路 Trace Context
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 2. 初始 Span 命名
		spanName := c.Request.URL.Path

		// 3. 启动 Span 注入标准属性
		ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(
			semconv.HTTPMethodKey.String(c.Request.Method),
			semconv.HTTPTargetKey.String(c.Request.URL.RequestURI()),
			semconv.NetHostNameKey.String(c.Request.Host),
			semconv.HTTPClientIPKey.String(c.ClientIP()),
		))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// 4. 路由模板覆盖（消除高基数索引爆炸）
		if fullPath := c.FullPath(); fullPath != "" {
			span.SetName(fullPath)
			span.SetAttributes(semconv.HTTPRouteKey.String(fullPath))
		}

		// 5. 记录状态码与错误追溯
		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(status))

		if status >= 500 || len(c.Errors) > 0 {
			span.SetStatus(codes.Error, "HTTP request failed")
			for _, e := range c.Errors {
				span.RecordError(e.Err)
			}
		}
	}
}
