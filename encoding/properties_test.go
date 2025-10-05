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
	err := props.AddProperty(PropertyID(0xFF), byte(1))
	assert.ErrorIs(t, err, ErrInvalidPropertyID)
}

func TestPropertyID_String(t *testing.T) {
	tests := []struct {
		id   PropertyID
		want string
	}{
		{PropPayloadFormatIndicator, "PayloadFormatIndicator"},
		{PropMessageExpiryInterval, "MessageExpiryInterval"},
		{PropContentType, "ContentType"},
		{PropUserProperty, "UserProperty"},
		{PropertyID(0xFF), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.id.String())
		})
	}
}

func TestPropertySerializer(t *testing.T) {
	props := &Properties{
		Properties: []Property{
			{ID: PropPayloadFormatIndicator, Value: byte(1)},
			{ID: PropContentType, Value: "application/json"},
		},
	}

	buf := make([]byte, 256)
	serializer := NewPropertySerializer(buf)

	n, err := serializer.Serialize(props)
	require.NoError(t, err)
	assert.Greater(t, n, 0)
	assert.Equal(t, buf[:n], serializer.Buffer()[:n])

	parsed, bytesRead, err := ParsePropertiesFromBytes(buf[:n])
	require.NoError(t, err)
	assert.Equal(t, n, bytesRead)
	assert.Len(t, parsed.Properties, 2)
}

func TestPropertyBuilder(t *testing.T) {
	tests := []struct {
		name          string
		buildFunc     func(*PropertyBuilder) *PropertyBuilder
		expectError   bool
		expectedProps int
		validate      func(*testing.T, *Properties)
	}{
		{
			name: "single_byte_property",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b.WithPayloadFormat(1)
			},
			expectedProps: 1,
			validate: func(t *testing.T, p *Properties) {
				assert.Equal(t, PropPayloadFormatIndicator, p.Properties[0].ID)
				assert.Equal(t, byte(1), p.Properties[0].Value)
			},
		},
		{
			name: "multiple_properties",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b.
					WithPayloadFormat(1).
					WithMessageExpiry(3600).
					WithContentType("application/json")
			},
			expectedProps: 3,
			validate: func(t *testing.T, p *Properties) {
				assert.Equal(t, PropPayloadFormatIndicator, p.Properties[0].ID)
				assert.Equal(t, PropMessageExpiryInterval, p.Properties[1].ID)
				assert.Equal(t, PropContentType, p.Properties[2].ID)
			},
		},
		{
			name: "user_properties",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b.
					WithUserProperty("key1", "value1").
					WithUserProperty("key2", "value2")
			},
			expectedProps: 2,
			validate: func(t *testing.T, p *Properties) {
				assert.Equal(t, PropUserProperty, p.Properties[0].ID)
				assert.Equal(t, PropUserProperty, p.Properties[1].ID)
			},
		},
		{
			name: "all_property_types",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b.
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
					WithResponseInfo("some info").
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
					WithSharedSubscriptionAvailable(1)
			},
			expectedProps: 27,
		},
		{
			name: "duplicate_non_multiple_property",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b.
					WithPayloadFormat(1).
					WithPayloadFormat(0)
			},
			expectError: true,
		},
		{
			name: "empty_properties",
			buildFunc: func(b *PropertyBuilder) *PropertyBuilder {
				return b
			},
			expectedProps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewPropertyBuilder()
			builder = tt.buildFunc(builder)
			props, err := builder.Build()

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, props.Properties, tt.expectedProps)
				if tt.validate != nil {
					tt.validate(t, props)
				}
			}
		})
	}
}

func TestCalculatePropertiesSize(t *testing.T) {
	tests := []struct {
		name     string
		props    *Properties
		expected int
	}{
		{
			name:     "empty_properties",
			props:    &Properties{},
			expected: 1,
		},
		{
			name: "single_byte_property",
			props: &Properties{
				Properties: []Property{
					{ID: PropPayloadFormatIndicator, Value: byte(1)},
				},
			},
			expected: 3,
		},
		{
			name: "two_byte_int_property",
			props: &Properties{
				Properties: []Property{
					{ID: PropServerKeepAlive, Value: uint16(60)},
				},
			},
			expected: 4,
		},
		{
			name: "four_byte_int_property",
			props: &Properties{
				Properties: []Property{
					{ID: PropMessageExpiryInterval, Value: uint32(3600)},
				},
			},
			expected: 6,
		},
		{
			name: "string_property",
			props: &Properties{
				Properties: []Property{
					{ID: PropContentType, Value: "text/plain"},
				},
			},
			expected: 14,
		},
		{
			name: "multiple_properties",
			props: &Properties{
				Properties: []Property{
					{ID: PropPayloadFormatIndicator, Value: byte(1)},
					{ID: PropMessageExpiryInterval, Value: uint32(3600)},
					{ID: PropContentType, Value: "text/plain"},
				},
			},
			expected: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := CalculatePropertiesSize(tt.props)
			assert.Equal(t, tt.expected, size)
		})
	}
}

func TestValidateProperty(t *testing.T) {
	tests := []struct {
		name        string
		id          PropertyID
		value       interface{}
		expectError bool
	}{
		{
			name:        "valid_byte",
			id:          PropPayloadFormatIndicator,
			value:       byte(1),
			expectError: false,
		},
		{
			name:        "invalid_byte_type",
			id:          PropPayloadFormatIndicator,
			value:       uint16(1),
			expectError: true,
		},
		{
			name:        "valid_two_byte_int",
			id:          PropServerKeepAlive,
			value:       uint16(60),
			expectError: false,
		},
		{
			name:        "invalid_two_byte_int_type",
			id:          PropServerKeepAlive,
			value:       uint32(60),
			expectError: true,
		},
		{
			name:        "valid_four_byte_int",
			id:          PropMessageExpiryInterval,
			value:       uint32(3600),
			expectError: false,
		},
		{
			name:        "invalid_four_byte_int_type",
			id:          PropMessageExpiryInterval,
			value:       uint16(3600),
			expectError: true,
		},
		{
			name:        "valid_varint",
			id:          PropSubscriptionIdentifier,
			value:       uint32(127),
			expectError: false,
		},
		{
			name:        "invalid_varint_too_large",
			id:          PropSubscriptionIdentifier,
			value:       uint32(268435456),
			expectError: true,
		},
		{
			name:        "valid_utf8_string",
			id:          PropContentType,
			value:       "text/plain",
			expectError: false,
		},
		{
			name:        "invalid_utf8_string_type",
			id:          PropContentType,
			value:       []byte("text/plain"),
			expectError: true,
		},
		{
			name:        "valid_utf8_pair",
			id:          PropUserProperty,
			value:       UTF8Pair{Key: "key", Value: "value"},
			expectError: false,
		},
		{
			name:        "invalid_utf8_pair_type",
			id:          PropUserProperty,
			value:       "key=value",
			expectError: true,
		},
		{
			name:        "valid_binary_data",
			id:          PropCorrelationData,
			value:       []byte{1, 2, 3, 4},
			expectError: false,
		},
		{
			name:        "invalid_binary_data_type",
			id:          PropCorrelationData,
			value:       "binary",
			expectError: true,
		},
		{
			name:        "invalid_property_id",
			id:          PropertyID(0xFF),
			value:       byte(1),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProperty(tt.id, tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPropertiesEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		properties []Property
	}{
		{
			name:       "empty",
			properties: []Property{},
		},
		{
			name: "all_property_types",
			properties: []Property{
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
		},
		{
			name: "large_varint",
			properties: []Property{
				{ID: PropSubscriptionIdentifier, Value: uint32(268435455)},
			},
		},
		{
			name: "empty_strings",
			properties: []Property{
				{ID: PropContentType, Value: ""},
				{ID: PropResponseTopic, Value: ""},
			},
		},
		{
			name: "empty_binary",
			properties: []Property{
				{ID: PropCorrelationData, Value: []byte{}},
			},
		},
		{
			name: "long_strings",
			properties: []Property{
				{ID: PropContentType, Value: "application/vnd.oasis.opendocument.text"},
				{ID: PropReasonString, Value: "This is a very long reason string that describes in detail what happened"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &Properties{Properties: tt.properties}

			buf := make([]byte, 4096)
			n, err := original.EncodePropertiesToBytes(buf)
			require.NoError(t, err)
			require.Greater(t, n, 0)

			decoded, bytesRead, err := ParsePropertiesFromBytes(buf[:n])
			require.NoError(t, err)
			assert.Equal(t, n, bytesRead)
			assert.Len(t, decoded.Properties, len(original.Properties))

			for i, prop := range original.Properties {
				assert.Equal(t, prop.ID, decoded.Properties[i].ID)
				assert.Equal(t, prop.Value, decoded.Properties[i].Value)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		testFunc    func(t *testing.T)
		expectError bool
	}{
		{
			name: "buffer_too_small",
			testFunc: func(t *testing.T) {
				props := &Properties{
					Properties: []Property{
						{ID: PropPayloadFormatIndicator, Value: byte(1)},
					},
				}
				buf := make([]byte, 1)
				_, err := props.EncodePropertiesToBytes(buf)
				assert.ErrorIs(t, err, ErrBufferTooSmall)
			},
		},
		{
			name: "parse_truncated_property_length",
			testFunc: func(t *testing.T) {
				data := []byte{}
				_, _, err := ParsePropertiesFromBytes(data)
				assert.ErrorIs(t, err, ErrUnexpectedEOF)
			},
		},
		{
			name: "parse_truncated_property_data",
			testFunc: func(t *testing.T) {
				data := []byte{0x05, 0x01}
				_, _, err := ParsePropertiesFromBytes(data)
				assert.Error(t, err)
			},
		},
		{
			name: "large_property_collection",
			testFunc: func(t *testing.T) {
				props := &Properties{Properties: []Property{}}
				for i := 0; i < 100; i++ {
					props.Properties = append(props.Properties, Property{
						ID:    PropUserProperty,
						Value: UTF8Pair{Key: "key", Value: "value"},
					})
				}

				buf := make([]byte, 10000)
				n, err := props.EncodePropertiesToBytes(buf)
				require.NoError(t, err)

				decoded, _, err := ParsePropertiesFromBytes(buf[:n])
				require.NoError(t, err)
				assert.Len(t, decoded.Properties, 100)
			},
		},
		{
			name: "property_with_max_varint",
			testFunc: func(t *testing.T) {
				props := &Properties{
					Properties: []Property{
						{ID: PropSubscriptionIdentifier, Value: uint32(268435455)},
					},
				}

				buf := make([]byte, 256)
				n, err := props.EncodePropertiesToBytes(buf)
				require.NoError(t, err)

				decoded, _, err := ParsePropertiesFromBytes(buf[:n])
				require.NoError(t, err)
				assert.Equal(t, uint32(268435455), decoded.Properties[0].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

func TestPropertiesGettersAndSetters(t *testing.T) {
	props := &Properties{}

	err := props.AddProperty(PropPayloadFormatIndicator, byte(1))
	require.NoError(t, err)

	err = props.AddProperty(PropContentType, "text/plain")
	require.NoError(t, err)

	err = props.AddProperty(PropUserProperty, UTF8Pair{Key: "k1", Value: "v1"})
	require.NoError(t, err)

	err = props.AddProperty(PropUserProperty, UTF8Pair{Key: "k2", Value: "v2"})
	require.NoError(t, err)

	prop := props.GetProperty(PropPayloadFormatIndicator)
	require.NotNil(t, prop)
	assert.Equal(t, byte(1), prop.Value)

	prop = props.GetProperty(PropContentType)
	require.NotNil(t, prop)
	assert.Equal(t, "text/plain", prop.Value)

	userProps := props.GetProperties(PropUserProperty)
	assert.Len(t, userProps, 2)

	prop = props.GetProperty(PropMessageExpiryInterval)
	assert.Nil(t, prop)
}

func TestComplexPropertyCombinations(t *testing.T) {
	tests := []struct {
		name       string
		properties []Property
	}{
		{
			name: "connect_properties",
			properties: []Property{
				{ID: PropSessionExpiryInterval, Value: uint32(3600)},
				{ID: PropReceiveMaximum, Value: uint16(100)},
				{ID: PropMaximumPacketSize, Value: uint32(65535)},
				{ID: PropTopicAliasMaximum, Value: uint16(10)},
				{ID: PropRequestResponseInformation, Value: byte(1)},
				{ID: PropRequestProblemInformation, Value: byte(1)},
				{ID: PropUserProperty, Value: UTF8Pair{Key: "client", Value: "mqtt-test"}},
				{ID: PropAuthenticationMethod, Value: "SCRAM-SHA-256"},
				{ID: PropAuthenticationData, Value: []byte{0x01, 0x02, 0x03}},
			},
		},
		{
			name: "connack_properties",
			properties: []Property{
				{ID: PropSessionExpiryInterval, Value: uint32(7200)},
				{ID: PropReceiveMaximum, Value: uint16(65535)},
				{ID: PropMaximumQoS, Value: byte(2)},
				{ID: PropRetainAvailable, Value: byte(1)},
				{ID: PropMaximumPacketSize, Value: uint32(268435455)},
				{ID: PropAssignedClientIdentifier, Value: "auto-generated-id"},
				{ID: PropTopicAliasMaximum, Value: uint16(20)},
				{ID: PropReasonString, Value: "Connection accepted"},
				{ID: PropWildcardSubscriptionAvailable, Value: byte(1)},
				{ID: PropSubscriptionIdentifierAvailable, Value: byte(1)},
				{ID: PropSharedSubscriptionAvailable, Value: byte(1)},
				{ID: PropServerKeepAlive, Value: uint16(120)},
				{ID: PropResponseInformation, Value: "response/info"},
				{ID: PropServerReference, Value: "mqtt.backup.example.com"},
				{ID: PropAuthenticationMethod, Value: "SCRAM-SHA-256"},
				{ID: PropAuthenticationData, Value: []byte{0xAA, 0xBB, 0xCC}},
			},
		},
		{
			name: "publish_properties",
			properties: []Property{
				{ID: PropPayloadFormatIndicator, Value: byte(1)},
				{ID: PropMessageExpiryInterval, Value: uint32(3600)},
				{ID: PropTopicAlias, Value: uint16(5)},
				{ID: PropResponseTopic, Value: "response/topic"},
				{ID: PropCorrelationData, Value: []byte{0x01, 0x02, 0x03, 0x04}},
				{ID: PropUserProperty, Value: UTF8Pair{Key: "priority", Value: "high"}},
				{ID: PropUserProperty, Value: UTF8Pair{Key: "source", Value: "sensor-1"}},
				{ID: PropContentType, Value: "application/json"},
			},
		},
		{
			name: "subscribe_properties",
			properties: []Property{
				{ID: PropSubscriptionIdentifier, Value: uint32(1)},
				{ID: PropUserProperty, Value: UTF8Pair{Key: "group", Value: "monitoring"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &Properties{Properties: tt.properties}

			buf := make([]byte, 4096)
			n, err := original.EncodePropertiesToBytes(buf)
			require.NoError(t, err)

			decoded, bytesRead, err := ParsePropertiesFromBytes(buf[:n])
			require.NoError(t, err)
			assert.Equal(t, n, bytesRead)
			assert.Len(t, decoded.Properties, len(original.Properties))

			for i := range original.Properties {
				assert.Equal(t, original.Properties[i].ID, decoded.Properties[i].ID)
				assert.Equal(t, original.Properties[i].Value, decoded.Properties[i].Value)
			}
		})
	}
}

func TestProperties_EmptyProperties(t *testing.T) {
	var buf bytes.Buffer
	props := &Properties{}
	err := props.EncodeProperties(&buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00}, buf.Bytes())
}
