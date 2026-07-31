package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOtelMiddleware_Execution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := gin.New()
	r.Use(OtelMiddleware())
	r.GET("/items/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "item "+c.Param("id"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/items/123", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
