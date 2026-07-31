package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/pkg/ws"
)

func TestSetupWSRouter_Health(t *testing.T) {
	cfg := &config.Config{
		WSPort: "8081",
	}
	hub := ws.NewHub()

	router := setupWSRouter(cfg, hub)
	if router == nil {
		t.Fatal("setupWSRouter returned nil")
	}

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", w.Code)
	}
}
