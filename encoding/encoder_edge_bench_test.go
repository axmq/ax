package encoding

import (
	"bytes"
	"strings"
	"testing"
)

func BenchmarkEncodeConnectPacket_Small(b *testing.B) {
	packet := &ConnectPacket{
		ProtocolName:    "MQTT",
		ProtocolVersion: ProtocolVersion50,
		CleanStart:      true,
		KeepAlive:       60,
		ClientID:        "test-client",
		Properties:      Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnectPacket_MaxClientID(b *testing.B) {
	packet := &ConnectPacket{
		ProtocolName:    "MQTT",
		ProtocolVersion: ProtocolVersion50,
		CleanStart:      true,
		KeepAlive:       60,
		ClientID:        strings.Repeat("a", MaxUTF8StringLen),
		Properties:      Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnectPacket_WithWill(b *testing.B) {
	packet := &ConnectPacket{
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
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnectPacket_FullFeatures(b *testing.B) {
	packet := &ConnectPacket{
		ProtocolName:    "MQTT",
		ProtocolVersion: ProtocolVersion50,
		CleanStart:      true,
		WillFlag:        true,
		WillQoS:         QoS2,
		WillRetain:      true,
		UsernameFlag:    true,
		PasswordFlag:    true,
		KeepAlive:       60,
		ClientID:        "test-client-123",
		WillTopic:       "will/topic",
		WillPayload:     []byte("goodbye message"),
		Username:        "username",
		Password:        []byte("password123"),
		Properties:      Properties{},
		WillProperties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS0_EmptyPayload(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     []byte{},
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS0_SmallPayload(b *testing.B) {
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

func BenchmarkEncodePublishPacket_QoS0_1KB(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     make([]byte, 1024),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS0_64KB(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     make([]byte, 64*1024),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS0_256KB(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     make([]byte, 256*1024),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS0_1MB(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   "test/topic",
		Payload:     make([]byte, 1024*1024),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_QoS1_SmallPayload(b *testing.B) {
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

func BenchmarkEncodePublishPacket_QoS2_SmallPayload(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS2, Retain: true, DUP: true},
		TopicName:   "test/topic",
		PacketID:    5678,
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

func BenchmarkEncodePublishPacket_MaxTopicLength(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS0},
		TopicName:   strings.Repeat("t", MaxUTF8StringLen),
		Payload:     []byte("data"),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeSubscribePacket_SingleTopic(b *testing.B) {
	packet := &SubscribePacket{
		PacketID: 1,
		Subscriptions: []Subscription{
			{TopicFilter: "test/topic", QoS: QoS1},
		},
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeSubscribePacket_MultipleTopics(b *testing.B) {
	packet := &SubscribePacket{
		PacketID: 1,
		Subscriptions: []Subscription{
			{TopicFilter: "topic/1", QoS: QoS0},
			{TopicFilter: "topic/2", QoS: QoS1},
			{TopicFilter: "topic/3", QoS: QoS2},
			{TopicFilter: "topic/4", QoS: QoS1, NoLocal: true},
			{TopicFilter: "topic/5", QoS: QoS2, RetainAsPublished: true},
		},
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeSubscribePacket_WithOptions(b *testing.B) {
	packet := &SubscribePacket{
		PacketID: 1,
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
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeUnsubscribePacket_SingleTopic(b *testing.B) {
	packet := &UnsubscribePacket{
		PacketID:     1,
		TopicFilters: []string{"test/topic"},
		Properties:   Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeUnsubscribePacket_MultipleTopics(b *testing.B) {
	packet := &UnsubscribePacket{
		PacketID:     1,
		TopicFilters: []string{"topic/1", "topic/2", "topic/3", "topic/4", "topic/5"},
		Properties:   Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePubackPacket(b *testing.B) {
	packet := &PubackPacket{
		PacketID:   1234,
		ReasonCode: ReasonSuccess,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePubrecPacket(b *testing.B) {
	packet := &PubrecPacket{
		PacketID:   1234,
		ReasonCode: ReasonSuccess,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePubrelPacket(b *testing.B) {
	packet := &PubrelPacket{
		PacketID:   1234,
		ReasonCode: ReasonSuccess,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePubcompPacket(b *testing.B) {
	packet := &PubcompPacket{
		PacketID:   1234,
		ReasonCode: ReasonSuccess,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeSubackPacket(b *testing.B) {
	packet := &SubackPacket{
		PacketID:    1,
		ReasonCodes: []ReasonCode{ReasonGrantedQoS0, ReasonGrantedQoS1, ReasonGrantedQoS2},
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeUnsubackPacket(b *testing.B) {
	packet := &UnsubackPacket{
		PacketID:    1,
		ReasonCodes: []ReasonCode{ReasonSuccess, ReasonSuccess},
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePingreqPacket(b *testing.B) {
	packet := &PingreqPacket{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePingrespPacket(b *testing.B) {
	packet := &PingrespPacket{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeDisconnectPacket_Normal(b *testing.B) {
	packet := &DisconnectPacket{
		ReasonCode: ReasonNormalDisconnection,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeDisconnectPacket_WithReason(b *testing.B) {
	packet := &DisconnectPacket{
		ReasonCode: ReasonProtocolError,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeAuthPacket(b *testing.B) {
	packet := &AuthPacket{
		ReasonCode: ReasonContinueAuthentication,
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodeConnackPacket(b *testing.B) {
	packet := &ConnackPacket{
		SessionPresent: false,
		ReasonCode:     ReasonSuccess,
		Properties:     Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = packet.Encode(&buf)
	}
}

func BenchmarkEncodePublishPacket_Parallel(b *testing.B) {
	packet := &PublishPacket{
		FixedHeader: FixedHeader{QoS: QoS1},
		TopicName:   "test/topic",
		PacketID:    1234,
		Payload:     make([]byte, 1024),
		Properties:  Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf bytes.Buffer
			_ = packet.Encode(&buf)
		}
	})
}

func BenchmarkEncodeSubscribePacket_Parallel(b *testing.B) {
	packet := &SubscribePacket{
		PacketID: 1,
		Subscriptions: []Subscription{
			{TopicFilter: "test/topic", QoS: QoS1},
		},
		Properties: Properties{},
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf bytes.Buffer
			_ = packet.Encode(&buf)
		}
	})
}

func BenchmarkEncodeVariedPayloads(b *testing.B) {
	sizes := []int{0, 1, 10, 100, 1024, 16384, 65535}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			packet := &PublishPacket{
				FixedHeader: FixedHeader{QoS: QoS0},
				TopicName:   "test/topic",
				Payload:     make([]byte, size),
				Properties:  Properties{},
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				_ = packet.Encode(&buf)
			}
		})
	}
}
