package message

import (
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name       string
		packetID   uint16
		topic      string
		payload    []byte
		qos        encoding.QoS
		retain     bool
		properties map[string]interface{}
		wantExpiry bool
	}{
		{
			name:       "qos 0 message without properties",
			packetID:   1,
			topic:      "test/topic",
			payload:    []byte("test payload"),
			qos:        encoding.QoS0,
			retain:     false,
			properties: nil,
			wantExpiry: false,
		},
		{
			name:     "qos 1 message with expiry",
			packetID: 2,
			topic:    "test/topic",
			payload:  []byte("test payload"),
			qos:      encoding.QoS1,
			retain:   true,
			properties: map[string]interface{}{
				"MessageExpiryInterval": uint32(60),
			},
			wantExpiry: true,
		},
		{
			name:     "qos 2 message with properties",
			packetID: 3,
			topic:    "test/topic",
			payload:  []byte("test payload"),
			qos:      encoding.QoS2,
			retain:   false,
			properties: map[string]interface{}{
				"MessageExpiryInterval": uint32(120),
				"ContentType":           "application/json",
			},
			wantExpiry: true,
		},
		{
			name:       "empty payload",
			packetID:   4,
			topic:      "test/topic",
			payload:    []byte{},
			qos:        encoding.QoS1,
			retain:     false,
			properties: nil,
			wantExpiry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.packetID, tt.topic, tt.payload, tt.qos, tt.retain, tt.properties)

			require.NotNil(t, msg)
			assert.Equal(t, tt.packetID, msg.PacketID)
			assert.Equal(t, tt.topic, msg.Topic)
			assert.Equal(t, tt.payload, msg.Payload)
			assert.Equal(t, tt.qos, msg.QoS)
			assert.Equal(t, tt.retain, msg.Retain)
			assert.False(t, msg.DUP)
			assert.Equal(t, 0, msg.AttemptCount)
			assert.Equal(t, tt.wantExpiry, msg.MessageExpirySet)
			assert.False(t, msg.CreatedAt.IsZero())
			assert.False(t, msg.LastAttemptAt.IsZero())
		})
	}
}

func TestMessage_IsExpired(t *testing.T) {
	tests := []struct {
		name           string
		expiryInterval uint32
		setExpiry      bool
		waitDuration   time.Duration
		wantExpired    bool
	}{
		{
			name:           "not expired - no expiry set",
			expiryInterval: 0,
			setExpiry:      false,
			waitDuration:   0,
			wantExpired:    false,
		},
		{
			name:           "not expired - zero expiry interval",
			expiryInterval: 0,
			setExpiry:      true,
			waitDuration:   0,
			wantExpired:    false,
		},
		{
			name:           "not expired - within expiry time",
			expiryInterval: 60,
			setExpiry:      true,
			waitDuration:   0,
			wantExpired:    false,
		},
		{
			name:           "expired - past expiry time",
			expiryInterval: 1,
			setExpiry:      true,
			waitDuration:   2 * time.Second,
			wantExpired:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := make(map[string]interface{})
			if tt.setExpiry {
				properties["MessageExpiryInterval"] = tt.expiryInterval
			}

			msg := NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)

			if tt.waitDuration > 0 {
				msg.CreatedAt = time.Now().Add(-tt.waitDuration)
			}

			assert.Equal(t, tt.wantExpired, msg.IsExpired())
		})
	}
}

func TestMessage_RemainingExpiry(t *testing.T) {
	tests := []struct {
		name             string
		expiryInterval   uint32
		setExpiry        bool
		elapsed          time.Duration
		wantRemaining    uint32
		wantRemainingMin uint32
	}{
		{
			name:             "no expiry set",
			expiryInterval:   0,
			setExpiry:        false,
			elapsed:          0,
			wantRemaining:    0,
			wantRemainingMin: 0,
		},
		{
			name:             "zero expiry interval",
			expiryInterval:   0,
			setExpiry:        true,
			elapsed:          0,
			wantRemaining:    0,
			wantRemainingMin: 0,
		},
		{
			name:             "no time elapsed",
			expiryInterval:   60,
			setExpiry:        true,
			elapsed:          0,
			wantRemaining:    60,
			wantRemainingMin: 59,
		},
		{
			name:             "partial time elapsed",
			expiryInterval:   60,
			setExpiry:        true,
			elapsed:          30 * time.Second,
			wantRemaining:    30,
			wantRemainingMin: 29,
		},
		{
			name:             "expired",
			expiryInterval:   10,
			setExpiry:        true,
			elapsed:          15 * time.Second,
			wantRemaining:    0,
			wantRemainingMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := make(map[string]interface{})
			if tt.setExpiry {
				properties["MessageExpiryInterval"] = tt.expiryInterval
			}

			msg := NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)

			if tt.elapsed > 0 {
				msg.CreatedAt = time.Now().Add(-tt.elapsed)
			}

			remaining := msg.RemainingExpiry()
			assert.GreaterOrEqual(t, remaining, tt.wantRemainingMin)
			assert.LessOrEqual(t, remaining, tt.wantRemaining)
		})
	}
}

func TestMessage_MarkAttempt(t *testing.T) {
	msg := NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)

	assert.Equal(t, 0, msg.AttemptCount)
	assert.False(t, msg.DUP)

	initialTime := msg.LastAttemptAt

	time.Sleep(10 * time.Millisecond)
	msg.MarkAttempt()

	assert.Equal(t, 1, msg.AttemptCount)
	assert.False(t, msg.DUP)
	assert.True(t, msg.LastAttemptAt.After(initialTime))

	msg.MarkAttempt()
	assert.Equal(t, 2, msg.AttemptCount)
	assert.True(t, msg.DUP)

	msg.MarkAttempt()
	assert.Equal(t, 3, msg.AttemptCount)
	assert.True(t, msg.DUP)
}

func TestMessage_Clone(t *testing.T) {
	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(60),
		"ContentType":           "application/json",
	}

	original := NewMessage(1, "test/topic", []byte("payload"), encoding.QoS2, true, properties)
	original.MarkAttempt()
	original.MarkAttempt()

	cloned := original.Clone()

	require.NotNil(t, cloned)
	assert.Equal(t, original.PacketID, cloned.PacketID)
	assert.Equal(t, original.Topic, cloned.Topic)
	assert.Equal(t, original.Payload, cloned.Payload)
	assert.Equal(t, original.QoS, cloned.QoS)
	assert.Equal(t, original.Retain, cloned.Retain)
	assert.Equal(t, original.DUP, cloned.DUP)
	assert.Equal(t, original.AttemptCount, cloned.AttemptCount)
	assert.Equal(t, original.ExpiryInterval, cloned.ExpiryInterval)
	assert.Equal(t, original.MessageExpirySet, cloned.MessageExpirySet)

	cloned.Payload[0] = 'X'
	assert.NotEqual(t, original.Payload[0], cloned.Payload[0])

	cloned.Properties["NewProp"] = "value"
	_, exists := original.Properties["NewProp"]
	assert.False(t, exists)
}

func TestMessage_CloneWithNilProperties(t *testing.T) {
	original := NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)
	cloned := original.Clone()

	require.NotNil(t, cloned)
	assert.NotNil(t, cloned.Properties)
	assert.Equal(t, 0, len(cloned.Properties))
}

func TestMessage_AllQoSLevels(t *testing.T) {
	tests := []struct {
		name string
		qos  encoding.QoS
	}{
		{
			name: "qos 0",
			qos:  encoding.QoS0,
		},
		{
			name: "qos 1",
			qos:  encoding.QoS1,
		},
		{
			name: "qos 2",
			qos:  encoding.QoS2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(1, "test/topic", []byte("payload"), tt.qos, false, nil)
			assert.Equal(t, tt.qos, msg.QoS)
		})
	}
}

func TestMessage_LargePayload(t *testing.T) {
	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	msg := NewMessage(1, "test/topic", largePayload, encoding.QoS1, false, nil)
	assert.Equal(t, len(largePayload), len(msg.Payload))

	cloned := msg.Clone()
	assert.Equal(t, len(largePayload), len(cloned.Payload))
	assert.Equal(t, msg.Payload, cloned.Payload)
}
