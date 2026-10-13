package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MohsenDowlati/wassup/internal/core/ports"
	goredis "github.com/redis/go-redis/v9"
)

const slidingWindowLua = `
local key          = KEYS[1]
local now          = tonumber(ARGV[1])
local window       = tonumber(ARGV[2])
local limit        = tonumber(ARGV[3])
local uniqueMember = ARGV[4]

local clearBefore = now - window
redis.call('ZREMRANGEBYSCORE', key, '-inf', clearBefore)

local currentRequests = redis.call('ZCARD', key)

if currentRequests < limit then
    redis.call('ZADD', key, now, uniqueMember)
    redis.call('PEXPIRE', key, window)
    return {1, limit - currentRequests - 1, 0}
else
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retryAfter = 0
    if #oldest > 0 then
        retryAfter = (tonumber(oldest[2]) + window) - now
        if retryAfter < 0 then retryAfter = 0 end
    end
    return {0, 0, retryAfter}
end
`

type RedisRateLimiter struct {
	client *goredis.Client
	script *goredis.Script
}

func NewRedisRateLimiter(client *goredis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		script: goredis.NewScript(slidingWindowLua),
	}
}

func (r *RedisRateLimiter) Allow(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, int64, time.Duration, error) {
	nowMs := time.Now().UTC().UnixMilli()
	windowMs := window.Milliseconds()

	randSuffix := make([]byte, 4)
	if _, err := rand.Read(randSuffix); err != nil {
		return false, 0, 0, fmt.Errorf("failed to generate entropy: %w", err)
	}
	uniqueMember := fmt.Sprintf("%d:%s", nowMs, hex.EncodeToString(randSuffix))

	res, err := r.script.Run(ctx, r.client, []string{"rl:" + key}, nowMs, windowMs, limit, uniqueMember).Result()
	if err != nil {
		return false, 0, 0, fmt.Errorf("redis rate limiter eval failed: %w", err)
	}

	results, ok := res.([]interface{})
	if !ok || len(results) < 3 {
		return false, 0, 0, fmt.Errorf("unexpected script output shape: %v", res)
	}

	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)
	retryAfterMs := results[2].(int64)

	return allowed, remaining, time.Duration(retryAfterMs) * time.Millisecond, nil
}

var _ ports.DistributedRateLimiter = (*RedisRateLimiter)(nil)
