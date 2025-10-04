package encoding

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFixedHeader_ValidPackets(t *testing.T) {
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
			name:           "CONNECT with zero remaining length",
			input:          []byte{0x10, 0x00},
			expectedType:   CONNECT,
			expectedFlags:  0x00,
			expectedRemLen: 0,
		},
		{
			name:           "CONNACK with small remaining length",
			input:          []byte{0x20, 0x02},
			expectedType:   CONNACK,
			expectedFlags:  0x00,
			expectedRemLen: 2,
		},
		{
			name:           "PUBLISH QoS0 without DUP or Retain",
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
			name:           "PUBLISH QoS1 with DUP and Retain",
			input:          []byte{0x3B, 0x08},
			expectedType:   PUBLISH,
			expectedFlags:  0x0B,
			expectedRemLen: 8,
			expectedDUP:    true,
			expectedQoS:    QoS1,
			expectedRetain: true,
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
		{
			name:           "AUTH",
			input:          []byte{0xF0, 0x00},
			expectedType:   AUTH,
			expectedFlags:  0x00,
			expectedRemLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			header, err := ParseFixedHeader(r)
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

func TestParseFixedHeader_VariableByteInteger(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedRemLen uint32
	}{
		{
			name:           "1-byte: 0",
			input:          []byte{0x10, 0x00},
			expectedRemLen: 0,
		},
		{
			name:           "1-byte: 127",
			input:          []byte{0x10, 0x7F},
			expectedRemLen: 127,
		},
		{
			name:           "2-byte: 128",
			input:          []byte{0x10, 0x80, 0x01},
			expectedRemLen: 128,
		},
		{
			name:           "2-byte: 16383",
			input:          []byte{0x10, 0xFF, 0x7F},
			expectedRemLen: 16383,
		},
		{
			name:           "3-byte: 16384",
			input:          []byte{0x10, 0x80, 0x80, 0x01},
			expectedRemLen: 16384,
		},
		{
			name:           "3-byte: 2097151",
			input:          []byte{0x10, 0xFF, 0xFF, 0x7F},
			expectedRemLen: 2097151,
		},
		{
			name:           "4-byte: 2097152",
			input:          []byte{0x10, 0x80, 0x80, 0x80, 0x01},
			expectedRemLen: 2097152,
		},
		{
			name:           "4-byte: 268435455 (max)",
			input:          []byte{0x10, 0xFF, 0xFF, 0xFF, 0x7F},
			expectedRemLen: 268435455,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			header, err := ParseFixedHeader(r)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRemLen, header.RemainingLength)
		})
	}
}

func TestParseFixedHeader_InvalidPackets(t *testing.T) {
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
			name:        "Only first byte",
			input:       []byte{0x10},
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Reserved packet type (0)",
			input:       []byte{0x00, 0x00},
			expectedErr: ErrInvalidReservedType,
		},
		{
			name:        "Invalid packet type (16)",
			input:       []byte{0xFF, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "CONNECT with invalid flags",
			input:       []byte{0x11, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "CONNACK with invalid flags",
			input:       []byte{0x21, 0x00},
			expectedErr: ErrInvalidFlags,
		},
		{
			name:        "PUBLISH with invalid QoS (3)",
			input:       []byte{0x36, 0x00},
			expectedErr: ErrInvalidQoS,
		},
		{
			name:        "PUBACK with invalid flags",
			input:       []byte{0x41, 0x00},
			expectedErr: ErrInvalidFlags,
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
		{
			name:        "Malformed remaining length - 5 bytes",
			input:       []byte{0x10, 0x80, 0x80, 0x80, 0x80, 0x01},
			expectedErr: ErrMalformedVariableByteInteger,
		},
		{
			name:        "Incomplete variable byte integer",
			input:       []byte{0x10, 0x80},
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Incomplete 3-byte variable byte integer",
			input:       []byte{0x10, 0x80, 0x80},
			expectedErr: ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			_, err := ParseFixedHeader(r)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestParseFixedHeaderFromBytes(t *testing.T) {
	tests := []struct {
		name              string
		input             []byte
		expectedType      PacketType
		expectedRemLen    uint32
		expectedBytesRead int
		expectError       bool
		expectedErr       error
	}{
		{
			name:              "CONNECT",
			input:             []byte{0x10, 0x0A},
			expectedType:      CONNECT,
			expectedRemLen:    10,
			expectedBytesRead: 2,
		},
		{
			name:              "PUBLISH with 2-byte remaining length",
			input:             []byte{0x30, 0x80, 0x01},
			expectedType:      PUBLISH,
			expectedRemLen:    128,
			expectedBytesRead: 3,
		},
		{
			name:              "PUBLISH with 4-byte remaining length",
			input:             []byte{0x30, 0xFF, 0xFF, 0xFF, 0x7F},
			expectedType:      PUBLISH,
			expectedRemLen:    268435455,
			expectedBytesRead: 5,
		},
		{
			name:        "Empty input",
			input:       []byte{},
			expectError: true,
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Only one byte",
			input:       []byte{0x10},
			expectError: true,
			expectedErr: ErrUnexpectedEOF,
		},
		{
			name:        "Reserved type",
			input:       []byte{0x00, 0x00},
			expectError: true,
			expectedErr: ErrInvalidReservedType,
		},
		{
			name:        "Invalid QoS",
			input:       []byte{0x36, 0x00},
			expectError: true,
			expectedErr: ErrInvalidQoS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, bytesRead, err := ParseFixedHeaderFromBytes(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, header.Type)
			assert.Equal(t, tt.expectedRemLen, header.RemainingLength)
			assert.Equal(t, tt.expectedBytesRead, bytesRead)
		})
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		typ      PacketType
		expected string
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
			assert.Equal(t, tt.expected, tt.typ.String())
		})
	}
}

func TestQoSString(t *testing.T) {
	tests := []struct {
		qos      QoS
		expected string
	}{
		{QoS0, "QoS0"},
		{QoS1, "QoS1"},
		{QoS2, "QoS2"},
		{QoS(3), "INVALID"},
		{QoS(255), "INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.qos.String())
		})
	}
}

func TestQoSIsValid(t *testing.T) {
	tests := []struct {
		qos      QoS
		expected bool
	}{
		{QoS0, true},
		{QoS1, true},
		{QoS2, true},
		{QoS(3), false},
		{QoS(4), false},
		{QoS(255), false},
	}

	for _, tt := range tests {
		t.Run(tt.qos.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.qos.IsValid())
		})
	}
}

func TestParsePUBLISHFlags(t *testing.T) {
	tests := []struct {
		name      string
		flags     byte
		expectDUP bool
		expectQoS QoS
		expectRet bool
		expectErr bool
	}{
		{"DUP=0 QoS=0 Retain=0", 0x00, false, QoS0, false, false},
		{"DUP=0 QoS=0 Retain=1", 0x01, false, QoS0, true, false},
		{"DUP=0 QoS=1 Retain=0", 0x02, false, QoS1, false, false},
		{"DUP=0 QoS=1 Retain=1", 0x03, false, QoS1, true, false},
		{"DUP=0 QoS=2 Retain=0", 0x04, false, QoS2, false, false},
		{"DUP=0 QoS=2 Retain=1", 0x05, false, QoS2, true, false},
		{"DUP=0 QoS=3 Retain=0", 0x06, false, QoS(3), false, true},
		{"DUP=0 QoS=3 Retain=1", 0x07, false, QoS(3), true, true},
		{"DUP=1 QoS=0 Retain=0", 0x08, true, QoS0, false, false},
		{"DUP=1 QoS=0 Retain=1", 0x09, true, QoS0, true, false},
		{"DUP=1 QoS=1 Retain=0", 0x0A, true, QoS1, false, false},
		{"DUP=1 QoS=1 Retain=1", 0x0B, true, QoS1, true, false},
		{"DUP=1 QoS=2 Retain=0", 0x0C, true, QoS2, false, false},
		{"DUP=1 QoS=2 Retain=1", 0x0D, true, QoS2, true, false},
		{"DUP=1 QoS=3 Retain=0", 0x0E, true, QoS(3), false, true},
		{"DUP=1 QoS=3 Retain=1", 0x0F, true, QoS(3), true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte{0x30 | tt.flags, 0x00}
			r := bytes.NewReader(input)
			header, err := ParseFixedHeader(r)

			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidQoS)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectDUP, header.DUP)
			assert.Equal(t, tt.expectQoS, header.QoS)
			assert.Equal(t, tt.expectRet, header.Retain)
		})
	}
}

func TestEOFHandling(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"Empty", []byte{}},
		{"Only type byte", []byte{0x10}},
		{"Incomplete VBI 1", []byte{0x10, 0x80}},
		{"Incomplete VBI 2", []byte{0x10, 0x80, 0x80}},
		{"Incomplete VBI 3", []byte{0x10, 0x80, 0x80, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			_, err := ParseFixedHeader(r)
			require.Error(t, err)
			assert.True(t, err == ErrUnexpectedEOF || err == io.EOF)
		})
	}
}
