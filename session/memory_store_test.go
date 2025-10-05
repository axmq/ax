package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.sessions)
	assert.False(t, store.closed)
}

func TestMemoryStore_Save(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func() *MemoryStore
		session     *Session
		expectError bool
	}{
		{
			name: "save new session",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			session:     New("client1", true, 300, 5),
			expectError: false,
		},
		{
			name: "update existing session",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				return store
			},
			session:     New("client1", false, 600, 5),
			expectError: false,
		},
		{
			name: "save to closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			session:     New("client1", true, 300, 5),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			err := store.Save(context.Background(), tt.session)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrStoreClosed, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryStore_Load(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func() *MemoryStore
		clientID    string
		expectError error
	}{
		{
			name: "load existing session",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				return store
			},
			clientID:    "client1",
			expectError: nil,
		},
		{
			name: "load non-existent session",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			clientID:    "client1",
			expectError: ErrSessionNotFound,
		},
		{
			name: "load from closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			clientID:    "client1",
			expectError: ErrStoreClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			session, err := store.Load(context.Background(), tt.clientID)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.clientID, session.ClientID)
			}
		})
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func() *MemoryStore
		clientID    string
		expectError bool
	}{
		{
			name: "delete existing session",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				return store
			},
			clientID:    "client1",
			expectError: false,
		},
		{
			name: "delete non-existent session",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			clientID:    "client1",
			expectError: false,
		},
		{
			name: "delete from closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			clientID:    "client1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			err := store.Delete(context.Background(), tt.clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func() *MemoryStore
		clientID    string
		expectExist bool
		expectError bool
	}{
		{
			name: "check existing session",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				return store
			},
			clientID:    "client1",
			expectExist: true,
			expectError: false,
		},
		{
			name: "check non-existent session",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			clientID:    "client1",
			expectExist: false,
			expectError: false,
		},
		{
			name: "check in closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			clientID:    "client1",
			expectExist: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			exists, err := store.Exists(context.Background(), tt.clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExist, exists)
			}
		})
	}
}

func TestMemoryStore_List(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func() *MemoryStore
		expectedCount int
		expectError   bool
	}{
		{
			name: "list empty store",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "list store with sessions",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				_ = store.Save(context.Background(), New("client2", true, 300, 5))
				_ = store.Save(context.Background(), New("client3", true, 300, 5))
				return store
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "list closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			clientIDs, err := store.List(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, clientIDs, tt.expectedCount)
			}
		})
	}
}

func TestMemoryStore_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func() *MemoryStore
		expectError bool
	}{
		{
			name: "close open store",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			expectError: false,
		},
		{
			name: "close already closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			err := store.Close()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrStoreClosed, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, store.closed)
				assert.Nil(t, store.sessions)
			}
		})
	}
}

func TestMemoryStore_Count(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func() *MemoryStore
		expectedCount int64
		expectError   bool
	}{
		{
			name: "count empty store",
			setupStore: func() *MemoryStore {
				return NewMemoryStore()
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "count store with sessions",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				_ = store.Save(context.Background(), New("client2", true, 300, 5))
				return store
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "count closed store",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				_ = store.Close()
				return store
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			count, err := store.Count(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

func TestMemoryStore_CountByState(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func() *MemoryStore
		state         State
		expectedCount int64
	}{
		{
			name: "count active sessions",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				s1 := New("client1", true, 300, 5)
				s1.SetActive()
				s2 := New("client2", true, 300, 5)
				s2.SetActive()
				s3 := New("client3", true, 300, 5)
				s3.SetDisconnected()
				_ = store.Save(context.Background(), s1)
				_ = store.Save(context.Background(), s2)
				_ = store.Save(context.Background(), s3)
				return store
			},
			state:         StateActive,
			expectedCount: 2,
		},
		{
			name: "count disconnected sessions",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				s1 := New("client1", true, 300, 5)
				s1.SetActive()
				s2 := New("client2", true, 300, 5)
				s2.SetDisconnected()
				_ = store.Save(context.Background(), s1)
				_ = store.Save(context.Background(), s2)
				return store
			},
			state:         StateDisconnected,
			expectedCount: 1,
		},
		{
			name: "count with no matching state",
			setupStore: func() *MemoryStore {
				store := NewMemoryStore()
				s1 := New("client1", true, 300, 5)
				s1.SetActive()
				_ = store.Save(context.Background(), s1)
				return store
			},
			state:         StateExpired,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			count, err := store.CountByState(context.Background(), tt.state)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestMemoryStore_ContextCancellation(t *testing.T) {
	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Save(ctx, New("client1", true, 300, 5))
	assert.Error(t, err)

	_, err = store.Load(ctx, "client1")
	assert.Error(t, err)

	err = store.Delete(ctx, "client1")
	assert.Error(t, err)

	_, err = store.Exists(ctx, "client1")
	assert.Error(t, err)

	_, err = store.List(ctx)
	assert.Error(t, err)

	_, err = store.Count(ctx)
	assert.Error(t, err)

	_, err = store.CountByState(ctx, StateActive)
	assert.Error(t, err)
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := context.Background()
			for j := 0; j < 100; j++ {
				clientID := "client"
				session := New(clientID, true, 300, 5)
				_ = store.Save(ctx, session)
				_, _ = store.Load(ctx, clientID)
				_, _ = store.Exists(ctx, clientID)
				_, _ = store.Count(ctx)
				_ = store.Delete(ctx, clientID)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMemoryStore_SessionPersistence(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session := New("client1", true, 300, 5)
	session.SetActive()
	session.AddSubscription(&Subscription{
		TopicFilter: "test/topic",
		QoS:         1,
	})
	session.SetWillMessage(&WillMessage{
		Topic:   "will/topic",
		Payload: []byte("offline"),
		QoS:     1,
		Retain:  true,
	}, 0)

	err := store.Save(ctx, session)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "client1")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, session.ClientID, loaded.ClientID)
	assert.Equal(t, session.State, loaded.State)
	assert.Len(t, loaded.Subscriptions, 1)
	assert.NotNil(t, loaded.WillMessage)
	assert.Equal(t, "will/topic", loaded.WillMessage.Topic)
}

func TestMemoryStore_Timeout(t *testing.T) {
	store := NewMemoryStore()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	err := store.Save(ctx, New("client1", true, 300, 5))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
