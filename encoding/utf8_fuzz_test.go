package encoding

import (
	"testing"
	"unicode/utf8"
)

// FuzzValidateUTF8String fuzzes the ValidateUTF8String function
func FuzzValidateUTF8String(f *testing.F) {
	// Add seed corpus
	f.Add([]byte("Hello, World!"))
	f.Add([]byte(""))
	f.Add([]byte("🌍"))
	f.Add([]byte("你好"))
	f.Add([]byte{0x00})             // null
	f.Add([]byte{0xFF, 0xFE})       // invalid UTF-8
	f.Add([]byte{0xEF, 0xBF, 0xBE}) // U+FFFE
	f.Add([]byte{0xED, 0xA0, 0x80}) // surrogate
	f.Add([]byte("Hello\x00World"))
	f.Add([]byte("Test\x01\x02\x03"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// The function should never panic
		err := ValidateUTF8String(data)

		// If no error, the string should be valid UTF-8
		if err == nil {
			if !utf8.Valid(data) {
				t.Errorf("ValidateUTF8String returned nil but data is not valid UTF-8")
			}

			// Check for null bytes
			for _, b := range data {
				if b == 0 {
					t.Errorf("ValidateUTF8String returned nil but data contains null byte")
				}
			}

			// Verify no disallowed code points
			for _, r := range string(data) {
				if r >= 0xD800 && r <= 0xDFFF {
					t.Errorf("ValidateUTF8String returned nil but data contains surrogate: U+%04X", r)
				}
				if (r&0xFFFF) == 0xFFFE || (r&0xFFFF) == 0xFFFF {
					t.Errorf("ValidateUTF8String returned nil but data contains non-character: U+%04X", r)
				}
				if r >= 0xFDD0 && r <= 0xFDEF {
					t.Errorf("ValidateUTF8String returned nil but data contains non-character: U+%04X", r)
				}
			}
		}

		// If error, it should be one of our defined errors
		if err != nil {
			switch err {
			case ErrInvalidUTF8, ErrNullCharacter, ErrSurrogateCodePoint,
				ErrNonCharacterCodePoint, ErrInvalidCodePoint:
				// Valid error
			default:
				t.Errorf("ValidateUTF8String returned unexpected error: %v", err)
			}
		}
	})
}

// FuzzValidateUTF8StringStrict fuzzes the strict validation function
func FuzzValidateUTF8StringStrict(f *testing.F) {
	// Add seed corpus
	f.Add([]byte("Hello, World!"))
	f.Add([]byte(""))
	f.Add([]byte("Test\tString"))
	f.Add([]byte("Line1\nLine2"))
	f.Add([]byte{0x01})       // control character
	f.Add([]byte{0x7F})       // DEL
	f.Add([]byte{0xC2, 0x80}) // U+0080

	f.Fuzz(func(t *testing.T, data []byte) {
		// The function should never panic
		err := ValidateUTF8StringStrict(data)

		// If no error from strict, regular validation should also pass
		if err == nil {
			if regularErr := ValidateUTF8String(data); regularErr != nil {
				t.Errorf("ValidateUTF8StringStrict returned nil but ValidateUTF8String returned: %v", regularErr)
			}
		}

		// If error, it should be one of our defined errors
		if err != nil {
			switch err {
			case ErrInvalidUTF8, ErrNullCharacter, ErrSurrogateCodePoint,
				ErrNonCharacterCodePoint, ErrInvalidCodePoint, ErrControlCharacter:
				// Valid error
			default:
				t.Errorf("ValidateUTF8StringStrict returned unexpected error: %v", err)
			}
		}
	})
}
