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

func BenchmarkEncodeProperties_SingleByte(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := props.EncodeProperties(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_SingleByte(b *testing.B) {
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
	buf := make([]byte, 256)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_Complex(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "application/json"},
			{ID: PropResponseTopic, Value: "response/topic"},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
			{ID: PropSubscriptionIdentifier, Value: uint32(100)},
			{ID: PropSessionExpiryInterval, Value: uint32(7200)},
			{ID: PropServerKeepAlive, Value: uint16(60)},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "app", Value: "test"}},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "version", Value: "1.0"}},
		},
	}
	buf := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPropertySerializer_SingleByte(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	buf := make([]byte, 128)
	serializer := NewPropertySerializer(buf)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(props)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPropertySerializer_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "application/json"},
		},
	}
	buf := make([]byte, 256)
	serializer := NewPropertySerializer(buf)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(props)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPropertyBuilder_Simple(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder := NewPropertyBuilder()
		_, err := builder.
			WithPayloadFormat(1).
			WithContentType("text/plain").
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPropertyBuilder_Complex(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder := NewPropertyBuilder()
		_, err := builder.
			WithPayloadFormat(1).
			WithMessageExpiry(3600).
			WithContentType("application/json").
			WithResponseTopic("response/topic").
			WithCorrelationData([]byte{1, 2, 3, 4}).
			WithUserProperty("app", "test").
			WithUserProperty("version", "1.0").
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundtrip_SingleByte(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	encodeBuf := make([]byte, 128)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := props.EncodePropertiesToBytes(encodeBuf)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = ParsePropertiesFromBytes(encodeBuf[:n])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundtrip_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	encodeBuf := make([]byte, 256)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := props.EncodePropertiesToBytes(encodeBuf)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = ParsePropertiesFromBytes(encodeBuf[:n])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundtrip_Complex(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "application/json"},
			{ID: PropResponseTopic, Value: "response/topic"},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
			{ID: PropSubscriptionIdentifier, Value: uint32(100)},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "app", Value: "test"}},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "version", Value: "1.0"}},
		},
	}
	encodeBuf := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := props.EncodePropertiesToBytes(encodeBuf)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = ParsePropertiesFromBytes(encodeBuf[:n])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalculatePropertiesSize_Empty(b *testing.B) {
	props := &Properties{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculatePropertiesSize(props)
	}
}

func BenchmarkCalculatePropertiesSize_Single(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculatePropertiesSize(props)
	}
}

func BenchmarkCalculatePropertiesSize_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
			{ID: PropResponseTopic, Value: "response/topic"},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculatePropertiesSize(props)
	}
}

func BenchmarkValidateProperty(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ValidateProperty(PropPayloadFormatIndicator, byte(1))
	}
}

func BenchmarkAddProperty_Byte(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		props := &Properties{}
		err := props.AddProperty(PropPayloadFormatIndicator, byte(1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddProperty_String(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		props := &Properties{}
		err := props.AddProperty(PropContentType, "application/json")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddProperty_UserProperty(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		props := &Properties{}
		err := props.AddProperty(PropUserProperty, UTF8Pair{Key: "key", Value: "value"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetProperty(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
			{ID: PropResponseTopic, Value: "response/topic"},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = props.GetProperty(PropContentType)
	}
}

func BenchmarkGetProperties_Single(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = props.GetProperties(PropContentType)
	}
}

func BenchmarkGetProperties_Multiple(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropUserProperty, Value: UTF8Pair{Key: "k1", Value: "v1"}},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "k2", Value: "v2"}},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "k3", Value: "v3"}},
			{ID: PropContentType, Value: "text/plain"},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = props.GetProperties(PropUserProperty)
	}
}

func BenchmarkParsePropertiesFromBytes_LargeCollection(b *testing.B) {
	props := &Properties{Properties: []Property{}}
	for i := 0; i < 50; i++ {
		props.Properties = append(props.Properties, Property{
			ID:    PropUserProperty,
			Value: UTF8Pair{Key: "key", Value: "value"},
		})
	}

	buf := make([]byte, 4096)
	n, err := props.EncodePropertiesToBytes(buf)
	if err != nil {
		b.Fatal(err)
	}
	data := buf[:n]

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := ParsePropertiesFromBytes(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_LargeCollection(b *testing.B) {
	props := &Properties{Properties: []Property{}}
	for i := 0; i < 50; i++ {
		props.Properties = append(props.Properties, Property{
			ID:    PropUserProperty,
			Value: UTF8Pair{Key: "key", Value: "value"},
		})
	}

	buf := make([]byte, 4096)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPropertyBuilder_AllProperties(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder := NewPropertyBuilder()
		_, err := builder.
			WithPayloadFormat(1).
			WithMessageExpiry(3600).
			WithContentType("text/plain").
			WithResponseTopic("response/topic").
			WithCorrelationData([]byte{1, 2, 3, 4}).
			WithSubscriptionIdentifier(100).
			WithSessionExpiry(7200).
			WithAssignedClientID("client123").
			WithServerKeepAlive(60).
			WithAuthenticationMethod("SCRAM-SHA-256").
			WithAuthenticationData([]byte{0xAA, 0xBB}).
			WithRequestProblemInfo(1).
			WithWillDelay(30).
			WithRequestResponseInfo(1).
			WithResponseInfo("info").
			WithServerReference("mqtt.example.com").
			WithReasonString("Success").
			WithReceiveMaximum(100).
			WithTopicAliasMaximum(10).
			WithTopicAlias(5).
			WithMaximumQoS(2).
			WithRetainAvailable(1).
			WithUserProperty("app", "test").
			WithMaximumPacketSize(65535).
			WithWildcardSubscriptionAvailable(1).
			WithSubscriptionIdentifierAvailable(1).
			WithSharedSubscriptionAvailable(1).
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_ConnectPacket(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropSessionExpiryInterval, Value: uint32(3600)},
			{ID: PropReceiveMaximum, Value: uint16(100)},
			{ID: PropMaximumPacketSize, Value: uint32(65535)},
			{ID: PropTopicAliasMaximum, Value: uint16(10)},
			{ID: PropRequestResponseInformation, Value: byte(1)},
			{ID: PropRequestProblemInformation, Value: byte(1)},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "client", Value: "mqtt-test"}},
		},
	}
	buf := make([]byte, 512)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodePropertiesToBytes_PublishPacket(b *testing.B) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropTopicAlias, Value: uint16(5)},
			{ID: PropResponseTopic, Value: "response/topic"},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "priority", Value: "high"}},
			{ID: PropContentType, Value: "application/json"},
		},
	}
	buf := make([]byte, 512)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := props.EncodePropertiesToBytes(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
