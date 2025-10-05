//go:build integration

package session

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

func setupRedisStore(t *testing.T) *RedisStore {
	store, err := NewRedisStore(RedisStoreConfig{
		Addr: getRedisAddr(),
		DB:   15, // Use DB 15 for testing
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	require.NotNil(t, store)

	ctx := context.Background()
	_ = store.Flush(ctx)

	return store
}

func TestNewRedisStore(t *testing.T) {
	tests := []struct {
		name        string
		config      RedisStoreConfig
		expectError bool
	}{
		{
			name: "create new redis store",
			config: RedisStoreConfig{
				Addr: getRedisAddr(),
				DB:   15,
			},
			expectError: false,
		},
		{
			name: "create store with TTL",
			config: RedisStoreConfig{
				Addr: getRedisAddr(),
				DB:   15,
				TTL:  1 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "create store with invalid address",
			config: RedisStoreConfig{
				Addr: "invalid:99999",
				DB:   0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewRedisStore(tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				if err != nil {
					t.Skipf("Redis not available: %v", err)
				}
				assert.NoError(t, err)
				require.NotNil(t, store)
				assert.NoError(t, store.Close())
			}
		})
	}
}

func TestRedisStore_Save(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) *RedisStore
		session     *Session
		expectError bool
	}{
		{
			name: "save new session",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
			},
			session:     New("client1", true, 300, 5),
			expectError: false,
		},
		{
			name: "save session with subscriptions",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
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
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
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
			name: "save session with pending messages",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
			},
			session: func() *Session {
				s := New("client4", false, 300, 5)
				s.AddPendingPublish(&PendingMessage{
					PacketID:  1,
					Topic:     "test/topic",
					Payload:   []byte("data"),
					QoS:       1,
					Timestamp: time.Now(),
				})
				s.AddPendingPubrel(2)
				s.AddPendingPubcomp(3)
				return s
			}(),
			expectError: false,
		},
		{
			name: "update existing session",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
				s := New("client5", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store
			},
			session:     New("client5", false, 600, 5),
			expectError: false,
		},
		{
			name: "save to closed store",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
				_ = store.Close()
				return store
			},
			session:     New("client6", true, 300, 5),
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

func TestRedisStore_Load(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*RedisStore, string)
		expectError error
	}{
		{
			name: "load existing session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectError: nil,
		},
		{
			name: "load non-existent session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				return store, "nonexistent"
			},
			expectError: ErrSessionNotFound,
		},
		{
			name: "load from closed store",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
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

func TestRedisStore_Delete(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*RedisStore, string)
		expectError bool
	}{
		{
			name: "delete existing session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectError: false,
		},
		{
			name: "delete non-existent session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				return store, "nonexistent"
			},
			expectError: false,
		},
		{
			name: "delete from closed store",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
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

func TestRedisStore_Exists(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) (*RedisStore, string)
		expectExist bool
		expectError bool
	}{
		{
			name: "check existing session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				s := New("client1", true, 300, 5)
				_ = store.Save(context.Background(), s)
				return store, "client1"
			},
			expectExist: true,
			expectError: false,
		},
		{
			name: "check non-existent session",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
				return store, "nonexistent"
			},
			expectExist: false,
			expectError: false,
		},
		{
			name: "check in closed store",
			setupStore: func(t *testing.T) (*RedisStore, string) {
				store := setupRedisStore(t)
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

func TestRedisStore_List(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *RedisStore
		expectedCount int
		expectError   bool
	}{
		{
			name: "list empty store",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "list store with sessions",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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

func TestRedisStore_Count(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *RedisStore
		expectedCount int64
		expectError   bool
	}{
		{
			name: "count empty store",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "count store with sessions",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
				_ = store.Save(context.Background(), New("client1", true, 300, 5))
				_ = store.Save(context.Background(), New("client2", true, 300, 5))
				return store
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "count closed store",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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

func TestRedisStore_CountByState(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(*testing.T) *RedisStore
		state         State
		expectedCount int64
	}{
		{
			name: "count active sessions",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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

func TestRedisStore_SessionPersistence(t *testing.T) {
	store := setupRedisStore(t)
	defer store.Close()

	ctx := context.Background()

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
	session.AddPendingPubrel(2)
	session.AddPendingPubcomp(3)

	err := store.Save(ctx, session)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "client1")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, session.ClientID, loaded.ClientID)
	assert.Equal(t, session.CleanStart, loaded.CleanStart)
	assert.Equal(t, session.GetExpiryInterval(), loaded.GetExpiryInterval())
	assert.Len(t, loaded.Subscriptions, 1)
	assert.NotNil(t, loaded.WillMessage)
	assert.Equal(t, "will/topic", loaded.WillMessage.Topic)
	assert.Len(t, loaded.PendingPublish, 1)
	assert.True(t, loaded.HasPendingPubrel(2))
	assert.True(t, loaded.HasPendingPubcomp(3))
}

func TestRedisStore_ConcurrentAccess(t *testing.T) {
	store := setupRedisStore(t)
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
			}
		}(i)
	}

	wg.Wait()
}

func TestRedisStore_ContextCancellation(t *testing.T) {
	store := setupRedisStore(t)
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

func TestRedisStore_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*testing.T) *RedisStore
		expectError bool
	}{
		{
			name: "close open store",
			setupStore: func(t *testing.T) *RedisStore {
				return setupRedisStore(t)
			},
			expectError: false,
		},
		{
			name: "close already closed store",
			setupStore: func(t *testing.T) *RedisStore {
				store := setupRedisStore(t)
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

func TestRedisStore_Flush(t *testing.T) {
	store := setupRedisStore(t)
	defer store.Close()

	ctx := context.Background()

	_ = store.Save(ctx, New("client1", true, 300, 5))
	_ = store.Save(ctx, New("client2", true, 300, 5))
	_ = store.Save(ctx, New("client3", true, 300, 5))

	count, _ := store.Count(ctx)
	assert.Equal(t, int64(3), count)

	err := store.Flush(ctx)
	require.NoError(t, err)

	count, _ = store.Count(ctx)
	assert.Equal(t, int64(0), count)
}

func TestRedisStore_TTL(t *testing.T) {
	store, err := NewRedisStore(RedisStoreConfig{
		Addr: getRedisAddr(),
		DB:   15,
		TTL:  1 * time.Second,
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.Flush(ctx)

	session := New("client1", true, 300, 5)
	err = store.Save(ctx, session)
	require.NoError(t, err)

	exists, _ := store.Exists(ctx, "client1")
	assert.True(t, exists)

	time.Sleep(2 * time.Second)

	exists, _ = store.Exists(ctx, "client1")
	assert.False(t, exists)
}

func TestRedisStore_MultipleUpdates(t *testing.T) {
	store := setupRedisStore(t)
	defer store.Close()

	ctx := context.Background()

	session := New("client1", false, 300, 5)
	_ = store.Save(ctx, session)

	for i := 0; i < 10; i++ {
		loaded, err := store.Load(ctx, "client1")
		require.NoError(t, err)

		loaded.AddSubscription(&Subscription{
			TopicFilter: "test/topic",
			QoS:         1,
		})

		_ = store.Save(ctx, loaded)
	}

	final, err := store.Load(ctx, "client1")
	require.NoError(t, err)
	assert.Len(t, final.Subscriptions, 1)
}

func BenchmarkRedisStore_Save(b *testing.B) {
	store, err := NewRedisStore(RedisStoreConfig{
		Addr: getRedisAddr(),
		DB:   15,
	})
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.Flush(ctx)
	session := New("client1", true, 300, 5)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Save(ctx, session)
	}
}

func BenchmarkRedisStore_Load(b *testing.B) {
	store, err := NewRedisStore(RedisStoreConfig{
		Addr: getRedisAddr(),
		DB:   15,
	})
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.Flush(ctx)
	_ = store.Save(ctx, New("client1", true, 300, 5))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = store.Load(ctx, "client1")
	}
}

func BenchmarkRedisStore_SaveLoad(b *testing.B) {
	store, err := NewRedisStore(RedisStoreConfig{
		Addr: getRedisAddr(),
		DB:   15,
	})
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.Flush(ctx)
	session := New("client1", true, 300, 5)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Save(ctx, session)
		_, _ = store.Load(ctx, "client1")
	}
}
