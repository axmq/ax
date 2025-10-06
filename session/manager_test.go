package session

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axmq/ax/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWillPublisher struct {
	mu        sync.Mutex
	published []*WillMessage
	clientIDs []string
}

func (m *mockWillPublisher) PublishWill(ctx context.Context, will *WillMessage, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, will)
	m.clientIDs = append(m.clientIDs, clientID)
	return nil
}

func (m *mockWillPublisher) getPublishedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.published)
}

func (m *mockWillPublisher) getPublished() []*WillMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*WillMessage, len(m.published))
	copy(result, m.published)
	return result
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name   string
		config ManagerConfig
	}{
		{
			name: "create manager with defaults",
			config: ManagerConfig{
				Store: store.NewMemoryStore[*Session](),
			},
		},
		{
			name: "create manager with custom config",
			config: ManagerConfig{
				Store:               store.NewMemoryStore[*Session](),
				ExpiryCheckInterval: 10 * time.Second,
				AssignedIDPrefix:    "custom-",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)
			require.NotNil(t, manager)
			assert.NotNil(t, manager.store)
			assert.NotNil(t, manager.activeSessions)
			assert.NotNil(t, manager.expiryCheckTicker)
			assert.NotNil(t, manager.stopCh)

			err := manager.Close()
			assert.NoError(t, err)
		})
	}
}

func TestManager_CreateSession(t *testing.T) {
	tests := []struct {
		name            string
		setupManager    func() *Manager
		clientID        string
		cleanStart      bool
		expiryInterval  uint32
		protocolVersion byte
		expectedPresent bool
		expectError     bool
	}{
		{
			name: "create new session",
			setupManager: func() *Manager {
				return NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
			},
			clientID:        "client1",
			cleanStart:      true,
			expiryInterval:  300,
			protocolVersion: 5,
			expectedPresent: false,
			expectError:     false,
		},
		{
			name: "resume existing session",
			setupManager: func() *Manager {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				_, _, _ = m.CreateSession(context.Background(), "client1", false, 300, 5)
				_ = m.DisconnectSession(context.Background(), "client1", false)
				return m
			},
			clientID:        "client1",
			cleanStart:      false,
			expiryInterval:  300,
			protocolVersion: 5,
			expectedPresent: true,
			expectError:     false,
		},
		{
			name: "clean start with existing session",
			setupManager: func() *Manager {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				s, _, _ := m.CreateSession(context.Background(), "client1", false, 300, 5)
				s.AddSubscription(&Subscription{TopicFilter: "test/topic", QoS: 1})
				_ = m.DisconnectSession(context.Background(), "client1", false)
				return m
			},
			clientID:        "client1",
			cleanStart:      true,
			expiryInterval:  300,
			protocolVersion: 5,
			expectedPresent: false,
			expectError:     false,
		},
		{
			name: "create session with clean start",
			setupManager: func() *Manager {
				return NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
			},
			clientID:        "client2",
			cleanStart:      true,
			expiryInterval:  0,
			protocolVersion: 4,
			expectedPresent: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()
			defer manager.Close()

			session, present, err := manager.CreateSession(
				context.Background(),
				tt.clientID,
				tt.cleanStart,
				tt.expiryInterval,
				tt.protocolVersion,
			)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.clientID, session.ClientID)
				assert.Equal(t, tt.expectedPresent, present)
				assert.Equal(t, StateActive, session.GetState())
			}
		})
	}
}

func TestManager_GetSession(t *testing.T) {
	tests := []struct {
		name         string
		setupManager func() *Manager
		clientID     string
		expectError  bool
	}{
		{
			name: "get active session",
			setupManager: func() *Manager {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				_, _, _ = m.CreateSession(context.Background(), "client1", true, 300, 5)
				return m
			},
			clientID:    "client1",
			expectError: false,
		},
		{
			name: "get disconnected session",
			setupManager: func() *Manager {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				_, _, _ = m.CreateSession(context.Background(), "client1", false, 300, 5)
				_ = m.DisconnectSession(context.Background(), "client1", false)
				return m
			},
			clientID:    "client1",
			expectError: false,
		},
		{
			name: "get non-existent session",
			setupManager: func() *Manager {
				return NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
			},
			clientID:    "client1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()
			defer manager.Close()

			session, err := manager.GetSession(context.Background(), tt.clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.clientID, session.ClientID)
			}
		})
	}
}

func TestManager_DisconnectSession(t *testing.T) {
	tests := []struct {
		name         string
		setupManager func() (*Manager, *mockWillPublisher)
		clientID     string
		sendWill     bool
		expectWill   bool
		expectError  bool
	}{
		{
			name: "disconnect without will",
			setupManager: func() (*Manager, *mockWillPublisher) {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				_, _, _ = m.CreateSession(context.Background(), "client1", true, 300, 5)
				return m, nil
			},
			clientID:    "client1",
			sendWill:    false,
			expectWill:  false,
			expectError: false,
		},
		{
			name: "disconnect with will no delay",
			setupManager: func() (*Manager, *mockWillPublisher) {
				wp := &mockWillPublisher{}
				m := NewManager(ManagerConfig{
					Store:         store.NewMemoryStore[*Session](),
					WillPublisher: wp,
				})
				s, _, _ := m.CreateSession(context.Background(), "client1", true, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "client/status",
					Payload: []byte("offline"),
				}, 0)
				return m, wp
			},
			clientID:    "client1",
			sendWill:    true,
			expectWill:  true,
			expectError: false,
		},
		{
			name: "disconnect with will delay",
			setupManager: func() (*Manager, *mockWillPublisher) {
				wp := &mockWillPublisher{}
				m := NewManager(ManagerConfig{
					Store:         store.NewMemoryStore[*Session](),
					WillPublisher: wp,
				})
				s, _, _ := m.CreateSession(context.Background(), "client1", false, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "client/status",
					Payload: []byte("offline"),
				}, 60)
				return m, wp
			},
			clientID:    "client1",
			sendWill:    true,
			expectWill:  false,
			expectError: false,
		},
		{
			name: "disconnect clean start session",
			setupManager: func() (*Manager, *mockWillPublisher) {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				_, _, _ = m.CreateSession(context.Background(), "client1", true, 300, 5)
				return m, nil
			},
			clientID:    "client1",
			sendWill:    false,
			expectWill:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, wp := tt.setupManager()
			defer manager.Close()

			err := manager.DisconnectSession(context.Background(), tt.clientID, tt.sendWill)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectWill {
					require.NotNil(t, wp)
					assert.Equal(t, 1, wp.getPublishedCount())
				} else if wp != nil {
					assert.Equal(t, 0, wp.getPublishedCount())
				}
			}
		})
	}
}

func TestManager_RemoveSession(t *testing.T) {
	manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
	defer manager.Close()

	_, _, err := manager.CreateSession(context.Background(), "client1", true, 300, 5)
	require.NoError(t, err)

	err = manager.RemoveSession(context.Background(), "client1")
	assert.NoError(t, err)

	_, err = manager.GetSession(context.Background(), "client1")
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
}

func TestManager_TakeoverSession(t *testing.T) {
	tests := []struct {
		name         string
		setupManager func() *Manager
		clientID     string
		expectError  bool
	}{
		{
			name: "takeover existing session with will",
			setupManager: func() *Manager {
				m := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
				s, _, _ := m.CreateSession(context.Background(), "client1", true, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "client/status",
					Payload: []byte("offline"),
				}, 0)
				return m
			},
			clientID:    "client1",
			expectError: false,
		},
		{
			name: "takeover non-existent session",
			setupManager: func() *Manager {
				return NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
			},
			clientID:    "client1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()
			defer manager.Close()

			err := manager.TakeoverSession(context.Background(), tt.clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_GenerateClientID(t *testing.T) {
	tests := []struct {
		name   string
		config ManagerConfig
		prefix string
	}{
		{
			name: "generate with default prefix",
			config: ManagerConfig{
				Store: store.NewMemoryStore[*Session](),
			},
			prefix: "auto-",
		},
		{
			name: "generate with custom prefix",
			config: ManagerConfig{
				Store:            store.NewMemoryStore[*Session](),
				AssignedIDPrefix: "mqtt-",
			},
			prefix: "mqtt-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)
			defer manager.Close()

			clientID, err := manager.GenerateClientID(context.Background())
			assert.NoError(t, err)
			assert.NotEmpty(t, clientID)
			assert.True(t, strings.HasPrefix(clientID, tt.prefix))
			assert.Greater(t, len(clientID), len(tt.prefix))
		})
	}
}

func TestManager_ExpiryChecker(t *testing.T) {
	wp := &mockWillPublisher{}
	manager := NewManager(ManagerConfig{
		Store:               store.NewMemoryStore[*Session](),
		ExpiryCheckInterval: 100 * time.Millisecond,
		WillPublisher:       wp,
	})
	defer manager.Close()

	s1, _, err := manager.CreateSession(context.Background(), "client1", false, 1, 5)
	require.NoError(t, err)
	s1.SetWillMessage(&WillMessage{
		Topic:   "client1/status",
		Payload: []byte("offline"),
	}, 0)

	err = manager.DisconnectSession(context.Background(), "client1", true)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	_, err = manager.GetSession(context.Background(), "client1")
	assert.Error(t, err)
}

func TestManager_DelayedWillPublish(t *testing.T) {
	wp := &mockWillPublisher{}
	manager := NewManager(ManagerConfig{
		Store:               store.NewMemoryStore[*Session](),
		ExpiryCheckInterval: 100 * time.Millisecond,
		WillPublisher:       wp,
	})
	defer manager.Close()

	s, _, err := manager.CreateSession(context.Background(), "client1", false, 10, 5)
	require.NoError(t, err)
	s.SetWillMessage(&WillMessage{
		Topic:   "client1/status",
		Payload: []byte("offline"),
	}, 1)

	err = manager.DisconnectSession(context.Background(), "client1", true)
	require.NoError(t, err)

	assert.Equal(t, 0, wp.getPublishedCount())

	time.Sleep(2 * time.Second)

	assert.Equal(t, 1, wp.getPublishedCount())
	published := wp.getPublished()
	assert.Equal(t, "client1/status", published[0].Topic)
}

func TestManager_GetActiveSessionCount(t *testing.T) {
	manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
	defer manager.Close()

	assert.Equal(t, 0, manager.GetActiveSessionCount())

	_, _, _ = manager.CreateSession(context.Background(), "client1", true, 300, 5)
	_, _, _ = manager.CreateSession(context.Background(), "client2", true, 300, 5)
	assert.Equal(t, 2, manager.GetActiveSessionCount())

	_ = manager.DisconnectSession(context.Background(), "client1", false)
	assert.Equal(t, 1, manager.GetActiveSessionCount())
}

func TestManager_GetAllActiveSessions(t *testing.T) {
	manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
	defer manager.Close()

	_, _, _ = manager.CreateSession(context.Background(), "client1", true, 300, 5)
	_, _, _ = manager.CreateSession(context.Background(), "client2", true, 300, 5)

	sessions := manager.GetAllActiveSessions()
	assert.Len(t, sessions, 2)
	assert.Contains(t, sessions, "client1")
	assert.Contains(t, sessions, "client2")
}

func TestManager_SessionRecovery(t *testing.T) {
	tests := []struct {
		name          string
		setupSession  func(*Manager) *Session
		reconnect     func(*Manager, string) (*Session, bool, error)
		expectPresent bool
	}{
		{
			name: "recover session with subscriptions",
			setupSession: func(m *Manager) *Session {
				s, _, _ := m.CreateSession(context.Background(), "client1", false, 300, 5)
				s.AddSubscription(&Subscription{TopicFilter: "test/topic", QoS: 1})
				_ = m.DisconnectSession(context.Background(), "client1", false)
				return s
			},
			reconnect: func(m *Manager, clientID string) (*Session, bool, error) {
				return m.CreateSession(context.Background(), clientID, false, 300, 5)
			},
			expectPresent: true,
		},
		{
			name: "recover session with pending messages",
			setupSession: func(m *Manager) *Session {
				s, _, _ := m.CreateSession(context.Background(), "client2", false, 300, 5)
				s.AddPendingPublish(&PendingMessage{
					PacketID: 1,
					Topic:    "test/topic",
					Payload:  []byte("test"),
				})
				_ = m.DisconnectSession(context.Background(), "client2", false)
				return s
			},
			reconnect: func(m *Manager, clientID string) (*Session, bool, error) {
				return m.CreateSession(context.Background(), clientID, false, 300, 5)
			},
			expectPresent: true,
		},
		{
			name: "clean start prevents recovery",
			setupSession: func(m *Manager) *Session {
				s, _, _ := m.CreateSession(context.Background(), "client3", false, 300, 5)
				s.AddSubscription(&Subscription{TopicFilter: "test/topic", QoS: 1})
				_ = m.DisconnectSession(context.Background(), "client3", false)
				return s
			},
			reconnect: func(m *Manager, clientID string) (*Session, bool, error) {
				return m.CreateSession(context.Background(), clientID, true, 300, 5)
			},
			expectPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
			defer manager.Close()

			originalSession := tt.setupSession(manager)
			clientID := originalSession.ClientID

			recoveredSession, present, err := tt.reconnect(manager, clientID)
			require.NoError(t, err)
			require.NotNil(t, recoveredSession)
			assert.Equal(t, tt.expectPresent, present)

			if tt.expectPresent {
				assert.Equal(t, originalSession.ClientID, recoveredSession.ClientID)
			}
		})
	}
}

func TestManager_ConcurrentOperations(t *testing.T) {
	manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
	defer manager.Close()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < 50; j++ {
				clientID := "client1"
				_, _, _ = manager.CreateSession(ctx, clientID, false, 300, 5)
				_, _ = manager.GetSession(ctx, clientID)
				_ = manager.DisconnectSession(ctx, clientID, false)
			}
		}(i)
	}

	wg.Wait()
}

func TestManager_Close(t *testing.T) {
	manager := NewManager(ManagerConfig{Store: store.NewMemoryStore[*Session]()})
	_, _, _ = manager.CreateSession(context.Background(), "client1", true, 300, 5)

	err := manager.Close()
	assert.NoError(t, err)
}
