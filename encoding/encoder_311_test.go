package encoding

import (
	"bytes"
	"testing"
)

func TestEncodeConnectPacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnectPacket311
		wantErr bool
	}{
		{
			name: "basic connect with clean session",
			packet: &ConnectPacket311{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion311,
				CleanSession:    true,
				KeepAlive:       60,
				ClientID:        "test-client",
			},
			wantErr: false,
		},
		{
			name: "connect with will message",
			packet: &ConnectPacket311{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion311,
				CleanSession:    true,
				WillFlag:        true,
				WillQoS:         QoS1,
				WillRetain:      true,
				KeepAlive:       60,
				ClientID:        "test-client",
				WillTopic:       "will/topic",
				WillPayload:     []byte("goodbye"),
			},
			wantErr: false,
		},
		{
			name: "connect with username and password",
			packet: &ConnectPacket311{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion311,
				CleanSession:    true,
				UsernameFlag:    true,
				PasswordFlag:    true,
				KeepAlive:       60,
				ClientID:        "test-client",
				Username:        "user",
				Password:        []byte("pass"),
			},
			wantErr: false,
		},
		{
			name: "connect with empty client ID and clean session",
			packet: &ConnectPacket311{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion311,
				CleanSession:    true,
				KeepAlive:       60,
				ClientID:        "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != CONNECT {
					t.Errorf("Expected packet type CONNECT, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodeConnackPacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnackPacket311
		wantErr bool
	}{
		{
			name: "connection accepted",
			packet: &ConnackPacket311{
				SessionPresent: false,
				ReturnCode:     ConnectAccepted311,
			},
			wantErr: false,
		},
		{
			name: "session present",
			packet: &ConnackPacket311{
				SessionPresent: true,
				ReturnCode:     ConnectAccepted311,
			},
			wantErr: false,
		},
		{
			name: "bad username or password",
			packet: &ConnackPacket311{
				SessionPresent: false,
				ReturnCode:     ConnectRefusedBadUsernamePassword311,
			},
			wantErr: false,
		},
		{
			name: "not authorized",
			packet: &ConnackPacket311{
				SessionPresent: false,
				ReturnCode:     ConnectRefusedNotAuthorized311,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != CONNACK {
					t.Errorf("Expected packet type CONNACK, got %v", fh.Type)
				}
				if fh.RemainingLength != 2 {
					t.Errorf("Expected remaining length 2, got %d", fh.RemainingLength)
				}
			}
		})
	}
}

func TestEncodePublishPacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *PublishPacket311
		wantErr bool
	}{
		{
			name: "QoS 0 publish",
			packet: &PublishPacket311{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     []byte("hello"),
			},
			wantErr: false,
		},
		{
			name: "QoS 1 publish with packet ID",
			packet: &PublishPacket311{
				FixedHeader: FixedHeader{QoS: QoS1},
				TopicName:   "test/topic",
				PacketID:    1234,
				Payload:     []byte("hello"),
			},
			wantErr: false,
		},
		{
			name: "QoS 2 publish with DUP and retain",
			packet: &PublishPacket311{
				FixedHeader: FixedHeader{QoS: QoS2, DUP: true, Retain: true},
				TopicName:   "test/topic",
				PacketID:    5678,
				Payload:     []byte("retained message"),
			},
			wantErr: false,
		},
		{
			name: "empty payload",
			packet: &PublishPacket311{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     []byte{},
			},
			wantErr: false,
		},
		{
			name: "large payload",
			packet: &PublishPacket311{
				FixedHeader: FixedHeader{QoS: QoS1},
				TopicName:   "test/topic",
				PacketID:    9999,
				Payload:     make([]byte, 10000),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != PUBLISH {
					t.Errorf("Expected packet type PUBLISH, got %v", fh.Type)
				}
				if fh.QoS != tt.packet.FixedHeader.QoS {
					t.Errorf("Expected QoS %v, got %v", tt.packet.FixedHeader.QoS, fh.QoS)
				}
				if fh.DUP != tt.packet.FixedHeader.DUP {
					t.Errorf("Expected DUP %v, got %v", tt.packet.FixedHeader.DUP, fh.DUP)
				}
				if fh.Retain != tt.packet.FixedHeader.Retain {
					t.Errorf("Expected Retain %v, got %v", tt.packet.FixedHeader.Retain, fh.Retain)
				}
			}
		})
	}
}

func TestEncodeSubscribePacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *SubscribePacket311
		wantErr bool
	}{
		{
			name: "single subscription",
			packet: &SubscribePacket311{
				PacketID: 1234,
				Subscriptions: []Subscription311{
					{
						TopicFilter: "test/topic",
						QoS:         QoS1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple subscriptions",
			packet: &SubscribePacket311{
				PacketID: 5678,
				Subscriptions: []Subscription311{
					{
						TopicFilter: "test/topic1",
						QoS:         QoS0,
					},
					{
						TopicFilter: "test/topic2",
						QoS:         QoS1,
					},
					{
						TopicFilter: "test/topic3",
						QoS:         QoS2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "wildcard subscription",
			packet: &SubscribePacket311{
				PacketID: 9999,
				Subscriptions: []Subscription311{
					{
						TopicFilter: "test/#",
						QoS:         QoS1,
					},
					{
						TopicFilter: "test/+/subtopic",
						QoS:         QoS2,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != SUBSCRIBE {
					t.Errorf("Expected packet type SUBSCRIBE, got %v", fh.Type)
				}
				if fh.Flags != 0x02 {
					t.Errorf("Expected flags 0x02, got 0x%02x", fh.Flags)
				}
			}
		})
	}
}

func TestEncodeSubackPacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *SubackPacket311
		wantErr bool
	}{
		{
			name: "successful subscriptions",
			packet: &SubackPacket311{
				PacketID:    1234,
				ReturnCodes: []byte{0x00, 0x01, 0x02}, // QoS 0, 1, 2
			},
			wantErr: false,
		},
		{
			name: "failure response",
			packet: &SubackPacket311{
				PacketID:    5678,
				ReturnCodes: []byte{0x80}, // Failure
			},
			wantErr: false,
		},
		{
			name: "mixed success and failure",
			packet: &SubackPacket311{
				PacketID:    9999,
				ReturnCodes: []byte{0x00, 0x01, 0x80, 0x02},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != SUBACK {
					t.Errorf("Expected packet type SUBACK, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodeUnsubscribePacket311(t *testing.T) {
	tests := []struct {
		name    string
		packet  *UnsubscribePacket311
		wantErr bool
	}{
		{
			name: "single topic filter",
			packet: &UnsubscribePacket311{
				PacketID:     1234,
				TopicFilters: []string{"test/topic"},
			},
			wantErr: false,
		},
		{
			name: "multiple topic filters",
			packet: &UnsubscribePacket311{
				PacketID:     5678,
				TopicFilters: []string{"test/topic1", "test/topic2", "test/topic3"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.packet.Encode(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader311(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader311() error = %v", err)
					return
				}
				if fh.Type != UNSUBSCRIBE {
					t.Errorf("Expected packet type UNSUBSCRIBE, got %v", fh.Type)
				}
				if fh.Flags != 0x02 {
					t.Errorf("Expected flags 0x02, got 0x%02x", fh.Flags)
				}
			}
		})
	}
}

func TestEncodeAckPackets311(t *testing.T) {
	tests := []struct {
		name       string
		packetType PacketType
		packetID   uint16
		encode     func(uint16) ([]byte, error)
	}{
		{
			name:       "PUBACK",
			packetType: PUBACK,
			packetID:   1234,
			encode: func(id uint16) ([]byte, error) {
				var buf bytes.Buffer
				p := &PubackPacket311{PacketID: id}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
		},
		{
			name:       "PUBREC",
			packetType: PUBREC,
			packetID:   5678,
			encode: func(id uint16) ([]byte, error) {
				var buf bytes.Buffer
				p := &PubrecPacket311{PacketID: id}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
		},
		{
			name:       "PUBREL",
			packetType: PUBREL,
			packetID:   9999,
			encode: func(id uint16) ([]byte, error) {
				var buf bytes.Buffer
				p := &PubrelPacket311{PacketID: id}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
		},
		{
			name:       "PUBCOMP",
			packetType: PUBCOMP,
			packetID:   1111,
			encode: func(id uint16) ([]byte, error) {
				var buf bytes.Buffer
				p := &PubcompPacket311{PacketID: id}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
		},
		{
			name:       "UNSUBACK",
			packetType: UNSUBACK,
			packetID:   2222,
			encode: func(id uint16) ([]byte, error) {
				var buf bytes.Buffer
				p := &UnsubackPacket311{PacketID: id}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.encode(tt.packetID)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if len(data) == 0 {
				t.Fatal("Expected non-empty encoded data")
			}

			// Parse and verify
			reader := bytes.NewReader(data)
			fh, err := ParseFixedHeader311(reader)
			if err != nil {
				t.Fatalf("ParseFixedHeader311() error = %v", err)
			}

			if fh.Type != tt.packetType {
				t.Errorf("Expected packet type %v, got %v", tt.packetType, fh.Type)
			}

			if fh.RemainingLength != 2 {
				t.Errorf("Expected remaining length 2, got %d", fh.RemainingLength)
			}

			// Special check for PUBREL flags
			if tt.packetType == PUBREL && fh.Flags != 0x02 {
				t.Errorf("Expected PUBREL flags 0x02, got 0x%02x", fh.Flags)
			}
		})
	}
}

func TestEncodeDisconnectPacket311(t *testing.T) {
	packet := &DisconnectPacket311{}
	var buf bytes.Buffer

	err := packet.Encode(&buf)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if buf.Len() != 2 {
		t.Errorf("Expected 2 bytes for DISCONNECT, got %d", buf.Len())
	}

	fh, err := ParseFixedHeader311(&buf)
	if err != nil {
		t.Fatalf("ParseFixedHeader311() error = %v", err)
	}

	if fh.Type != DISCONNECT {
		t.Errorf("Expected packet type DISCONNECT, got %v", fh.Type)
	}

	if fh.RemainingLength != 0 {
		t.Errorf("Expected remaining length 0, got %d", fh.RemainingLength)
	}
}

func BenchmarkEncodePublishQoS0_311(b *testing.B) {
	packet := &PublishPacket311{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     []byte("hello world"),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishQoS1_311(b *testing.B) {
	packet := &PublishPacket311{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     []byte("hello world"),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnectPacket311(b *testing.B) {
	packet := &ConnectPacket311{
		ProtocolName:    "MQTT",
		ProtocolVersion: ProtocolVersion311,
		CleanSession:    true,
		KeepAlive:       60,
		ClientID:        "benchmark-client",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeSubscribePacket311(b *testing.B) {
	packet := &SubscribePacket311{
		PacketID: 1234,
		Subscriptions: []Subscription311{
			{TopicFilter: "test/topic1", QoS: QoS1},
			{TopicFilter: "test/topic2", QoS: QoS2},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func TestEncode311PacketSizes(t *testing.T) {
	tests := []struct {
		name         string
		encode       func() ([]byte, error)
		expectedSize int
	}{
		{
			name: "PINGREQ",
			encode: func() ([]byte, error) {
				var buf bytes.Buffer
				p := &PingreqPacket{}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
			expectedSize: 2,
		},
		{
			name: "PINGRESP",
			encode: func() ([]byte, error) {
				var buf bytes.Buffer
				p := &PingrespPacket{}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
			expectedSize: 2,
		},
		{
			name: "DISCONNECT",
			encode: func() ([]byte, error) {
				var buf bytes.Buffer
				p := &DisconnectPacket311{}
				err := p.Encode(&buf)
				return buf.Bytes(), err
			},
			expectedSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.encode()
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if len(data) != tt.expectedSize {
				t.Errorf("Expected size %d, got %d", tt.expectedSize, len(data))
			}
		})
	}
}
