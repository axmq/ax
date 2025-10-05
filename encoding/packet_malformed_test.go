package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMalformedFixedHeader tests various malformed fixed header scenarios
func TestMalformedFixedHeader(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedErr error
		reasonCode  ReasonCode
	}{
		{
			name:        "Reserved packet type 0",
			input:       []byte{0x00, 0x00},
			expectedErr: ErrInvalidReservedType,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "Invalid packet type 16 (MQTT 5.0)",
			input:       []byte{0xFF, 0x00}, // Type 15 with flags, should fail on flags
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "CONNECT with invalid flags",
			input:       []byte{0x1F, 0x00}, // All flags set
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "CONNACK with invalid flags",
			input:       []byte{0x2F, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBLISH with invalid QoS 3",
			input:       []byte{0x36, 0x00}, // QoS = 3
			expectedErr: ErrInvalidQoS,
			reasonCode:  ReasonMalformedPacket,
		},
		{
			name:        "PUBLISH with invalid QoS 3 and other flags",
			input:       []byte{0x3F, 0x00}, // QoS = 3, DUP=1, RETAIN=1
			expectedErr: ErrInvalidQoS,
			reasonCode:  ReasonMalformedPacket,
		},
		{
			name:        "PUBACK with invalid flags",
			input:       []byte{0x4F, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBREC with invalid flags",
			input:       []byte{0x5F, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBREL with wrong flags (should be 0x02)",
			input:       []byte{0x60, 0x00}, // Flags = 0x00
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBREL with wrong flags (0x01)",
			input:       []byte{0x61, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBREL with wrong flags (0x03)",
			input:       []byte{0x63, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PUBCOMP with invalid flags",
			input:       []byte{0x7F, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "SUBSCRIBE with wrong flags (should be 0x02)",
			input:       []byte{0x80, 0x00}, // Flags = 0x00
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "SUBSCRIBE with wrong flags (0x01)",
			input:       []byte{0x81, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "SUBACK with invalid flags",
			input:       []byte{0x9F, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "UNSUBSCRIBE with wrong flags (should be 0x02)",
			input:       []byte{0xA0, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "UNSUBACK with invalid flags",
			input:       []byte{0xBF, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PINGREQ with invalid flags",
			input:       []byte{0xCF, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "PINGRESP with invalid flags",
			input:       []byte{0xDF, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "DISCONNECT with invalid flags",
			input:       []byte{0xEF, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "AUTH with invalid flags",
			input:       []byte{0xFF, 0x00},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name:        "Malformed variable byte integer - 5 bytes",
			input:       []byte{0x10, 0x80, 0x80, 0x80, 0x80, 0x01},
			expectedErr: ErrMalformedVariableByteInteger,
			reasonCode:  ReasonMalformedPacket,
		},
		{
			name:        "Incomplete variable byte integer - 1 byte",
			input:       []byte{0x10, 0x80},
			expectedErr: ErrUnexpectedEOF,
			reasonCode:  ReasonUnspecifiedError,
		},
		{
			name:        "Incomplete variable byte integer - 2 bytes",
			input:       []byte{0x10, 0x80, 0x80},
			expectedErr: ErrUnexpectedEOF,
			reasonCode:  ReasonUnspecifiedError,
		},
		{
			name:        "Incomplete variable byte integer - 3 bytes",
			input:       []byte{0x10, 0x80, 0x80, 0x80},
			expectedErr: ErrUnexpectedEOF,
			reasonCode:  ReasonUnspecifiedError,
		},
		{
			name:        "Empty input",
			input:       []byte{},
			expectedErr: ErrUnexpectedEOF,
			reasonCode:  ReasonUnspecifiedError,
		},
		{
			name:        "Only first byte",
			input:       []byte{0x10},
			expectedErr: ErrUnexpectedEOF,
			reasonCode:  ReasonUnspecifiedError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			_, err := ParseFixedHeader(r)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)

			// Verify reason code mapping
			reasonCode := GetReasonCode(err)
			assert.Equal(t, tt.reasonCode, reasonCode, "Reason code mismatch")
		})
	}
}

// TestMalformedFixedHeaderMQTT311 tests MQTT 3.1.1 specific malformed cases
func TestMalformedFixedHeaderMQTT311(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedErr error
	}{
		{
			name:        "AUTH packet not allowed in MQTT 3.1.1",
			input:       []byte{0xF0, 0x00},
			expectedErr: ErrInvalidType,
		},
		{
			name:        "Reserved type 0",
			input:       []byte{0x00, 0x00},
			expectedErr: ErrInvalidReservedType,
		},
		{
			name:        "PUBLISH with invalid QoS",
			input:       []byte{0x36, 0x00},
			expectedErr: ErrInvalidQoS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			_, err := ParseFixedHeader311(r)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestMalformedFixedHeaderFromBytes tests byte slice parsing for malformed packets
func TestMalformedFixedHeaderFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedErr error
	}{
		{
			name:        "Empty input",
			input:       []byte{},
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Only one byte",
			input:       []byte{0x10},
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Reserved type",
			input:       []byte{0x00, 0x00},
			expectedErr: ErrInvalidReservedType,
		},
		{
			name:        "Invalid QoS in PUBLISH",
			input:       []byte{0x36, 0x00},
			expectedErr: ErrInvalidQoS,
		},
		{
			name:        "SUBSCRIBE with wrong flags",
			input:       []byte{0x80, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "PUBREL with wrong flags",
			input:       []byte{0x60, 0x00},
			expectedErr: ErrInvalidFlags,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFixedHeaderFromBytes(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestEncodeFixedHeaderValidation tests validation during encoding
func TestEncodeFixedHeaderValidation(t *testing.T) {
	tests := []struct {
		name        string
		header      FixedHeader
		expectedErr error
		reasonCode  ReasonCode
	}{
		{
			name: "Reserved packet type",
			header: FixedHeader{
				Type:            Reserved,
				Flags:           0x00,
				RemainingLength: 0,
			},
			expectedErr: ErrInvalidReservedType,
			reasonCode:  ReasonProtocolError,
		},
		{
			name: "Invalid packet type (too high for MQTT 5.0)",
			header: FixedHeader{
				Type:            PacketType(16),
				Flags:           0x00,
				RemainingLength: 0,
			},
			expectedErr: ErrInvalidType,
			reasonCode:  ReasonProtocolError,
		},
		{
			name: "CONNECT with invalid flags",
			header: FixedHeader{
				Type:            CONNECT,
				Flags:           0x0F,
				RemainingLength: 10,
			},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name: "PUBLISH with invalid QoS",
			header: FixedHeader{
				Type:            PUBLISH,
				Flags:           0x06, // QoS = 3
				QoS:             QoS(3),
				RemainingLength: 10,
			},
			expectedErr: ErrInvalidQoS,
			reasonCode:  ReasonMalformedPacket,
		},
		{
			name: "SUBSCRIBE with wrong flags",
			header: FixedHeader{
				Type:            SUBSCRIBE,
				Flags:           0x00, // Should be 0x02
				RemainingLength: 10,
			},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
		{
			name: "PUBREL with wrong flags",
			header: FixedHeader{
				Type:            PUBREL,
				Flags:           0x00, // Should be 0x02
				RemainingLength: 10,
			},
			expectedErr: ErrInvalidFlags,
			reasonCode:  ReasonProtocolError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.header.EncodeFixedHeader(&buf)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)

			// Verify reason code
			reasonCode := GetReasonCode(err)
			assert.Equal(t, tt.reasonCode, reasonCode)
		})
	}
}

// TestEncodeFixedHeaderToBytesValidation tests byte slice encoding validation
func TestEncodeFixedHeaderToBytesValidation(t *testing.T) {
	tests := []struct {
		name        string
		header      FixedHeader
		bufSize     int
		expectedErr error
	}{
		{
			name: "Buffer too small",
			header: FixedHeader{
				Type:            CONNECT,
				Flags:           0x00,
				RemainingLength: 0,
			},
			bufSize:     1,
			expectedErr: ErrBufferTooSmall,
		},
		{
			name: "Reserved type",
			header: FixedHeader{
				Type:            Reserved,
				Flags:           0x00,
				RemainingLength: 0,
			},
			bufSize:     10,
			expectedErr: ErrInvalidReservedType,
		},
		{
			name: "Invalid QoS in PUBLISH",
			header: FixedHeader{
				Type:            PUBLISH,
				Flags:           0x06,
				QoS:             QoS(3),
				RemainingLength: 0,
			},
			bufSize:     10,
			expectedErr: ErrInvalidQoS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			_, err := tt.header.EncodeFixedHeaderToBytes(buf)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestQoSValidation tests QoS level validation
func TestQoSValidation(t *testing.T) {
	tests := []struct {
		name    string
		qos     QoS
		isValid bool
	}{
		{"QoS 0", QoS0, true},
		{"QoS 1", QoS1, true},
		{"QoS 2", QoS2, true},
		{"Invalid QoS 3", QoS(3), false},
		{"Invalid QoS 4", QoS(4), false},
		{"Invalid QoS 255", QoS(255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.qos.IsValid())
		})
	}
}

// TestPacketTypeString tests string representation
func TestPacketTypeString(t *testing.T) {
	tests := []struct {
		packetType PacketType
		expected   string
	}{
		{Reserved, "RESERVED"},
		{CONNECT, "CONNECT"},
		{CONNACK, "CONNACK"},
		{PUBLISH, "PUBLISH"},
		{PUBACK, "PUBACK"},
		{PUBREC, "PUBREC"},
		{PUBREL, "PUBREL"},
		{PUBCOMP, "PUBCOMP"},
		{SUBSCRIBE, "SUBSCRIBE"},
		{SUBACK, "SUBACK"},
		{UNSUBSCRIBE, "UNSUBSCRIBE"},
		{UNSUBACK, "UNSUBACK"},
		{PINGREQ, "PINGREQ"},
		{PINGRESP, "PINGRESP"},
		{DISCONNECT, "DISCONNECT"},
		{AUTH, "AUTH"},
		{PacketType(16), "UNKNOWN"},
		{PacketType(255), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.packetType.String())
		})
	}
}
