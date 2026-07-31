package redisbus

import (
	"context"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

// MessageBus — 基于 Redis Pub/Sub 的跨进程消息总线（单连接多路复用 + 本地引用计数）
type MessageBus struct {
	rdb    *redis.Client
	prefix string

	// 共享单个 redis.PubSub 实例，解决多频道连接爆满问题
	pubsub *redis.PubSub

	mu          sync.Mutex
	channelRefs map[string]int // 记录频道本地订阅计数

	onMessage func(channelID string, data []byte)

	ctx    context.Context
	cancel context.CancelFunc
}

// NewMessageBus 创建基于多路复用的 Redis 消息总线
func NewMessageBus(rdb *redis.Client) *MessageBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &MessageBus{
		rdb:         rdb,
		prefix:      "chat:ch:",
		pubsub:      rdb.Subscribe(ctx),
		channelRefs: make(map[string]int),
		ctx:         ctx,
		cancel:      cancel,
	}

	go bus.listen()

	return bus
}

// SetOnMessage 设置收到消息时的回调句柄
func (b *MessageBus) SetOnMessage(handler func(channelID string, data []byte)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onMessage = handler
}

// Publish 向指定频道发布消息，透传请求 Context
func (b *MessageBus) Publish(ctx context.Context, channelID string, data []byte) error {
	if ctx == nil {
		ctx = b.ctx
	}
	return b.rdb.Publish(ctx, b.prefix+channelID, data).Err()
}

// Subscribe 订阅频道，首个订阅者触发真正的 Redis Subscribe
func (b *MessageBus) Subscribe(channelID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.channelRefs[channelID]++
	if b.channelRefs[channelID] == 1 {
		if err := b.pubsub.Subscribe(b.ctx, b.prefix+channelID); err != nil {
			b.channelRefs[channelID]--
			if b.channelRefs[channelID] <= 0 {
				delete(b.channelRefs, channelID)
			}
			return err
		}
	}
	return nil
}

// Unsubscribe 取消订阅频道，无订阅者时触发真正的 Redis Unsubscribe
func (b *MessageBus) Unsubscribe(channelID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if refs, ok := b.channelRefs[channelID]; ok {
		b.channelRefs[channelID] = refs - 1
		if b.channelRefs[channelID] <= 0 {
			delete(b.channelRefs, channelID)
			return b.pubsub.Unsubscribe(b.ctx, b.prefix+channelID)
		}
	}
	return nil
}

// listen 监听 PubSub 连接中的所有频道消息
func (b *MessageBus) listen() {
	ch := b.pubsub.Channel()
	for msg := range ch {
		b.mu.Lock()
		handler := b.onMessage
		b.mu.Unlock()

		if handler != nil {
			channelID := strings.TrimPrefix(msg.Channel, b.prefix)
			// 异步回调分发，防止主订阅循环被业务逻辑卡死
			go handler(channelID, []byte(msg.Payload))
		}
	}
}

// Close 优雅关闭消息总线
func (b *MessageBus) Close() error {
	b.cancel()

	b.mu.Lock()
	b.channelRefs = make(map[string]int)
	b.mu.Unlock()

	return b.pubsub.Close()
}
