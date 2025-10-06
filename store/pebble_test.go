package store

import (
	"context"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPebbleStore(t *testing.T) {
	tests := []struct {
		name    string
		config  PebbleStoreConfig
		wantErr bool
	}{
		{
			name: "create with default options",
			config: PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			},
			wantErr: false,
		},
		{
			name: "create with custom options",
			config: PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "custom:",
				Opts:   &pebble.Options{ErrorIfExists: false},
			},
			wantErr: false,
		},
		{
			name: "create with empty prefix",
			config: PebbleStoreConfig{
				Path: t.TempDir(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewPebbleStore[testData](tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				if store != nil {
					store.Close()
				}
			}
		})
	}
}

func TestNewPebbleStore_InvalidPath(t *testing.T) {
	config := PebbleStoreConfig{
		Path:   "/invalid/path/that/does/not/exist/and/cannot/be/created",
		Prefix: "test:",
	}

	_, err := NewPebbleStore[testData](config)
	assert.Error(t, err)
}

func TestNewPebbleStore_ErrorIfExists(t *testing.T) {
	tmpDir := t.TempDir()

	store1, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(t, err)
	store1.Close()

	_, err = NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
		Opts:   &pebble.Options{ErrorIfExists: true},
	})
	assert.Error(t, err)
}

func TestPebbleStore_Save(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

			err = store.Save(context.Background(), tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPebbleStore_SaveInvalidValue(t *testing.T) {
	store, err := NewPebbleStore[chan int](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ch := make(chan int)
	err = store.Save(context.Background(), "key1", ch)
	assert.Error(t, err)
}

func TestPebbleStore_SaveWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.Error(t, err)
}

func TestPebbleStore_SaveAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	err = store.Save(context.Background(), "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_Load(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

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

func TestPebbleStore_LoadCorruptedData(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	fullKey := store.makeKey("corrupt")
	err = store.db.Set(fullKey, []byte("invalid cbor data"), pebble.Sync)
	require.NoError(t, err)

	_, err = store.Load(context.Background(), "corrupt")
	assert.Error(t, err)
}

func TestPebbleStore_LoadWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Load(ctx, "key1")
	assert.Error(t, err)
}

func TestPebbleStore_LoadAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	_, err = store.Load(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_Delete(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

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

func TestPebbleStore_DeleteWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = store.Delete(ctx, "key1")
	assert.Error(t, err)
}

func TestPebbleStore_DeleteAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	err = store.Delete(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_Exists(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			got, err := store.Exists(context.Background(), tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPebbleStore_ExistsWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Exists(ctx, "key1")
	assert.Error(t, err)
}

func TestPebbleStore_ExistsAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	_, err = store.Exists(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_List(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			keys, err := store.List(context.Background())
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.wantKeys, keys)
		})
	}
}

func TestPebbleStore_ListIteratorError(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	require.NoError(t, err)

	keys, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Contains(t, keys, "key1")

	store.Close()
}

func TestPebbleStore_ListWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.List(ctx)
	assert.Error(t, err)
}

func TestPebbleStore_ListAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	_, err = store.List(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_Count(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: "test:",
			})
			require.NoError(t, err)
			defer store.Close()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			count, err := store.Count(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.want, count)
		})
	}
}

func TestPebbleStore_CountIteratorError(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	require.NoError(t, err)

	count, err := store.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	store.Close()
}

func TestPebbleStore_CountWithCanceledContext(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Count(ctx)
	assert.Error(t, err)
}

func TestPebbleStore_CountAfterClose(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	store.Close()

	_, err = store.Count(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_Close(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)

	err = store.Close()
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestPebbleStore_MakeKey(t *testing.T) {
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
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: tt.prefix,
			})
			require.NoError(t, err)
			defer store.Close()

			got := store.makeKey(tt.key)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestPebbleStore_MakeKeyWithDifferentSizes(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "long prefix and key",
			prefix: "very:long:prefix:with:colons:",
			key:    "very_long_key_name_with_underscores",
			want:   "very:long:prefix:with:colons:very_long_key_name_with_underscores",
		},
		{
			name:   "unicode characters",
			prefix: "测试:",
			key:    "键",
			want:   "测试:键",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewPebbleStore[testData](PebbleStoreConfig{
				Path:   t.TempDir(),
				Prefix: tt.prefix,
			})
			require.NoError(t, err)
			defer store.Close()

			got := store.makeKey(tt.key)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestPebbleStore_SaveAndLoadWithSpecialCharacters(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	tests := []struct {
		key   string
		value testData
	}{
		{
			key:   "key/with/slashes",
			value: testData{ID: "1", Name: "Alice", Age: 30},
		},
		{
			key:   "key:with:colons",
			value: testData{ID: "2", Name: "Bob", Age: 25},
		},
		{
			key:   "key with spaces",
			value: testData{ID: "3", Name: "Charlie", Age: 35},
		},
		{
			key:   "key\nwith\nnewlines",
			value: testData{ID: "4", Name: "David", Age: 40},
		},
	}

	for _, tt := range tests {
		err = store.Save(ctx, tt.key, tt.value)
		require.NoError(t, err)

		loaded, err := store.Load(ctx, tt.key)
		require.NoError(t, err)
		assert.Equal(t, tt.value, loaded)
	}
}

func TestPebbleStore_LargeDataset(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	count := 1000

	for i := 0; i < count; i++ {
		key := string(rune(i))
		value := testData{ID: key, Name: "User", Age: i}
		err = store.Save(ctx, key, value)
		require.NoError(t, err)
	}

	actualCount, err := store.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(count), actualCount)

	keys, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, count, len(keys))
}

func TestPebbleStore_DeleteAndRestore(t *testing.T) {
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   t.TempDir(),
		Prefix: "test:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	key := "test_key"
	value := testData{ID: "1", Name: "Alice", Age: 30}

	err = store.Save(ctx, key, value)
	require.NoError(t, err)

	err = store.Delete(ctx, key)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

	err = store.Save(ctx, key, value)
	require.NoError(t, err)

	exists, err = store.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

func BenchmarkPebbleStore_Save(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(b, err)
	defer store.Close()

	ctx := context.Background()
	data := testData{ID: "1", Name: "Alice", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(ctx, "key", data)
	}
}

func BenchmarkPebbleStore_Load(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(b, err)
	defer store.Close()

	ctx := context.Background()
	store.Save(ctx, "key", testData{ID: "1", Name: "Alice", Age: 30})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Load(ctx, "key")
	}
}

func BenchmarkPebbleStore_Delete(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(b, err)
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store.Save(ctx, "key", testData{ID: "1", Name: "Alice", Age: 30})
		b.StartTimer()
		store.Delete(ctx, "key")
	}
}

func BenchmarkPebbleStore_List(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(b, err)
	defer store.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		store.Save(ctx, string(rune(i)), testData{ID: string(rune(i)), Name: "User", Age: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List(ctx)
	}
}

func BenchmarkPebbleStore_Count(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewPebbleStore[testData](PebbleStoreConfig{
		Path:   tmpDir,
		Prefix: "test:",
	})
	require.NoError(b, err)
	defer store.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		store.Save(ctx, string(rune(i)), testData{ID: string(rune(i)), Name: "User", Age: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Count(ctx)
	}
}
