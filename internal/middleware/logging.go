package middleware

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
)

var fieldsPool = sync.Pool{
	New: func() any {
		f := make(logrus.Fields, 8)
		return &f
	},
}

// maskIP 对 IP 地址进行脱敏处理
func maskIP(ip string, isProd bool) string {
	if !isProd {
		return ip
	}

	// 处理 IPv4
	if idx := strings.LastIndex(ip, "."); idx > 0 {
		return ip[:idx] + ".***"
	}

	// 处理 IPv6
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) > 2 {
			return strings.Join(parts[:2], ":") + ":****"
		}
	}

	return ip
}

// LoggingMiddleware 开启应用日志，自动继承上下文 TraceID 并且剔除高频健康检查杂音
func LoggingMiddleware(cfgs ...*config.Config) gin.HandlerFunc {
	isProd := false
	if len(cfgs) > 0 && cfgs[0] != nil {
		isProd = cfgs[0].AppEnv == "production"
	} else {
		isProd = os.Getenv("APP_ENV") == "production"
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		// 忽略高频健康检查噪音日志
		if path == "/api/public/health" {
			return
		}

		duration := time.Since(start)

		fp := fieldsPool.Get().(*logrus.Fields)
		f := *fp

		f["method"] = c.Request.Method
		if rawQuery != "" {
			f["path"] = path + "?" + rawQuery
		} else {
			f["path"] = path
		}
		f["status"] = c.Writer.Status()
		// 保存为浮点数毫秒，便利 Kibana / Grafana 图表查询
		f["duration_ms"] = float64(duration.Microseconds()) / 1000.0
		f["client_ip"] = maskIP(c.ClientIP(), isProd)
		f["user_agent"] = c.Request.UserAgent()

		if len(c.Errors) > 0 {
			f["errors"] = c.Errors.ByType(gin.ErrorTypePrivate).String()
		}

		// 使用 Ctx 自动带入 TraceID
		logEntry := logger.Ctx(c.Request.Context()).WithFields(f)

		status := c.Writer.Status()
		if status >= 500 {
			logEntry.Error("Server error during request")
		} else if status >= 400 {
			logEntry.Warn("Client error during request")
		} else {
			logEntry.Info("Request completed")
		}

		for k := range f {
			delete(f, k)
		}
		fieldsPool.Put(fp)
	}
}
