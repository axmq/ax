package qos

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDedupCache(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
	}{
		{
			name:    "small cache",
			maxSize: 10,
		},
		{
			name:    "medium cache",
			maxSize: 100,
		},
		{
			name:    "large cache",
			maxSize: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newDedupCache(tt.maxSize)
			require.NotNil(t, cache)
			assert.Equal(t, tt.maxSize, cache.maxSize)
			assert.NotNil(t, cache.entries)
			assert.Equal(t, 0, cache.size())
		})
	}
}

func TestDedupCache_AddAndExists(t *testing.T) {
	tests := []struct {
		name      string
		packetIDs []uint16
		checkID   uint16
		wantExist bool
	}{
		{
			name:      "single entry exists",
			packetIDs: []uint16{1},
			checkID:   1,
			wantExist: true,
		},
		{
			name:      "single entry not exists",
			packetIDs: []uint16{1},
			checkID:   2,
			wantExist: false,
		},
		{
			name:      "multiple entries exists",
			packetIDs: []uint16{1, 2, 3, 4, 5},
			checkID:   3,
			wantExist: true,
		},
		{
			name:      "multiple entries not exists",
			packetIDs: []uint16{1, 2, 3, 4, 5},
			checkID:   10,
			wantExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newDedupCache(100)

			for _, id := range tt.packetIDs {
				cache.add(id)
			}

			exists := cache.exists(tt.checkID)
			assert.Equal(t, tt.wantExist, exists)
			assert.Equal(t, len(tt.packetIDs), cache.size())
		})
	}
}

func TestDedupCache_Remove(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(1)
	cache.add(2)
	cache.add(3)

	assert.Equal(t, 3, cache.size())
	assert.True(t, cache.exists(2))

	cache.remove(2)

	assert.Equal(t, 2, cache.size())
	assert.False(t, cache.exists(2))
	assert.True(t, cache.exists(1))
	assert.True(t, cache.exists(3))
}

func TestDedupCache_EvictOldest(t *testing.T) {
	cache := newDedupCache(3)

	cache.add(1)
	time.Sleep(10 * time.Millisecond)
	cache.add(2)
	time.Sleep(10 * time.Millisecond)
	cache.add(3)

	assert.Equal(t, 3, cache.size())

	cache.add(4)

	assert.Equal(t, 3, cache.size())
	assert.False(t, cache.exists(1))
	assert.True(t, cache.exists(2))
	assert.True(t, cache.exists(3))
	assert.True(t, cache.exists(4))
}

func TestDedupCache_MaxSize(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		addCount int
		wantSize int
	}{
		{
			name:     "under limit",
			maxSize:  10,
			addCount: 5,
			wantSize: 5,
		},
		{
			name:     "at limit",
			maxSize:  10,
			addCount: 10,
			wantSize: 10,
		},
		{
			name:     "over limit",
			maxSize:  10,
			addCount: 15,
			wantSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newDedupCache(tt.maxSize)

			for i := 0; i < tt.addCount; i++ {
				cache.add(uint16(i + 1))
				time.Sleep(1 * time.Millisecond)
			}

			assert.Equal(t, tt.wantSize, cache.size())
		})
	}
}

func TestDedupCache_Cleanup(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(1)
	cache.add(2)

	cache.entries[1].timestamp = time.Now().Add(-10 * time.Minute)
	cache.entries[2].timestamp = time.Now().Add(-10 * time.Minute)

	cache.add(3)

	assert.Equal(t, 3, cache.size())

	cache.cleanup()

	assert.Equal(t, 1, cache.size())
	assert.False(t, cache.exists(1))
	assert.False(t, cache.exists(2))
	assert.True(t, cache.exists(3))
}

func TestDedupCache_CleanupRecentEntries(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(1)
	cache.add(2)
	cache.add(3)

	assert.Equal(t, 3, cache.size())

	cache.cleanup()

	assert.Equal(t, 3, cache.size())
	assert.True(t, cache.exists(1))
	assert.True(t, cache.exists(2))
	assert.True(t, cache.exists(3))
}

func TestDedupCache_Clear(t *testing.T) {
	cache := newDedupCache(100)

	for i := 0; i < 50; i++ {
		cache.add(uint16(i + 1))
	}

	assert.Equal(t, 50, cache.size())

	cache.clear()

	assert.Equal(t, 0, cache.size())
	for i := 0; i < 50; i++ {
		assert.False(t, cache.exists(uint16(i+1)))
	}
}

func TestDedupCache_DuplicateAdd(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(1)
	assert.Equal(t, 1, cache.size())

	cache.add(1)
	assert.Equal(t, 1, cache.size())

	cache.add(1)
	assert.Equal(t, 1, cache.size())
}

func TestDedupCache_RemoveNonExistent(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(1)
	assert.Equal(t, 1, cache.size())

	cache.remove(2)
	assert.Equal(t, 1, cache.size())
	assert.True(t, cache.exists(1))
}

func TestDedupCache_ConcurrentAdd(t *testing.T) {
	cache := newDedupCache(1000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.add(uint16(id))
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 100, cache.size())
}

func TestDedupCache_ConcurrentExists(t *testing.T) {
	cache := newDedupCache(1000)

	for i := 0; i < 100; i++ {
		cache.add(uint16(i))
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			exists := cache.exists(uint16(id))
			assert.True(t, exists)
		}(i)
	}

	wg.Wait()
}

func TestDedupCache_ConcurrentRemove(t *testing.T) {
	cache := newDedupCache(1000)

	for i := 0; i < 100; i++ {
		cache.add(uint16(i))
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.remove(uint16(id))
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, cache.size())
}

func TestDedupCache_ConcurrentMixed(t *testing.T) {
	cache := newDedupCache(1000)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.add(uint16(id))
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.exists(uint16(id))
		}(i)
	}

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.remove(uint16(id))
		}(i)
	}

	wg.Wait()
}

func TestDedupCache_EmptyCache(t *testing.T) {
	cache := newDedupCache(100)

	assert.Equal(t, 0, cache.size())
	assert.False(t, cache.exists(1))

	cache.remove(1)
	assert.Equal(t, 0, cache.size())

	cache.cleanup()
	assert.Equal(t, 0, cache.size())
}

func TestDedupCache_EvictOldestEmptyCache(t *testing.T) {
	cache := newDedupCache(10)
	cache.evictOldest()
	assert.Equal(t, 0, cache.size())
}

func TestDedupCache_PacketIDZero(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(0)
	assert.True(t, cache.exists(0))
	assert.Equal(t, 1, cache.size())

	cache.remove(0)
	assert.False(t, cache.exists(0))
	assert.Equal(t, 0, cache.size())
}

func TestDedupCache_MaxPacketID(t *testing.T) {
	cache := newDedupCache(100)

	cache.add(65535)
	assert.True(t, cache.exists(65535))
	assert.Equal(t, 1, cache.size())

	cache.remove(65535)
	assert.False(t, cache.exists(65535))
	assert.Equal(t, 0, cache.size())
}
