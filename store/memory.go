package store

import (
	"context"
	"sync"
)

// MemoryStore is an in-memory implementation of the Store interface
type MemoryStore[T any] struct {
	mu     sync.RWMutex
	data   map[string]T
	closed bool
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore[T any]() *MemoryStore[T] {
	return &MemoryStore[T]{
		data: make(map[string]T),
	}
}

// Save stores or updates a value
func (m *MemoryStore[T]) Save(ctx context.Context, key string, value T) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	m.data[key] = value
	return nil
}

// Load retrieves a value by key
func (m *MemoryStore[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return zero, ErrStoreClosed
	}

	value, ok := m.data[key]
	if !ok {
		return zero, ErrNotFound
	}

	return value, nil
}

// Delete removes a value
func (m *MemoryStore[T]) Delete(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	delete(m.data, key)
	return nil
}

// Exists checks if a key exists
func (m *MemoryStore[T]) Exists(ctx context.Context, key string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, ErrStoreClosed
	}

	_, ok := m.data[key]
	return ok, nil
}

// List returns all keys
func (m *MemoryStore[T]) List(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed
	}

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Close closes the store
func (m *MemoryStore[T]) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	m.closed = true
	m.data = nil
	return nil
}

// Count returns the total number of items
func (m *MemoryStore[T]) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, ErrStoreClosed
	}

	return int64(len(m.data)), nil
}
