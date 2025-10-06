package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testData struct {
	ID   string
	Name string
	Age  int
}

func TestMemoryStore_Save(t *testing.T) {
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
			store := NewMemoryStore[testData]()
			defer store.Close()

			err := store.Save(context.Background(), tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryStore_SaveWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Save(ctx, "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.Error(t, err)
}

func TestMemoryStore_SaveAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	err := store.Save(context.Background(), "key1", testData{ID: "1", Name: "Alice", Age: 30})
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_Load(t *testing.T) {
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
			store := NewMemoryStore[testData]()
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

func TestMemoryStore_LoadWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Load(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryStore_LoadAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	_, err := store.Load(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_Delete(t *testing.T) {
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
			store := NewMemoryStore[testData]()
			defer store.Close()

			for k, v := range tt.setupData {
				require.NoError(t, store.Save(context.Background(), k, v))
			}

			err := store.Delete(context.Background(), tt.key)
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

func TestMemoryStore_DeleteWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Delete(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryStore_DeleteAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	err := store.Delete(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_Exists(t *testing.T) {
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
			store := NewMemoryStore[testData]()
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

func TestMemoryStore_ExistsWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Exists(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryStore_ExistsAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	_, err := store.Exists(context.Background(), "key1")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_List(t *testing.T) {
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
			store := NewMemoryStore[testData]()
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

func TestMemoryStore_ListWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.List(ctx)
	assert.Error(t, err)
}

func TestMemoryStore_ListAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	_, err := store.List(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_Count(t *testing.T) {
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
			store := NewMemoryStore[testData]()
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

func TestMemoryStore_CountWithCanceledContext(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Count(ctx)
	assert.Error(t, err)
}

func TestMemoryStore_CountAfterClose(t *testing.T) {
	store := NewMemoryStore[testData]()
	store.Close()

	_, err := store.Count(context.Background())
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore[testData]()

	err := store.Close()
	assert.NoError(t, err)

	err = store.Close()
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestMemoryStore_ConcurrentOperations(t *testing.T) {
	store := NewMemoryStore[testData]()
	defer store.Close()

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

func BenchmarkMemoryStore_Save(b *testing.B) {
	store := NewMemoryStore[testData]()
	defer store.Close()
	ctx := context.Background()
	data := testData{ID: "1", Name: "Alice", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(ctx, "key", data)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	store := NewMemoryStore[testData]()
	defer store.Close()
	ctx := context.Background()
	store.Save(ctx, "key", testData{ID: "1", Name: "Alice", Age: 30})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Load(ctx, "key")
	}
}

func BenchmarkMemoryStore_Delete(b *testing.B) {
	store := NewMemoryStore[testData]()
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

func BenchmarkMemoryStore_List(b *testing.B) {
	store := NewMemoryStore[testData]()
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

func BenchmarkMemoryStore_Count(b *testing.B) {
	store := NewMemoryStore[testData]()
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
