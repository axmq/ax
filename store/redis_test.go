//go:build integration

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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

func setupRedis(t *testing.T) *redis.Options {
	opts := &redis.Options{
		Addr: getRedisAddr(),
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", opts.Addr, err)
	}

	client.Close()
	return opts
}

func cleanupRedis(store *RedisStore[testData]) {
	if store == nil {
		return
	}
	ctx := context.Background()
	keys, _ := store.List(ctx)
	for _, key := range keys {
		store.Delete(ctx, key)
	}
}

func TestNewRedisStore(t *testing.T) {
	tests := []struct {
		name    string
		config  func(*testing.T) RedisStoreConfig
		wantErr bool
	}{
		{
			name: "create with default options",
			config: func(t *testing.T) RedisStoreConfig {
				opts := setupRedis(t)
				return RedisStoreConfig{
					Prefix:  "test:",
					Options: opts,
				}
			},
			wantErr: false,
		},
		{
			name: "create with TTL",
			config: func(t *testing.T) RedisStoreConfig {
				opts := setupRedis(t)
				return RedisStoreConfig{
					Prefix:  "test:",
					TTL:     time.Minute,
					Options: opts,
				}
			},
			wantErr: false,
		},
		{
			name: "create with empty prefix",
			config: func(t *testing.T) RedisStoreConfig {
				opts := setupRedis(t)
				return RedisStoreConfig{
					Options: opts,
				}
			},
			wantErr: false,
		},
		{
			name: "create with manual addr",
			config: func(t *testing.T) RedisStoreConfig {
				addr := getRedisAddr()
				return RedisStoreConfig{
					Addr:   addr,
					Prefix: "test:",
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config(t)
			store, err := NewRedisStore[testData](config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				if store != nil {
					cleanupRedis(store)
					store.Close()
				}
			}
		})
	}
}

func TestNewRedisStore_ConnectionFailure(t *testing.T) {
	config := RedisStoreConfig{
		Addr:   "localhost:9999",
		Prefix: "test:",
	}

	_, err := NewRedisStore[testData](config)
	assert.Error(t, err)
}

func TestRedisStore_Save(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   testData
		wantErr bool
	}{
		{
			name:    "save new value",
			key:     "user1",
			value:   testData{ID: "1", Name: "Alice", Age: 30},
			wantErr: false,
		},
		{
			name:    "overwrite existing value",
			key:     "user1",
			value:   testData{ID: "1", Name: "Alice Updated", Age: 31},
			wantErr: false,
		},
		{
			name:    "save with empty key",
			key:     "",
			value:   testData{ID: "2", Name: "Bob", Age: 25},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			err = store.Save(context.Background(), tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisStore_SaveWithTTL(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		TTL:     1 * time.Second,
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()
	key := "ttl_key"
	value := testData{ID: "1", Name: "Alice", Age: 30}

	err = store.Save(ctx, key, value)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)

	time.Sleep(2 * time.Second)

	exists, err = store.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisStore_SaveWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.Error(t, err)
}

func TestRedisStore_SaveAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	err = store.Save(context.Background(), "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_Load(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]testData
		key       string
		want      testData
		wantErr   error
	}{
		{
			name:      "load existing value",
			setupData: map[string]testData{"user1": {ID: "1", Name: "Alice", Age: 30}},
			key:       "user1",
			want:      testData{ID: "1", Name: "Alice", Age: 30},
			wantErr:   nil,
		},
		{
			name:      "load non-existing value",
			setupData: map[string]testData{},
			key:       "user999",
			want:      testData{},
			wantErr:   ErrNotFound,
		},
		{
			name:      "load with empty key",
			setupData: map[string]testData{"": {ID: "0", Name: "Empty", Age: 0}},
			key:       "",
			want:      testData{ID: "0", Name: "Empty", Age: 0},
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			got, err := store.Load(context.Background(), tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRedisStore_LoadWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Load(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisStore_LoadAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	_, err = store.Load(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_Delete(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]testData
		key       string
		wantErr   bool
	}{
		{
			name:      "delete existing value",
			setupData: map[string]testData{"user1": {ID: "1", Name: "Alice", Age: 30}},
			key:       "user1",
			wantErr:   false,
		},
		{
			name:      "delete non-existing value",
			setupData: map[string]testData{},
			key:       "user999",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			err = store.Delete(context.Background(), tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				exists, _ := store.Exists(context.Background(), tt.key)
				assert.False(t, exists)
			}
		})
	}
}

func TestRedisStore_DeleteWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = store.Delete(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisStore_DeleteAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	err = store.Delete(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_Exists(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]testData
		key       string
		want      bool
	}{
		{
			name:      "existing key",
			setupData: map[string]testData{"user1": {ID: "1", Name: "Alice", Age: 30}},
			key:       "user1",
			want:      true,
		},
		{
			name:      "non-existing key",
			setupData: map[string]testData{},
			key:       "user999",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			got, err := store.Exists(context.Background(), tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisStore_ExistsWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Exists(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisStore_ExistsAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	_, err = store.Exists(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_List(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]testData
		wantKeys  []string
	}{
		{
			name: "list multiple keys",
			setupData: map[string]testData{
				"user1": {ID: "1", Name: "Alice", Age: 30},
				"user2": {ID: "2", Name: "Bob", Age: 25},
				"user3": {ID: "3", Name: "Charlie", Age: 35},
			},
			wantKeys: []string{"user1", "user2", "user3"},
		},
		{
			name:      "list empty store",
			setupData: map[string]testData{},
			wantKeys:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			keys, err := store.List(context.Background())
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.wantKeys, keys)
		})
	}
}

func TestRedisStore_ListWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.List(ctx)
	assert.Error(t, err)
}

func TestRedisStore_ListAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	_, err = store.List(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_Count(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]testData
		want      int64
	}{
		{
			name: "count multiple items",
			setupData: map[string]testData{
				"user1": {ID: "1", Name: "Alice", Age: 30},
				"user2": {ID: "2", Name: "Bob", Age: 25},
				"user3": {ID: "3", Name: "Charlie", Age: 35},
			},
			want: 3,
		},
		{
			name:      "count empty store",
			setupData: map[string]testData{},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  "test:",
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			count, err := store.Count(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.want, count)
		})
	}
}

func TestRedisStore_CountWithCanceledContext(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Count(ctx)
	assert.Error(t, err)
}

func TestRedisStore_CountAfterClose(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	cleanupRedis(store)
	store.Close()

	_, err = store.Count(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_Close(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)

	cleanupRedis(store)
	err = store.Close()
	assert.NoError(t, err)

	err = store.Close()
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestRedisStore_MakeKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "standard prefix and key",
			prefix: "test:",
			key:    "user1",
			want:   "test:user1",
		},
		{
			name:   "empty prefix uses default",
			prefix: "",
			key:    "user1",
			want:   "data:user1",
		},
		{
			name:   "empty key",
			prefix: "test:",
			key:    "",
			want:   "test:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := setupRedis(t)
			store, err := NewRedisStore[testData](RedisStoreConfig{
				Prefix:  tt.prefix,
				Options: opts,
			})
			require.NoError(t, err)
			defer func() {
				cleanupRedis(store)
				store.Close()
			}()

			got := store.makeKey(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisStore_ConcurrentOperations(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()
	iterations := 100

	done := make(chan bool)
	go func() {
		for i := 0; i < iterations; i++ {
			store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: i})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			store.Load(ctx, "key1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			store.Exists(ctx, "key1")
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestRedisStore_IndexMaintenance(t *testing.T) {
	opts := setupRedis(t)
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "test:",
		Options: opts,
	})
	require.NoError(t, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()

	err = store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	require.NoError(t, err)

	keys, err := store.List(ctx)
	require.NoError(t, err)
	assert.Contains(t, keys, "key1")

	err = store.Delete(ctx, "key1")
	require.NoError(t, err)

	keys, err = store.List(ctx)
	require.NoError(t, err)
	assert.NotContains(t, keys, "key1")
}

func BenchmarkRedisStore_Save(b *testing.B) {
	opts := &redis.Options{Addr: getRedisAddr()}
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "bench:",
		Options: opts,
	})
	require.NoError(b, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()
	data := testData{ID: "1", Name: "Alice", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(ctx, "key", data)
	}
}

func BenchmarkRedisStore_Load(b *testing.B) {
	opts := &redis.Options{Addr: getRedisAddr()}
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "bench:",
		Options: opts,
	})
	require.NoError(b, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()
	store.Save(ctx, "key", testData{ID: "1", Name: "Alice", Age: 30})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Load(ctx, "key")
	}
}

func BenchmarkRedisStore_Delete(b *testing.B) {
	opts := &redis.Options{Addr: getRedisAddr()}
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "bench:",
		Options: opts,
	})
	require.NoError(b, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store.Save(ctx, "key", testData{ID: "1", Name: "Alice", Age: 30})
		b.StartTimer()
		store.Delete(ctx, "key")
	}
}

func BenchmarkRedisStore_List(b *testing.B) {
	opts := &redis.Options{Addr: getRedisAddr()}
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "bench:",
		Options: opts,
	})
	require.NoError(b, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		store.Save(ctx, string(rune(i)), testData{ID: string(rune(i)), Name: "User", Age: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List(ctx)
	}
}

func BenchmarkRedisStore_Count(b *testing.B) {
	opts := &redis.Options{Addr: getRedisAddr()}
	store, err := NewRedisStore[testData](RedisStoreConfig{
		Prefix:  "bench:",
		Options: opts,
	})
	require.NoError(b, err)
	defer func() {
		cleanupRedis(store)
		store.Close()
	}()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		store.Save(ctx, string(rune(i)), testData{ID: string(rune(i)), Name: "User", Age: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Count(ctx)
	}
}
