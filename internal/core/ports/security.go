package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenExpired      = errors.New("security: token has expired")
	ErrTokenInvalidSig   = errors.New("security: token signature invalid")
	ErrTokenMalformed    = errors.New("security: token format is malformed")
	ErrRateLimitExceeded = errors.New("security: rate limit quota exceeded")
)

type UserIdentity struct {
	UserID    string    `json:"sub"`
	DeviceID  string    `json:"dev_id,omitempty"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

type TokenVerifier interface {
	Verify(tokenString string) (*UserIdentity, error)
}

type DistributedRateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, time.Duration, error)
}
