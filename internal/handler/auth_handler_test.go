package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

func TestAuthHandler_LoginTimingAttackProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "12345678901234567890123456789012",
		JWTExpireHours: 24,
	}
	userRepo := repository.NewInMemoryUserRepository()
	hub := ws.NewHub()
	h := NewAuthHandler(cfg, userRepo, hub)

	body, _ := json.Marshal(LoginRequest{
		Username: "non_existent_user",
		Password: "password123",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/api/public/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	// Ensure dummyBcryptHash is non-empty
	if len(dummyBcryptHash) == 0 {
		t.Error("expected dummyBcryptHash to be initialized")
	}
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "12345678901234567890123456789012",
		JWTExpireHours: 24,
	}
	userRepo := repository.NewInMemoryUserRepository()
	hash, _ := bcrypt.GenerateFromPassword([]byte("old_password"), bcrypt.DefaultCost)
	userRepo.CreateUser("testuser", string(hash))

	hub := ws.NewHub()
	h := NewAuthHandler(cfg, userRepo, hub)

	body, _ := json.Marshal(ChangePasswordRequest{
		OldPassword: "old_password",
		NewPassword: "new_password_123",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("username", "testuser")
	req, _ := http.NewRequest("PUT", "/api/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChangePassword(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify new password works
	u, _ := userRepo.GetUserByUsername("testuser")
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("new_password_123"))
	if err != nil {
		t.Errorf("new password verification failed: %v", err)
	}
}
