package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Manager manages session lifecycle, expiry, and recovery
type Manager struct {
	mu                sync.RWMutex
	store             Store
	activeSessions    map[string]*Session // clientID -> session for quick access
	expiryCheckTicker *time.Ticker
	stopCh            chan struct{}
	wg                sync.WaitGroup
	willPublisher     WillPublisher
	assignedIDPrefix  string
}

// WillPublisher defines the interface for publishing will messages
type WillPublisher interface {
	PublishWill(ctx context.Context, will *WillMessage, clientID string) error
}

// ManagerConfig configures the session manager
type ManagerConfig struct {
	Store               Store
	ExpiryCheckInterval time.Duration
	WillPublisher       WillPublisher
	AssignedIDPrefix    string
}

// NewManager creates a new session manager
func NewManager(config ManagerConfig) *Manager {
	if config.ExpiryCheckInterval == 0 {
		config.ExpiryCheckInterval = 30 * time.Second
	}
	if config.AssignedIDPrefix == "" {
		config.AssignedIDPrefix = "auto-"
	}

	m := &Manager{
		store:             config.Store,
		activeSessions:    make(map[string]*Session),
		expiryCheckTicker: time.NewTicker(config.ExpiryCheckInterval),
		stopCh:            make(chan struct{}),
		willPublisher:     config.WillPublisher,
		assignedIDPrefix:  config.AssignedIDPrefix,
	}

	m.wg.Add(1)
	go m.expiryChecker()

	return m
}

// CreateSession creates a new session or returns an existing one
func (m *Manager) CreateSession(ctx context.Context, clientID string, cleanStart bool, expiryInterval uint32, protocolVersion byte) (*Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingSession, err := m.store.Load(ctx, clientID)
	if err != nil && err != ErrSessionNotFound {
		return nil, false, err
	}

	sessionPresent := false

	if existingSession != nil && !existingSession.IsExpired() {
		if cleanStart {
			// Clean start with existing session - clear it
			existingSession.Clear()
			existingSession.CleanStart = true
			existingSession.ExpiryInterval = expiryInterval
			existingSession.SetActive()
			sessionPresent = false
		} else {
			// Resume existing session
			existingSession.SetActive()
			if expiryInterval > 0 {
				existingSession.UpdateExpiryInterval(expiryInterval)
			}
			sessionPresent = true
		}
		m.activeSessions[clientID] = existingSession
		if err := m.store.Save(ctx, existingSession); err != nil {
			return nil, false, err
		}
		return existingSession, sessionPresent, nil
	}

	// Create new session
	session := New(clientID, cleanStart, expiryInterval, protocolVersion)
	session.SetActive()
	m.activeSessions[clientID] = session

	if err := m.store.Save(ctx, session); err != nil {
		delete(m.activeSessions, clientID)
		return nil, false, err
	}

	return session, false, nil
}

// GetSession retrieves a session by client ID
func (m *Manager) GetSession(ctx context.Context, clientID string) (*Session, error) {
	m.mu.RLock()
	if session, ok := m.activeSessions[clientID]; ok {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	return m.store.Load(ctx, clientID)
}

// DisconnectSession marks a session as disconnected and handles will message
func (m *Manager) DisconnectSession(ctx context.Context, clientID string, sendWill bool) error {
	session, err := m.GetSession(ctx, clientID)
	if err != nil {
		return err
	}

	session.SetDisconnected()

	// Handle will message
	if sendWill && session.WillMessage != nil {
		if session.WillDelayInterval == 0 {
			// Publish immediately
			if m.willPublisher != nil {
				if err := m.willPublisher.PublishWill(ctx, session.WillMessage, clientID); err != nil {
					// Log error but don't fail disconnection
				}
			}
			session.ClearWillMessage()
		}
		// If delay > 0, will be handled by expiry checker
	} else {
		session.ClearWillMessage()
	}

	// Remove from active sessions
	m.mu.Lock()
	delete(m.activeSessions, clientID)
	m.mu.Unlock()

	// Clean session - remove immediately
	cleanStart := session.GetCleanStart()
	expiryInterval := session.GetExpiryInterval()
	if cleanStart || expiryInterval == 0 {
		return m.store.Delete(ctx, clientID)
	}

	return m.store.Save(ctx, session)
}

// RemoveSession removes a session completely
func (m *Manager) RemoveSession(ctx context.Context, clientID string) error {
	m.mu.Lock()
	delete(m.activeSessions, clientID)
	m.mu.Unlock()

	return m.store.Delete(ctx, clientID)
}

// TakeoverSession handles session takeover when a new connection uses an existing client ID
func (m *Manager) TakeoverSession(ctx context.Context, clientID string) error {
	session, err := m.GetSession(ctx, clientID)
	if err != nil {
		if err == ErrSessionNotFound {
			return nil
		}
		return err
	}

	// Clear will message on takeover
	session.ClearWillMessage()

	return nil
}

// GenerateClientID generates a unique client ID for clients that don't provide one
func (m *Manager) GenerateClientID(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		clientID := m.assignedIDPrefix + hex.EncodeToString(b)

		exists, err := m.store.Exists(ctx, clientID)
		if err != nil {
			return "", err
		}
		if !exists {
			return clientID, nil
		}
	}

	return "", ErrSessionAlreadyExists
}

// expiryChecker runs periodically to check for expired sessions
func (m *Manager) expiryChecker() {
	defer m.wg.Done()

	for {
		select {
		case <-m.expiryCheckTicker.C:
			m.checkExpiredSessions()
		case <-m.stopCh:
			return
		}
	}
}

// checkExpiredSessions checks and removes expired sessions
func (m *Manager) checkExpiredSessions() {
	ctx := context.Background()

	clientIDs, err := m.store.List(ctx)
	if err != nil {
		return
	}

	for _, clientID := range clientIDs {
		session, err := m.store.Load(ctx, clientID)
		if err != nil {
			continue
		}

		if session.IsExpired() {
			// Publish delayed will message if present
			if session.WillMessage != nil && session.ShouldPublishWill() {
				if m.willPublisher != nil {
					_ = m.willPublisher.PublishWill(ctx, session.WillMessage, clientID)
				}
			}

			// Remove expired session
			session.SetExpired()
			_ = m.store.Delete(ctx, clientID)
		} else if session.GetState() == StateDisconnected && session.WillMessage != nil {
			// Check if delayed will should be published
			if session.ShouldPublishWill() {
				if m.willPublisher != nil {
					_ = m.willPublisher.PublishWill(ctx, session.WillMessage, clientID)
				}
				session.ClearWillMessage()
				_ = m.store.Save(ctx, session)
			}
		}
	}
}

// Close closes the manager and stops background tasks
func (m *Manager) Close() error {
	close(m.stopCh)
	m.expiryCheckTicker.Stop()
	m.wg.Wait()

	return m.store.Close()
}

// GetActiveSessionCount returns the number of active sessions
func (m *Manager) GetActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.activeSessions)
}

// GetAllActiveSessions returns all active session client IDs
func (m *Manager) GetAllActiveSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clientIDs := make([]string, 0, len(m.activeSessions))
	for clientID := range m.activeSessions {
		clientIDs = append(clientIDs, clientID)
	}
	return clientIDs
}
