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

func TestReadTwoByteIntFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expected  uint16
		expectErr bool
	}{
		{
			name:     "valid two bytes",
			data:     []byte{0x12, 0x34, 0xFF},
			expected: 0x1234,
		},
		{
			name:      "insufficient data",
			data:      []byte{0x12},
			expectErr: true,
		},
		{
			name:      "empty data",
			data:      []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readTwoByteIntFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
				assert.Equal(t, 2, n)
			}
		})
	}
}

func TestReadFourByteIntFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expected  uint32
		expectErr bool
	}{
		{
			name:     "valid four bytes",
			data:     []byte{0x12, 0x34, 0x56, 0x78, 0xFF},
			expected: 0x12345678,
		},
		{
			name:      "insufficient data - 3 bytes",
			data:      []byte{0x12, 0x34, 0x56},
			expectErr: true,
		},
		{
			name:      "insufficient data - 1 byte",
			data:      []byte{0x12},
			expectErr: true,
		},
		{
			name:      "empty data",
			data:      []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readFourByteIntFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
				assert.Equal(t, 4, n)
			}
		})
	}
}

func TestReadUTF8StringFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expected  string
		expectErr bool
	}{
		{
			name:     "valid string",
			data:     []byte{0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0xFF},
			expected: "hello",
		},
		{
			name:     "empty string",
			data:     []byte{0x00, 0x00},
			expected: "",
		},
		{
			name:      "insufficient length bytes",
			data:      []byte{0x00},
			expectErr: true,
		},
		{
			name:      "incomplete string data",
			data:      []byte{0x00, 0x05, 'h', 'i'},
			expectErr: true,
		},
		{
			name:      "no data",
			data:      []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, _, err := readUTF8StringFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestReadUTF8PairFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectedK string
		expectedV string
		expectErr bool
	}{
		{
			name:      "valid pair",
			data:      []byte{0x00, 0x03, 'k', 'e', 'y', 0x00, 0x05, 'v', 'a', 'l', 'u', 'e'},
			expectedK: "key",
			expectedV: "value",
		},
		{
			name:      "empty key and value",
			data:      []byte{0x00, 0x00, 0x00, 0x00},
			expectedK: "",
			expectedV: "",
		},
		{
			name:      "incomplete key",
			data:      []byte{0x00, 0x05, 'k', 'e'},
			expectErr: true,
		},
		{
			name:      "incomplete value",
			data:      []byte{0x00, 0x03, 'k', 'e', 'y', 0x00, 0x05, 'v', 'a'},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readUTF8PairFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedK, val.Key)
				assert.Equal(t, tt.expectedV, val.Value)
				assert.Greater(t, n, 0)
			}
		})
	}
}

func TestReadBinaryDataFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expected  []byte
		expectErr bool
	}{
		{
			name:     "valid binary data",
			data:     []byte{0x00, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05, 0xFF},
			expected: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
		{
			name:     "empty binary data",
			data:     []byte{0x00, 0x00},
			expected: []byte{},
		},
		{
			name:      "insufficient length bytes",
			data:      []byte{0x00},
			expectErr: true,
		},
		{
			name:      "incomplete binary data",
			data:      []byte{0x00, 0x05, 0x01, 0x02},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readBinaryDataFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
				assert.Greater(t, n, 0)
			}
		})
	}
}

func TestWriteTwoByteIntToBytes(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		value     uint16
		expected  []byte
		expectErr bool
	}{
		{
			name:     "valid write",
			bufSize:  3,
			value:    0x1234,
			expected: []byte{0x12, 0x34},
		},
		{
			name:      "buffer too small",
			bufSize:   1,
			value:     0x1234,
			expectErr: true,
		},
		{
			name:      "zero buffer",
			bufSize:   0,
			value:     0x1234,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n, err := writeTwoByteIntToBytes(buf, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2, n)
				assert.Equal(t, tt.expected, buf[:n])
			}
		})
	}
}

func TestWriteFourByteIntToBytes(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		value     uint32
		expected  []byte
		expectErr bool
	}{
		{
			name:     "valid write",
			bufSize:  5,
			value:    0x12345678,
			expected: []byte{0x12, 0x34, 0x56, 0x78},
		},
		{
			name:      "buffer too small - 3 bytes",
			bufSize:   3,
			value:     0x12345678,
			expectErr: true,
		},
		{
			name:      "buffer too small - 1 byte",
			bufSize:   1,
			value:     0x12345678,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n, err := writeFourByteIntToBytes(buf, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 4, n)
				assert.Equal(t, tt.expected, buf[:n])
			}
		})
	}
}

func TestWriteUTF8StringToBytes(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		value     string
		expectErr bool
	}{
		{
			name:    "valid write",
			bufSize: 10,
			value:   "hello",
		},
		{
			name:    "empty string",
			bufSize: 2,
			value:   "",
		},
		{
			name:      "buffer too small",
			bufSize:   3,
			value:     "hello",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n, err := writeUTF8StringToBytes(buf, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2+len(tt.value), n)
			}
		})
	}
}

func TestWriteUTF8PairToBytes(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		key       string
		value     string
		expectErr bool
	}{
		{
			name:    "valid write",
			bufSize: 20,
			key:     "key",
			value:   "value",
		},
		{
			name:    "empty pair",
			bufSize: 4,
			key:     "",
			value:   "",
		},
		{
			name:      "buffer too small for key",
			bufSize:   3,
			key:       "key",
			value:     "value",
			expectErr: true,
		},
		{
			name:      "buffer too small for value",
			bufSize:   6,
			key:       "key",
			value:     "value",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			pair := UTF8Pair{Key: tt.key, Value: tt.value}
			n, err := writeUTF8PairToBytes(buf, pair)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				expectedLen := 2 + len(tt.key) + 2 + len(tt.value)
				assert.Equal(t, expectedLen, n)
			}
		})
	}
}

func TestWriteBinaryDataToBytes(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		data      []byte
		expectErr bool
	}{
		{
			name:    "valid write",
			bufSize: 10,
			data:    []byte{0x01, 0x02, 0x03},
		},
		{
			name:    "empty data",
			bufSize: 2,
			data:    []byte{},
		},
		{
			name:      "buffer too small",
			bufSize:   3,
			data:      []byte{0x01, 0x02, 0x03},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n, err := writeBinaryDataToBytes(buf, tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2+len(tt.data), n)
			}
		})
	}
}

func TestReadByteFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expected  byte
		expectErr bool
	}{
		{
			name:     "valid byte",
			data:     []byte{0x42, 0xFF},
			expected: 0x42,
		},
		{
			name:      "empty data",
			data:      []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readByteFromBytes(tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
				assert.Equal(t, 1, n)
			}
		})
	}
}

func TestWriteByteToBytes_BufferTooSmall(t *testing.T) {
	buf := make([]byte, 0)
	n, err := writeByteToBytes(buf, 0x42)
	assert.ErrorIs(t, err, ErrBufferTooSmall)
	assert.Equal(t, 0, n)
}

func TestParsePropertiesFromBytes_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid property length",
			data: []byte{0xFF},
		},
		{
			name: "incomplete property",
			data: []byte{0x02, 0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePropertiesFromBytes(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestEncodePropertiesToBytes_ErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		props Properties
		buf   []byte
	}{
		{
			name: "buffer too small",
			props: Properties{
				Properties: []Property{
					{ID: PropPayloadFormatIndicator, Value: byte(1)},
				},
			},
			buf: make([]byte, 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.props.EncodePropertiesToBytes(tt.buf)
			assert.Error(t, err)
		})
	}
}

func TestCalculateLength_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		prop     Property
		expected int
	}{
		{
			name:     "byte property",
			prop:     Property{ID: PropPayloadFormatIndicator, Value: byte(1)},
			expected: 2,
		},
		{
			name:     "two byte int",
			prop:     Property{ID: PropServerKeepAlive, Value: uint16(60)},
			expected: 3,
		},
		{
			name:     "four byte int",
			prop:     Property{ID: PropMessageExpiryInterval, Value: uint32(60)},
			expected: 5,
		},
		{
			name:     "UTF8 string",
			prop:     Property{ID: PropContentType, Value: "test"},
			expected: 7,
		},
		{
			name:     "UTF8 pair",
			prop:     Property{ID: PropUserProperty, Value: UTF8Pair{Key: "k", Value: "v"}},
			expected: 7,
		},
		{
			name:     "binary data",
			prop:     Property{ID: PropCorrelationData, Value: []byte{0x01, 0x02}},
			expected: 5,
		},
		{
			name:     "varint",
			prop:     Property{ID: PropSubscriptionIdentifier, Value: uint32(127)},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := &Properties{Properties: []Property{tt.prop}}
			length := props.calculateLength()
			assert.Equal(t, uint32(tt.expected), length)
		})
	}
}

func TestEncodePropertyToBytes_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		prop Property
		buf  []byte
	}{
		{
			name: "byte property buffer too small",
			prop: Property{ID: PropPayloadFormatIndicator, Value: byte(1)},
			buf:  make([]byte, 1),
		},
		{
			name: "two byte int buffer too small",
			prop: Property{ID: PropServerKeepAlive, Value: uint16(60)},
			buf:  make([]byte, 2),
		},
		{
			name: "four byte int buffer too small",
			prop: Property{ID: PropMessageExpiryInterval, Value: uint32(60)},
			buf:  make([]byte, 3),
		},
		{
			name: "UTF8 string buffer too small",
			prop: Property{ID: PropContentType, Value: "test"},
			buf:  make([]byte, 3),
		},
		{
			name: "binary data buffer too small",
			prop: Property{ID: PropCorrelationData, Value: []byte{0x01, 0x02}},
			buf:  make([]byte, 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encodePropertyToBytes(tt.buf, &tt.prop)
			assert.Error(t, err)
		})
	}
}

func TestReadByte_Error(t *testing.T) {
	r := bytes.NewReader([]byte{})
	_, err := readByte(r)
	assert.Error(t, err)
}

func TestReadTwoByteInt_Error(t *testing.T) {
	r := bytes.NewReader([]byte{0x01})
	_, err := readTwoByteInt(r)
	assert.Error(t, err)
}

func TestReadFourByteInt_Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "3 bytes",
			data: []byte{0x01, 0x02, 0x03},
		},
		{
			name: "0 bytes",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			_, err := readFourByteInt(r)
			assert.Error(t, err)
		})
	}
}

func TestReadUTF8String_Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "incomplete length",
			data: []byte{0x00},
		},
		{
			name: "incomplete string",
			data: []byte{0x00, 0x05, 'h', 'i'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			_, err := readUTF8String(r)
			assert.Error(t, err)
		})
	}
}

func TestReadUTF8Pair_Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "incomplete key",
			data: []byte{0x00, 0x05, 'k'},
		},
		{
			name: "incomplete value",
			data: []byte{0x00, 0x01, 'k', 0x00, 0x05, 'v'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			_, err := readUTF8Pair(r)
			assert.Error(t, err)
		})
	}
}

func TestReadBinaryData_Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "incomplete length",
			data: []byte{0x00},
		},
		{
			name: "incomplete data",
			data: []byte{0x00, 0x05, 0x01, 0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			_, err := readBinaryData(r)
			assert.Error(t, err)
		})
	}
}

func TestWriteUTF8String_EmptyString(t *testing.T) {
	var buf bytes.Buffer
	err := writeUTF8String(&buf, "")
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0x00}, buf.Bytes())
}

func TestWriteBinaryData_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	err := writeBinaryData(&buf, []byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0x00}, buf.Bytes())
}

func TestEncodeProperties_EmptyProperties(t *testing.T) {
	var buf bytes.Buffer
	props := &Properties{}
	err := props.EncodeProperties(&buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00}, buf.Bytes())
}
