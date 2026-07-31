package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/logger"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func RateLimitMiddleware(limiter RateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	// 友好的兜底：如果未注入限流器，直接放行（适用于本地开发或功能降级）
	if limiter == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// 1. 动态粒度的 Key 策略：优先使用用户隔离，其次退化为 IP 隔离
		var limitKey string
		if userID := c.GetString("user_id"); userID != "" {
			limitKey = "ratelimit:user:" + userID
		} else {
			limitKey = "ratelimit:ip:" + c.ClientIP()
		}

		// 2. 严格的微操作超时控制：防止 Redis 慢查询拖垮 API
		ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()

		allowed, err := limiter.Allow(ctx, limitKey, limit, window)
		if err != nil {
			// 3. 高可用 Fail-Open 原则：限流器故障/超时时打日志告警，但【放行请求】
			logger.Ctx(c.Request.Context()).WithError(err).WithField("limit_key", limitKey).Warn("Rate limiter error, bypassing request")
			c.Next()
			return
		}

		if !allowed {
			// 4. 标准 HTTP 429 响应头
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			return
		}

		c.Next()
	}
}
