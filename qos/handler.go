package qos

import (
	"context"
	"sync"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
)

// Config holds QoS handler configuration
type Config struct {
	MaxInflight       uint16
	RetryInterval     time.Duration
	MaxRetries        int
	RetryBackoff      float64
	MaxRetryInterval  time.Duration
	CleanupInterval   time.Duration
	AckTimeout        time.Duration
	EnableDedup       bool
	DedupWindowSize   int
	DedupCleanupCount int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxInflight:       65535,
		RetryInterval:     5 * time.Second,
		MaxRetries:        5,
		RetryBackoff:      2.0,
		MaxRetryInterval:  60 * time.Second,
		CleanupInterval:   30 * time.Second,
		AckTimeout:        30 * time.Second,
		EnableDedup:       true,
		DedupWindowSize:   1000,
		DedupCleanupCount: 100,
	}
}

// Handler manages QoS message delivery and acknowledgments
type Handler struct {
	config *Config

	mu            sync.RWMutex
	qos1Messages  map[uint16]*message.Message
	qos2Messages  map[uint16]*message.Message
	qos2Pubrel    map[uint16]struct{}
	qos2Received  map[uint16]struct{}
	dedupCache    *dedupCache
	nextPacketID  uint16
	inflightCount int
	callbacks     *callbacks
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closed        bool
}

// callbacks holds event handlers
type callbacks struct {
	onPublish  func(msg *message.Message) error
	onPuback   func(packetID uint16) error
	onPubrec   func(packetID uint16) error
	onPubrel   func(packetID uint16) error
	onPubcomp  func(packetID uint16) error
	onExpired  func(msg *message.Message)
	onMaxRetry func(msg *message.Message)
}

// NewHandler creates a new QoS handler
func NewHandler(config *Config) *Handler {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := &Handler{
		config:       config,
		qos1Messages: make(map[uint16]*message.Message),
		qos2Messages: make(map[uint16]*message.Message),
		qos2Pubrel:   make(map[uint16]struct{}),
		qos2Received: make(map[uint16]struct{}),
		nextPacketID: 1,
		callbacks:    &callbacks{},
		ctx:          ctx,
		cancel:       cancel,
	}

	if config.EnableDedup {
		h.dedupCache = newDedupCache(config.DedupWindowSize)
	}

	h.wg.Add(2)
	go h.retryLoop()
	go h.cleanupLoop()

	return h
}

// SetPublishCallback sets the callback for publishing messages
func (h *Handler) SetPublishCallback(cb func(msg *message.Message) error) {
	h.mu.Lock()
	h.callbacks.onPublish = cb
	h.mu.Unlock()
}

// SetPubackCallback sets the callback for PUBACK
func (h *Handler) SetPubackCallback(cb func(packetID uint16) error) {
	h.mu.Lock()
	h.callbacks.onPuback = cb
	h.mu.Unlock()
}

// SetPubrecCallback sets the callback for PUBREC
func (h *Handler) SetPubrecCallback(cb func(packetID uint16) error) {
	h.mu.Lock()
	h.callbacks.onPubrec = cb
	h.mu.Unlock()
}

// SetPubrelCallback sets the callback for PUBREL
func (h *Handler) SetPubrelCallback(cb func(packetID uint16) error) {
	h.mu.Lock()
	h.callbacks.onPubrel = cb
	h.mu.Unlock()
}

// SetPubcompCallback sets the callback for PUBCOMP
func (h *Handler) SetPubcompCallback(cb func(packetID uint16) error) {
	h.mu.Lock()
	h.callbacks.onPubcomp = cb
	h.mu.Unlock()
}

// SetExpiredCallback sets the callback for expired messages
func (h *Handler) SetExpiredCallback(cb func(msg *message.Message)) {
	h.mu.Lock()
	h.callbacks.onExpired = cb
	h.mu.Unlock()
}

// SetMaxRetryCallback sets the callback for max retry reached
func (h *Handler) SetMaxRetryCallback(cb func(msg *message.Message)) {
	h.mu.Lock()
	h.callbacks.onMaxRetry = cb
	h.mu.Unlock()
}

// HandlePublish handles incoming PUBLISH packet based on QoS level
func (h *Handler) HandlePublish(msg *message.Message) error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return ErrHandlerClosed
	}
	h.mu.Unlock()

	if msg.IsExpired() {
		return ErrMessageExpired
	}

	switch msg.QoS {
	case encoding.QoS0:
		return h.handleQoS0Publish(msg)
	case encoding.QoS1:
		return h.handleQoS1Publish(msg)
	case encoding.QoS2:
		return h.handleQoS2Publish(msg)
	default:
		return ErrInvalidQoS
	}
}

// handleQoS0Publish handles QoS 0 fire-and-forget delivery
func (h *Handler) handleQoS0Publish(msg *message.Message) error {
	h.mu.RLock()
	cb := h.callbacks.onPublish
	h.mu.RUnlock()

	if cb != nil {
		return cb(msg)
	}
	return nil
}

// handleQoS1Publish handles QoS 1 at-least-once delivery
func (h *Handler) handleQoS1Publish(msg *message.Message) error {
	h.mu.Lock()

	if h.config.EnableDedup && h.dedupCache.exists(msg.PacketID) {
		h.mu.Unlock()
		return h.sendPuback(msg.PacketID)
	}

	if h.config.EnableDedup {
		h.dedupCache.add(msg.PacketID)
	}

	cb := h.callbacks.onPublish
	h.mu.Unlock()

	var err error
	if cb != nil {
		err = cb(msg)
	}

	if err == nil {
		return h.sendPuback(msg.PacketID)
	}

	return err
}

// handleQoS2Publish handles QoS 2 exactly-once delivery (step 1: receive PUBLISH)
func (h *Handler) handleQoS2Publish(msg *message.Message) error {
	h.mu.Lock()

	if _, exists := h.qos2Received[msg.PacketID]; exists {
		h.mu.Unlock()
		return h.sendPubrec(msg.PacketID)
	}

	if h.config.EnableDedup && h.dedupCache.exists(msg.PacketID) {
		h.mu.Unlock()
		return h.sendPubrec(msg.PacketID)
	}

	h.qos2Received[msg.PacketID] = struct{}{}

	if h.config.EnableDedup {
		h.dedupCache.add(msg.PacketID)
	}

	cb := h.callbacks.onPublish
	h.mu.Unlock()

	var err error
	if cb != nil {
		err = cb(msg)
	}

	if err == nil {
		return h.sendPubrec(msg.PacketID)
	}

	return err
}

// HandlePuback handles incoming PUBACK packet (completes QoS 1 flow)
func (h *Handler) HandlePuback(packetID uint16) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return ErrHandlerClosed
	}

	msg, exists := h.qos1Messages[packetID]
	if !exists {
		return ErrPacketIDNotFound
	}

	delete(h.qos1Messages, packetID)
	h.inflightCount--

	if h.callbacks.onPuback != nil {
		return h.callbacks.onPuback(msg.PacketID)
	}

	return nil
}

// HandlePubrec handles incoming PUBREC packet (QoS 2 step 2)
func (h *Handler) HandlePubrec(packetID uint16) error {
	h.mu.Lock()

	if h.closed {
		h.mu.Unlock()
		return ErrHandlerClosed
	}

	msg, exists := h.qos2Messages[packetID]
	if !exists {
		h.mu.Unlock()
		return ErrPacketIDNotFound
	}

	delete(h.qos2Messages, packetID)
	h.qos2Pubrel[packetID] = struct{}{}

	cb := h.callbacks.onPubrec
	h.mu.Unlock()

	if cb != nil {
		if err := cb(packetID); err != nil {
			return err
		}
	}

	return h.sendPubrel(msg.PacketID)
}

// HandlePubrel handles incoming PUBREL packet (QoS 2 step 3)
func (h *Handler) HandlePubrel(packetID uint16) error {
	h.mu.Lock()

	if h.closed {
		h.mu.Unlock()
		return ErrHandlerClosed
	}

	if _, exists := h.qos2Received[packetID]; !exists {
		h.mu.Unlock()
		return h.sendPubcomp(packetID)
	}

	delete(h.qos2Received, packetID)

	cb := h.callbacks.onPubrel
	h.mu.Unlock()

	if cb != nil {
		if err := cb(packetID); err != nil {
			return err
		}
	}

	return h.sendPubcomp(packetID)
}

// HandlePubcomp handles incoming PUBCOMP packet (completes QoS 2 flow)
func (h *Handler) HandlePubcomp(packetID uint16) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return ErrHandlerClosed
	}

	if _, exists := h.qos2Pubrel[packetID]; !exists {
		return ErrPacketIDNotFound
	}

	delete(h.qos2Pubrel, packetID)
	h.inflightCount--

	if h.callbacks.onPubcomp != nil {
		return h.callbacks.onPubcomp(packetID)
	}

	return nil
}

// PublishQoS1 publishes a message with QoS 1 (at-least-once)
func (h *Handler) PublishQoS1(topic string, payload []byte, retain bool, properties map[string]interface{}) (uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return 0, ErrHandlerClosed
	}

	if h.inflightCount >= int(h.config.MaxInflight) {
		return 0, ErrQueueFull
	}

	packetID := h.allocatePacketID()
	msg := message.NewMessage(packetID, topic, payload, encoding.QoS1, retain, properties)

	if msg.IsExpired() {
		return 0, ErrMessageExpired
	}

	h.qos1Messages[packetID] = msg
	h.inflightCount++

	msg.MarkAttempt()
	if h.callbacks.onPublish != nil {
		if err := h.callbacks.onPublish(msg); err != nil {
			delete(h.qos1Messages, packetID)
			h.inflightCount--
			return 0, err
		}
	}

	return packetID, nil
}

// PublishQoS2 publishes a message with QoS 2 (exactly-once)
func (h *Handler) PublishQoS2(topic string, payload []byte, retain bool, properties map[string]interface{}) (uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return 0, ErrHandlerClosed
	}

	if h.inflightCount >= int(h.config.MaxInflight) {
		return 0, ErrQueueFull
	}

	packetID := h.allocatePacketID()
	msg := message.NewMessage(packetID, topic, payload, encoding.QoS2, retain, properties)

	if msg.IsExpired() {
		return 0, ErrMessageExpired
	}

	h.qos2Messages[packetID] = msg
	h.inflightCount++

	msg.MarkAttempt()
	if h.callbacks.onPublish != nil {
		if err := h.callbacks.onPublish(msg); err != nil {
			delete(h.qos2Messages, packetID)
			h.inflightCount--
			return 0, err
		}
	}

	return packetID, nil
}

// allocatePacketID allocates a new packet ID (must be called with lock held)
func (h *Handler) allocatePacketID() uint16 {
	for {
		packetID := h.nextPacketID
		h.nextPacketID++
		if h.nextPacketID == 0 {
			h.nextPacketID = 1
		}

		if _, exists := h.qos1Messages[packetID]; !exists {
			if _, exists := h.qos2Messages[packetID]; !exists {
				if _, exists := h.qos2Pubrel[packetID]; !exists {
					return packetID
				}
			}
		}
	}
}

// sendPuback sends a PUBACK packet
func (h *Handler) sendPuback(packetID uint16) error {
	h.mu.RLock()
	cb := h.callbacks.onPuback
	h.mu.RUnlock()

	if cb != nil {
		return cb(packetID)
	}
	return nil
}

// sendPubrec sends a PUBREC packet
func (h *Handler) sendPubrec(packetID uint16) error {
	h.mu.RLock()
	cb := h.callbacks.onPubrec
	h.mu.RUnlock()

	if cb != nil {
		return cb(packetID)
	}
	return nil
}

// sendPubrel sends a PUBREL packet
func (h *Handler) sendPubrel(packetID uint16) error {
	h.mu.RLock()
	cb := h.callbacks.onPubrel
	h.mu.RUnlock()

	if cb != nil {
		return cb(packetID)
	}
	return nil
}

// sendPubcomp sends a PUBCOMP packet
func (h *Handler) sendPubcomp(packetID uint16) error {
	h.mu.RLock()
	cb := h.callbacks.onPubcomp
	h.mu.RUnlock()

	if cb != nil {
		return cb(packetID)
	}
	return nil
}

// retryLoop handles message retry with exponential backoff
func (h *Handler) retryLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.RetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.retryMessages()
		}
	}
}

// retryMessages retries pending messages
func (h *Handler) retryMessages() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	for packetID, msg := range h.qos1Messages {
		if msg.IsExpired() {
			delete(h.qos1Messages, packetID)
			h.inflightCount--
			if h.callbacks.onExpired != nil {
				h.callbacks.onExpired(msg)
			}
			continue
		}

		retryInterval := h.calculateRetryInterval(msg.AttemptCount)
		if now.Sub(msg.LastAttemptAt) >= retryInterval {
			if msg.AttemptCount >= h.config.MaxRetries {
				delete(h.qos1Messages, packetID)
				h.inflightCount--
				if h.callbacks.onMaxRetry != nil {
					h.callbacks.onMaxRetry(msg)
				}
				continue
			}

			msg.MarkAttempt()
			if h.callbacks.onPublish != nil {
				h.callbacks.onPublish(msg)
			}
		}
	}

	for packetID, msg := range h.qos2Messages {
		if msg.IsExpired() {
			delete(h.qos2Messages, packetID)
			h.inflightCount--
			if h.callbacks.onExpired != nil {
				h.callbacks.onExpired(msg)
			}
			continue
		}

		retryInterval := h.calculateRetryInterval(msg.AttemptCount)
		if now.Sub(msg.LastAttemptAt) >= retryInterval {
			if msg.AttemptCount >= h.config.MaxRetries {
				delete(h.qos2Messages, packetID)
				h.inflightCount--
				if h.callbacks.onMaxRetry != nil {
					h.callbacks.onMaxRetry(msg)
				}
				continue
			}

			msg.MarkAttempt()
			if h.callbacks.onPublish != nil {
				h.callbacks.onPublish(msg)
			}
		}
	}
}

// calculateRetryInterval calculates retry interval with exponential backoff
func (h *Handler) calculateRetryInterval(attemptCount int) time.Duration {
	if attemptCount == 0 {
		return h.config.RetryInterval
	}

	backoffMultiplier := 1.0
	for i := 0; i < attemptCount-1; i++ {
		backoffMultiplier *= h.config.RetryBackoff
	}

	interval := time.Duration(float64(h.config.RetryInterval) * backoffMultiplier)
	if interval > h.config.MaxRetryInterval {
		interval = h.config.MaxRetryInterval
	}

	return interval
}

// cleanupLoop periodically cleans up expired data
func (h *Handler) cleanupLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.cleanup()
		}
	}
}

// cleanup removes expired messages and old deduplication entries
func (h *Handler) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	for packetID, msg := range h.qos1Messages {
		if msg.IsExpired() {
			delete(h.qos1Messages, packetID)
			h.inflightCount--
			if h.callbacks.onExpired != nil {
				h.callbacks.onExpired(msg)
			}
		}
	}

	for packetID, msg := range h.qos2Messages {
		if msg.IsExpired() {
			delete(h.qos2Messages, packetID)
			h.inflightCount--
			if h.callbacks.onExpired != nil {
				h.callbacks.onExpired(msg)
			}
		}
	}

	for packetID := range h.qos2Received {
		if len(h.qos2Received) > h.config.DedupCleanupCount {
			if now.Sub(time.Now()) > h.config.AckTimeout {
				delete(h.qos2Received, packetID)
			}
		}
	}

	if h.config.EnableDedup && h.dedupCache != nil {
		h.dedupCache.cleanup()
	}
}

// GetInflightCount returns the current inflight message count
func (h *Handler) GetInflightCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.inflightCount
}

// GetPendingQoS1Count returns the number of pending QoS 1 messages
func (h *Handler) GetPendingQoS1Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.qos1Messages)
}

// GetPendingQoS2Count returns the number of pending QoS 2 messages
func (h *Handler) GetPendingQoS2Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.qos2Messages)
}

// Close stops the handler and releases resources
func (h *Handler) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	h.mu.Unlock()

	h.cancel()
	h.wg.Wait()

	return nil
}
