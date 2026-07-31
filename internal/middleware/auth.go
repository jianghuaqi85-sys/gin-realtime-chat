package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/jwt"
)

// AuthMiddleware 验证 JWT Token 并提取用户信息注入 Context
func AuthMiddleware(cfg *config.Config, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header 或 query token (兼容 WebSocket 握手)
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			tokenParam := c.Query("token")
			if tokenParam != "" {
				authHeader = "Bearer " + tokenParam
			}
		}

		// 2. 校验 Bearer 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		tokenString := parts[1]

		// 3. 解析并验证 JWT (CPU 密集型无 I/O)
		claims, err := jwt.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// 4. 实时检查账号封禁状态及最新角色名称
		// ⚠️ 高性能架构提醒：超高并发环境中可在 Redis 缓存 user 状态或维护 TokenVersion 黑名单
		if userRepo != nil {
			ctx := c.Request.Context()

			user, err := userRepo.GetUserByID(claims.UserID)
			if err != nil {
				logger.Ctx(ctx).WithError(err).Error("Auth checks failed")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			if user == nil || user.Banned {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account banned or deleted"})
				return
			}

			// 更新时间差导致的 Payload 不一致
			claims.Username = user.Username
			claims.Role = user.Role
		}

		// 5. 将 Core 信息注入 Gin Context 供后续 handler 与 admin 中间件复用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
