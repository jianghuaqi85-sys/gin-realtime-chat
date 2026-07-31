package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/repository"
)

func TestAdminMiddleware_RoleInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminMiddleware(nil))
	r.GET("/admin-only", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Case 1: admin role -> 200
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/admin-only", nil)
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = req1
	c1.Set("role", "admin")
	r.ServeHTTP(w1, req1)

	// Case 2: normal user role -> 403
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/admin-only", nil)
	r.ServeHTTP(w2, req2)
}

func TestSuperAdminMiddleware_RoleFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewInMemoryUserRepository()
	userRepo.CreateUser("super_user", "hash")
	u, _ := userRepo.GetUserByUsername("super_user")
	userRepo.SetRole(u.ID, "super_admin")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", u.ID)
		c.Next()
	})
	r.Use(SuperAdminMiddleware(userRepo))
	r.GET("/super-only", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/super-only", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for super_admin fallback, got %d", w.Code)
	}
}
