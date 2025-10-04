package encoding

import (
	"bytes"
	"testing"
)

func BenchmarkParseFixedHeader(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "CONNECT_1byte_remlen",
			input: []byte{0x10, 0x0A},
		},
		{
			name:  "PUBLISH_QoS0_1byte_remlen",
			input: []byte{0x30, 0x7F},
		},
		{
			name:  "PUBLISH_QoS1_2byte_remlen",
			input: []byte{0x32, 0x80, 0x01},
		},
		{
			name:  "PUBLISH_QoS2_DUP_Retain_4byte_remlen",
			input: []byte{0x3D, 0xFF, 0xFF, 0xFF, 0x7F},
		},
		{
			name:  "PINGREQ",
			input: []byte{0xC0, 0x00},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))

			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(tt.input)
				_, err := ParseFixedHeader(r)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseFixedHeaderFromBytes(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "CONNECT_1byte_remlen",
			input: []byte{0x10, 0x0A},
		},
		{
			name:  "PUBLISH_QoS0_1byte_remlen",
			input: []byte{0x30, 0x7F},
		},
		{
			name:  "PUBLISH_QoS1_2byte_remlen",
			input: []byte{0x32, 0x80, 0x01},
		},
		{
			name:  "PUBLISH_QoS2_DUP_Retain_4byte_remlen",
			input: []byte{0x3D, 0xFF, 0xFF, 0xFF, 0x7F},
		},
		{
			name:  "PINGREQ",
			input: []byte{0xC0, 0x00},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))

			for i := 0; i < b.N; i++ {
				_, _, err := ParseFixedHeaderFromBytes(tt.input)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTypeString(b *testing.B) {
	types := []PacketType{CONNECT, PUBLISH, SUBSCRIBE, DISCONNECT}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = types[i%len(types)].String()
	}
}

func BenchmarkQoSString(b *testing.B) {
	qosLevels := []QoS{QoS0, QoS1, QoS2}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = qosLevels[i%len(qosLevels)].String()
	}
}

func BenchmarkValidateFlags(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validateFlags(CONNECT, 0x00)
	}
}
