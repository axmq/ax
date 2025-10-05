package encoding

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	MaxUTF8StringLen    = 65535
	MaxPacketID         = 65535
	MaxRealisticPayload = 65000
)

func TestEncodeConnectPacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnectPacket
		wantErr bool
	}{
		{
			name: "empty client ID",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				KeepAlive:       60,
				ClientID:        "",
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "max client ID length",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				KeepAlive:       60,
				ClientID:        strings.Repeat("a", MaxUTF8StringLen),
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "zero keep alive",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				KeepAlive:       0,
				ClientID:        "test",
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "max keep alive",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				KeepAlive:       65535,
				ClientID:        "test",
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "will message with large payload",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				WillFlag:        true,
				WillQoS:         QoS2,
				WillRetain:      true,
				KeepAlive:       60,
				ClientID:        "test",
				WillTopic:       "will/topic",
				WillPayload:     make([]byte, MaxRealisticPayload),
				Properties:      Properties{},
				WillProperties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "will message with empty payload",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				WillFlag:        true,
				WillQoS:         QoS0,
				WillRetain:      false,
				KeepAlive:       60,
				ClientID:        "test",
				WillTopic:       "will/topic",
				WillPayload:     []byte{},
				Properties:      Properties{},
				WillProperties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "max username length",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				UsernameFlag:    true,
				KeepAlive:       60,
				ClientID:        "test",
				Username:        strings.Repeat("u", MaxUTF8StringLen),
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "large password length",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				UsernameFlag:    true,
				PasswordFlag:    true,
				KeepAlive:       60,
				ClientID:        "test",
				Username:        "user",
				Password:        bytes.Repeat([]byte{0xFF}, MaxRealisticPayload),
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "all flags enabled with max data",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				WillFlag:        true,
				WillQoS:         QoS2,
				WillRetain:      true,
				UsernameFlag:    true,
				PasswordFlag:    true,
				KeepAlive:       30000,
				ClientID:        strings.Repeat("c", 1000),
				WillTopic:       strings.Repeat("t", 1000),
				WillPayload:     bytes.Repeat([]byte("will"), 1000),
				Username:        strings.Repeat("u", 1000),
				Password:        bytes.Repeat([]byte{0xAB}, 1000),
				Properties:      Properties{},
				WillProperties:  Properties{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, buf.Len(), 0)

				fh, err := ParseFixedHeader(&buf)
				require.NoError(t, err)
				assert.Equal(t, CONNECT, fh.Type)
			}
		})
	}
}

func TestEncodePublishPacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		packet  *PublishPacket
		wantErr bool
	}{
		{
			name: "QoS0 with empty topic",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "",
				Payload:     []byte("data"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS0 with max topic length",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   strings.Repeat("t", MaxUTF8StringLen),
				Payload:     []byte("data"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS0 with zero payload",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "topic",
				Payload:     []byte{},
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS0 with nil payload",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "topic",
				Payload:     nil,
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS0 with large payload 1MB",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "topic",
				Payload:     make([]byte, 1024*1024),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS1 with packet ID 1",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS1},
				TopicName:   "topic",
				PacketID:    1,
				Payload:     []byte("data"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS1 with max packet ID",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS1},
				TopicName:   "topic",
				PacketID:    MaxPacketID,
				Payload:     []byte("data"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS2 with retain and DUP flags",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{
					QoS:    QoS2,
					Retain: true,
					DUP:    true,
				},
				TopicName:  "topic",
				PacketID:   12345,
				Payload:    []byte("data"),
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "binary payload with all byte values",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "binary",
				Payload:     createAllByteValues(),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "single byte payload",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "topic",
				Payload:     []byte{0x42},
				Properties:  Properties{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, buf.Len(), 0)

				fh, err := ParseFixedHeader(&buf)
				require.NoError(t, err)
				assert.Equal(t, PUBLISH, fh.Type)
			}
		})
	}
}

func TestEncodeSubscribePacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		packet  *SubscribePacket
		wantErr bool
	}{
		{
			name: "single subscription QoS0",
			packet: &SubscribePacket{
				PacketID: 1,
				Subscriptions: []Subscription{
					{TopicFilter: "test/topic", QoS: QoS0},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "single subscription QoS2 with all options",
			packet: &SubscribePacket{
				PacketID: 100,
				Subscriptions: []Subscription{
					{
						TopicFilter:       "test/topic",
						QoS:               QoS2,
						NoLocal:           true,
						RetainAsPublished: true,
						RetainHandling:    2,
					},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "max subscriptions with varied QoS",
			packet: &SubscribePacket{
				PacketID: MaxPacketID,
				Subscriptions: []Subscription{
					{TopicFilter: "topic/0", QoS: QoS0},
					{TopicFilter: "topic/1", QoS: QoS1},
					{TopicFilter: "topic/2", QoS: QoS2},
					{TopicFilter: "topic/3", QoS: QoS0, NoLocal: true},
					{TopicFilter: "topic/4", QoS: QoS1, RetainAsPublished: true},
					{TopicFilter: "topic/5", QoS: QoS2, RetainHandling: 1},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "subscription with wildcard plus",
			packet: &SubscribePacket{
				PacketID: 1,
				Subscriptions: []Subscription{
					{TopicFilter: "test/+/topic", QoS: QoS1},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "subscription with wildcard hash",
			packet: &SubscribePacket{
				PacketID: 1,
				Subscriptions: []Subscription{
					{TopicFilter: "test/#", QoS: QoS2},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "max topic filter length",
			packet: &SubscribePacket{
				PacketID: 1,
				Subscriptions: []Subscription{
					{TopicFilter: strings.Repeat("t", MaxUTF8StringLen), QoS: QoS1},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, buf.Len(), 0)

				fh, err := ParseFixedHeader(&buf)
				require.NoError(t, err)
				assert.Equal(t, SUBSCRIBE, fh.Type)
				assert.Equal(t, byte(0x02), fh.Flags)
			}
		})
	}
}

func TestEncodeUnsubscribePacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		packet  *UnsubscribePacket
		wantErr bool
	}{
		{
			name: "single topic filter",
			packet: &UnsubscribePacket{
				PacketID:     1,
				TopicFilters: []string{"test/topic"},
				Properties:   Properties{},
			},
			wantErr: false,
		},
		{
			name: "multiple topic filters",
			packet: &UnsubscribePacket{
				PacketID:     MaxPacketID,
				TopicFilters: []string{"topic/1", "topic/2", "topic/3"},
				Properties:   Properties{},
			},
			wantErr: false,
		},
		{
			name: "max topic filter length",
			packet: &UnsubscribePacket{
				PacketID:     1,
				TopicFilters: []string{strings.Repeat("t", MaxUTF8StringLen)},
				Properties:   Properties{},
			},
			wantErr: false,
		},
		{
			name: "wildcard patterns",
			packet: &UnsubscribePacket{
				PacketID:     1,
				TopicFilters: []string{"test/+/topic", "test/#"},
				Properties:   Properties{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, buf.Len(), 0)

				fh, err := ParseFixedHeader(&buf)
				require.NoError(t, err)
				assert.Equal(t, UNSUBSCRIBE, fh.Type)
				assert.Equal(t, byte(0x02), fh.Flags)
			}
		})
	}
}

func TestEncodeAckPackets_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		encodeFunc func() ([]byte, error)
		packetType PacketType
	}{
		{
			name: "PUBACK with reason success",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PubackPacket{
					PacketID:   1,
					ReasonCode: ReasonSuccess,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PUBACK,
		},
		{
			name: "PUBACK with error reason code",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PubackPacket{
					PacketID:   MaxPacketID,
					ReasonCode: ReasonUnspecifiedError,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PUBACK,
		},
		{
			name: "PUBREC with packet ID 1",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PubrecPacket{
					PacketID:   1,
					ReasonCode: ReasonSuccess,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PUBREC,
		},
		{
			name: "PUBREL with required flags",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PubrelPacket{
					PacketID:   100,
					ReasonCode: ReasonSuccess,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PUBREL,
		},
		{
			name: "PUBCOMP with max packet ID",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PubcompPacket{
					PacketID:   MaxPacketID,
					ReasonCode: ReasonSuccess,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PUBCOMP,
		},
		{
			name: "SUBACK with multiple reason codes",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&SubackPacket{
					PacketID:    1,
					ReasonCodes: []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2},
					Properties:  Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: SUBACK,
		},
		{
			name: "SUBACK with failure reason codes",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&SubackPacket{
					PacketID:    1,
					ReasonCodes: []ReasonCode{ReasonGrantedQoS1, ReasonUnspecifiedError, ReasonNotAuthorized},
					Properties:  Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: SUBACK,
		},
		{
			name: "UNSUBACK with success codes",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&UnsubackPacket{
					PacketID:    MaxPacketID,
					ReasonCodes: []ReasonCode{ReasonSuccess, ReasonSuccess},
					Properties:  Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: UNSUBACK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.encodeFunc()
			require.NoError(t, err)
			assert.Greater(t, len(data), 0)

			fh, _, err := ParseFixedHeaderFromBytes(data)
			require.NoError(t, err)
			assert.Equal(t, tt.packetType, fh.Type)
		})
	}
}

func TestEncodeControlPackets_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		encodeFunc func() ([]byte, error)
		packetType PacketType
	}{
		{
			name: "PINGREQ",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PingreqPacket{}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PINGREQ,
		},
		{
			name: "PINGRESP",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&PingrespPacket{}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: PINGRESP,
		},
		{
			name: "DISCONNECT with normal disconnection",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&DisconnectPacket{
					ReasonCode: ReasonNormalDisconnection,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: DISCONNECT,
		},
		{
			name: "DISCONNECT with error reason code",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&DisconnectPacket{
					ReasonCode: ReasonProtocolError,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: DISCONNECT,
		},
		{
			name: "AUTH with continue authentication",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&AuthPacket{
					ReasonCode: ReasonContinueAuthentication,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: AUTH,
		},
		{
			name: "AUTH with re-authenticate",
			encodeFunc: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&AuthPacket{
					ReasonCode: ReasonReAuthenticate,
					Properties: Properties{},
				}).Encode(&buf)
				return buf.Bytes(), err
			},
			packetType: AUTH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.encodeFunc()
			require.NoError(t, err)
			assert.Greater(t, len(data), 0)

			fh, _, err := ParseFixedHeaderFromBytes(data)
			require.NoError(t, err)
			assert.Equal(t, tt.packetType, fh.Type)
		})
	}
}

func TestEncodeConnackPacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnackPacket
		wantErr bool
	}{
		{
			name: "success without session",
			packet: &ConnackPacket{
				SessionPresent: false,
				ReasonCode:     ReasonSuccess,
				Properties:     Properties{},
			},
			wantErr: false,
		},
		{
			name: "success with session present",
			packet: &ConnackPacket{
				SessionPresent: true,
				ReasonCode:     ReasonSuccess,
				Properties:     Properties{},
			},
			wantErr: false,
		},
		{
			name: "all error reason codes",
			packet: &ConnackPacket{
				SessionPresent: false,
				ReasonCode:     ReasonBanned,
				Properties:     Properties{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, buf.Len(), 0)

				fh, err := ParseFixedHeader(&buf)
				require.NoError(t, err)
				assert.Equal(t, CONNACK, fh.Type)
			}
		})
	}
}

func TestEncodePublishPacket_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "empty payload",
			payload: []byte{},
		},
		{
			name:    "1 byte payload",
			payload: []byte{0x42},
		},
		{
			name:    "127 bytes payload",
			payload: make([]byte, 127),
		},
		{
			name:    "128 bytes payload",
			payload: make([]byte, 128),
		},
		{
			name:    "16383 bytes payload",
			payload: make([]byte, 16383),
		},
		{
			name:    "16384 bytes payload",
			payload: make([]byte, 16384),
		},
		{
			name:    "65535 bytes payload",
			payload: make([]byte, 65535),
		},
		{
			name:    "1MB payload",
			payload: make([]byte, 1024*1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     tt.payload,
				Properties:  Properties{},
			}

			var buf bytes.Buffer
			err := packet.Encode(&buf)
			require.NoError(t, err)

			fh, err := ParseFixedHeader(&buf)
			require.NoError(t, err)
			assert.Equal(t, PUBLISH, fh.Type)
			assert.Greater(t, int(fh.RemainingLength), 0)
		})
	}
}

func createAllByteValues() []byte {
	result := make([]byte, 256)
	for i := 0; i < 256; i++ {
		result[i] = byte(i)
	}
	return result
}
