package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/jwt"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "12345678901234567890123456789012",
		JWTExpireHours: 24,
	}

	userRepo := repository.NewInMemoryUserRepository()
	userRepo.CreateUser("user_auth", "hash")
	u, _ := userRepo.GetUserByUsername("user_auth")

	token, _ := jwt.GenerateToken(cfg.JWTSecret, u.ID, u.Username, u.Role, 24)

	r := gin.New()
	r.Use(AuthMiddleware(cfg, userRepo))
	r.GET("/protected", func(c *gin.Context) {
		userID := c.GetString("user_id")
		c.String(http.StatusOK, userID)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != u.ID {
		t.Errorf("expected user_id = %s, got %s", u.ID, w.Body.String())
	}
}

func TestAuthMiddleware_BannedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "12345678901234567890123456789012",
		JWTExpireHours: 24,
	}

	userRepo := repository.NewInMemoryUserRepository()
	userRepo.CreateUser("banned_user", "hash")
	u, _ := userRepo.GetUserByUsername("banned_user")
	userRepo.SetBanned(u.ID, true)

	token, _ := jwt.GenerateToken(cfg.JWTSecret, u.ID, u.Username, u.Role, 24)

	r := gin.New()
	r.Use(AuthMiddleware(cfg, userRepo))
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for banned user, got %d", w.Code)
	}
}
