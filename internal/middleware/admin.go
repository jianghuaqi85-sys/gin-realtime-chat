package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/internal/repository"
)

// getRole 提取 DRY 逻辑：优先从 Context 获取，如果没有则查库兜底
func getRole(c *gin.Context, userRepo repository.UserRepository) (string, bool) {
	role := c.GetString("role")
	if role != "" {
		return role, true
	}

	if userRepo == nil {
		return "", false
	}

	userID := c.GetString("user_id")
	if userID == "" {
		return "", false
	}

	ctx := c.Request.Context()
	user, err := userRepo.GetUserByID(userID)
	if err != nil {
		logger.Ctx(ctx).WithError(err).WithField("user_id", userID).Error("Failed to fetch user for role fallback")
		return "", false
	}
	if user == nil {
		return "", false
	}

	// 优化：将查到的 Role 写入 Context，供后续 Handler 直接复用，避免重复查库
	c.Set("role", user.Role)
	return user.Role, true
}

// AdminMiddleware 验证管理员权限 (包含 admin 和 super_admin)
func AdminMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := getRole(c, userRepo)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized or internal error"})
			return
		}

		if role != "admin" && role != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}

		c.Next()
	}
}

// SuperAdminMiddleware 验证超级管理员权限
func SuperAdminMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := getRole(c, userRepo)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized or internal error"})
			return
		}

		if role != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "super admin only"})
			return
		}

		c.Next()
	}
}
