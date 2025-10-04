package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseFixedHeader311_ValidPackets tests parsing of valid MQTT 3.1.1 packets
func TestParseFixedHeader311_ValidPackets(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedType   PacketType
		expectedFlags  byte
		expectedRemLen uint32
		expectedDUP    bool
		expectedQoS    QoS
		expectedRetain bool
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
			expectedDUP:    false,
			expectedQoS:    QoS0,
			expectedRetain: false,
		},
		{
			name:           "PUBLISH QoS1 with Retain",
			input:          []byte{0x33, 0x05},
			expectedType:   PUBLISH,
			expectedFlags:  0x03,
			expectedRemLen: 5,
			expectedDUP:    false,
			expectedQoS:    QoS1,
			expectedRetain: true,
		},
		{
			name:           "PUBLISH QoS2 with DUP",
			input:          []byte{0x3C, 0x07},
			expectedType:   PUBLISH,
			expectedFlags:  0x0C,
			expectedRemLen: 7,
			expectedDUP:    true,
			expectedQoS:    QoS2,
			expectedRetain: false,
		},
		{
			name:           "PUBACK",
			input:          []byte{0x40, 0x02},
			expectedType:   PUBACK,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "PUBREC",
			input:          []byte{0x50, 0x02},
			expectedType:   PUBREC,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "PUBREL with required flags 0010",
			input:          []byte{0x62, 0x02},
			expectedType:   PUBREL,
			expectedFlags:  0x02,
			expectedRemLen: 2,
		},
		{
			name:           "PUBCOMP",
			input:          []byte{0x70, 0x02},
			expectedType:   PUBCOMP,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "SUBSCRIBE with required flags 0010",
			input:          []byte{0x82, 0x05},
			expectedType:   SUBSCRIBE,
			expectedFlags:  0x02,
			expectedRemLen: 5,
		},
		{
			name:           "SUBACK",
			input:          []byte{0x90, 0x03},
			expectedType:   SUBACK,
			expectedFlags:  0x00,
			expectedRemLen: 3,
		},
		{
			name:           "UNSUBSCRIBE with required flags 0010",
			input:          []byte{0xA2, 0x04},
			expectedType:   UNSUBSCRIBE,
			expectedFlags:  0x02,
			expectedRemLen: 4,
		},
		{
			name:           "UNSUBACK",
			input:          []byte{0xB0, 0x02},
			expectedType:   UNSUBACK,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "PINGREQ",
			input:          []byte{0xC0, 0x00},
			expectedType:   PINGREQ,
			expectedFlags:  0x00,
			expectedRemLen: 0,
		},
		{
			name:           "PINGRESP",
			input:          []byte{0xD0, 0x00},
			expectedType:   PINGRESP,
			expectedFlags:  0x00,
			expectedRemLen: 0,
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
			header, err := ParseFixedHeader311(r)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedType, header.Type)
			assert.Equal(t, tt.expectedFlags, header.Flags)
			assert.Equal(t, tt.expectedRemLen, header.RemainingLength)

			if tt.expectedType == PUBLISH {
				assert.Equal(t, tt.expectedDUP, header.DUP)
				assert.Equal(t, tt.expectedQoS, header.QoS)
				assert.Equal(t, tt.expectedRetain, header.Retain)
			}
		})
	}
}

// TestParseFixedHeader311_InvalidPackets tests error handling for invalid MQTT 3.1.1 packets
func TestParseFixedHeader311_InvalidPackets(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedErr error
	}{
		{
			name:        "Reserved packet type (0)",
			input:       []byte{0x00, 0x00},
			expectedErr: ErrInvalidReservedType,
		},
		{
			name:        "AUTH packet (not in MQTT 3.1.1)",
			input:       []byte{0xF0, 0x00},
			expectedErr: ErrInvalidType,
		},
		{
			name:        "Invalid packet type (16)",
			input:       []byte{0xFF, 0x00},
			expectedErr: ErrInvalidType, // Type 15 (AUTH) is > DISCONNECT for 3.1.1
		},
		{
			name:        "CONNECT with invalid flags",
			input:       []byte{0x11, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "PUBLISH with invalid QoS (3)",
			input:       []byte{0x36, 0x00},
			expectedErr: ErrInvalidQoS,
		},
		{
			name:        "PUBREL with invalid flags (not 0x02)",
			input:       []byte{0x60, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "SUBSCRIBE with invalid flags (not 0x02)",
			input:       []byte{0x80, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "UNSUBSCRIBE with invalid flags (not 0x02)",
			input:       []byte{0xA0, 0x00},
			expectedErr: ErrInvalidFlags,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			header, err := ParseFixedHeader311(r)
			assert.Nil(t, header)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestParseFixedHeaderFromBytes311_ValidPackets tests parsing from byte slices
func TestParseFixedHeaderFromBytes311_ValidPackets(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedType   PacketType
		expectedOffset int
	}{
		{
			name:           "CONNECT with 1-byte length",
			input:          []byte{0x10, 0x0A},
			expectedType:   CONNECT,
			expectedOffset: 2,
		},
		{
			name:           "PUBLISH with 2-byte length",
			input:          []byte{0x30, 0x80, 0x01},
			expectedType:   PUBLISH,
			expectedOffset: 3,
		},
		{
			name:           "SUBSCRIBE with 3-byte length",
			input:          []byte{0x82, 0x80, 0x80, 0x01},
			expectedType:   SUBSCRIBE,
			expectedOffset: 4,
		},
		{
			name:           "DISCONNECT",
			input:          []byte{0xE0, 0x00},
			expectedType:   DISCONNECT,
			expectedOffset: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, offset, err := ParseFixedHeaderFromBytes311(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, header.Type)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

// TestParseFixedHeaderFromBytes311_RejectsAUTH tests that AUTH packets are rejected in 3.1.1
func TestParseFixedHeaderFromBytes311_RejectsAUTH(t *testing.T) {
	// AUTH packet (type 15)
	input := []byte{0xF0, 0x00}
	header, offset, err := ParseFixedHeaderFromBytes311(input)
	assert.Nil(t, header)
	assert.Equal(t, 0, offset)
	assert.ErrorIs(t, err, ErrInvalidType)
}

// TestEncodeFixedHeader311_ValidPackets tests encoding valid MQTT 3.1.1 packets
func TestEncodeFixedHeader311_ValidPackets(t *testing.T) {
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
			name: "PUBLISH QoS1 with Retain",
			header: &FixedHeader{
				Type:            PUBLISH,
				Flags:           0x03,
				RemainingLength: 20,
				DUP:             false,
				QoS:             QoS1,
				Retain:          true,
			},
			expected: []byte{0x33, 0x14},
		},
		{
			name: "PUBREL",
			header: &FixedHeader{
				Type:            PUBREL,
				Flags:           0x02,
				RemainingLength: 2,
			},
			expected: []byte{0x62, 0x02},
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
			err := tt.header.EncodeFixedHeader311(&buf)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.Bytes())
		})
	}
}

// TestEncodeFixedHeader311_RejectsAUTH tests that AUTH packets are rejected in 3.1.1
func TestEncodeFixedHeader311_RejectsAUTH(t *testing.T) {
	header := &FixedHeader{
		Type:            AUTH,
		Flags:           0x00,
		RemainingLength: 0,
	}

	var buf bytes.Buffer
	err := header.EncodeFixedHeader311(&buf)
	assert.ErrorIs(t, err, ErrInvalidType)
	assert.Equal(t, 0, buf.Len())
}

// TestEncodeFixedHeaderToBytes311_ValidPackets tests encoding to byte slices
func TestEncodeFixedHeaderToBytes311_ValidPackets(t *testing.T) {
	tests := []struct {
		name           string
		header         *FixedHeader
		expected       []byte
		expectedOffset int
	}{
		{
			name: "CONNECT",
			header: &FixedHeader{
				Type:            CONNECT,
				Flags:           0x00,
				RemainingLength: 10,
			},
			expected:       []byte{0x10, 0x0A},
			expectedOffset: 2,
		},
		{
			name: "PUBLISH with 2-byte length",
			header: &FixedHeader{
				Type:            PUBLISH,
				Flags:           0x00,
				RemainingLength: 128,
				QoS:             QoS0,
			},
			expected:       []byte{0x30, 0x80, 0x01},
			expectedOffset: 3,
		},
		{
			name: "SUBSCRIBE with 3-byte length",
			header: &FixedHeader{
				Type:            SUBSCRIBE,
				Flags:           0x02,
				RemainingLength: 16384,
			},
			expected:       []byte{0x82, 0x80, 0x80, 0x01},
			expectedOffset: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 5)
			offset, err := tt.header.EncodeFixedHeaderToBytes311(buf)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expected, buf[:offset])
		})
	}
}

// TestEncodeFixedHeaderToBytes311_RejectsAUTH tests that AUTH packets are rejected in 3.1.1
func TestEncodeFixedHeaderToBytes311_RejectsAUTH(t *testing.T) {
	header := &FixedHeader{
		Type:            AUTH,
		Flags:           0x00,
		RemainingLength: 0,
	}

	buf := make([]byte, 5)
	offset, err := header.EncodeFixedHeaderToBytes311(buf)
	assert.ErrorIs(t, err, ErrInvalidType)
	assert.Equal(t, 0, offset)
}

// TestVersionCompatibility tests that MQTT 5.0 packets work with 5.0 parser but not 3.1.1 parser
func TestVersionCompatibility(t *testing.T) {
	authPacket := []byte{0xF0, 0x00}

	// Should work with MQTT 5.0 parser
	r := bytes.NewReader(authPacket)
	header, err := ParseFixedHeader(r)
	require.NoError(t, err)
	assert.Equal(t, AUTH, header.Type)

	// Should fail with MQTT 3.1.1 parser
	r = bytes.NewReader(authPacket)
	header, err = ParseFixedHeader311(r)
	assert.Nil(t, header)
	assert.ErrorIs(t, err, ErrInvalidType)
}

// TestRoundTrip311 tests encoding and then decoding produces the same result
func TestRoundTrip311(t *testing.T) {
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
			name: "PUBLISH QoS2 with DUP and Retain",
			header: &FixedHeader{
				Type:            PUBLISH,
				Flags:           0x0D,
				RemainingLength: 100,
				DUP:             true,
				QoS:             QoS2,
				Retain:          true,
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
		{
			name: "DISCONNECT",
			header: &FixedHeader{
				Type:            DISCONNECT,
				Flags:           0x00,
				RemainingLength: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			var buf bytes.Buffer
			err := tt.header.EncodeFixedHeader311(&buf)
			require.NoError(t, err)

			// Decode
			decoded, err := ParseFixedHeader311(&buf)
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
