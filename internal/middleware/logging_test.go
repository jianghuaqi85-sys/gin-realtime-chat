package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
)

func TestLoggingMiddleware_Execution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Init("info")

	cfg := &config.Config{AppEnv: "development"}
	r := gin.New()
	r.Use(LoggingMiddleware(cfg))

	r.GET("/test-log", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	r.GET("/api/public/health", func(c *gin.Context) {
		c.String(http.StatusOK, "healthy")
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test-log", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/public/health", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 for health endpoint, got %d", w2.Code)
	}
}

func TestMaskIP(t *testing.T) {
	ip4 := "192.168.1.100"
	maskedProd := maskIP(ip4, true)
	if maskedProd != "192.168.1.***" {
		t.Errorf("expected 192.168.1.*** in prod, got %s", maskedProd)
	}

	maskedDev := maskIP(ip4, false)
	if maskedDev != ip4 {
		t.Errorf("expected %s in dev, got %s", ip4, maskedDev)
	}
}
