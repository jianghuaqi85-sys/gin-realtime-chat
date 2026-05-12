package middleware

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/example/gin-high-performance/internal/logger"
)

var fieldsPool = sync.Pool{
	New: func() any {
		f := make(map[string]any, 6)
		return &f
	},
}

// maskIP 对 IP 地址进行脱敏处理
// 开发环境显示完整 IP，生产环境隐藏最后一位
func maskIP(ip string) string {
	// 开发环境不做脱敏
	if os.Getenv("APP_ENV") != "production" {
		return ip
	}

	// 处理 IPv4
	if idx := strings.LastIndex(ip, "."); idx > 0 {
		return ip[:idx] + ".***"
	}

	// 处理 IPv6（简化处理）
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) > 2 {
			return strings.Join(parts[:2], ":") + ":****"
		}
	}

	return ip
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		fp := fieldsPool.Get().(*map[string]any)
		f := *fp
		f["method"] = c.Request.Method
		f["path"] = c.Request.URL.Path
		f["status"] = c.Writer.Status()
		f["duration"] = duration.String()
		f["client_ip"] = maskIP(c.ClientIP())
		f["user_agent"] = c.Request.UserAgent()

		logger.Get().WithFields(f).Info("request completed")

		for k := range f {
			delete(f, k)
		}
		fieldsPool.Put(fp)
	}
}
