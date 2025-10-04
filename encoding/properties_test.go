package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProperties_EmptyProperties(t *testing.T) {
	// Property length of 0
	data := []byte{0x00}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), props.Length)
	assert.Len(t, props.Properties, 0)
}

func TestParseProperties_SingleByteProperty(t *testing.T) {
	// Property length: 2 bytes
	// Property ID: PayloadFormatIndicator (0x01)
	// Value: 1 (UTF-8)
	data := []byte{0x02, 0x01, 0x01}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(2), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropPayloadFormatIndicator, props.Properties[0].ID)
	assert.Equal(t, byte(1), props.Properties[0].Value)
}

func TestParseProperties_TwoByteIntProperty(t *testing.T) {
	// Property length: 3 bytes
	// Property ID: ServerKeepAlive (0x13)
	// Value: 60 seconds
	data := []byte{0x03, 0x13, 0x00, 0x3C}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(3), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropServerKeepAlive, props.Properties[0].ID)
	assert.Equal(t, uint16(60), props.Properties[0].Value)
}

func TestParseProperties_FourByteIntProperty(t *testing.T) {
	// Property length: 5 bytes
	// Property ID: MessageExpiryInterval (0x02)
	// Value: 3600 seconds
	data := []byte{0x05, 0x02, 0x00, 0x00, 0x0E, 0x10}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(5), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropMessageExpiryInterval, props.Properties[0].ID)
	assert.Equal(t, uint32(3600), props.Properties[0].Value)
}

func TestParseProperties_UTF8StringProperty(t *testing.T) {
	// Property length: 13 bytes (1 ID + 2 length + 10 string)
	// Property ID: ContentType (0x03)
	// Value: "text/plain"
	data := []byte{0x0D, 0x03, 0x00, 0x0A, 't', 'e', 'x', 't', '/', 'p', 'l', 'a', 'i', 'n'}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(13), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropContentType, props.Properties[0].ID)
	assert.Equal(t, "text/plain", props.Properties[0].Value)
}

func TestParseProperties_UTF8PairProperty(t *testing.T) {
	// Property length: 11 bytes
	// Property ID: UserProperty (0x26)
	// Key: "foo" (length 3)
	// Value: "bar" (length 3)
	data := []byte{0x0B, 0x26, 0x00, 0x03, 'f', 'o', 'o', 0x00, 0x03, 'b', 'a', 'r'}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(11), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropUserProperty, props.Properties[0].ID)

	pair, ok := props.Properties[0].Value.(UTF8Pair)
	require.True(t, ok)
	assert.Equal(t, "foo", pair.Key)
	assert.Equal(t, "bar", pair.Value)
}

func TestParseProperties_BinaryDataProperty(t *testing.T) {
	// Property length: 7 bytes
	// Property ID: CorrelationData (0x09)
	// Value: {0x01, 0x02, 0x03, 0x04}
	data := []byte{0x07, 0x09, 0x00, 0x04, 0x01, 0x02, 0x03, 0x04}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(7), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropCorrelationData, props.Properties[0].ID)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, props.Properties[0].Value)
}

func TestParseProperties_VarIntProperty(t *testing.T) {
	// Property length: 2 bytes
	// Property ID: SubscriptionIdentifier (0x0B)
	// Value: 127
	data := []byte{0x02, 0x0B, 0x7F}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(2), props.Length)
	assert.Len(t, props.Properties, 1)
	assert.Equal(t, PropSubscriptionIdentifier, props.Properties[0].ID)
	assert.Equal(t, uint32(127), props.Properties[0].Value)
}

func TestParseProperties_MultipleProperties(t *testing.T) {
	// Multiple properties
	// 1. PayloadFormatIndicator (0x01): 1 = 2 bytes (1 ID + 1 value)
	// 2. MessageExpiryInterval (0x02): 3600 = 5 bytes (1 ID + 4 value)
	// 3. ContentType (0x03): "text/plain" = 13 bytes (1 ID + 2 length + 10 string)
	// Total: 2 + 5 + 13 = 20 bytes
	data := []byte{
		0x14,       // Property length: 20 bytes
		0x01, 0x01, // PayloadFormatIndicator = 1
		0x02, 0x00, 0x00, 0x0E, 0x10, // MessageExpiryInterval = 3600
		0x03, 0x00, 0x0A, 't', 'e', 'x', 't', '/', 'p', 'l', 'a', 'i', 'n', // ContentType = "text/plain"
	}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Equal(t, uint32(20), props.Length)
	assert.Len(t, props.Properties, 3)

	assert.Equal(t, PropPayloadFormatIndicator, props.Properties[0].ID)
	assert.Equal(t, byte(1), props.Properties[0].Value)

	assert.Equal(t, PropMessageExpiryInterval, props.Properties[1].ID)
	assert.Equal(t, uint32(3600), props.Properties[1].Value)

	assert.Equal(t, PropContentType, props.Properties[2].ID)
	assert.Equal(t, "text/plain", props.Properties[2].Value)
}

func TestParseProperties_MultipleUserProperties(t *testing.T) {
	// Multiple user properties (allowed)
	data := []byte{
		0x16,                                                       // Property length: 22 bytes
		0x26, 0x00, 0x03, 'f', 'o', 'o', 0x00, 0x03, 'b', 'a', 'r', // UserProperty: foo=bar
		0x26, 0x00, 0x03, 'k', 'e', 'y', 0x00, 0x03, 'v', 'a', 'l', // UserProperty: key=val
	}
	r := bytes.NewReader(data)

	props, err := ParseProperties(r)
	require.NoError(t, err)
	assert.Len(t, props.Properties, 2)
	assert.Equal(t, PropUserProperty, props.Properties[0].ID)
	assert.Equal(t, PropUserProperty, props.Properties[1].ID)
}

func TestParseProperties_InvalidPropertyID(t *testing.T) {
	// Invalid property ID: 0xFF
	data := []byte{0x02, 0xFF, 0x00}
	r := bytes.NewReader(data)

	_, err := ParseProperties(r)
	assert.ErrorIs(t, err, ErrInvalidPropertyID)
}

func TestParseProperties_UnexpectedEOF(t *testing.T) {
	// Property length indicates 5 bytes but only 3 provided
	data := []byte{0x05, 0x02, 0x00}
	r := bytes.NewReader(data)

	_, err := ParseProperties(r)
	// When using io.LimitedReader, we get io.EOF when trying to read beyond the limit
	// which is then converted to ErrUnexpectedEOF by our read functions
	assert.Error(t, err)
}

func TestParsePropertiesFromBytes(t *testing.T) {
	// Property length: 2 bytes
	// Property ID: PayloadFormatIndicator (0x01)
	// Value: 1
	data := []byte{0x02, 0x01, 0x01}

	props, bytesRead, err := ParsePropertiesFromBytes(data)
	require.NoError(t, err)
	assert.Equal(t, 3, bytesRead)
	assert.Equal(t, uint32(2), props.Length)
	assert.Len(t, props.Properties, 1)
}

func TestEncodeProperties(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}

	var buf bytes.Buffer
	err := props.EncodeProperties(&buf)
	require.NoError(t, err)

	// Parse it back
	parsed, err := ParseProperties(&buf)
	require.NoError(t, err)
	assert.Len(t, parsed.Properties, 3)
	assert.Equal(t, PropPayloadFormatIndicator, parsed.Properties[0].ID)
	assert.Equal(t, byte(1), parsed.Properties[0].Value)
}

func TestEncodePropertiesToBytes(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
		},
	}

	buf := make([]byte, 128)
	n, err := props.EncodePropertiesToBytes(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0)

	// Parse it back
	parsed, bytesRead, err := ParsePropertiesFromBytes(buf[:n])
	require.NoError(t, err)
	assert.Equal(t, n, bytesRead)
	assert.Len(t, parsed.Properties, 1)
}

func TestProperties_GetProperty(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropContentType, Value: "text/plain"},
		},
	}

	prop := props.GetProperty(PropContentType)
	require.NotNil(t, prop)
	assert.Equal(t, "text/plain", prop.Value)

	prop = props.GetProperty(PropSessionExpiryInterval)
	assert.Nil(t, prop)
}

func TestProperties_GetProperties(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropUserProperty, Value: UTF8Pair{Key: "foo", Value: "bar"}},
			{ID: PropContentType, Value: "text/plain"},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "key", Value: "val"}},
		},
	}

	userProps := props.GetProperties(PropUserProperty)
	assert.Len(t, userProps, 2)

	contentProps := props.GetProperties(PropContentType)
	assert.Len(t, contentProps, 1)
}

func TestProperties_AddProperty(t *testing.T) {
	props := &Properties{}

	err := props.AddProperty(PropPayloadFormatIndicator, byte(1))
	require.NoError(t, err)
	assert.Len(t, props.Properties, 1)

	// Try to add duplicate non-multiple property
	err = props.AddProperty(PropPayloadFormatIndicator, byte(0))
	assert.ErrorIs(t, err, ErrDuplicateProperty)

	// Add multiple user properties (allowed)
	err = props.AddProperty(PropUserProperty, UTF8Pair{Key: "foo", Value: "bar"})
	require.NoError(t, err)
	err = props.AddProperty(PropUserProperty, UTF8Pair{Key: "key", Value: "val"})
	require.NoError(t, err)
	assert.Len(t, props.Properties, 3)
}

func TestProperties_AddProperty_InvalidID(t *testing.T) {
	props := &Properties{}

	err := props.AddProperty(PropertyID(0xFF), byte(0))
	assert.ErrorIs(t, err, ErrInvalidPropertyID)
}

func TestPropertyID_String(t *testing.T) {
	tests := []struct {
		id       PropertyID
		expected string
	}{
		{PropPayloadFormatIndicator, "PayloadFormatIndicator"},
		{PropContentType, "ContentType"},
		{PropUserProperty, "UserProperty"},
		{PropertyID(0xFF), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.id.String())
		})
	}
}

func TestEncodeDecodeRoundTrip_AllPropertyTypes(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropServerKeepAlive, Value: uint16(60)},
			{ID: PropMessageExpiryInterval, Value: uint32(3600)},
			{ID: PropSubscriptionIdentifier, Value: uint32(127)},
			{ID: PropContentType, Value: "text/plain"},
			{ID: PropUserProperty, Value: UTF8Pair{Key: "foo", Value: "bar"}},
			{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}

	var buf bytes.Buffer
	err := props.EncodeProperties(&buf)
	require.NoError(t, err)

	parsed, err := ParseProperties(&buf)
	require.NoError(t, err)
	assert.Len(t, parsed.Properties, 7)

	// Verify each property
	assert.Equal(t, byte(1), parsed.Properties[0].Value)
	assert.Equal(t, uint16(60), parsed.Properties[1].Value)
	assert.Equal(t, uint32(3600), parsed.Properties[2].Value)
	assert.Equal(t, uint32(127), parsed.Properties[3].Value)
	assert.Equal(t, "text/plain", parsed.Properties[4].Value)

	pair := parsed.Properties[5].Value.(UTF8Pair)
	assert.Equal(t, "foo", pair.Key)
	assert.Equal(t, "bar", pair.Value)

	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, parsed.Properties[6].Value)
}
