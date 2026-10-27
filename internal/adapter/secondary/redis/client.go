package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClientFromEnv() (*redis.Client, error) {
	addr := getEnv("REDIS_ADDR", "127.0.0.1:6379")
	password := getEnv("REDIS_PASSWORD", "")
	username := getEnv("REDIS_USERNAME", "")
	db := getEnvAsInt("REDIS_DB", 0)
	poolSize := getEnvAsInt("REDIS_POOL_SIZE", 100)
	minIdleConns := getEnvAsInt("REDIS_MIN_IDLE_CONNS", 20)
	dialTimeout := time.Duration(getEnvAsInt("REDIS_DIAL_TIMEOUT_MS", 5000)) * time.Millisecond
	readTimeout := time.Duration(getEnvAsInt("REDIS_READ_TIMEOUT_MS", 3000)) * time.Millisecond
	writeTimeout := time.Duration(getEnvAsInt("REDIS_WRITE_TIMEOUT_MS", 3000)) * time.Millisecond

	opts := &redis.Options{
		Addr:         addr,
		Username:     username,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed at %s: %w", addr, err)
	}

	return client, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}
