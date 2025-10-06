package message

import (
	"time"

	"github.com/axmq/ax/encoding"
)

// Message represents a QoS message with all necessary metadata
type Message struct {
	PacketID         uint16
	Topic            string
	Payload          []byte
	QoS              encoding.QoS
	Retain           bool
	DUP              bool
	Properties       map[string]interface{}
	CreatedAt        time.Time
	LastAttemptAt    time.Time
	AttemptCount     int
	ExpiryInterval   uint32
	MessageExpirySet bool
}

// NewMessage creates a new QoS message
func NewMessage(packetID uint16, topic string, payload []byte, qos encoding.QoS, retain bool, properties map[string]interface{}) *Message {
	now := time.Now()
	msg := &Message{
		PacketID:      packetID,
		Topic:         topic,
		Payload:       payload,
		QoS:           qos,
		Retain:        retain,
		DUP:           false,
		Properties:    properties,
		CreatedAt:     now,
		LastAttemptAt: now,
		AttemptCount:  0,
	}

	if properties != nil {
		if expiry, ok := properties["MessageExpiryInterval"].(uint32); ok {
			msg.ExpiryInterval = expiry
			msg.MessageExpirySet = true
		}
	}

	return msg
}

// IsExpired checks if the message has expired
func (m *Message) IsExpired() bool {
	if !m.MessageExpirySet || m.ExpiryInterval == 0 {
		return false
	}
	return time.Since(m.CreatedAt) >= time.Duration(m.ExpiryInterval)*time.Second
}

// RemainingExpiry returns the remaining expiry time in seconds
func (m *Message) RemainingExpiry() uint32 {
	if !m.MessageExpirySet || m.ExpiryInterval == 0 {
		return 0
	}
	elapsed := uint32(time.Since(m.CreatedAt).Seconds())
	if elapsed >= m.ExpiryInterval {
		return 0
	}
	return m.ExpiryInterval - elapsed
}

// MarkAttempt marks a delivery attempt
func (m *Message) MarkAttempt() {
	m.AttemptCount++
	m.LastAttemptAt = time.Now()
	if m.AttemptCount > 1 {
		m.DUP = true
	}
}

// Clone creates a deep copy of the message
func (m *Message) Clone() *Message {
	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)

	properties := make(map[string]interface{})
	for k, v := range m.Properties {
		properties[k] = v
	}

	return &Message{
		PacketID:         m.PacketID,
		Topic:            m.Topic,
		Payload:          payload,
		QoS:              m.QoS,
		Retain:           m.Retain,
		DUP:              m.DUP,
		Properties:       properties,
		CreatedAt:        m.CreatedAt,
		LastAttemptAt:    m.LastAttemptAt,
		AttemptCount:     m.AttemptCount,
		ExpiryInterval:   m.ExpiryInterval,
		MessageExpirySet: m.MessageExpirySet,
	}
}
