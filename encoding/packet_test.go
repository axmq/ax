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

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name       string
		packetType PacketType
		flags      byte
		expectErr  bool
	}{
		{
			name:       "CONNECT valid flags",
			packetType: CONNECT,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "CONNECT invalid flags",
			packetType: CONNECT,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "CONNACK valid flags",
			packetType: CONNACK,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "CONNACK invalid flags",
			packetType: CONNACK,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "PUBREL valid flags",
			packetType: PUBREL,
			flags:      0x02,
			expectErr:  false,
		},
		{
			name:       "PUBREL invalid flags",
			packetType: PUBREL,
			flags:      0x00,
			expectErr:  true,
		},
		{
			name:       "SUBSCRIBE valid flags",
			packetType: SUBSCRIBE,
			flags:      0x02,
			expectErr:  false,
		},
		{
			name:       "SUBSCRIBE invalid flags",
			packetType: SUBSCRIBE,
			flags:      0x00,
			expectErr:  true,
		},
		{
			name:       "UNSUBSCRIBE valid flags",
			packetType: UNSUBSCRIBE,
			flags:      0x02,
			expectErr:  false,
		},
		{
			name:       "UNSUBSCRIBE invalid flags",
			packetType: UNSUBSCRIBE,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "PUBACK valid flags",
			packetType: PUBACK,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "PUBACK invalid flags",
			packetType: PUBACK,
			flags:      0x03,
			expectErr:  true,
		},
		{
			name:       "PUBREC valid flags",
			packetType: PUBREC,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "PUBREC invalid flags",
			packetType: PUBREC,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "PUBCOMP valid flags",
			packetType: PUBCOMP,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "PUBCOMP invalid flags",
			packetType: PUBCOMP,
			flags:      0x02,
			expectErr:  true,
		},
		{
			name:       "SUBACK valid flags",
			packetType: SUBACK,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "SUBACK invalid flags",
			packetType: SUBACK,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "UNSUBACK valid flags",
			packetType: UNSUBACK,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "UNSUBACK invalid flags",
			packetType: UNSUBACK,
			flags:      0x02,
			expectErr:  true,
		},
		{
			name:       "PINGREQ valid flags",
			packetType: PINGREQ,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "PINGREQ invalid flags",
			packetType: PINGREQ,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "PINGRESP valid flags",
			packetType: PINGRESP,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "PINGRESP invalid flags",
			packetType: PINGRESP,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "DISCONNECT valid flags",
			packetType: DISCONNECT,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "DISCONNECT invalid flags",
			packetType: DISCONNECT,
			flags:      0x02,
			expectErr:  true,
		},
		{
			name:       "AUTH valid flags",
			packetType: AUTH,
			flags:      0x00,
			expectErr:  false,
		},
		{
			name:       "AUTH invalid flags",
			packetType: AUTH,
			flags:      0x01,
			expectErr:  true,
		},
		{
			name:       "PUBLISH allows any flags",
			packetType: PUBLISH,
			flags:      0x0D,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlags(tt.packetType, tt.flags)
			if tt.expectErr {
				assert.ErrorIs(t, err, ErrInvalidFlags)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseConnectPacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectErr bool
	}{
		{
			name:      "EOF on protocol name",
			data:      []byte{0x00},
			expectErr: true,
		},
		{
			name:      "EOF on protocol version",
			data:      []byte{0x00, 0x04, 'M', 'Q', 'T', 'T'},
			expectErr: true,
		},
		{
			name:      "EOF on connect flags",
			data:      []byte{0x00, 0x04, 'M', 'Q', 'T', 'T', 0x05},
			expectErr: true,
		},
		{
			name:      "EOF on keep alive",
			data:      []byte{0x00, 0x04, 'M', 'Q', 'T', 'T', 0x05, 0x02, 0x00},
			expectErr: true,
		},
		{
			name:      "EOF on properties",
			data:      []byte{0x00, 0x04, 'M', 'Q', 'T', 'T', 0x05, 0x02, 0x00, 0x3C},
			expectErr: true,
		},
		{
			name:      "EOF on client ID",
			data:      []byte{0x00, 0x04, 'M', 'Q', 'T', 'T', 0x05, 0x02, 0x00, 0x3C, 0x00},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            CONNECT,
				RemainingLength: uint32(len(tt.data)),
			}

			_, err := ParseConnectPacket(r, fh)
			assert.Error(t, err)
		})
	}
}

func TestParseConnectPacket_WillErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "EOF on will properties",
			data: []byte{
				0x00, 0x04, 'M', 'Q', 'T', 'T',
				0x05,
				0x04,
				0x00, 0x3C,
				0x00,
				0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
			},
		},
		{
			name: "EOF on will topic",
			data: []byte{
				0x00, 0x04, 'M', 'Q', 'T', 'T',
				0x05,
				0x04,
				0x00, 0x3C,
				0x00,
				0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
				0x00,
			},
		},
		{
			name: "EOF on will payload",
			data: []byte{
				0x00, 0x04, 'M', 'Q', 'T', 'T',
				0x05,
				0x04,
				0x00, 0x3C,
				0x00,
				0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
				0x00,
				0x00, 0x04, 'w', 'i', 'l', 'l',
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            CONNECT,
				RemainingLength: uint32(len(tt.data)),
			}

			_, err := ParseConnectPacket(r, fh)
			assert.Error(t, err)
		})
	}
}

func TestParseConnectPacket_UsernamePasswordErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "EOF on username",
			data: []byte{
				0x00, 0x04, 'M', 'Q', 'T', 'T',
				0x05,
				0x80,
				0x00, 0x3C,
				0x00,
				0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
			},
		},
		{
			name: "EOF on password",
			data: []byte{
				0x00, 0x04, 'M', 'Q', 'T', 'T',
				0x05,
				0xC0,
				0x00, 0x3C,
				0x00,
				0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
				0x00, 0x04, 'u', 's', 'e', 'r',
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            CONNECT,
				RemainingLength: uint32(len(tt.data)),
			}

			_, err := ParseConnectPacket(r, fh)
			assert.Error(t, err)
		})
	}
}

func TestBuildPublishFlags(t *testing.T) {
	tests := []struct {
		name     string
		dup      bool
		qos      QoS
		retain   bool
		expected byte
	}{
		{"QoS0", false, QoS0, false, 0x00},
		{"QoS0 Retain", false, QoS0, true, 0x01},
		{"QoS1", false, QoS1, false, 0x02},
		{"QoS1 Retain", false, QoS1, true, 0x03},
		{"QoS2", false, QoS2, false, 0x04},
		{"QoS2 Retain", false, QoS2, true, 0x05},
		{"DUP QoS0", true, QoS0, false, 0x08},
		{"DUP QoS0 Retain", true, QoS0, true, 0x09},
		{"DUP QoS1", true, QoS1, false, 0x0A},
		{"DUP QoS1 Retain", true, QoS1, true, 0x0B},
		{"DUP QoS2", true, QoS2, false, 0x0C},
		{"DUP QoS2 Retain", true, QoS2, true, 0x0D},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := &FixedHeader{
				DUP:    tt.dup,
				QoS:    tt.qos,
				Retain: tt.retain,
			}
			result := fh.BuildPublishFlags()
			assert.Equal(t, tt.expected, result)
		})
	}
}
