package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	numBuckets       = 256
	numChannelShards = 64
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	authTimeout      = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WSMessage — WebSocket JSON 消息协议
type WSMessage struct {
	Type      string `json:"type"`
	ChannelID string `json:"channel_id,omitempty"`
	Content   string `json:"content,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Token     string `json:"token,omitempty"`
	MessageID string `json:"id,omitempty"`
}

type Client struct {
	conn          *websocket.Conn
	send          chan []byte
	hub           *Hub
	bucket        *Bucket
	userID        string
	username      string
	readLimit     int64
	channels      map[string]bool
	mu            sync.Mutex
	authenticated bool
	stopCh        chan struct{}
	closeOnce     sync.Once
}

func (c *Client) GetUserID() string   { return c.userID }
func (c *Client) GetUsername() string { return c.username }

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.stopCh != nil {
			close(c.stopCh)
		}
		if c.conn != nil {
			c.conn.Close()
		}
	})
}

func (c *Client) SendMessage(msg []byte) {
	if c.stopCh != nil {
		select {
		case <-c.stopCh:
			return
		default:
		}
	}

	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) JoinChannel(channelID string) {
	c.mu.Lock()
	c.channels[channelID] = true
	c.mu.Unlock()
}

func (c *Client) LeaveChannel(channelID string) {
	c.mu.Lock()
	delete(c.channels, channelID)
	c.mu.Unlock()
}

func (c *Client) IsInChannel(channelID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channels[channelID]
}

type Bucket struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stopCh     chan struct{}
	mu         sync.RWMutex
}

func newBucket() *Bucket {
	return &Bucket{
		broadcast:  make(chan []byte, 2048),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		stopCh:     make(chan struct{}),
		clients:    make(map[*Client]bool),
	}
}

func (b *Bucket) Run() {
	for {
		select {
		case <-b.stopCh:
			return
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				client.Close()
			}
			b.mu.Unlock()
		case message := <-b.broadcast:
			var stale []*Client
			b.mu.RLock()
			for client := range b.clients {
				select {
				case <-client.stopCh:
					stale = append(stale, client)
				case client.send <- message:
				default:
					stale = append(stale, client)
				}
			}
			b.mu.RUnlock()
			if len(stale) > 0 {
				b.mu.Lock()
				for _, client := range stale {
					if _, ok := b.clients[client]; ok {
						delete(b.clients, client)
						client.Close()
					}
				}
				b.mu.Unlock()
			}
		}
	}
}

// channelShard — 频道分片锁，减少全局锁竞争
type channelShard struct {
	mu       sync.RWMutex
	channels map[string]map[*Client]bool
}

// MessageBus 跨进程消息总线接口
type MessageBus interface {
	Publish(ctx context.Context, channelID string, data []byte) error
	Subscribe(channelID string) error
	Unsubscribe(channelID string) error
	Close() error
	SetOnMessage(handler func(channelID string, data []byte))
}

type Hub struct {
	buckets   []*Bucket
	shards    []*channelShard
	OnMessage func(client *Client, data []byte)
	bus       MessageBus
}

func NewHub() *Hub {
	h := &Hub{
		buckets: make([]*Bucket, numBuckets),
		shards:  make([]*channelShard, numChannelShards),
	}
	for i := 0; i < numBuckets; i++ {
		h.buckets[i] = newBucket()
		go h.buckets[i].Run()
	}
	for i := 0; i < numChannelShards; i++ {
		h.shards[i] = &channelShard{
			channels: make(map[string]map[*Client]bool),
		}
	}
	return h
}

// SetBus 设置跨进程消息总线（可选，为 nil 则单进程模式）
func (h *Hub) SetBus(bus MessageBus) {
	h.bus = bus
	bus.SetOnMessage(func(channelID string, data []byte) {
		h.BroadcastToChannel(channelID, data)
	})
}

func (h *Hub) getChannelShard(channelID string) *channelShard {
	hasher := fnv.New32a()
	hasher.Write([]byte(channelID))
	return h.shards[hasher.Sum32()%numChannelShards]
}

func (h *Hub) Register(client *Client) {
	bucket := h.getBucket(client.userID)
	client.bucket = bucket
	bucket.register <- client
}

// Unregister 优化锁获取顺序，消除嵌套持锁倒置风险
func (h *Hub) Unregister(client *Client) {
	client.mu.Lock()
	channelsToLeave := make([]string, 0, len(client.channels))
	for chID := range client.channels {
		channelsToLeave = append(channelsToLeave, chID)
	}
	client.channels = make(map[string]bool)
	client.mu.Unlock()

	for _, chID := range channelsToLeave {
		shard := h.getChannelShard(chID)
		shard.mu.Lock()
		if clients, ok := shard.channels[chID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(shard.channels, chID)
			}
		}
		shard.mu.Unlock()

		if h.bus != nil {
			_ = h.bus.Unsubscribe(chID)
		}
	}

	if client.bucket != nil {
		client.bucket.unregister <- client
	}

	// 广播用户下线事件
	offlineMsg, _ := json.Marshal(WSMessage{
		Type:    "user_offline",
		UserID:  client.userID,
		Content: client.username,
	})
	h.Broadcast(offlineMsg)
}

func (h *Hub) JoinChannel(client *Client, channelID string) {
	shard := h.getChannelShard(channelID)
	shard.mu.Lock()
	if shard.channels[channelID] == nil {
		shard.channels[channelID] = make(map[*Client]bool)
	}
	shard.channels[channelID][client] = true
	shard.mu.Unlock()
	client.JoinChannel(channelID)

	if h.bus != nil {
		_ = h.bus.Subscribe(channelID)
	}
}

func (h *Hub) LeaveChannel(client *Client, channelID string) {
	shard := h.getChannelShard(channelID)
	shard.mu.Lock()
	if clients, ok := shard.channels[channelID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(shard.channels, channelID)
		}
	}
	shard.mu.Unlock()
	client.LeaveChannel(channelID)

	if h.bus != nil {
		_ = h.bus.Unsubscribe(channelID)
	}
}

// BroadcastToChannel — 按桶分组并行投递
func (h *Hub) BroadcastToChannel(channelID string, msg []byte) {
	shard := h.getChannelShard(channelID)
	shard.mu.RLock()
	clients := shard.channels[channelID]
	if len(clients) == 0 {
		shard.mu.RUnlock()
		return
	}
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	shard.mu.RUnlock()

	bucketGroups := make(map[*Bucket][]*Client)
	for _, c := range targets {
		if c.bucket != nil {
			bucketGroups[c.bucket] = append(bucketGroups[c.bucket], c)
		}
	}
	for bucket, group := range bucketGroups {
		bucket.mu.RLock()
		for _, c := range group {
			c.SendMessage(msg)
		}
		bucket.mu.RUnlock()
	}
}

func (h *Hub) Broadcast(msg []byte) error {
	for _, bucket := range h.buckets {
		select {
		case bucket.broadcast <- msg:
		default:
		}
	}
	return nil
}

func (h *Hub) Close() {
	if h.bus != nil {
		_ = h.bus.Close()
	}
	for _, bucket := range h.buckets {
		close(bucket.stopCh)
		bucket.mu.Lock()
		for client := range bucket.clients {
			client.Close()
			delete(bucket.clients, client)
		}
		bucket.mu.Unlock()
	}
}

func (h *Hub) OnlineCount() int {
	count := 0
	for _, bucket := range h.buckets {
		bucket.mu.RLock()
		count += len(bucket.clients)
		bucket.mu.RUnlock()
	}
	return count
}

type OnlineUser struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func (h *Hub) OnlineUsers() []OnlineUser {
	seen := make(map[string]bool)
	var users []OnlineUser
	for _, bucket := range h.buckets {
		bucket.mu.RLock()
		for client := range bucket.clients {
			if !seen[client.userID] {
				seen[client.userID] = true
				users = append(users, OnlineUser{UserID: client.userID, Username: client.username})
			}
		}
		bucket.mu.RUnlock()
	}
	return users
}

func (h *Hub) DisconnectUser(userID string) int {
	count := 0
	bucket := h.getBucket(userID)
	var targets []*Client
	bucket.mu.Lock()
	for client := range bucket.clients {
		if client.userID == userID {
			targets = append(targets, client)
			delete(bucket.clients, client)
			count++
		}
	}
	bucket.mu.Unlock()

	for _, client := range targets {
		client.Close()
	}
	return count
}

func (h *Hub) UpdateUsername(userID, newUsername string) {
	bucket := h.getBucket(userID)
	bucket.mu.Lock()
	for client := range bucket.clients {
		if client.userID == userID {
			client.username = newUsername
		}
	}
	bucket.mu.Unlock()
}

func (h *Hub) BroadcastSystemAll(content string) {
	msg, _ := json.Marshal(WSMessage{
		Type:      "system",
		Content:   content,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
	for _, bucket := range h.buckets {
		bucket.mu.RLock()
		for client := range bucket.clients {
			if client.stopCh != nil {
				select {
				case <-client.stopCh:
					continue
				default:
				}
			}
			select {
			case client.send <- msg:
			default:
			}
		}
		bucket.mu.RUnlock()
	}
}

func (h *Hub) getBucket(userID string) *Bucket {
	hasher := fnv.New32a()
	hasher.Write([]byte(userID))
	return h.buckets[hasher.Sum32()%numBuckets]
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadLimit(c.readLimit)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	errCount := 0
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			errCount++
			if errCount >= 5 {
				log.Printf("[WARN] 客户端 %s 连续发送非法 JSON，强制断开", c.userID)
				break
			}
			continue
		}
		errCount = 0

		switch msg.Type {
		case "join":
			if msg.ChannelID != "" {
				hub := c.hub
				hub.JoinChannel(c, msg.ChannelID)
				sysMsg, _ := json.Marshal(WSMessage{
					Type:      "system",
					ChannelID: msg.ChannelID,
					Content:   c.username + " 加入了频道",
					CreatedAt: time.Now().Format(time.RFC3339),
				})
				hub.BroadcastToChannel(msg.ChannelID, sysMsg)
			}

		case "leave":
			if msg.ChannelID != "" {
				c.hub.LeaveChannel(c, msg.ChannelID)
				sysMsg, _ := json.Marshal(WSMessage{
					Type:      "system",
					ChannelID: msg.ChannelID,
					Content:   c.username + " 离开了频道",
					CreatedAt: time.Now().Format(time.RFC3339),
				})
				c.hub.BroadcastToChannel(msg.ChannelID, sysMsg)
			}

		case "message":
			if msg.ChannelID != "" && msg.Content != "" {
				if c.hub.OnMessage != nil {
					c.hub.OnMessage(c, raw)
				}
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type TokenValidator func(token string) (userID, username string, err error)

func ServeWS(hub *Hub, origin string, readLimit int, validateToken TokenValidator, w http.ResponseWriter, r *http.Request) error {
	if !checkOrigin(origin, r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	if readLimit <= 0 {
		readLimit = 4096
	}

	client := &Client{
		conn:          conn,
		send:          make(chan []byte, 256),
		hub:           hub,
		readLimit:     int64(readLimit),
		channels:      make(map[string]bool),
		authenticated: false,
		stopCh:        make(chan struct{}),
	}

	if err := client.authenticate(validateToken); err != nil {
		errMsg, _ := json.Marshal(WSMessage{Type: "error", Content: err.Error()})
		conn.WriteMessage(websocket.TextMessage, errMsg)
		conn.Close()
		return nil
	}

	hub.Register(client)

	authOK, _ := json.Marshal(WSMessage{Type: "auth_ok", Content: client.username})
	client.SendMessage(authOK)

	// 广播用户上线事件
	onlineMsg, _ := json.Marshal(WSMessage{
		Type:    "user_online",
		UserID:  client.userID,
		Content: client.username,
	})
	hub.Broadcast(onlineMsg)

	go client.WritePump()
	client.ReadPump()

	return nil
}

func (c *Client) authenticate(validateToken TokenValidator) error {
	c.conn.SetReadDeadline(time.Now().Add(authTimeout))
	c.conn.SetReadLimit(c.readLimit)

	_, raw, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("未收到认证消息")
	}

	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil || msg.Type != "auth" {
		return fmt.Errorf("第一条消息必须是 auth 类型")
	}

	if msg.Token == "" {
		return fmt.Errorf("token 不能为空")
	}

	userID, username, err := validateToken(msg.Token)
	if err != nil {
		return fmt.Errorf("认证失败: %v", err)
	}

	c.userID = userID
	c.username = username
	c.authenticated = true
	return nil
}

func checkOrigin(allowedOrigin string, r *http.Request) bool {
	origin := r.Header.Get("Origin")

	if os.Getenv("APP_ENV") != "production" {
		return true
	}

	if origin == "" {
		return false
	}

	if allowedOrigin != "" {
		return origin == allowedOrigin
	}

	host := r.Host
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	originHost := origin
	originHost = strings.TrimPrefix(originHost, "http://")
	originHost = strings.TrimPrefix(originHost, "https://")
	if idx := strings.Index(originHost, ":"); idx > 0 {
		originHost = originHost[:idx]
	}
	if idx := strings.Index(originHost, "/"); idx > 0 {
		originHost = originHost[:idx]
	}

	return originHost == host
}
