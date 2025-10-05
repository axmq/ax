package encoding

import (
	"bytes"
	"testing"
)

// TestUTF8ValidationIntegration tests that UTF-8 validation is properly integrated
// into the property parsing functions
func TestUTF8ValidationIntegration(t *testing.T) {
	tests := []struct {
		name        string
		propertyID  PropertyID
		data        []byte
		expectError error
	}{
		{
			name:       "Valid UTF-8 string property",
			propertyID: PropContentType,
			data: []byte{
				0x03,       // Property ID: ContentType
				0x00, 0x0A, // Length: 10
				't', 'e', 'x', 't', '/', 'p', 'l', 'a', 'i', 'n',
			},
			expectError: nil,
		},
		{
			name:       "UTF-8 string with emoji",
			propertyID: PropReasonString,
			data: []byte{
				0x1F,       // Property ID: ReasonString
				0x00, 0x04, // Length: 4
				0xF0, 0x9F, 0x98, 0x80, // 😀 emoji
			},
			expectError: nil,
		},
		{
			name:       "String with null character - should fail",
			propertyID: PropContentType,
			data: []byte{
				0x03,       // Property ID: ContentType
				0x00, 0x05, // Length: 5
				't', 'e', 0x00, 's', 't', // Contains null
			},
			expectError: ErrNullCharacter,
		},
		{
			name:       "String with invalid UTF-8 - should fail",
			propertyID: PropContentType,
			data: []byte{
				0x03,       // Property ID: ContentType
				0x00, 0x03, // Length: 3
				0xFF, 0xFE, 0xFD, // Invalid UTF-8
			},
			expectError: ErrInvalidUTF8,
		},
		{
			name:       "String with non-character U+FFFE - should fail",
			propertyID: PropReasonString,
			data: []byte{
				0x1F,       // Property ID: ReasonString
				0x00, 0x03, // Length: 3
				0xEF, 0xBF, 0xBE, // U+FFFE
			},
			expectError: ErrNonCharacterCodePoint,
		},
		{
			name:       "Valid user property pair",
			propertyID: PropUserProperty,
			data: []byte{
				0x26,       // Property ID: UserProperty
				0x00, 0x03, // Key length: 3
				'k', 'e', 'y',
				0x00, 0x05, // Value length: 5
				'v', 'a', 'l', 'u', 'e',
			},
			expectError: nil,
		},
		{
			name:       "User property with null in key - should fail",
			propertyID: PropUserProperty,
			data: []byte{
				0x26,       // Property ID: UserProperty
				0x00, 0x03, // Key length: 3
				'k', 0x00, 'y', // Null in key
				0x00, 0x05, // Value length: 5
				'v', 'a', 'l', 'u', 'e',
			},
			expectError: ErrNullCharacter,
		},
		{
			name:       "User property with null in value - should fail",
			propertyID: PropUserProperty,
			data: []byte{
				0x26,       // Property ID: UserProperty
				0x00, 0x03, // Key length: 3
				'k', 'e', 'y',
				0x00, 0x05, // Value length: 5
				'v', 0x00, 'l', 'u', 'e', // Null in value
			},
			expectError: ErrNullCharacter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with io.Reader-based parsing
			reader := bytes.NewReader(tt.data)
			prop, err := parseProperty(reader)

			if tt.expectError != nil {
				if err != tt.expectError {
					t.Errorf("parseProperty() error = %v, want %v", err, tt.expectError)
				}
			} else {
				if err != nil {
					t.Errorf("parseProperty() unexpected error = %v", err)
				}
				if prop.ID != tt.propertyID {
					t.Errorf("parseProperty() property ID = %v, want %v", prop.ID, tt.propertyID)
				}
			}

			// Test with byte-slice-based parsing
			prop2, _, err2 := parsePropertyFromBytes(tt.data)

			if tt.expectError != nil {
				if err2 != tt.expectError {
					t.Errorf("parsePropertyFromBytes() error = %v, want %v", err2, tt.expectError)
				}
			} else {
				if err2 != nil {
					t.Errorf("parsePropertyFromBytes() unexpected error = %v", err2)
				}
				if prop2.ID != tt.propertyID {
					t.Errorf("parsePropertyFromBytes() property ID = %v, want %v", prop2.ID, tt.propertyID)
				}
			}
		})
	}
}

// TestUTF8ValidationInFullPropertyParsing tests UTF-8 validation in complete property collections
func TestUTF8ValidationInFullPropertyParsing(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError error
	}{
		{
			name: "Valid properties collection",
			data: []byte{
				0x0E, // Property length: 14 bytes (property data only, not including this varint)
				// ContentType property: 1 + 2 + 4 = 7 bytes
				0x03,       // ContentType (1 byte)
				0x00, 0x04, // String length: 4 (2 bytes)
				't', 'e', 's', 't', // String data (4 bytes)
				// UserProperty: 1 + 2 + 1 + 2 + 1 = 7 bytes
				0x26,            // UserProperty (1 byte)
				0x00, 0x01, 'a', // Key: "a" (2 + 1 = 3 bytes)
				0x00, 0x01, 'b', // Value: "b" (2 + 1 = 3 bytes)
			},
			expectError: nil,
		},
		{
			name: "Properties with invalid UTF-8",
			data: []byte{
				0x07,       // Property length: 7 bytes
				0x03,       // ContentType (1 byte)
				0x00, 0x04, // String length: 4 (2 bytes)
				0xFF, 0xFE, 0xFD, 0xFC, // Invalid UTF-8 (4 bytes)
			},
			expectError: ErrInvalidUTF8,
		},
		{
			name: "Multiple valid properties",
			data: []byte{
				0x18, // Property length: 24 bytes
				// ReasonString property: 1 + 2 + 7 = 10 bytes
				0x1F,       // ReasonString (1 byte)
				0x00, 0x07, // String length: 7 (2 bytes)
				'S', 'u', 'c', 'c', 'e', 's', 's', // String data (7 bytes)
				// UserProperty: 1 + 2 + 4 + 2 + 5 = 14 bytes
				0x26,                           // UserProperty (1 byte)
				0x00, 0x04, 't', 'e', 's', 't', // Key: "test" (2 + 4 = 6 bytes)
				0x00, 0x05, 'v', 'a', 'l', 'u', 'e', // Value: "value" (2 + 5 = 7 bytes)
			},
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with io.Reader
			reader := bytes.NewReader(tt.data)
			props, err := ParseProperties(reader)

			if tt.expectError != nil {
				if err != tt.expectError {
					t.Errorf("ParseProperties() error = %v, want %v", err, tt.expectError)
				}
			} else {
				if err != nil {
					t.Errorf("ParseProperties() unexpected error = %v", err)
				}
				if props == nil {
					t.Error("ParseProperties() returned nil properties")
				}
			}

			// Test with byte slice
			props2, _, err2 := ParsePropertiesFromBytes(tt.data)

			if tt.expectError != nil {
				if err2 != tt.expectError {
					t.Errorf("ParsePropertiesFromBytes() error = %v, want %v", err2, tt.expectError)
				}
			} else {
				if err2 != nil {
					t.Errorf("ParsePropertiesFromBytes() unexpected error = %v", err2)
				}
				if props2 == nil {
					t.Error("ParsePropertiesFromBytes() returned nil properties")
				}
			}
		})
	}
}
