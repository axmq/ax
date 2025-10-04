package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConnackPacket(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		expectedSession bool
		expectedReason  ReasonCode
		expectError     bool
	}{
		{
			name: "Success with session present",
			data: []byte{
				0x01, // Session present
				0x00, // Reason: Success
				0x00, // Properties length: 0
			},
			expectedSession: true,
			expectedReason:  ReasonSuccess,
		},
		{
			name: "Success without session",
			data: []byte{
				0x00, // No session present
				0x00, // Reason: Success
				0x00, // Properties length: 0
			},
			expectedSession: false,
			expectedReason:  ReasonSuccess,
		},
		{
			name: "Connection refused - bad username/password",
			data: []byte{
				0x00, // No session present
				0x86, // Reason: Bad username or password
				0x00, // Properties length: 0
			},
			expectedSession: false,
			expectedReason:  ReasonBadUsernameOrPassword,
		},
		{
			name: "Invalid flags - reserved bits set",
			data: []byte{
				0x02, // Invalid: reserved bit set
				0x00,
				0x00,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            CONNACK,
				RemainingLength: uint32(len(tt.data)),
			}

			pkt, err := ParseConnackPacket(r, fh)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedSession, pkt.SessionPresent)
			assert.Equal(t, tt.expectedReason, pkt.ReasonCode)
		})
	}
}

func TestParsePublishPacket_QoS0(t *testing.T) {
	// PUBLISH packet with QoS 0
	// Topic: "test/topic"
	// Payload: "hello"
	data := []byte{
		0x00, 0x0A, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // Topic name
		0x00,                    // Properties length: 0
		'h', 'e', 'l', 'l', 'o', // Payload
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            PUBLISH,
		QoS:             QoS0,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParsePublishPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, "test/topic", pkt.TopicName)
	assert.Equal(t, uint16(0), pkt.PacketID)
	assert.Equal(t, []byte("hello"), pkt.Payload)
}

func TestParsePublishPacket_QoS1(t *testing.T) {
	// PUBLISH packet with QoS 1
	// Topic: "test/topic"
	// Packet ID: 1234
	// Payload: "hello"
	data := []byte{
		0x00, 0x0A, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // Topic name
		0x04, 0xD2, // Packet ID: 1234
		0x00,                    // Properties length: 0
		'h', 'e', 'l', 'l', 'o', // Payload
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            PUBLISH,
		QoS:             QoS1,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParsePublishPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, "test/topic", pkt.TopicName)
	assert.Equal(t, uint16(1234), pkt.PacketID)
	assert.Equal(t, []byte("hello"), pkt.Payload)
}

func TestParsePublishPacket_WithProperties(t *testing.T) {
	// PUBLISH packet with properties
	data := []byte{
		0x00, 0x05, 't', 'e', 's', 't', '1', // Topic name: "test1"
		0x00, 0x01, // Packet ID: 1
		0x02, 0x01, 0x01, // Properties: PayloadFormatIndicator = 1
		'h', 'i', // Payload
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            PUBLISH,
		QoS:             QoS1,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParsePublishPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, "test1", pkt.TopicName)
	assert.Len(t, pkt.Properties.Properties, 1)
	assert.Equal(t, PropPayloadFormatIndicator, pkt.Properties.Properties[0].ID)
}

func TestParsePubackPacket(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		remainingLen   uint32
		expectedPktID  uint16
		expectedReason ReasonCode
	}{
		{
			name: "Minimal PUBACK - no reason code",
			data: []byte{
				0x00, 0x01, // Packet ID: 1
			},
			remainingLen:   2,
			expectedPktID:  1,
			expectedReason: ReasonSuccess,
		},
		{
			name: "PUBACK with reason code",
			data: []byte{
				0x00, 0x01, // Packet ID: 1
				0x00, // Reason: Success
			},
			remainingLen:   3,
			expectedPktID:  1,
			expectedReason: ReasonSuccess,
		},
		{
			name: "PUBACK with reason code and properties",
			data: []byte{
				0x00, 0x01, // Packet ID: 1
				0x00, // Reason: Success
				0x00, // Properties length: 0
			},
			remainingLen:   4,
			expectedPktID:  1,
			expectedReason: ReasonSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            PUBACK,
				RemainingLength: tt.remainingLen,
			}

			pkt, err := ParsePubackPacket(r, fh)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPktID, pkt.PacketID)
			assert.Equal(t, tt.expectedReason, pkt.ReasonCode)
		})
	}
}

func TestParseSubscribePacket(t *testing.T) {
	// SUBSCRIBE packet with multiple subscriptions
	data := []byte{
		0x00, 0x0A, // Packet ID: 10
		0x00, // Properties length: 0
		// First subscription
		0x00, 0x07, 't', 'e', 's', 't', '/', '#', '1', // Topic filter: "test/#1"
		0x01, // Options: QoS 1
		// Second subscription
		0x00, 0x05, 't', 'o', 'p', 'i', 'c', // Topic filter: "topic"
		0x06, // Options: QoS 2, NoLocal
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            SUBSCRIBE,
		Flags:           0x02,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseSubscribePacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, uint16(10), pkt.PacketID)
	assert.Len(t, pkt.Subscriptions, 2)

	// First subscription
	assert.Equal(t, "test/#1", pkt.Subscriptions[0].TopicFilter)
	assert.Equal(t, QoS1, pkt.Subscriptions[0].QoS)
	assert.False(t, pkt.Subscriptions[0].NoLocal)

	// Second subscription
	assert.Equal(t, "topic", pkt.Subscriptions[1].TopicFilter)
	assert.Equal(t, QoS2, pkt.Subscriptions[1].QoS)
	assert.True(t, pkt.Subscriptions[1].NoLocal)
}

func TestParseSubackPacket(t *testing.T) {
	// SUBACK packet with multiple reason codes
	data := []byte{
		0x00, 0x0A, // Packet ID: 10
		0x00, // Properties length: 0
		0x00, // Reason code: Granted QoS 0
		0x01, // Reason code: Granted QoS 1
		0x02, // Reason code: Granted QoS 2
		0x80, // Reason code: Unspecified error
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            SUBACK,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseSubackPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, uint16(10), pkt.PacketID)
	assert.Len(t, pkt.ReasonCodes, 4)
	assert.Equal(t, ReasonGrantedQoS0, pkt.ReasonCodes[0])
	assert.Equal(t, ReasonGrantedQoS1, pkt.ReasonCodes[1])
	assert.Equal(t, ReasonGrantedQoS2, pkt.ReasonCodes[2])
	assert.Equal(t, ReasonUnspecifiedError, pkt.ReasonCodes[3])
}

func TestParseUnsubscribePacket(t *testing.T) {
	// UNSUBSCRIBE packet with multiple topic filters
	data := []byte{
		0x00, 0x05, // Packet ID: 5
		0x00,                                          // Properties length: 0
		0x00, 0x07, 't', 'e', 's', 't', '/', '#', '1', // Topic filter: "test/#1"
		0x00, 0x05, 't', 'o', 'p', 'i', 'c', // Topic filter: "topic"
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            UNSUBSCRIBE,
		Flags:           0x02,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseUnsubscribePacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, uint16(5), pkt.PacketID)
	assert.Len(t, pkt.TopicFilters, 2)
	assert.Equal(t, "test/#1", pkt.TopicFilters[0])
	assert.Equal(t, "topic", pkt.TopicFilters[1])
}

func TestParseUnsubackPacket(t *testing.T) {
	// UNSUBACK packet
	data := []byte{
		0x00, 0x05, // Packet ID: 5
		0x00, // Properties length: 0
		0x00, // Reason code: Success
		0x11, // Reason code: No subscription existed
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            UNSUBACK,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseUnsubackPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, uint16(5), pkt.PacketID)
	assert.Len(t, pkt.ReasonCodes, 2)
	assert.Equal(t, ReasonSuccess, pkt.ReasonCodes[0])
	assert.Equal(t, ReasonNoSubscriptionExisted, pkt.ReasonCodes[1])
}

func TestParseDisconnectPacket(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		remainingLen   uint32
		expectedReason ReasonCode
	}{
		{
			name:           "Normal disconnection - no reason code",
			data:           []byte{},
			remainingLen:   0,
			expectedReason: ReasonNormalDisconnection,
		},
		{
			name: "Disconnection with reason code",
			data: []byte{
				0x00, // Reason: Normal disconnection
			},
			remainingLen:   1,
			expectedReason: ReasonNormalDisconnection,
		},
		{
			name: "Disconnection with reason and properties",
			data: []byte{
				0x8E, // Reason: Session taken over
				0x00, // Properties length: 0
			},
			remainingLen:   2,
			expectedReason: ReasonSessionTakenOver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            DISCONNECT,
				RemainingLength: tt.remainingLen,
			}

			pkt, err := ParseDisconnectPacket(r, fh)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedReason, pkt.ReasonCode)
		})
	}
}

func TestParseAuthPacket(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		remainingLen   uint32
		expectedReason ReasonCode
		expectError    bool
	}{
		{
			name:         "AUTH with no data - invalid",
			data:         []byte{},
			remainingLen: 0,
			expectError:  true,
		},
		{
			name: "AUTH with reason code",
			data: []byte{
				0x18, // Reason: Continue authentication
			},
			remainingLen:   1,
			expectedReason: ReasonContinueAuthentication,
		},
		{
			name: "AUTH with reason and properties",
			data: []byte{
				0x19, // Reason: Re-authenticate
				0x00, // Properties length: 0
			},
			remainingLen:   2,
			expectedReason: ReasonReAuthenticate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			fh := &FixedHeader{
				Type:            AUTH,
				RemainingLength: tt.remainingLen,
			}

			pkt, err := ParseAuthPacket(r, fh)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedReason, pkt.ReasonCode)
		})
	}
}

func TestParsePingreqPacket(t *testing.T) {
	fh := &FixedHeader{
		Type:            PINGREQ,
		RemainingLength: 0,
	}

	pkt, err := ParsePingreqPacket(fh)
	require.NoError(t, err)
	assert.Equal(t, PINGREQ, pkt.FixedHeader.Type)
}

func TestParsePingreqPacket_InvalidRemainingLength(t *testing.T) {
	fh := &FixedHeader{
		Type:            PINGREQ,
		RemainingLength: 1,
	}

	_, err := ParsePingreqPacket(fh)
	assert.ErrorIs(t, err, ErrMalformedPacket)
}

func TestParsePingrespPacket(t *testing.T) {
	fh := &FixedHeader{
		Type:            PINGRESP,
		RemainingLength: 0,
	}

	pkt, err := ParsePingrespPacket(fh)
	require.NoError(t, err)
	assert.Equal(t, PINGRESP, pkt.FixedHeader.Type)
}

func TestReasonCode_String(t *testing.T) {
	tests := []struct {
		code     ReasonCode
		expected string
	}{
		{ReasonSuccess, "Success"},
		{ReasonGrantedQoS1, "GrantedQoS1"},
		{ReasonMalformedPacket, "MalformedPacket"},
		{ReasonNotAuthorized, "NotAuthorized"},
		{ReasonCode(0xFF), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.String())
		})
	}
}

func TestParseConnectPacket(t *testing.T) {
	// CONNECT packet with minimal fields
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T', // Protocol name: "MQTT"
		0x05,       // Protocol version: 5
		0x02,       // Connect flags: Clean start
		0x00, 0x3C, // Keep alive: 60 seconds
		0x00,                                     // Properties length: 0
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't', // Client ID: "client"
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseConnectPacket(r, fh)
	require.NoError(t, err)
	assert.Equal(t, "MQTT", pkt.ProtocolName)
	assert.Equal(t, ProtocolVersion50, pkt.ProtocolVersion)
	assert.True(t, pkt.CleanStart)
	assert.Equal(t, uint16(60), pkt.KeepAlive)
	assert.Equal(t, "client", pkt.ClientID)
}

func TestParseConnectPacket_WithWill(t *testing.T) {
	// CONNECT packet with will message
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T', // Protocol name
		0x05,       // Protocol version: 5
		0x2E,       // Connect flags: Clean start (0x02), Will flag (0x04), Will QoS 1 (0x08), Will Retain (0x20) = 0x2E
		0x00, 0x3C, // Keep alive: 60
		0x00,                                     // Properties length: 0
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't', // Client ID
		0x00,                                                         // Will properties length: 0
		0x00, 0x0A, 'w', 'i', 'l', 'l', '/', 't', 'o', 'p', 'i', 'c', // Will topic (10 chars)
		0x00, 0x07, 'g', 'o', 'o', 'd', 'b', 'y', 'e', // Will payload (7 chars)
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseConnectPacket(r, fh)
	require.NoError(t, err)
	assert.True(t, pkt.WillFlag)
	assert.Equal(t, QoS1, pkt.WillQoS)
	assert.Equal(t, "will/topic", pkt.WillTopic)
	assert.Equal(t, []byte("goodbye"), pkt.WillPayload)
}

func TestParseConnectPacket_WithUsernamePassword(t *testing.T) {
	// CONNECT packet with username and password
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T', // Protocol name
		0x05,       // Protocol version: 5
		0xC2,       // Connect flags: Clean start, Username, Password
		0x00, 0x3C, // Keep alive: 60
		0x00,                                     // Properties length: 0
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't', // Client ID
		0x00, 0x04, 'u', 's', 'e', 'r', // Username
		0x00, 0x04, 'p', 'a', 's', 's', // Password
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	pkt, err := ParseConnectPacket(r, fh)
	require.NoError(t, err)
	assert.True(t, pkt.UsernameFlag)
	assert.True(t, pkt.PasswordFlag)
	assert.Equal(t, "user", pkt.Username)
	assert.Equal(t, []byte("pass"), pkt.Password)
}

func TestParseConnectPacket_InvalidProtocolName(t *testing.T) {
	// CONNECT with wrong protocol name
	data := []byte{
		0x00, 0x06, 'M', 'Q', 'I', 's', 'd', 'p', // Wrong protocol name
		0x05,       // Version
		0x02,       // Flags
		0x00, 0x3C, // Keep alive
		0x00, // Properties
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	_, err := ParseConnectPacket(r, fh)
	assert.ErrorIs(t, err, ErrInvalidProtocolName)
}

func TestParseConnectPacket_InvalidProtocolVersion(t *testing.T) {
	// CONNECT with unsupported version
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T',
		0x03, // Version 3 (not 5)
		0x02,
		0x00, 0x3C,
		0x00,
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	_, err := ParseConnectPacket(r, fh)
	assert.ErrorIs(t, err, ErrInvalidProtocolVersion)
}

func TestParseConnectPacket_ReservedBitSet(t *testing.T) {
	// CONNECT with reserved bit set (malformed)
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T',
		0x05,
		0x03, // Flags with reserved bit set
		0x00, 0x3C,
		0x00,
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
	}

	r := bytes.NewReader(data)
	fh := &FixedHeader{
		Type:            CONNECT,
		RemainingLength: uint32(len(data)),
	}

	_, err := ParseConnectPacket(r, fh)
	assert.ErrorIs(t, err, ErrMalformedPacket)
}
