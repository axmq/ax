package session

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPebbleStore(t *testing.T) (*PebbleStore, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_pebble")

	store, err := NewPebbleStore(PebbleStoreConfig{
		Path: dbPath,
	})
	require.NoError(t, err)
	require.NotNil(t, store)

	return store, dbPath
}

func TestNewPebbleStore(t *testing.T) {
	tests := []struct {
		name        string
		config      PebbleStoreConfig
		expectError bool
	}{
		{
			name: "create new pebble store",
			config: PebbleStoreConfig{
				Path: filepath.Join(t.TempDir(), "test1"),
			},
			expectError: false,
		},
		{
			name: "create store with existing path",
			config: PebbleStoreConfig{
				Path: filepath.Join(t.TempDir(), "test2"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewPebbleStore(tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, store)
				assert.NoError(t, store.Close())
			}
		})
	}
}

func TestPebbleStore_Save(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) *PebbleStore
		session     *Session
		expectError bool
	}{
		{
			name: "save new session",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			session:     New("client1", true, 300, 5),
			expectError: false,
		},
		{
			name: "save session with subscriptions",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			session: func() *Session {
				s := New("client2", false, 600, 5)
				s.AddSubscription(&Subscription{
					TopicFilter: "test/topic",
					QoS:         1,
				})
				return s
			}(),
			expectError: false,
		},
		{
			name: "save session with will message",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			session: func() *Session {
				s := New("client3", false, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "client/status",
					Payload: []byte("offline"),
					QoS:     1,
					Retain:  true,
				}, 60)
				return s
			}(),
			expectError: false,
		},
		{
			name: "update existing session",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				s := New("client4", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store
			},
			session:     New("client4", false, 600, 5),
			expectError: false,
		},
		{
			name: "save to closed store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store
			},
			session:     New("client5", true, 300, 5),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)
			defer store.Close()

			err := store.Save(context.Background(), tt.session)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPebbleStore_Load(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*PebbleStore, string)
		clientID    string
		expectError error
	}{
		{
			name: "load existing session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectError: nil,
		},
		{
			name: "load non-existent session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				return store, "nonexistent"
			},
			expectError: ErrSessionNotFound,
		},
		{
			name: "load from closed store",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store, "client1"
			},
			expectError: ErrStoreClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, clientID := tt.setupStore(t)
			defer store.Close()

			session, err := store.Load(context.Background(), clientID)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, clientID, session.ClientID)
			}
		})
	}
}

func TestPebbleStore_Delete(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*PebbleStore, string)
		expectError bool
	}{
		{
			name: "delete existing session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectError: false,
		},
		{
			name: "delete non-existent session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				return store, "nonexistent"
			},
			expectError: false,
		},
		{
			name: "delete from closed store",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store, "client1"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, clientID := tt.setupStore(t)
			defer store.Close()

			err := store.Delete(context.Background(), clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPebbleStore_Exists(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*PebbleStore, string)
		expectExist bool
		expectError bool
	}{
		{
			name: "check existing session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectExist: true,
			expectError: false,
		},
		{
			name: "check non-existent session",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				return store, "nonexistent"
			},
			expectExist: false,
			expectError: false,
		},
		{
			name: "check in closed store",
			setupStore: func(t *testing.T) (*PebbleStore, string) {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store, "client1"
			},
			expectExist: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, clientID := tt.setupStore(t)
			defer store.Close()

			exists, err := store.Exists(context.Background(), clientID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExist, exists)
			}
		})
	}
}

func TestPebbleStore_List(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *PebbleStore
		expectedCount int
		expectError   bool
	}{
		{
			name: "list empty store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "list store with sessions",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
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
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)
			defer store.Close()

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

func TestPebbleStore_Count(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *PebbleStore
		expectedCount int64
		expectError   bool
	}{
		{
			name: "count empty store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "count store with sessions",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				_ = store.Save(context.Background(), New("client2", true, 300, 5))
				return store
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "count closed store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)
			defer store.Close()

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

func TestPebbleStore_CountByState(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *PebbleStore
		state         State
		expectedCount int64
	}{
		{
			name: "count active sessions",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
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
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)
			defer store.Close()

			count, err := store.CountByState(context.Background(), tt.state)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestPebbleStore_SessionPersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "persistence_test")

	store1, err := NewPebbleStore(PebbleStoreConfig{Path: dbPath})
	require.NoError(t, err)

	session := New("client1", false, 600, 5)
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
	}, 30)
	session.AddPendingPublish(&PendingMessage{
		PacketID:  1,
		Topic:     "pending/topic",
		Payload:   []byte("data"),
		QoS:       1,
		Timestamp: time.Now(),
	})

	err = store1.Save(context.Background(), session)
	require.NoError(t, err)

	err = store1.Close()
	require.NoError(t, err)

	store2, err := NewPebbleStore(PebbleStoreConfig{Path: dbPath})
	require.NoError(t, err)
	defer store2.Close()

	loaded, err := store2.Load(context.Background(), "client1")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, session.ClientID, loaded.ClientID)
	assert.Equal(t, session.CleanStart, loaded.CleanStart)
	assert.Equal(t, session.GetExpiryInterval(), loaded.GetExpiryInterval())
	assert.Len(t, loaded.Subscriptions, 1)
	assert.NotNil(t, loaded.WillMessage)
	assert.Equal(t, "will/topic", loaded.WillMessage.Topic)
	assert.Len(t, loaded.PendingPublish, 1)
}

func TestPebbleStore_ConcurrentAccess(t *testing.T) {
	store, _ := setupPebbleStore(t)
	defer store.Close()

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				session := New("client1", false, 300, 5)
				_ = store.Save(ctx, session)
				_, _ = store.Load(ctx, "client1")
				_, _ = store.Exists(ctx, "client1")
				_ = store.Delete(ctx, "client1")
			}
		}(i)
	}

	wg.Wait()
}

func TestPebbleStore_ContextCancellation(t *testing.T) {
	store, _ := setupPebbleStore(t)
	defer store.Close()

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
}

func TestPebbleStore_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) *PebbleStore
		expectError bool
	}{
		{
			name: "close open store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				return store
			},
			expectError: false,
		},
		{
			name: "close already closed store",
			setupStore: func(t *testing.T) *PebbleStore {
				store, _ := setupPebbleStore(t)
				_ = store.Close()
				return store
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)

			err := store.Close()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrStoreClosed, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPebbleStore_Reopen(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "reopen_test")

	store1, err := NewPebbleStore(PebbleStoreConfig{Path: dbPath})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_ = store1.Save(context.Background(), New("client1", true, 300, 5))
	}

	count1, _ := store1.Count(context.Background())
	assert.Equal(t, int64(1), count1)

	require.NoError(t, store1.Close())

	store2, err := NewPebbleStore(PebbleStoreConfig{Path: dbPath})
	require.NoError(t, err)
	defer store2.Close()

	count2, _ := store2.Count(context.Background())
	assert.Equal(t, int64(1), count2)

	exists, _ := store2.Exists(context.Background(), "client1")
	assert.True(t, exists)
}

func TestPebbleStore_CleanupOnDelete(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "cleanup_test")

	store, err := NewPebbleStore(PebbleStoreConfig{Path: dbPath})
	require.NoError(t, err)

	_ = store.Save(context.Background(), New("client1", true, 300, 5))
	_ = store.Save(context.Background(), New("client2", true, 300, 5))

	count1, _ := store.Count(context.Background())
	assert.Equal(t, int64(2), count1)

	_ = store.Delete(context.Background(), "client1")

	count2, _ := store.Count(context.Background())
	assert.Equal(t, int64(1), count2)

	exists, _ := store.Exists(context.Background(), "client1")
	assert.False(t, exists)

	exists, _ = store.Exists(context.Background(), "client2")
	assert.True(t, exists)

	require.NoError(t, store.Close())
}

func BenchmarkPebbleStore_Save(b *testing.B) {
	tempDir := b.TempDir()
	store, _ := NewPebbleStore(PebbleStoreConfig{
		Path: filepath.Join(tempDir, "bench_save"),
	})
	defer store.Close()

	ctx := context.Background()
	session := New("client1", true, 300, 5)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Save(ctx, session)
	}
}

func BenchmarkPebbleStore_Load(b *testing.B) {
	tempDir := b.TempDir()
	store, _ := NewPebbleStore(PebbleStoreConfig{
		Path: filepath.Join(tempDir, "bench_load"),
	})
	defer store.Close()

	ctx := context.Background()
	_ = store.Save(ctx, New("client1", true, 300, 5))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = store.Load(ctx, "client1")
	}
}

func BenchmarkPebbleStore_SaveLoad(b *testing.B) {
	tempDir := b.TempDir()
	store, _ := NewPebbleStore(PebbleStoreConfig{
		Path: filepath.Join(tempDir, "bench_saveload"),
	})
	defer store.Close()

	ctx := context.Background()
	session := New("client1", true, 300, 5)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Save(ctx, session)
		_, _ = store.Load(ctx, "client1")
	}
}
