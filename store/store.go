package store

import (
	"context"
)

// Store defines a generic key-value store interface that can be used
// for various purposes (sessions, messages, metadata, etc.)
type Store[T any] interface {
	Reader[T]
	Metrics

	// Save stores or updates a value by key
	Save(ctx context.Context, key string, value T) error

	// Delete removes a value by key
	Delete(ctx context.Context, key string) error

	// Close closes the store
	Close() error
}

type Reader[T any] interface {
	// Load retrieves a value by key
	Load(ctx context.Context, key string) (T, error)

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// List returns all keys
	List(ctx context.Context) ([]string, error)
}

// Metrics provides metrics about the store
type Metrics interface {
	// Count returns the total number of items
	Count(ctx context.Context) (int64, error)
}
