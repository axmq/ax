package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisSessionPrefix = "session:"
	redisSessionIndex  = "sessions:index"
)

// RedisStore is a Redis-based implementation of the Store interface
type RedisStore struct {
	client *redis.Client
	mu     sync.RWMutex
	closed bool
	ttl    time.Duration // Optional TTL for keys
}

// RedisStoreConfig configures the Redis store
type RedisStoreConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration // Optional: TTL for session keys (0 = no TTL)
	Options  *redis.Options
}

// NewRedisStore creates a new Redis-based session store
func NewRedisStore(config RedisStoreConfig) (*RedisStore, error) {
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

	return &RedisStore{
		client: client,
		ttl:    config.TTL,
	}, nil
}

// makeRedisKey creates a Redis key for a client ID
func makeRedisKey(clientID string) string {
	return redisSessionPrefix + clientID
}

// Save stores or updates a session
func (r *RedisStore) Save(ctx context.Context, session *Session) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrStoreClosed
	}
	r.mu.RUnlock()

	data := sessionToData(session)
	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := makeRedisKey(session.GetClientID())

	pipe := r.client.Pipeline()

	// Save session data
	if r.ttl > 0 {
		pipe.Set(ctx, key, value, r.ttl)
	} else {
		pipe.Set(ctx, key, value, 0)
	}

	// Add to index set
	pipe.SAdd(ctx, redisSessionIndex, session.GetClientID())

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// Load retrieves a session by client ID
func (r *RedisStore) Load(ctx context.Context, clientID string) (*Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	r.mu.RUnlock()

	key := makeRedisKey(clientID)
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	var data sessionData
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return dataToSession(&data), nil
}

// Delete removes a session
func (r *RedisStore) Delete(ctx context.Context, clientID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrStoreClosed
	}
	r.mu.RUnlock()

	key := makeRedisKey(clientID)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, redisSessionIndex, clientID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// Exists checks if a session exists
func (r *RedisStore) Exists(ctx context.Context, clientID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return false, ErrStoreClosed
	}
	r.mu.RUnlock()

	key := makeRedisKey(clientID)
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

// List returns all session client IDs
func (r *RedisStore) List(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	r.mu.RUnlock()

	members, err := r.client.SMembers(ctx, redisSessionIndex).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	return members, nil
}

// Close closes the store
func (r *RedisStore) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	r.closed = true
	return r.client.Close()
}

// Count returns the total number of sessions
func (r *RedisStore) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	r.mu.RUnlock()

	count, err := r.client.SCard(ctx, redisSessionIndex).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}

// CountByState returns the number of sessions in a given state
func (r *RedisStore) CountByState(ctx context.Context, state State) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	r.mu.RUnlock()

	clientIDs, err := r.List(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, clientID := range clientIDs {
		session, err := r.Load(ctx, clientID)
		if err != nil {
			continue
		}
		if session.GetState() == state {
			count++
		}
	}

	return count, nil
}

// Flush removes all sessions from the store (useful for testing)
func (r *RedisStore) Flush(ctx context.Context) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrStoreClosed
	}
	r.mu.RUnlock()

	clientIDs, err := r.List(ctx)
	if err != nil {
		return err
	}

	if len(clientIDs) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, clientID := range clientIDs {
		pipe.Del(ctx, makeRedisKey(clientID))
	}
	pipe.Del(ctx, redisSessionIndex)

	_, err = pipe.Exec(ctx)
	return err
}
