package session

import (
	"sync"
	"time"
)

// State represents the session state
type State byte

const (
	StateNew          State = iota // Session is newly created
	StateActive                    // Session is active with a connected client
	StateDisconnected              // Session is disconnected but not expired
	StateExpired                   // Session has expired
)

// WillMessage represents the MQTT will message
type WillMessage struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Retain     bool
	Properties map[string]interface{}
}

// Session represents an MQTT session
type Session struct {
	mu sync.RWMutex

	ClientID          string
	CleanStart        bool
	State             State
	ExpiryInterval    uint32 // Session expiry interval in seconds (0 = no expiry for persistent session)
	CreatedAt         time.Time
	LastAccessedAt    time.Time
	DisconnectedAt    time.Time
	WillMessage       *WillMessage
	WillDelayInterval uint32 // Will delay interval in seconds

	// Subscription data
	Subscriptions map[string]*Subscription // topic filter -> subscription

	// QoS message state
	PendingPublish map[uint16]*PendingMessage // PacketID -> message (QoS 1,2 outbound not acked)
	PendingPubrel  map[uint16]struct{}        // PacketID -> marker (QoS 2 inbound waiting for PUBREL)
	PendingPubcomp map[uint16]struct{}        // PacketID -> marker (QoS 2 outbound waiting for PUBCOMP)

	// Packet ID generator
	nextPacketID uint16

	// Maximum packet size
	MaxPacketSize uint32

	// Receive maximum (max inflight)
	ReceiveMaximum uint16

	// Protocol version
	ProtocolVersion byte
}

// Subscription represents a topic subscription
type Subscription struct {
	TopicFilter            string
	QoS                    byte
	NoLocal                bool
	RetainAsPublished      bool
	RetainHandling         byte
	SubscriptionIdentifier uint32
	SubscribedAt           time.Time
}

// PendingMessage represents a message waiting for acknowledgment
type PendingMessage struct {
	PacketID   uint16
	Topic      string
	Payload    []byte
	QoS        byte
	Retain     bool
	DUP        bool
	Properties map[string]interface{}
	Timestamp  time.Time
}

// New creates a new session
func New(clientID string, cleanStart bool, expiryInterval uint32, protocolVersion byte) *Session {
	now := time.Now()
	return &Session{
		ClientID:        clientID,
		CleanStart:      cleanStart,
		State:           StateNew,
		ExpiryInterval:  expiryInterval,
		CreatedAt:       now,
		LastAccessedAt:  now,
		Subscriptions:   make(map[string]*Subscription),
		PendingPublish:  make(map[uint16]*PendingMessage),
		PendingPubrel:   make(map[uint16]struct{}),
		PendingPubcomp:  make(map[uint16]struct{}),
		nextPacketID:    1,
		ReceiveMaximum:  65535, // Default maximum
		ProtocolVersion: protocolVersion,
	}
}

// SetActive marks the session as active
func (s *Session) SetActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateActive
	s.LastAccessedAt = time.Now()
}

// SetDisconnected marks the session as disconnected
func (s *Session) SetDisconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateDisconnected
	s.DisconnectedAt = time.Now()
}

// SetExpired marks the session as expired
func (s *Session) SetExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateExpired
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ExpiryInterval == 0 && !s.CleanStart {
		return false // Persistent session with no expiry
	}

	if s.State == StateDisconnected && s.ExpiryInterval > 0 {
		return time.Since(s.DisconnectedAt) > time.Duration(s.ExpiryInterval)*time.Second
	}

	return s.State == StateExpired
}

// Touch updates the last accessed time
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccessedAt = time.Now()
}

// SetWillMessage sets the will message for the session
func (s *Session) SetWillMessage(will *WillMessage, delayInterval uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WillMessage = will
	s.WillDelayInterval = delayInterval
}

// ClearWillMessage clears the will message
func (s *Session) ClearWillMessage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WillMessage = nil
}

// GetWillMessage returns the will message if present
func (s *Session) GetWillMessage() *WillMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WillMessage
}

// ShouldPublishWill checks if will message should be published
func (s *Session) ShouldPublishWill() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.WillMessage == nil {
		return false
	}

	if s.WillDelayInterval == 0 {
		return true
	}

	return time.Since(s.DisconnectedAt) >= time.Duration(s.WillDelayInterval)*time.Second
}

// AddSubscription adds a subscription to the session
func (s *Session) AddSubscription(sub *Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Subscriptions[sub.TopicFilter] = sub
}

// RemoveSubscription removes a subscription from the session
func (s *Session) RemoveSubscription(topicFilter string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Subscriptions, topicFilter)
}

// GetSubscription returns a subscription by topic filter
func (s *Session) GetSubscription(topicFilter string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.Subscriptions[topicFilter]
	return sub, ok
}

// GetAllSubscriptions returns all subscriptions
func (s *Session) GetAllSubscriptions() map[string]*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs := make(map[string]*Subscription, len(s.Subscriptions))
	for k, v := range s.Subscriptions {
		subs[k] = v
	}
	return subs
}

// ClearSubscriptions removes all subscriptions
func (s *Session) ClearSubscriptions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Subscriptions = make(map[string]*Subscription)
}

// NextPacketID generates the next packet ID
func (s *Session) NextPacketID() uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		id := s.nextPacketID
		s.nextPacketID++
		if s.nextPacketID == 0 {
			s.nextPacketID = 1
		}

		// Check if ID is already in use
		if _, ok := s.PendingPublish[id]; !ok {
			if _, ok := s.PendingPubrel[id]; !ok {
				if _, ok := s.PendingPubcomp[id]; !ok {
					return id
				}
			}
		}
	}
}

// AddPendingPublish adds a pending publish message
func (s *Session) AddPendingPublish(msg *PendingMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingPublish[msg.PacketID] = msg
}

// RemovePendingPublish removes a pending publish message
func (s *Session) RemovePendingPublish(packetID uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PendingPublish, packetID)
}

// GetPendingPublish returns a pending publish message
func (s *Session) GetPendingPublish(packetID uint16) (*PendingMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msg, ok := s.PendingPublish[packetID]
	return msg, ok
}

// GetAllPendingPublish returns all pending publish messages
func (s *Session) GetAllPendingPublish() map[uint16]*PendingMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := make(map[uint16]*PendingMessage, len(s.PendingPublish))
	for k, v := range s.PendingPublish {
		msgs[k] = v
	}
	return msgs
}

// AddPendingPubrel adds a pending PUBREL marker
func (s *Session) AddPendingPubrel(packetID uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingPubrel[packetID] = struct{}{}
}

// RemovePendingPubrel removes a pending PUBREL marker
func (s *Session) RemovePendingPubrel(packetID uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PendingPubrel, packetID)
}

// HasPendingPubrel checks if a PUBREL is pending
func (s *Session) HasPendingPubrel(packetID uint16) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.PendingPubrel[packetID]
	return ok
}

// AddPendingPubcomp adds a pending PUBCOMP marker
func (s *Session) AddPendingPubcomp(packetID uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingPubcomp[packetID] = struct{}{}
}

// RemovePendingPubcomp removes a pending PUBCOMP marker
func (s *Session) RemovePendingPubcomp(packetID uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PendingPubcomp, packetID)
}

// HasPendingPubcomp checks if a PUBCOMP is pending
func (s *Session) HasPendingPubcomp(packetID uint16) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.PendingPubcomp[packetID]
	return ok
}

// Clear clears all session data
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Subscriptions = make(map[string]*Subscription)
	s.PendingPublish = make(map[uint16]*PendingMessage)
	s.PendingPubrel = make(map[uint16]struct{})
	s.PendingPubcomp = make(map[uint16]struct{})
	s.WillMessage = nil
}

// GetState returns the current state
func (s *Session) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// GetClientID returns the client ID
func (s *Session) GetClientID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientID
}

// GetCleanStart returns the clean start flag
func (s *Session) GetCleanStart() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CleanStart
}

// GetExpiryInterval returns the expiry interval
func (s *Session) GetExpiryInterval() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ExpiryInterval
}

// UpdateExpiryInterval updates the session expiry interval
func (s *Session) UpdateExpiryInterval(interval uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExpiryInterval = interval
}
