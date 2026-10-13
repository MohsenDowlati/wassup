package ports

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, windowSeconds int64) (bool, error)
}
