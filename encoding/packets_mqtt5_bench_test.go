package encoding

import (
	"bytes"
	"testing"
)

func BenchmarkParseConnackPacket(b *testing.B) {
	data := []byte{0x01, 0x00, 0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: CONNACK, RemainingLength: uint32(len(data))}
		_, err := ParseConnackPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePublishPacket_QoS0(b *testing.B) {
	data := []byte{
		0x00, 0x0A, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c',
		0x00,
		'h', 'e', 'l', 'l', 'o',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: PUBLISH, QoS: QoS0, RemainingLength: uint32(len(data))}
		_, err := ParsePublishPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePublishPacket_QoS1(b *testing.B) {
	data := []byte{
		0x00, 0x0A, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c',
		0x04, 0xD2,
		0x00,
		'h', 'e', 'l', 'l', 'o',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: PUBLISH, QoS: QoS1, RemainingLength: uint32(len(data))}
		_, err := ParsePublishPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePublishPacket_WithProperties(b *testing.B) {
	data := []byte{
		0x00, 0x05, 't', 'e', 's', 't', '1',
		0x00, 0x01,
		0x02, 0x01, 0x01,
		'h', 'i',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: PUBLISH, QoS: QoS1, RemainingLength: uint32(len(data))}
		_, err := ParsePublishPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePubackPacket(b *testing.B) {
	data := []byte{0x00, 0x01, 0x00, 0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: PUBACK, RemainingLength: 4}
		_, err := ParsePubackPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSubscribePacket(b *testing.B) {
	data := []byte{
		0x00, 0x0A,
		0x00,
		0x00, 0x07, 't', 'e', 's', 't', '/', '#', '1',
		0x01,
		0x00, 0x05, 't', 'o', 'p', 'i', 'c',
		0x06,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: SUBSCRIBE, Flags: 0x02, RemainingLength: uint32(len(data))}
		_, err := ParseSubscribePacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSubackPacket(b *testing.B) {
	data := []byte{0x00, 0x0A, 0x00, 0x00, 0x01, 0x02, 0x80}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: SUBACK, RemainingLength: uint32(len(data))}
		_, err := ParseSubackPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseConnectPacket(b *testing.B) {
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T',
		0x05,
		0x02,
		0x00, 0x3C,
		0x00,
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: CONNECT, RemainingLength: uint32(len(data))}
		_, err := ParseConnectPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseConnectPacket_WithWill(b *testing.B) {
	data := []byte{
		0x00, 0x04, 'M', 'Q', 'T', 'T',
		0x05,
		0x2E,
		0x00, 0x3C,
		0x00,
		0x00, 0x06, 'c', 'l', 'i', 'e', 'n', 't',
		0x00,
		0x00, 0x0A, 'w', 'i', 'l', 'l', '/', 't', 'o', 'p', 'i', 'c',
		0x00, 0x07, 'g', 'o', 'o', 'd', 'b', 'y', 'e',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: CONNECT, RemainingLength: uint32(len(data))}
		_, err := ParseConnectPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseDisconnectPacket(b *testing.B) {
	data := []byte{0x00, 0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: DISCONNECT, RemainingLength: 2}
		_, err := ParseDisconnectPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseAuthPacket(b *testing.B) {
	data := []byte{0x18, 0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		fh := &FixedHeader{Type: AUTH, RemainingLength: 2}
		_, err := ParseAuthPacket(r, fh)
		if err != nil {
			b.Fatal(err)
		}
	}
}
