package store

import (
	"context"
	"errors"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/fxamacker/cbor/v2"
)

// PebbleStore is a Pebble-based implementation of the Store interface
type PebbleStore[T any] struct {
	db     *pebble.DB
	mu     sync.RWMutex
	closed bool
	prefix []byte
}

// PebbleStoreConfig configures the Pebble store
type PebbleStoreConfig struct {
	Path   string
	Prefix string // Optional prefix for keys (useful when sharing a DB)
	Opts   *pebble.Options
}

// NewPebbleStore creates a new Pebble-based store
func NewPebbleStore[T any](config PebbleStoreConfig) (*PebbleStore[T], error) {
	opts := config.Opts
	if opts == nil {
		opts = &pebble.Options{
			ErrorIfExists: false,
		}
	}

	db, err := pebble.Open(config.Path, opts)
	if err != nil {
		return nil, err
	}

	prefix := []byte(config.Prefix)
	if len(prefix) == 0 {
		prefix = []byte("data:")
	}

	return &PebbleStore[T]{
		db:     db,
		prefix: prefix,
	}, nil
}

// makeKey creates a key with the prefix
func (p *PebbleStore[T]) makeKey(key string) []byte {
	fullKey := make([]byte, len(p.prefix)+len(key))
	copy(fullKey, p.prefix)
	copy(fullKey[len(p.prefix):], key)
	return fullKey
}

// Save stores or updates a value
func (p *PebbleStore[T]) Save(ctx context.Context, key string, value T) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrStoreClosed
	}
	p.mu.RUnlock()

	data, err := cbor.Marshal(value)
	if err != nil {
		return err
	}

	fullKey := p.makeKey(key)
	return p.db.Set(fullKey, data, pebble.Sync)
}

// Load retrieves a value by key
func (p *PebbleStore[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return zero, ErrStoreClosed
	}
	p.mu.RUnlock()

	fullKey := p.makeKey(key)
	data, closer, err := p.db.Get(fullKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return zero, ErrNotFound
		}
		return zero, err
	}
	defer closer.Close()

	var value T
	if err := cbor.Unmarshal(data, &value); err != nil {
		return zero, err
	}

	return value, nil
}

// Delete removes a value
func (p *PebbleStore[T]) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrStoreClosed
	}
	p.mu.RUnlock()

	fullKey := p.makeKey(key)
	return p.db.Delete(fullKey, pebble.Sync)
}

// Exists checks if a key exists
func (p *PebbleStore[T]) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return false, ErrStoreClosed
	}
	p.mu.RUnlock()

	fullKey := p.makeKey(key)
	_, closer, err := p.db.Get(fullKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	closer.Close()
	return true, nil
}

// List returns all keys
func (p *PebbleStore[T]) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	p.mu.RUnlock()

	var keys []string

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: p.prefix,
		UpperBound: append(p.prefix, 0xff),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		keyStr := string(key[len(p.prefix):])
		keys = append(keys, keyStr)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return keys, nil
}

// Close closes the store
func (p *PebbleStore[T]) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrStoreClosed
	}

	p.closed = true
	return p.db.Close()
}

// Count returns the total number of items
func (p *PebbleStore[T]) Count(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	p.mu.RUnlock()

	var count int64

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: p.prefix,
		UpperBound: append(p.prefix, 0xff),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	return count, nil
}
