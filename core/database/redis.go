package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDriverImpl implements RedisDriver for Redis.
type RedisDriverImpl struct {
	client *redis.Client
	name   string
}

// NewRedisDriver creates a new Redis driver.
func NewRedisDriver(name string) *RedisDriverImpl {
	return &RedisDriverImpl{
		name: name,
	}
}

// Name returns the driver name.
func (d *RedisDriverImpl) Name() string {
	return d.name
}

// Connect opens the Redis connection.
func (d *RedisDriverImpl) Connect(ctx context.Context, cfg Config) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       0,
	}

	if dsn, ok := cfg.Options["dsn"]; ok && dsn != "" {
		parsedOpts, err := redis.ParseURL(dsn)
		if err != nil {
			return fmt.Errorf("parsing Redis URL: %w", err)
		}
		opts = parsedOpts
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close() //nolint:errcheck
		return fmt.Errorf("pinging Redis: %w", err)
	}

	d.client = client
	return nil
}

// Get retrieves a value by key.
func (d *RedisDriverImpl) Get(ctx context.Context, key string) (string, error) {
	if d.client == nil {
		return "", fmt.Errorf("redis: not connected")
	}
	val, err := d.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("get %q: %w", key, err)
	}
	return val, nil
}

// Set sets a key-value pair with optional TTL.
func (d *RedisDriverImpl) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if d.client == nil {
		return fmt.Errorf("redis: not connected")
	}
	return d.client.Set(ctx, key, value, ttl).Err()
}

// Del deletes one or more keys.
func (d *RedisDriverImpl) Del(ctx context.Context, keys ...string) (int64, error) {
	if d.client == nil {
		return 0, fmt.Errorf("redis: not connected")
	}
	return d.client.Del(ctx, keys...).Result()
}

// Keys returns keys matching a pattern.
func (d *RedisDriverImpl) Keys(ctx context.Context, pattern string) ([]string, error) {
	if d.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	return d.client.Keys(ctx, pattern).Result()
}

// HGetAll returns all fields of a hash.
func (d *RedisDriverImpl) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if d.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	return d.client.HGetAll(ctx, key).Result()
}

// LRange returns a range of elements from a list.
func (d *RedisDriverImpl) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if d.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	return d.client.LRange(ctx, key, start, stop).Result()
}

// Info returns Redis server information.
func (d *RedisDriverImpl) Info(ctx context.Context) (string, error) {
	if d.client == nil {
		return "", fmt.Errorf("redis: not connected")
	}
	return d.client.Info(ctx).Result()
}

// GetSchema returns a description of the Redis data.
func (d *RedisDriverImpl) GetSchema(ctx context.Context) (string, error) {
	if d.client == nil {
		return "", fmt.Errorf("redis: not connected")
	}

	// Get sample keys to understand data patterns
	keys, err := d.client.Keys(ctx, "*").Result()
	if err != nil {
		return "", fmt.Errorf("listing keys: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Redis key-value store\n")

	if len(keys) == 0 {
		sb.WriteString("No keys found\n")
		return sb.String(), nil
	}

	// Analyze key patterns
	patterns := make(map[string]int)
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			patterns[parts[0]+":*"]++
		} else {
			patterns[key]++
		}
	}

	sb.WriteString("Key patterns:\n")
	for pattern, count := range patterns {
		sb.WriteString(fmt.Sprintf("  %s (%d keys)\n", pattern, count))
	}

	// Sample keys (max 10)
	sb.WriteString("Sample keys:\n")
	limit := 10
	if len(keys) < limit {
		limit = len(keys)
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("  %s\n", keys[i]))
	}
	if len(keys) > 10 {
		sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(keys)-10))
	}

	return sb.String(), nil
}

// Ping checks the connection.
func (d *RedisDriverImpl) Ping(ctx context.Context) error {
	if d.client == nil {
		return fmt.Errorf("redis: not connected")
	}
	return d.client.Ping(ctx).Err()
}

// Close closes the connection.
func (d *RedisDriverImpl) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}
