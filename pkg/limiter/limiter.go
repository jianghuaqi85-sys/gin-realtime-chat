package limiter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Lua 脚本：滑动窗口核心逻辑
// 1. 清理窗口外的数据
// 2. 统计当前窗口内的请求数
// 3. 判断是否超限：未超限则 ZADD 并返回 1，超限则不记录并返回 0 (规避被拒请求占用配额雪球效应)
// 4. 重置 PEXPIRE 毫秒级 TTL
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowStart = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
local ttl = tonumber(ARGV[5]) -- 毫秒

-- 移除时间窗口之前的旧数据
redis.call('ZREMRANGEBYSCORE', key, '-inf', windowStart)

-- 计算当前窗口内存在的请求数
local count = tonumber(redis.call('ZCARD', key))

if count < limit then
    -- 未超限：写入 ZSET 并设置毫秒级过期时间
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, ttl)
    return 1
else
    -- 超限：不写入 ZSET，但刷新过期时间防 key 提前失效
    redis.call('PEXPIRE', key, ttl)
    return 0
end
`

var script = redis.NewScript(slidingWindowScript)

type Limiter struct {
	redis *redis.Client
}

func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{redis: rdb}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	windowStart := nowMs - window.Milliseconds()
	ttlMs := window.Milliseconds()

	// 保证 ZSET 成员的绝对唯一性：纳秒时间戳 + 8字节十六进制随机串
	member := fmt.Sprintf("%d-%s", now.UnixNano(), generateRandomString(4))

	key = "rate_limit:" + key

	res, err := script.Run(ctx, l.redis, []string{key}, nowMs, windowStart, limit, member, ttlMs).Result()
	if err != nil {
		return false, err
	}

	return res.(int64) == 1, nil
}

// generateRandomString 生成指定字节数的十六进制随机字符串
func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
