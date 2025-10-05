package encoding

import (
	"bytes"
	"testing"
)

func TestEncodeConnectPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnectPacket
		wantErr bool
	}{
		{
			name: "basic connect with clean start",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				KeepAlive:       60,
				ClientID:        "test-client",
				Properties:      Properties{},
			},
			wantErr: false,
		},
		{
			name: "connect with will message",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				WillFlag:        true,
				WillQoS:         QoS1,
				WillRetain:      true,
				KeepAlive:       60,
				ClientID:        "test-client",
				WillTopic:       "will/topic",
				WillPayload:     []byte("goodbye"),
				Properties:      Properties{},
				WillProperties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "connect with username and password",
			packet: &ConnectPacket{
				ProtocolName:    "MQTT",
				ProtocolVersion: ProtocolVersion50,
				CleanStart:      true,
				UsernameFlag:    true,
				PasswordFlag:    true,
				KeepAlive:       60,
				ClientID:        "test-client",
				Username:        "user",
				Password:        []byte("pass"),
				Properties:      Properties{},
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
				// Verify we can parse it back
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != CONNECT {
					t.Errorf("Expected packet type CONNECT, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodeConnackPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *ConnackPacket
		wantErr bool
	}{
		{
			name: "successful connection",
			packet: &ConnackPacket{
				SessionPresent: false,
				ReasonCode:     ReasonSuccess,
				Properties:     Properties{},
			},
			wantErr: false,
		},
		{
			name: "session present",
			packet: &ConnackPacket{
				SessionPresent: true,
				ReasonCode:     ReasonSuccess,
				Properties:     Properties{},
			},
			wantErr: false,
		},
		{
			name: "connection refused",
			packet: &ConnackPacket{
				SessionPresent: false,
				ReasonCode:     ReasonNotAuthorized,
				Properties:     Properties{},
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
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != CONNACK {
					t.Errorf("Expected packet type CONNACK, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodePublishPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *PublishPacket
		wantErr bool
	}{
		{
			name: "QoS 0 publish",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     []byte("hello"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS 1 publish with packet ID",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS1},
				TopicName:   "test/topic",
				PacketID:    1234,
				Payload:     []byte("hello"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "QoS 2 publish with retain",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS2, Retain: true},
				TopicName:   "test/topic",
				PacketID:    5678,
				Payload:     []byte("retained message"),
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "empty payload",
			packet: &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     []byte{},
				Properties:  Properties{},
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
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != PUBLISH {
					t.Errorf("Expected packet type PUBLISH, got %v", fh.Type)
				}
				if fh.QoS != tt.packet.FixedHeader.QoS {
					t.Errorf("Expected QoS %v, got %v", tt.packet.FixedHeader.QoS, fh.QoS)
				}
			}
		})
	}
}

func TestEncodePubackPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *PubackPacket
		wantErr bool
	}{
		{
			name: "success puback",
			packet: &PubackPacket{
				PacketID:   1234,
				ReasonCode: ReasonSuccess,
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "puback with error",
			packet: &PubackPacket{
				PacketID:   5678,
				ReasonCode: ReasonNotAuthorized,
				Properties: Properties{},
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
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != PUBACK {
					t.Errorf("Expected packet type PUBACK, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodeSubscribePacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *SubscribePacket
		wantErr bool
	}{
		{
			name: "single subscription",
			packet: &SubscribePacket{
				PacketID: 1234,
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						QoS:         QoS1,
					},
				},
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "multiple subscriptions with options",
			packet: &SubscribePacket{
				PacketID: 5678,
				Subscriptions: []Subscription{
					{
						TopicFilter:       "test/topic1",
						QoS:               QoS1,
						NoLocal:           true,
						RetainAsPublished: true,
						RetainHandling:    1,
					},
					{
						TopicFilter: "test/topic2",
						QoS:         QoS2,
					},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() > 0 {
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
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

func TestEncodeSubackPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *SubackPacket
		wantErr bool
	}{
		{
			name: "successful subscriptions",
			packet: &SubackPacket{
				PacketID:    1234,
				ReasonCodes: []ReasonCode{ReasonGrantedQoS1, ReasonGrantedQoS2},
				Properties:  Properties{},
			},
			wantErr: false,
		},
		{
			name: "mixed success and failure",
			packet: &SubackPacket{
				PacketID:    5678,
				ReasonCodes: []ReasonCode{ReasonGrantedQoS1, ReasonNotAuthorized},
				Properties:  Properties{},
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
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != SUBACK {
					t.Errorf("Expected packet type SUBACK, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodePingreqPacket(t *testing.T) {
	packet := &PingreqPacket{}
	var buf bytes.Buffer

	err := packet.Encode(&buf)
	if err != nil {
		t.Errorf("Encode() error = %v", err)
		return
	}

	if buf.Len() != 2 {
		t.Errorf("Expected 2 bytes, got %d", buf.Len())
	}

	fh, err := ParseFixedHeader(&buf)
	if err != nil {
		t.Errorf("ParseFixedHeader() error = %v", err)
		return
	}
	if fh.Type != PINGREQ {
		t.Errorf("Expected packet type PINGREQ, got %v", fh.Type)
	}
	if fh.RemainingLength != 0 {
		t.Errorf("Expected remaining length 0, got %d", fh.RemainingLength)
	}
}

func TestEncodePingrespPacket(t *testing.T) {
	packet := &PingrespPacket{}
	var buf bytes.Buffer

	err := packet.Encode(&buf)
	if err != nil {
		t.Errorf("Encode() error = %v", err)
		return
	}

	if buf.Len() != 2 {
		t.Errorf("Expected 2 bytes, got %d", buf.Len())
	}

	fh, err := ParseFixedHeader(&buf)
	if err != nil {
		t.Errorf("ParseFixedHeader() error = %v", err)
		return
	}
	if fh.Type != PINGRESP {
		t.Errorf("Expected packet type PINGRESP, got %v", fh.Type)
	}
}

func TestEncodeDisconnectPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *DisconnectPacket
		wantErr bool
	}{
		{
			name: "normal disconnection",
			packet: &DisconnectPacket{
				ReasonCode: ReasonNormalDisconnection,
				Properties: Properties{},
			},
			wantErr: false,
		},
		{
			name: "disconnect with reason",
			packet: &DisconnectPacket{
				ReasonCode: ReasonServerShuttingDown,
				Properties: Properties{},
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
				fh, err := ParseFixedHeader(&buf)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != DISCONNECT {
					t.Errorf("Expected packet type DISCONNECT, got %v", fh.Type)
				}
			}
		})
	}
}

func TestEncodeAuthPacket(t *testing.T) {
	packet := &AuthPacket{
		ReasonCode: ReasonContinueAuthentication,
		Properties: Properties{},
	}

	var buf bytes.Buffer
	err := packet.Encode(&buf)
	if err != nil {
		t.Errorf("Encode() error = %v", err)
		return
	}

	if buf.Len() < 3 {
		t.Errorf("Expected at least 3 bytes, got %d", buf.Len())
	}

	fh, err := ParseFixedHeader(&buf)
	if err != nil {
		t.Errorf("ParseFixedHeader() error = %v", err)
		return
	}
	if fh.Type != AUTH {
		t.Errorf("Expected packet type AUTH, got %v", fh.Type)
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	t.Run("PUBLISH roundtrip", func(t *testing.T) {
		original := &PublishPacket{
			FixedHeader: FixedHeader{QoS: QoS1},
			TopicName:   "test/topic",
			PacketID:    1234,
			Payload:     []byte("test payload"),
			Properties:  Properties{},
		}

		var buf bytes.Buffer
		err := original.Encode(&buf)
		if err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		fh, err := ParseFixedHeader(&buf)
		if err != nil {
			t.Fatalf("ParseFixedHeader() error = %v", err)
		}

		decoded, err := ParsePublishPacket(&buf, fh)
		if err != nil {
			t.Fatalf("ParsePublishPacket() error = %v", err)
		}

		if decoded.TopicName != original.TopicName {
			t.Errorf("TopicName mismatch: got %v, want %v", decoded.TopicName, original.TopicName)
		}
		if decoded.PacketID != original.PacketID {
			t.Errorf("PacketID mismatch: got %v, want %v", decoded.PacketID, original.PacketID)
		}
		if !bytes.Equal(decoded.Payload, original.Payload) {
			t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, original.Payload)
		}
	})
}

func BenchmarkEncodePublishQoS0(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     []byte("hello world"),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishQoS1(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     []byte("hello world"),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnectPacket(b *testing.B) {
	packet := &ConnectPacket{
		ProtocolName:    "MQTT",
		ProtocolVersion: ProtocolVersion50,
		CleanStart:      true,
		KeepAlive:       60,
		ClientID:        "benchmark-client",
		Properties:      Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishToBuffer(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     []byte("hello world"),
		Properties:  Properties{},
	}

	buf := make([]byte, 256)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = packet.EncodeTo(buf)
	}
}

func TestEncodeToBuffer(t *testing.T) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     []byte("test"),
		Properties:  Properties{},
	}

	buf := make([]byte, 256)
	n, err := packet.EncodeTo(buf)
	if err != nil {
		t.Fatalf("EncodeTo() error = %v", err)
	}

	if n <= 0 {
		t.Errorf("Expected positive bytes written, got %d", n)
	}

	// Verify the encoded data is valid
	reader := bytes.NewReader(buf[:n])
	fh, err := ParseFixedHeader(reader)
	if err != nil {
		t.Fatalf("ParseFixedHeader() error = %v", err)
	}

	if fh.Type != PUBLISH {
		t.Errorf("Expected PUBLISH packet type, got %v", fh.Type)
	}
}

func TestEncodeBufferTooSmall(t *testing.T) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     make([]byte, 1000),
		Properties:  Properties{},
	}

	buf := make([]byte, 10)
	_, err := packet.EncodeTo(buf)
	if err == nil {
		t.Error("Expected error for buffer too small, got nil")
	}
}

func TestWriteReasonCodes(t *testing.T) {
	tests := []struct {
		name        string
		reasonCodes []ReasonCode
		want        []byte
		wantErr     bool
	}{
		{
			name:        "empty slice",
			reasonCodes: []ReasonCode{},
			want:        []byte{},
			wantErr:     false,
		},
		{
			name:        "single reason code",
			reasonCodes: []ReasonCode{ReasonSuccess},
			want:        []byte{0x00},
			wantErr:     false,
		},
		{
			name:        "multiple success codes",
			reasonCodes: []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2},
			want:        []byte{0x00, 0x01, 0x02},
			wantErr:     false,
		},
		{
			name:        "mixed success and error codes",
			reasonCodes: []ReasonCode{ReasonGrantedQoS1, ReasonNotAuthorized, ReasonGrantedQoS2},
			want:        []byte{0x01, 0x87, 0x02},
			wantErr:     false,
		},
		{
			name:        "all error codes",
			reasonCodes: []ReasonCode{ReasonNotAuthorized, ReasonTopicFilterInvalid, ReasonPacketIdentifierInUse},
			want:        []byte{0x87, 0x8F, 0x91},
			wantErr:     false,
		},
		{
			name:        "large number of codes",
			reasonCodes: []ReasonCode{ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess},
			want:        []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr:     false,
		},
		{
			name:        "various reason codes",
			reasonCodes: []ReasonCode{ReasonUnspecifiedError, ReasonMalformedPacket, ReasonProtocolError, ReasonImplementationSpecificError},
			want:        []byte{0x80, 0x81, 0x82, 0x83},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeReasonCodes(&buf, tt.reasonCodes)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeReasonCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := buf.Bytes()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("writeReasonCodes() = %v, want %v", got, tt.want)
			}
			if len(got) != len(tt.reasonCodes) {
				t.Errorf("writeReasonCodes() wrote %d bytes, want %d", len(got), len(tt.reasonCodes))
			}
		})
	}
}

func TestEncodeAckPacketWithReasonCodes(t *testing.T) {
	tests := []struct {
		name        string
		packetType  PacketType
		flags       byte
		packetID    uint16
		reasonCodes []ReasonCode
		properties  Properties
		wantErr     bool
		validate    func(*testing.T, []byte)
	}{
		{
			name:        "SUBACK with single success code",
			packetType:  SUBACK,
			flags:       0,
			packetID:    1234,
			reasonCodes: []ReasonCode{ReasonGrantedQoS1},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != SUBACK {
					t.Errorf("packet type = %v, want SUBACK", fh.Type)
				}
				var pidBytes [2]byte
				if _, err := r.Read(pidBytes[:]); err != nil {
					t.Errorf("failed to read packet ID: %v", err)
					return
				}
				pid := uint16(pidBytes[0])<<8 | uint16(pidBytes[1])
				if pid != 1234 {
					t.Errorf("packet ID = %d, want 1234", pid)
				}
			},
		},
		{
			name:        "SUBACK with multiple reason codes",
			packetType:  SUBACK,
			flags:       0,
			packetID:    5678,
			reasonCodes: []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != SUBACK {
					t.Errorf("packet type = %v, want SUBACK", fh.Type)
				}
			},
		},
		{
			name:        "UNSUBACK with success codes",
			packetType:  UNSUBACK,
			flags:       0,
			packetID:    9999,
			reasonCodes: []ReasonCode{ReasonSuccess, ReasonSuccess},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != UNSUBACK {
					t.Errorf("packet type = %v, want UNSUBACK", fh.Type)
				}
			},
		},
		{
			name:        "SUBACK with mixed success and error codes",
			packetType:  SUBACK,
			flags:       0,
			packetID:    111,
			reasonCodes: []ReasonCode{ReasonGrantedQoS1, ReasonNotAuthorized, ReasonGrantedQoS2, ReasonTopicFilterInvalid},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				expectedRemaining := uint32(2 + 1 + 4)
				if fh.RemainingLength != expectedRemaining {
					t.Errorf("remaining length = %d, want %d", fh.RemainingLength, expectedRemaining)
				}
			},
		},
		{
			name:        "UNSUBACK with error codes",
			packetType:  UNSUBACK,
			flags:       0,
			packetID:    222,
			reasonCodes: []ReasonCode{ReasonNoSubscriptionExisted, ReasonTopicFilterInvalid},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				_, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
				}
			},
		},
		{
			name:        "SUBACK with empty reason codes",
			packetType:  SUBACK,
			flags:       0,
			packetID:    333,
			reasonCodes: []ReasonCode{},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				expectedRemaining := uint32(2 + 1 + 0)
				if fh.RemainingLength != expectedRemaining {
					t.Errorf("remaining length = %d, want %d", fh.RemainingLength, expectedRemaining)
				}
			},
		},
		{
			name:        "SUBACK with maximum packet ID",
			packetType:  SUBACK,
			flags:       0,
			packetID:    0xFFFF,
			reasonCodes: []ReasonCode{ReasonGrantedQoS1},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Type != SUBACK {
					t.Errorf("packet type = %v, want SUBACK", fh.Type)
				}
				var pidBytes [2]byte
				if _, err := r.Read(pidBytes[:]); err != nil {
					t.Errorf("failed to read packet ID: %v", err)
					return
				}
				pid := uint16(pidBytes[0])<<8 | uint16(pidBytes[1])
				if pid != 0xFFFF {
					t.Errorf("packet ID = %d, want 65535", pid)
				}
			},
		},
		{
			name:        "UNSUBACK with all QoS granted codes",
			packetType:  UNSUBACK,
			flags:       0,
			packetID:    444,
			reasonCodes: []ReasonCode{ReasonSuccess, ReasonNoSubscriptionExisted},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				_, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
				}
			},
		},
		{
			name:        "SUBACK with standard flags",
			packetType:  SUBACK,
			flags:       0x00,
			packetID:    555,
			reasonCodes: []ReasonCode{ReasonGrantedQoS2},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				if fh.Flags != 0x00 {
					t.Errorf("flags = 0x%02x, want 0x00", fh.Flags)
				}
			},
		},
		{
			name:        "SUBACK with many reason codes",
			packetType:  SUBACK,
			flags:       0,
			packetID:    666,
			reasonCodes: []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2, ReasonNotAuthorized, ReasonTopicFilterInvalid, ReasonPacketIdentifierInUse, ReasonQuotaExceeded, ReasonSharedSubscriptionsNotSupported},
			properties:  Properties{},
			wantErr:     false,
			validate: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("encoded data is empty")
					return
				}
				r := bytes.NewReader(data)
				fh, err := ParseFixedHeader(r)
				if err != nil {
					t.Errorf("ParseFixedHeader() error = %v", err)
					return
				}
				expectedRemaining := uint32(2 + 1 + 8)
				if fh.RemainingLength != expectedRemaining {
					t.Errorf("remaining length = %d, want %d", fh.RemainingLength, expectedRemaining)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := encodeAckPacketWithReasonCodes(&buf, tt.packetType, tt.flags, tt.packetID, tt.reasonCodes, &tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeAckPacketWithReasonCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, buf.Bytes())
			}
		})
	}
}

func TestEncodeAckPacketWithReasonCodesRemainingLength(t *testing.T) {
	tests := []struct {
		name              string
		packetID          uint16
		reasonCodes       []ReasonCode
		expectedRemaining uint32
	}{
		{
			name:              "one reason code",
			packetID:          1,
			reasonCodes:       []ReasonCode{ReasonSuccess},
			expectedRemaining: 2 + 1 + 1,
		},
		{
			name:              "three reason codes",
			packetID:          2,
			reasonCodes:       []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2},
			expectedRemaining: 2 + 1 + 3,
		},
		{
			name:              "five reason codes",
			packetID:          3,
			reasonCodes:       []ReasonCode{ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess, ReasonSuccess},
			expectedRemaining: 2 + 1 + 5,
		},
		{
			name:              "no reason codes",
			packetID:          4,
			reasonCodes:       []ReasonCode{},
			expectedRemaining: 2 + 1 + 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			props := Properties{}
			err := encodeAckPacketWithReasonCodes(&buf, SUBACK, 0, tt.packetID, tt.reasonCodes, &props)
			if err != nil {
				t.Fatalf("encodeAckPacketWithReasonCodes() error = %v", err)
			}

			r := bytes.NewReader(buf.Bytes())
			fh, err := ParseFixedHeader(r)
			if err != nil {
				t.Fatalf("ParseFixedHeader() error = %v", err)
			}

			if fh.RemainingLength != tt.expectedRemaining {
				t.Errorf("remaining length = %d, want %d", fh.RemainingLength, tt.expectedRemaining)
			}
		})
	}
}

func TestEncodeAckPacketWithReasonCodesPacketIDEncoding(t *testing.T) {
	testCases := []uint16{0, 1, 255, 256, 1234, 32767, 65535}

	for _, packetID := range testCases {
		t.Run(string(rune(packetID)), func(t *testing.T) {
			var buf bytes.Buffer
			props := Properties{}
			reasonCodes := []ReasonCode{ReasonSuccess}

			err := encodeAckPacketWithReasonCodes(&buf, SUBACK, 0, packetID, reasonCodes, &props)
			if err != nil {
				t.Fatalf("encodeAckPacketWithReasonCodes() error = %v", err)
			}

			r := bytes.NewReader(buf.Bytes())
			fh, err := ParseFixedHeader(r)
			if err != nil {
				t.Fatalf("ParseFixedHeader() error = %v", err)
			}
			if fh.Type != SUBACK {
				t.Errorf("packet type = %v, want SUBACK", fh.Type)
			}

			var pidBytes [2]byte
			if _, err := r.Read(pidBytes[:]); err != nil {
				t.Fatalf("failed to read packet ID: %v", err)
			}

			decodedPID := uint16(pidBytes[0])<<8 | uint16(pidBytes[1])
			if decodedPID != packetID {
				t.Errorf("decoded packet ID = %d, want %d", decodedPID, packetID)
			}
		})
	}
}

func TestEncodeAckPacketWithReasonCodesReasonCodesOrder(t *testing.T) {
	reasonCodes := []ReasonCode{
		ReasonGrantedQoS0,
		ReasonGrantedQoS1,
		ReasonGrantedQoS2,
		ReasonNotAuthorized,
		ReasonTopicFilterInvalid,
	}

	var buf bytes.Buffer
	props := Properties{}
	err := encodeAckPacketWithReasonCodes(&buf, SUBACK, 0, 1, reasonCodes, &props)
	if err != nil {
		t.Fatalf("encodeAckPacketWithReasonCodes() error = %v", err)
	}

	r := bytes.NewReader(buf.Bytes())
	_, err = ParseFixedHeader(r)
	if err != nil {
		t.Fatalf("ParseFixedHeader() error = %v", err)
	}

	var pidBytes [2]byte
	if _, err := r.Read(pidBytes[:]); err != nil {
		t.Fatalf("failed to read packet ID: %v", err)
	}

	propsLen, err := readByte(r)
	if err != nil {
		t.Fatalf("failed to read properties length: %v", err)
	}
	if propsLen != 0 {
		t.Errorf("expected empty properties (length 0), got %d", propsLen)
	}

	for i, expectedCode := range reasonCodes {
		code, err := readByte(r)
		if err != nil {
			t.Fatalf("failed to read reason code %d: %v", i, err)
		}
		if ReasonCode(code) != expectedCode {
			t.Errorf("reason code %d = %v, want %v", i, ReasonCode(code), expectedCode)
		}
	}
}
