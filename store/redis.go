package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-based implementation of the Store interface
type RedisStore[T any] struct {
	client *redis.Client
	mu     sync.RWMutex
	closed bool
	ttl    time.Duration // Optional TTL for keys
	prefix string
	index  string // Set key for indexing all keys
}

// RedisStoreConfig configures the Redis store
type RedisStoreConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string        // Optional prefix for keys (e.g., "session:", "message:")
	TTL      time.Duration // Optional: TTL for keys (0 = no TTL)
	Options  *redis.Options
}

// NewRedisStore creates a new Redis-based store
func NewRedisStore[T any](config RedisStoreConfig) (*RedisStore[T], error) {
	var client *redis.Client

	if config.Options != nil {
		client = redis.NewClient(config.Options)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "data:"
	}

	return &RedisStore[T]{
		client: client,
		ttl:    config.TTL,
		prefix: prefix,
		index:  prefix + "index",
	}, nil
}

// makeKey creates a Redis key with the prefix
func (r *RedisStore[T]) makeKey(key string) string {
	return r.prefix + key
}

// Save stores or updates a value
func (r *RedisStore[T]) Save(ctx context.Context, key string, value T) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrStoreClosed
	}
	r.mu.RUnlock()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := r.makeKey(key)

	pipe := r.client.Pipeline()

	// Save data
	if r.ttl > 0 {
		pipe.Set(ctx, fullKey, data, r.ttl)
	} else {
		pipe.Set(ctx, fullKey, data, 0)
	}

	// Add to index set
	pipe.SAdd(ctx, r.index, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save value: %w", err)
	}

	return nil
}

// Load retrieves a value by key
func (r *RedisStore[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return zero, ErrStoreClosed
	}
	r.mu.RUnlock()

	fullKey := r.makeKey(key)
	data, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, ErrNotFound
		}
		return zero, fmt.Errorf("failed to load value: %w", err)
	}

	var value T
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return zero, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

// Delete removes a value
func (r *RedisStore[T]) Delete(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrStoreClosed
	}
	r.mu.RUnlock()

	fullKey := r.makeKey(key)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, fullKey)
	pipe.SRem(ctx, r.index, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

// Exists checks if a key exists
func (r *RedisStore[T]) Exists(ctx context.Context, key string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return false, ErrStoreClosed
	}
	r.mu.RUnlock()

	fullKey := r.makeKey(key)
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

// List returns all keys
func (r *RedisStore[T]) List(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	r.mu.RUnlock()

	keys, err := r.client.SMembers(ctx, r.index).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	return keys, nil
}

// Close closes the store
func (r *RedisStore[T]) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	r.closed = true
	return r.client.Close()
}

// Count returns the total number of items
func (r *RedisStore[T]) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	r.mu.RUnlock()

	count, err := r.client.SCard(ctx, r.index).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count items: %w", err)
	}

	return count, nil
}
