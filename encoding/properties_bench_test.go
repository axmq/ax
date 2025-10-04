package encoding

import (
	"bytes"
	"testing"
)

func BenchmarkParseProperties_Empty(b *testing.B) {
	data := []byte{0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		_, err := ParseProperties(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseProperties_SingleByte(b *testing.B) {
	data := []byte{0x02, 0x01, 0x01}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		_, err := ParseProperties(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseProperties_Multiple(b *testing.B) {
	data := []byte{
		0x14,
		0x01, 0x01,
		0x02, 0x00, 0x00, 0x0E, 0x10,
		0x03, 0x00, 0x0A, 't', 'e', 'x', 't', '/', 'p', 'l', 'a', 'i', 'n',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		_, err := ParseProperties(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePropertiesFromBytes_Empty(b *testing.B) {
	data := []byte{0x00}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := ParsePropertiesFromBytes(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePropertiesFromBytes_SingleByte(b *testing.B) {
	data := []byte{0x02, 0x01, 0x01}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := ParsePropertiesFromBytes(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePropertiesFromBytes_Multiple(b *testing.B) {
	data := []byte{
		0x14,
		0x01, 0x01,
		0x02, 0x00, 0x00, 0x0E, 0x10,
		0x03, 0x00, 0x0A, 't', 'e', 'x', 't', '/', 'p', 'l', 'a', 'i', 'n',
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := ParsePropertiesFromBytes(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeProperties_Empty(b *testing.B) {
	props := &Properties{}
	var buf bytes.Buffer
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := props.EncodeProperties(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeProperties_Single(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	var buf bytes.Buffer
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := props.EncodeProperties(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeProperties_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	var buf bytes.Buffer
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := props.EncodeProperties(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_Single(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	buf := make([]byte, 128)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	buf := make([]byte, 128)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProperties_GetProperty(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = props.GetProperty(PropContentType)
	}
}

func BenchmarkProperties_AddProperty(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		props := &Properties{}
		_ = props.AddProperty(PropPayloadFormatIndicator, byte(1))
		_ = props.AddProperty(PropMessageExpiryInterval, uint32(3600))
		_ = props.AddProperty(PropContentType, "text/plain")
	}
}
