package redisbus

import (
	"context"
	"log"
	"sync"

	"github.com/go-redis/redis/v8"
)

// MessageBus — 基于 Redis Pub/Sub 的跨进程消息总线
type MessageBus struct {
	rdb         *redis.Client
	prefix      string
	mu          sync.RWMutex
	subscribers map[string]*redis.PubSub
	onMessage   func(channelID string, data []byte)
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewMessageBus(rdb *redis.Client) *MessageBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &MessageBus{
		rdb:         rdb,
		prefix:      "chat:ch:",
		subscribers: make(map[string]*redis.PubSub),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetOnMessage 设置收到消息时的回调
func (b *MessageBus) SetOnMessage(handler func(channelID string, data []byte)) {
	b.onMessage = handler
}

// Publish 向指定频道发布消息
func (b *MessageBus) Publish(channelID string, data []byte) error {
	return b.rdb.Publish(b.ctx, b.prefix+channelID, data).Err()
}

// Subscribe 订阅指定频道，收到消息后调用 onMessage 回调
func (b *MessageBus) Subscribe(channelID string) {
	b.mu.Lock()
	if _, ok := b.subscribers[channelID]; ok {
		b.mu.Unlock()
		return
	}

	pubsub := b.rdb.Subscribe(b.ctx, b.prefix+channelID)
	b.subscribers[channelID] = pubsub
	b.mu.Unlock()

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			if b.onMessage != nil {
				b.onMessage(channelID, []byte(msg.Payload))
			}
		}
	}()
}

// Unsubscribe 取消订阅指定频道
func (b *MessageBus) Unsubscribe(channelID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if pubsub, ok := b.subscribers[channelID]; ok {
		pubsub.Close()
		delete(b.subscribers, channelID)
	}
}

// Close 关闭所有订阅
func (b *MessageBus) Close() {
	b.cancel()
	b.mu.Lock()
	for id, pubsub := range b.subscribers {
		if err := pubsub.Close(); err != nil {
			log.Printf("[WARN] 关闭 Redis 订阅 %s 失败: %v", id, err)
		}
	}
	b.subscribers = make(map[string]*redis.PubSub)
	b.mu.Unlock()
}
