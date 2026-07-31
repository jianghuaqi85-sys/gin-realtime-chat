package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

func TestAdminHandler_Stats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewInMemoryUserRepository()
	userRepo.CreateUser("user1", "hash1")

	hub := ws.NewHub()
	h := NewAdminHandler(userRepo, nil, nil, hub)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/api/admin/stats", nil)
	c.Request = req

	h.Stats(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if total, ok := res["users_total"].(float64); !ok || total < 1 {
		t.Errorf("expected users_total >= 1, got %v", res["users_total"])
	}
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewInMemoryUserRepository()
	userRepo.CreateUser("user_to_del", "hash")
	u, _ := userRepo.GetUserByUsername("user_to_del")

	hub := ws.NewHub()
	h := NewAdminHandler(userRepo, nil, nil, hub)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: u.ID}}
	req, _ := http.NewRequest("DELETE", "/api/admin/users/"+u.ID, nil)
	c.Request = req

	h.DeleteUser(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify user is removed from repo
	delUser, _ := userRepo.GetUserByID(u.ID)
	if delUser != nil {
		t.Errorf("expected user to be deleted from repo")
	}
}
