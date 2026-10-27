package security_test

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redisAdapter "github.com/MohsenDowlati/wassup/internal/adapter/secondary/redis"
)

func TestSlidingWindowRateLimiter_ConcurrentRaces(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	limiter := redisAdapter.NewRedisRateLimiter(rdb)

	ctx := context.Background()
	key := "test:user:123"
	var quota int64 = 50
	window := 1 * time.Second

	var wg sync.WaitGroup
	concurrentClients := 100
	var allowedCount int64
	var blockedCount int64

	wg.Add(concurrentClients)
	for i := 0; i < concurrentClients; i++ {
		go func(workerID int) {
			defer wg.Done()
			allowed, _, _, err := limiter.Allow(ctx, key, quota, window)
			if err != nil {
				t.Errorf("worker %d unexpected error: %v", workerID, err)
				return
			}
			if allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&blockedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, quota, allowedCount, "allowed requests must match exact quota")
	assert.Equal(t, int64(concurrentClients)-quota, blockedCount, "excess requests must be blocked")

	s.FastForward(window + 100*time.Millisecond)

	allowed, _, _, err := limiter.Allow(ctx, key, quota, window)
	require.NoError(t, err)
	assert.True(t, allowed, "request must be permitted after window expiration")
}
