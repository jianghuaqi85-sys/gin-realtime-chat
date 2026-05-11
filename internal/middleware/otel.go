package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func OtelMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer("gin-middleware")

	return func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), c.Request.URL.Path,
			trace.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", c.Request.URL.Path),
			))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		span.SetAttributes(attribute.Int("status_code", c.Writer.Status()))
	}
}
