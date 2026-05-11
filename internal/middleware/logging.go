package middleware

import (
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
		f["client_ip"] = c.ClientIP()
		f["user_agent"] = c.Request.UserAgent()

		logger.Get().WithFields(f).Info("request completed")

		for k := range f {
			delete(f, k)
		}
		fieldsPool.Put(fp)
	}
}
