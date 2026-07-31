package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockLimiter struct {
	allowed bool
	err     error
	lastKey string
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.lastKey = key
	return m.allowed, m.err
}

func TestRateLimitMiddleware_FailOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lim := &mockLimiter{err: errors.New("redis down")}

	r := gin.New()
	r.Use(RateLimitMiddleware(lim, 10, time.Minute))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected Fail-Open 200 OK on limiter error, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_RateLimitedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lim := &mockLimiter{allowed: false}

	r := gin.New()
	r.Use(RateLimitMiddleware(lim, 10, time.Minute))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 StatusTooManyRequests, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After header = '1', got '%s'", w.Header().Get("Retry-After"))
	}
}

func TestRateLimitMiddleware_UserVsIPKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lim := &mockLimiter{allowed: true}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user_123")
		c.Next()
	})
	r.Use(RateLimitMiddleware(lim, 10, time.Minute))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if lim.lastKey != "ratelimit:user:user_123" {
		t.Errorf("expected key 'ratelimit:user:user_123', got '%s'", lim.lastKey)
	}
}
