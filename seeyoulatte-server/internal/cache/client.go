package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	redislib "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb    *redislib.Client
	logger *slog.Logger
}

func NewClient(logger *slog.Logger) *Client {
	// Parse Redis port
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6395"
	}

	// Build Redis address
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	// Parse Redis DB number
	dbNum := 0
	if db := os.Getenv("REDIS_DB"); db != "" {
		if parsed, err := strconv.Atoi(db); err == nil {
			dbNum = parsed
		}
	}

	// Parse pool size
	poolSize := 10
	if ps := os.Getenv("REDIS_POOL_SIZE"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil {
			poolSize = parsed
		}
	}

	// Parse min idle connections
	minIdleConns := 5
	if mic := os.Getenv("REDIS_MIN_IDLE_CONNS"); mic != "" {
		if parsed, err := strconv.Atoi(mic); err == nil {
			minIdleConns = parsed
		}
	}

	rdb := redislib.NewClient(&redislib.Options{
		Addr:            addr,
		Password:        os.Getenv("REDIS_PASSWORD"),
		DB:              dbNum,
		MaxRetries:      3,
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		ConnMaxIdleTime: 300 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	})

	return &Client{
		rdb:    rdb,
		logger: logger,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	// Test the connection
	_, err := c.rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	c.logger.Info("Redis client connected successfully",
		slog.String("addr", c.rdb.Options().Addr))
	return nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	result := c.rdb.SetNX(ctx, key, value, expiration)
	return result.Val(), result.Err()
}

func (c *Client) Pipeline() redislib.Pipeliner {
	return c.rdb.Pipeline()
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result := c.rdb.Exists(ctx, key)
	return result.Val() > 0, result.Err()
}

// TTL returns the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

// Expire sets a timeout on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// Keys finds all keys matching a pattern (use carefully in production)
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.rdb.Keys(ctx, pattern).Result()
}

// GetSetJSON is a helper for JSON operations
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Set(ctx, key, jsonData, expiration)
}

// GetJSON is a helper for JSON retrieval
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	jsonStr, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(jsonStr), dest)
}

// Helper methods for payment processor cache key management
// There are 3 primary keys for mappings:
//
// -- Data --
// 1. paymentprocessor:customer:{customerId} → full customer payment processor data
//
// -- Key Mappings --
// 2. paymentprocessor:customer:{customerId}:userid → userId (customerId → userId lookup)
// 3. paymentprocessor:user:{userId}:customerid → customerId (userId → customerId lookup)

const (
	cacheKeyCustomerData       = "paymentprocessor:customer:%s"
	cacheKeyCustomerIdToUserId = "paymentprocessor:customer:%s:userid"
	cacheKeyUserIdToCustomerId = "paymentprocessor:user:%s:customerid"
)

// GetCustomerDataKey returns the cache key for full customer data
func (c *Client) GetCustomerDataKey(customerId string) string {
	return fmt.Sprintf(cacheKeyCustomerData, customerId)
}

// GetUserIdFromCustomerIdKey returns the cache key for customerId to userId mapping
func (c *Client) GetUserIdFromCustomerIdKey(customerId string) string {
	return fmt.Sprintf(cacheKeyCustomerIdToUserId, customerId)
}

// GetCustomerIdFromUserIdKey returns the cache key for userId to customerId mapping
func (c *Client) GetCustomerIdFromUserIdKey(userId string) string {
	return fmt.Sprintf(cacheKeyUserIdToCustomerId, userId)
}