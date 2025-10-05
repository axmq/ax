package session

import (
	"context"
	"sync"
)

// MemoryStore is an in-memory implementation of the Store interface
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	closed   bool
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Save stores or updates a session
func (m *MemoryStore) Save(ctx context.Context, session *Session) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	m.sessions[session.GetClientID()] = session
	return nil
}

// Load retrieves a session by client ID
func (m *MemoryStore) Load(ctx context.Context, clientID string) (*Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed
	}

	session, ok := m.sessions[clientID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// Delete removes a session
func (m *MemoryStore) Delete(ctx context.Context, clientID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	delete(m.sessions, clientID)
	return nil
}

// Exists checks if a session exists
func (m *MemoryStore) Exists(ctx context.Context, clientID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, ErrStoreClosed
	}

	_, ok := m.sessions[clientID]
	return ok, nil
}

// List returns all session client IDs
func (m *MemoryStore) List(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed
	}

	clientIDs := make([]string, 0, len(m.sessions))
	for clientID := range m.sessions {
		clientIDs = append(clientIDs, clientID)
	}

	return clientIDs, nil
}

// Close closes the store
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	m.closed = true
	m.sessions = nil
	return nil
}

// Count returns the total number of sessions
func (m *MemoryStore) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, ErrStoreClosed
	}

	return int64(len(m.sessions)), nil
}

// CountByState returns the number of sessions in a given state
func (m *MemoryStore) CountByState(ctx context.Context, state State) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, ErrStoreClosed
	}

	var count int64
	for _, session := range m.sessions {
		if session.GetState() == state {
			count++
		}
	}

	return count, nil
}
