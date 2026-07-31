package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

func TestMain(m *testing.M) {
	logger.Init("info")
	os.Exit(m.Run())
}

func TestSetupRouter_PublicHealth(t *testing.T) {
	cfg := &config.Config{
		AppPort: "8080",
	}
	userRepo := repository.NewInMemoryUserRepository()
	hub := ws.NewHub()

	router := setupRouter(cfg, userRepo, nil, nil, hub, nil, nil)
	if router == nil {
		t.Fatal("setupRouter returned nil")
	}

	req, _ := http.NewRequest("GET", "/api/public/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/public/health, got %d", w.Code)
	}
}

func TestSetupRouter_ChatHTMLResponse(t *testing.T) {
	cfg := &config.Config{
		AppPort: "8080",
	}
	userRepo := repository.NewInMemoryUserRepository()
	channelRepo := repository.NewMySQLChannelRepository(nil)
	messageRepo := repository.NewMySQLMessageRepository(nil)
	hub := ws.NewHub()

	router := setupRouter(cfg, userRepo, channelRepo, messageRepo, hub, nil, nil)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /, got %d", w.Code)
	}
	if w.Body.String() != chatHTML {
		t.Error("GET / response body does not match chatHTML constant")
	}
}
