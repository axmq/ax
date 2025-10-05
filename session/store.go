package session

import (
	"context"
	"errors"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrStoreClosed          = errors.New("store is closed")
)

// Store defines the interface for session persistence
type Store interface {
	// Save stores or updates a session
	Save(ctx context.Context, session *Session) error

	// Load retrieves a session by client ID
	Load(ctx context.Context, clientID string) (*Session, error)

	// Delete removes a session
	Delete(ctx context.Context, clientID string) error

	// Exists checks if a session exists
	Exists(ctx context.Context, clientID string) (bool, error)

	// List returns all session client IDs
	List(ctx context.Context) ([]string, error)

	// Close closes the store
	Close() error
}

// StoreMetrics provides metrics about the store
type StoreMetrics interface {
	// Count returns the total number of sessions
	Count(ctx context.Context) (int64, error)

	// CountByState returns the number of sessions in a given state
	CountByState(ctx context.Context, state State) (int64, error)
}
