package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubDisconnectUserConcurrencyPanic(t *testing.T) {
	hub := NewHub()

	// 创建测试 HTTP 服务
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := ServeWS(hub, "", 4096, func(token string) (string, string, error) {
			return "user_1", "testuser", nil
		}, w, r)
		if err != nil {
			t.Logf("ServeWS error: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 建立 10 个客户端并发连接
	var clients []*websocket.Conn
	for i := 0; i < 10; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Dial failed: %v", err)
		}
		defer conn.Close()

		// 第一条消息必须是 auth 消息
		authMsg, _ := json.Marshal(WSMessage{
			Type:  "auth",
			Token: "valid_token",
		})
		if err := conn.WriteMessage(websocket.TextMessage, authMsg); err != nil {
			t.Fatalf("Write auth failed: %v", err)
		}
		clients = append(clients, conn)
	}

	// 等待连接建立与注册完成
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	// 协程 1: 高频广播消息
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			hub.Broadcast([]byte("test_broadcast_msg"))
			hub.BroadcastToChannel("test_channel", []byte("channel_msg"))
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// 协程 2: 并发调用 DisconnectUser
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			hub.DisconnectUser("user_1")
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()
}

func TestDirectCloseSendPanic(t *testing.T) {
	hub := NewHub()
	client := &Client{
		send:     make(chan []byte, 10),
		userID:   "user_test",
		username: "test",
		channels: make(map[string]bool),
		hub:      hub,
		stopCh:   make(chan struct{}),
	}

	bucket := hub.getBucket("user_test")
	bucket.mu.Lock()
	bucket.clients[client] = true
	bucket.mu.Unlock()

	var wg sync.WaitGroup
	// 线程 1: 模拟 DisconnectUser (原代码执行 close(client.send))
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Millisecond)
		hub.DisconnectUser("user_test")
	}()

	// 线程 2: 模拟并发向 client 发送消息
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			client.SendMessage([]byte("hello"))
			hub.BroadcastSystemAll("system notice")
			time.Sleep(100 * time.Microsecond)
		}
	}()

	wg.Wait()
}
