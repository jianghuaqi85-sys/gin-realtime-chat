package redisbus

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestMessageBusInitAndClose(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	bus := NewMessageBus(rdb)
	if bus == nil {
		t.Fatal("expected NewMessageBus to return non-nil instance")
	}

	bus.SetOnMessage(func(channelID string, data []byte) {})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = bus.Publish(ctx, "test_channel", []byte("hello"))

	if err := bus.Close(); err != nil {
		t.Logf("Close returned: %v", err)
	}
}
