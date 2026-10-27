package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/MohsenDowlati/wassup/internal/core/ports"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	sessionTTL = 60 * time.Second

	userSessionPrefix = "user:%s:gateways"
)

type SessionRegistry struct {
	client *redis.Client
}

func NewSessionRegistry(client *redis.Client) ports.SessionRegistry {
	return &SessionRegistry{
		client: client,
	}
}

func (r *SessionRegistry) RegisterSession(ctx context.Context, userID, gatewayID string) error {
	_ = fmt.Sprintf(userSessionPrefix, userID)

	pipe := r.client.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("user:%s:gateway", userID), gatewayID, sessionTTL)
	pipe.Set(ctx, fmt.Sprintf("user:%s:heartbeat", userID), time.Now().Unix(), sessionTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis: failed to register session: %w", err)
	}
	return nil
}

func (r *SessionRegistry) UnregisterSession(ctx context.Context, userID, gatewayID string) error {
	script := `
		local current = redis.call('GET', KEYS[1])
		if current == ARGV[1] then
			redis.call('DEL', KEYS[1])
			redis.call('DEL', KEYS[2])
			return 1
		else
			return 0
		end
	`

	key := fmt.Sprintf("user:%s:gateway", userID)
	heartbeatKey := fmt.Sprintf("user:%s:heartbeat", userID)

	err := r.client.Eval(ctx, script, []string{key, heartbeatKey}, gatewayID).Err()
	if err != nil {
		return fmt.Errorf("redis: failed to unregister session: %w", err)
	}
	return nil
}

func (r *SessionRegistry) LocateUser(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("user:%s:gateway", userID)
	gatewayID, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis: locate user failed: %w", err)
	}
	return gatewayID, nil
}

func (r *SessionRegistry) RefreshSession(ctx context.Context, userID, gatewayID string) error {
	script := `
		local current = redis.call('GET', KEYS[1])
		if current == ARGV[1] then
			redis.call('EXPIRE', KEYS[1], ARGV[2])
			return 1
		else
			return 0
		end
	`

	key := fmt.Sprintf("user:%s:gateway", userID)
	err := r.client.Eval(ctx, script, []string{key}, gatewayID, int(sessionTTL.Seconds())).Err()
	if err != nil {
		return fmt.Errorf("redis: failed to refresh lease: %w", err)
	}
	return nil
}

func (r *SessionRegistry) FindNode(ctx context.Context, userID string) (string, bool, error) {
	nodeID, err := r.client.Get(ctx, "user:node:"+userID).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return nodeID, true, nil
}
