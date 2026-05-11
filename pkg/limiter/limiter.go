package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// Lua 脚本：ZREM 过期 + ZADD 新增 + ZCARD 计数 + EXPIRE 续期，一次 EVAL 完成
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowStart = tonumber(ARGV[2])
local member = ARGV[3]
local ttl = tonumber(ARGV[4])

redis.call('ZREMRANGEBYSCORE', key, '-inf', windowStart)
redis.call('ZADD', key, now, member)
local count = redis.call('ZCARD', key)
redis.call('EXPIRE', key, ttl)

return count
`

var script = redis.NewScript(slidingWindowScript)

type Limiter struct {
	redis *redis.Client
}

func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{redis: rdb}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())
	member := time.Now().UnixNano()
	ttl := int(window.Seconds())

	key = "rate_limit:" + key

	count, err := script.Run(ctx, l.redis, []string{key}, now, windowStart, member, ttl).Int64()
	if err != nil {
		return false, err
	}

	return count <= int64(limit), nil
}
