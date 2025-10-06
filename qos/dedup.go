package qos

import (
	"sync"
	"time"
)

// dedupCache implements a deduplication cache for QoS 2 messages
type dedupCache struct {
	mu       sync.RWMutex
	entries  map[uint16]*dedupEntry
	maxSize  int
	cleanups int
}

// dedupEntry represents a deduplication cache entry
type dedupEntry struct {
	packetID  uint16
	timestamp time.Time
}

// newDedupCache creates a new deduplication cache
func newDedupCache(maxSize int) *dedupCache {
	return &dedupCache{
		entries: make(map[uint16]*dedupEntry),
		maxSize: maxSize,
	}
}

// add adds a packet ID to the deduplication cache
func (dc *dedupCache) add(packetID uint16) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if len(dc.entries) >= dc.maxSize {
		dc.evictOldest()
	}

	dc.entries[packetID] = &dedupEntry{
		packetID:  packetID,
		timestamp: time.Now(),
	}
}

// exists checks if a packet ID exists in the cache
func (dc *dedupCache) exists(packetID uint16) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	_, exists := dc.entries[packetID]
	return exists
}

// remove removes a packet ID from the cache
func (dc *dedupCache) remove(packetID uint16) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	delete(dc.entries, packetID)
}

// evictOldest evicts the oldest entry from the cache
func (dc *dedupCache) evictOldest() {
	var oldestPacketID uint16
	var oldestTime time.Time
	first := true

	for packetID, entry := range dc.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestPacketID = packetID
			oldestTime = entry.timestamp
			first = false
		}
	}

	if !first {
		delete(dc.entries, oldestPacketID)
	}
}

// cleanup removes entries older than 5 minutes
func (dc *dedupCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	toRemove := make([]uint16, 0)

	for packetID, entry := range dc.entries {
		if entry.timestamp.Before(threshold) {
			toRemove = append(toRemove, packetID)
		}
	}

	for _, packetID := range toRemove {
		delete(dc.entries, packetID)
	}
}

// size returns the current size of the cache
func (dc *dedupCache) size() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.entries)
}

// clear clears all entries from the cache
func (dc *dedupCache) clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.entries = make(map[uint16]*dedupEntry)
}
