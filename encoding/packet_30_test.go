package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseFixedHeader30_ValidPackets tests parsing of valid MQTT 3.0 packets
func TestParseFixedHeader30_ValidPackets(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedType   PacketType
		expectedFlags  byte
		expectedRemLen uint32
	}{
		{
			name:           "CONNECT",
			input:          []byte{0x10, 0x00},
			expectedType:   CONNECT,
			expectedFlags:  0x00,
			expectedRemLen: 0,
		},
		{
			name:           "CONNACK",
			input:          []byte{0x20, 0x02},
			expectedType:   CONNACK,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "PUBLISH QoS0",
			input:          []byte{0x30, 0x0A},
			expectedType:   PUBLISH,
			expectedFlags:  0x00,
			expectedRemLen: 10,
		},
		{
			name:           "SUBSCRIBE",
			input:          []byte{0x82, 0x05},
			expectedType:   SUBSCRIBE,
			expectedFlags:  0x02,
			expectedRemLen: 5,
		},
		{
			name:           "DISCONNECT",
			input:          []byte{0xE0, 0x00},
			expectedType:   DISCONNECT,
			expectedFlags:  0x00,
			expectedRemLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			header, err := ParseFixedHeaderWithVersion(r, ProtocolVersion30)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedType, header.Type)
			assert.Equal(t, tt.expectedFlags, header.Flags)
			assert.Equal(t, tt.expectedRemLen, header.RemainingLength)
		})
	}
}

// TestParseFixedHeader30_RejectsAUTH tests that AUTH packets are rejected in MQTT 3.0
func TestParseFixedHeader30_RejectsAUTH(t *testing.T) {
	// AUTH packet (type 15)
	input := []byte{0xF0, 0x00}
	r := bytes.NewReader(input)
	header, err := ParseFixedHeaderWithVersion(r, ProtocolVersion30)
	assert.Nil(t, header)
	assert.ErrorIs(t, err, ErrInvalidType)
}

// TestEncodeFixedHeader30_ValidPackets tests encoding valid MQTT 3.0 packets
func TestEncodeFixedHeader30_ValidPackets(t *testing.T) {
	tests := []struct {
		name     string
		header   *FixedHeader
		expected []byte
	}{
		{
			name: "CONNECT",
			header: &FixedHeader{
				Type:            CONNECT,
				Flags:           0x00,
				RemainingLength: 10,
			},
			expected: []byte{0x10, 0x0A},
		},
		{
			name: "SUBSCRIBE",
			header: &FixedHeader{
				Type:            SUBSCRIBE,
				Flags:           0x02,
				RemainingLength: 128,
			},
			expected: []byte{0x82, 0x80, 0x01},
		},
		{
			name: "DISCONNECT",
			header: &FixedHeader{
				Type:            DISCONNECT,
				Flags:           0x00,
				RemainingLength: 0,
			},
			expected: []byte{0xE0, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.header.EncodeFixedHeaderWithVersion(&buf, ProtocolVersion30)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.Bytes())
		})
	}
}

// TestEncodeFixedHeader30_RejectsAUTH tests that AUTH packets are rejected in MQTT 3.0
func TestEncodeFixedHeader30_RejectsAUTH(t *testing.T) {
	header := &FixedHeader{
		Type:            AUTH,
		Flags:           0x00,
		RemainingLength: 0,
	}

	var buf bytes.Buffer
	err := header.EncodeFixedHeaderWithVersion(&buf, ProtocolVersion30)
	assert.ErrorIs(t, err, ErrInvalidType)
	assert.Equal(t, 0, buf.Len())
}

// TestProtocolVersion30Compatibility tests MQTT 3.0 compatibility
func TestProtocolVersion30Compatibility(t *testing.T) {
	authPacket := []byte{0xF0, 0x00}

	// Should work with MQTT 5.0 parser
	r := bytes.NewReader(authPacket)
	header, err := ParseFixedHeaderWithVersion(r, ProtocolVersion50)
	require.NoError(t, err)
	assert.Equal(t, AUTH, header.Type)

	// Should fail with MQTT 3.0 parser
	r = bytes.NewReader(authPacket)
	header, err = ParseFixedHeaderWithVersion(r, ProtocolVersion30)
	assert.Nil(t, header)
	assert.ErrorIs(t, err, ErrInvalidType)

	// Should fail with MQTT 3.1.1 parser
	r = bytes.NewReader(authPacket)
	header, err = ParseFixedHeaderWithVersion(r, ProtocolVersion311)
	assert.Nil(t, header)
	assert.ErrorIs(t, err, ErrInvalidType)
}

// TestRoundTrip30 tests encoding and then decoding produces the same result for MQTT 3.0
func TestRoundTrip30(t *testing.T) {
	tests := []struct {
		name   string
		header *FixedHeader
	}{
		{
			name: "CONNECT",
			header: &FixedHeader{
				Type:            CONNECT,
				Flags:           0x00,
				RemainingLength: 42,
			},
		},
		{
			name: "PUBLISH QoS2",
			header: &FixedHeader{
				Type:            PUBLISH,
				Flags:           0x04,
				RemainingLength: 100,
				DUP:             false,
				QoS:             QoS2,
				Retain:          false,
			},
		},
		{
			name: "SUBSCRIBE",
			header: &FixedHeader{
				Type:            SUBSCRIBE,
				Flags:           0x02,
				RemainingLength: 16383,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			var buf bytes.Buffer
			err := tt.header.EncodeFixedHeaderWithVersion(&buf, ProtocolVersion30)
			require.NoError(t, err)

			// Decode
			decoded, err := ParseFixedHeaderWithVersion(&buf, ProtocolVersion30)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tt.header.Type, decoded.Type)
			assert.Equal(t, tt.header.Flags, decoded.Flags)
			assert.Equal(t, tt.header.RemainingLength, decoded.RemainingLength)

			if tt.header.Type == PUBLISH {
				assert.Equal(t, tt.header.DUP, decoded.DUP)
				assert.Equal(t, tt.header.QoS, decoded.QoS)
				assert.Equal(t, tt.header.Retain, decoded.Retain)
			}
		})
	}
}
