package interfaces

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache interface for interacting with the cache implementation
type Cache interface {
	// Core Redis operations
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Pipeline() redis.Pipeliner
	Close() error
	Ping(ctx context.Context) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Keys(ctx context.Context, pattern string) ([]string, error)

	// JSON helpers
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error

	// Payment processor specific key helpers
	GetUserIdFromCustomerIdKey(customerId string) string
	GetCustomerIdFromUserIdKey(userId string) string
	GetCustomerDataKey(customerId string) string
}