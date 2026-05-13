package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/example/gin-high-performance/internal/repository"
)

func AdminMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)

		// token 中没有 role 时回退查库（兼容旧 token）
		if roleStr == "" && userRepo != nil {
			userID, exists := c.Get("user_id")
			if !exists {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			user, err := userRepo.GetUserByID(userID.(string))
			if err != nil || user == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}
			roleStr = user.Role
		}

		if roleStr != "admin" && roleStr != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}

		c.Next()
	}
}

func SuperAdminMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)

		// token 中没有 role 时回退查库（兼容旧 token）
		if roleStr == "" && userRepo != nil {
			userID, exists := c.Get("user_id")
			if !exists {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			user, err := userRepo.GetUserByID(userID.(string))
			if err != nil || user == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}
			roleStr = user.Role
		}

		if roleStr != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "super admin only"})
			return
		}

		c.Next()
	}
}
