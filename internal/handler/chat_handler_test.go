package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/pkg/ws"
)

func TestChatHandler_CreateChannelNameTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(nil, nil, ws.NewHub(), nil)

	longName := strings.Repeat("a", 35) // > MaxChannelNameLen (32)
	body, _ := json.Marshal(CreateChannelRequest{
		Name: longName,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user1")
	req, _ := http.NewRequest("POST", "/api/admin/channels", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.CreateChannel(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for long channel name, got %d", w.Code)
	}
}

func TestChatHandler_EditMessageTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(nil, nil, ws.NewHub(), nil)

	longContent := strings.Repeat("a", 4097) // > MaxMessageLength (4096)
	body, _ := json.Marshal(struct {
		Content string `json:"content" binding:"required"`
	}{
		Content: longContent,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user1")
	c.Params = gin.Params{{Key: "id", Value: "msg1"}}
	req, _ := http.NewRequest("PUT", "/api/messages/msg1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.EditMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for long message content, got %d", w.Code)
	}
}
