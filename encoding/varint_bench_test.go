package encoding

import (
	"bytes"
	"testing"
)

func BenchmarkEncodeVariableByteInteger(b *testing.B) {
	tests := []struct {
		name  string
		value uint32
	}{
		{"1byte_0", 0},
		{"1byte_127", 127},
		{"2byte_128", 128},
		{"2byte_16383", 16383},
		{"3byte_16384", 16384},
		{"3byte_2097151", 2097151},
		{"4byte_2097152", 2097152},
		{"4byte_max", 268435455},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := EncodeVariableByteInteger(tt.value)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncodeVariableByteIntegerTo(b *testing.B) {
	tests := []struct {
		name  string
		value uint32
	}{
		{"1byte", 127},
		{"2byte", 16383},
		{"3byte", 2097151},
		{"4byte", 268435455},
	}

	buf := make([]byte, 10)

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := EncodeVariableByteIntegerTo(buf, 0, tt.value)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDecodeVariableByteInteger(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"1byte", []byte{0x00}},
		{"2byte", []byte{0x80, 0x01}},
		{"3byte", []byte{0x80, 0x80, 0x01}},
		{"4byte", []byte{0xFF, 0xFF, 0xFF, 0x7F}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))

			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(tt.input)
				_, err := DecodeVariableByteInteger(r)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDecodeVariableByteIntegerFromBytes(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"1byte", []byte{0x00}},
		{"2byte", []byte{0x80, 0x01}},
		{"3byte", []byte{0x80, 0x80, 0x01}},
		{"4byte", []byte{0xFF, 0xFF, 0xFF, 0x7F}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))

			for i := 0; i < b.N; i++ {
				_, _, err := DecodeVariableByteIntegerFromBytes(tt.input)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSizeVariableByteInteger(b *testing.B) {
	values := []uint32{0, 127, 128, 16383, 16384, 2097151, 2097152, 268435455}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SizeVariableByteInteger(values[i%len(values)])
	}
}

func BenchmarkEncodeDecodeRoundTrip(b *testing.B) {
	tests := []struct {
		name  string
		value uint32
	}{
		{"1byte", 127},
		{"2byte", 16383},
		{"3byte", 2097151},
		{"4byte", 268435455},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				encoded, err := EncodeVariableByteInteger(tt.value)
				if err != nil {
					b.Fatal(err)
				}

				decoded, _, err := DecodeVariableByteIntegerFromBytes(encoded)
				if err != nil {
					b.Fatal(err)
				}

				if decoded != tt.value {
					b.Fatal("round trip failed")
				}
			}
		})
	}
}
