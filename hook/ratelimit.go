package hook

import (
	"sync"
	"time"
)

const (
	// _defaultExpiryWindowMultiplier defines how many window periods to wait before cleaning up inactive rate limiters.
	// A limiter is considered inactive if it hasn't been accessed for (window * _defaultExpiryWindowMultiplier).
	_defaultExpiryWindowMultiplier = 3
	// _defaultCleanupInterval defines how often the cleanup process runs to remove old limiters.
	// It should be at least as long as the window duration to ensure proper cleanup.
	// This value is overridden in the startCleanup method based on the window duration.
	_defaultCleanupInterval = 2
)

// RateLimitHook provides rate limiting for MQTT operations
type RateLimitHook struct {
	*Base
	mu           sync.RWMutex
	limiters     map[string]*rateLimiter
	maxRate      int
	window       time.Duration
	cleanupTimer *time.Timer
}

type rateLimiter struct {
	count       int
	windowStart time.Time
	lastAccess  time.Time
}

// NewRateLimitHook creates a new rate limiting hook
// maxRate: maximum number of operations allowed
// window: time window for rate limiting (e.g., 1 minute)
func NewRateLimitHook(maxRate int, window time.Duration) *RateLimitHook {
	h := &RateLimitHook{
		Base:     &Base{id: "rate-limit"},
		limiters: make(map[string]*rateLimiter),
		maxRate:  maxRate,
		window:   window,
	}
	h.startCleanup()
	return h
}

// ID returns the hook identifier
func (h *RateLimitHook) ID() string {
	return h.id
}

// Provides indicates this hook provides publish rate limiting
func (h *RateLimitHook) Provides(event Event) bool {
	return event == OnPublish
}

// Stop stops the rate limiter and cleanup timer
func (h *RateLimitHook) Stop() error {
	if h.cleanupTimer != nil {
		h.cleanupTimer.Stop()
	}
	return nil
}

// OnPublish checks if the client has exceeded the rate limit
func (h *RateLimitHook) OnPublish(client *Client, _ *PublishPacket) error {
	if client == nil {
		return ErrRatelimitClientNil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	limiter, exists := h.limiters[client.ID]

	if !exists || now.Sub(limiter.windowStart) > h.window {
		h.limiters[client.ID] = &rateLimiter{
			count:       1,
			windowStart: now,
			lastAccess:  now,
		}
		if h.maxRate < 1 {
			return ErrRateLimitExceeded
		}
		return nil
	}

	limiter.lastAccess = now
	limiter.count++

	if limiter.count > h.maxRate {
		return ErrRateLimitExceeded
	}

	return nil
}

// SetMaxRate updates the maximum rate limit
func (h *RateLimitHook) SetMaxRate(maxRate int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.maxRate = maxRate
}

// SetWindow updates the time window
func (h *RateLimitHook) SetWindow(window time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.window = window
}

// GetMaxRate returns the current maximum rate
func (h *RateLimitHook) GetMaxRate() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.maxRate
}

// GetWindow returns the current time window
func (h *RateLimitHook) GetWindow() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.window
}

// GetClientCount returns the current count for a specific client
func (h *RateLimitHook) GetClientCount(clientID string) (int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	limiter, exists := h.limiters[clientID]
	if !exists {
		return 0, false
	}
	return limiter.count, true
}

// ResetClient resets the rate limit for a specific client
func (h *RateLimitHook) ResetClient(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.limiters, clientID)
}

// ResetAll resets all rate limiters
func (h *RateLimitHook) ResetAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.limiters = make(map[string]*rateLimiter)
}

// ActiveClients returns the number of clients currently being tracked
func (h *RateLimitHook) ActiveClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.limiters)
}

// startCleanup starts a background goroutine to clean up old limiters
func (h *RateLimitHook) startCleanup() {
	cleanupInterval := h.window * _defaultCleanupInterval
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	h.cleanupTimer = time.AfterFunc(cleanupInterval, func() {
		h.cleanup()
		h.startCleanup()
	})
}

// cleanup removes old rate limiters that haven't been accessed recently
func (h *RateLimitHook) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	expiry := h.window * _defaultExpiryWindowMultiplier

	for clientID, limiter := range h.limiters {
		if now.Sub(limiter.lastAccess) > expiry {
			delete(h.limiters, clientID)
		}
	}
}

// MultiLevelRateLimitHook provides rate limiting at multiple levels
type MultiLevelRateLimitHook struct {
	*Base
	mu             sync.RWMutex
	perClientLimit int
	perTopicLimit  int
	globalLimit    int
	window         time.Duration
	clientLimiters map[string]*rateLimiter
	topicLimiters  map[string]*rateLimiter
	globalLimiter  *rateLimiter
	cleanupTimer   *time.Timer
}

// NewMultiLevelRateLimitHook creates a multi-level rate limiter
func NewMultiLevelRateLimitHook(perClientLimit, perTopicLimit, globalLimit int, window time.Duration) *MultiLevelRateLimitHook {
	h := &MultiLevelRateLimitHook{
		Base:           &Base{id: "multi-level-rate-limit"},
		perClientLimit: perClientLimit,
		perTopicLimit:  perTopicLimit,
		globalLimit:    globalLimit,
		window:         window,
		clientLimiters: make(map[string]*rateLimiter),
		topicLimiters:  make(map[string]*rateLimiter),
		globalLimiter: &rateLimiter{
			windowStart: time.Now(),
		},
	}
	h.startCleanup()
	return h
}

// ID returns the hook identifier
func (h *MultiLevelRateLimitHook) ID() string {
	return h.id
}

// Provides indicates this hook provides publish rate limiting
func (h *MultiLevelRateLimitHook) Provides(event Event) bool {
	return event == OnPublish
}

// Stop stops the cleanup timer
func (h *MultiLevelRateLimitHook) Stop() error {
	if h.cleanupTimer != nil {
		h.cleanupTimer.Stop()
	}
	return nil
}

// OnPublish checks rate limits at all levels
func (h *MultiLevelRateLimitHook) OnPublish(client *Client, packet *PublishPacket) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// Check global limit
	if h.globalLimit > 0 {
		if now.Sub(h.globalLimiter.windowStart) > h.window {
			h.globalLimiter.count = 1
			h.globalLimiter.windowStart = now
		} else {
			h.globalLimiter.count++
			if h.globalLimiter.count > h.globalLimit {
				return ErrGlobalRateLimitExceeded
			}
		}
	}

	// Check per-client limit
	if h.perClientLimit > 0 {
		if err := h.checkLimit(client.ID, h.perClientLimit, now, h.clientLimiters, ErrClientRateLimitExceeded); err != nil {
			return err
		}
	}

	// Check per-topic limit
	if h.perTopicLimit > 0 {
		if err := h.checkLimit(packet.Topic, h.perTopicLimit, now, h.topicLimiters, ErrTopicRateLimitExceeded); err != nil {
			return err
		}
	}

	return nil
}

// checkLimit checks and updates a specific limit
func (h *MultiLevelRateLimitHook) checkLimit(key string, maxRate int, now time.Time, limiters map[string]*rateLimiter, errType error) error {
	limiter, exists := limiters[key]

	if !exists || now.Sub(limiter.windowStart) > h.window {
		limiters[key] = &rateLimiter{
			count:       1,
			windowStart: now,
			lastAccess:  now,
		}
		return nil
	}

	limiter.lastAccess = now
	limiter.count++

	if limiter.count > maxRate {
		return errType
	}

	return nil
}

// GetClientCount returns the current count for a client
func (h *MultiLevelRateLimitHook) GetClientCount(clientID string) (int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	limiter, exists := h.clientLimiters[clientID]
	if !exists {
		return 0, false
	}
	return limiter.count, true
}

// GetTopicCount returns the current count for a topic
func (h *MultiLevelRateLimitHook) GetTopicCount(topic string) (int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	limiter, exists := h.topicLimiters[topic]
	if !exists {
		return 0, false
	}
	return limiter.count, true
}

// GetGlobalCount returns the current global count
func (h *MultiLevelRateLimitHook) GetGlobalCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.globalLimiter.count
}

// ResetAll resets all rate limiters
func (h *MultiLevelRateLimitHook) ResetAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clientLimiters = make(map[string]*rateLimiter)
	h.topicLimiters = make(map[string]*rateLimiter)
	h.globalLimiter = &rateLimiter{
		windowStart: time.Now(),
	}
}

// startCleanup starts background cleanup
func (h *MultiLevelRateLimitHook) startCleanup() {
	cleanupInterval := h.window * _defaultCleanupInterval
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	h.cleanupTimer = time.AfterFunc(cleanupInterval, func() {
		h.cleanup()
		h.startCleanup()
	})
}

// cleanup removes old limiters
func (h *MultiLevelRateLimitHook) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	expiry := h.window * _defaultExpiryWindowMultiplier

	for key, limiter := range h.clientLimiters {
		if now.Sub(limiter.lastAccess) > expiry {
			delete(h.clientLimiters, key)
		}
	}

	for key, limiter := range h.topicLimiters {
		if now.Sub(limiter.lastAccess) > expiry {
			delete(h.topicLimiters, key)
		}
	}
}
