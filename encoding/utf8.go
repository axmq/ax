package encoding

import (
	"unicode/utf8"
)

// ValidateUTF8String validates a UTF-8 encoded string according to MQTT specification.
// MQTT 5.0 Section 1.5.4 specifies that UTF-8 Encoded Strings must:
// - Be valid UTF-8 as defined in RFC 3629
// - Not include null character U+0000
// - Not include code points between U+D800 and U+DFFF (UTF-16 surrogates)
// - Should not include U+0001 to U+001F or U+007F to U+009F (control characters)
// - Should not include non-character code points U+FFFE and U+FFFF
func ValidateUTF8String(data []byte) error {
	// Quick check for null bytes
	for _, b := range data {
		if b == 0 {
			return ErrNullCharacter
		}
	}

	// Validate UTF-8 encoding and check for disallowed code points
	if !utf8.Valid(data) {
		return ErrInvalidUTF8
	}

	// Check each rune for disallowed code points
	i := 0
	for i < len(data) {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError {
			// This shouldn't happen since we already validated, but check anyway
			if size == 1 {
				return ErrInvalidUTF8
			}
		}

		// Check for disallowed code points
		if err := validateCodePoint(r); err != nil {
			return err
		}

		i += size
	}

	return nil
}

// validateCodePoint checks if a Unicode code point is allowed in MQTT UTF-8 strings
func validateCodePoint(r rune) error {
	// U+0000 null character
	if r == 0x0000 {
		return ErrNullCharacter
	}

	// U+D800 to U+DFFF are UTF-16 surrogates (invalid in UTF-8)
	if r >= 0xD800 && r <= 0xDFFF {
		return ErrSurrogateCodePoint
	}

	// U+FFFE and U+FFFF are non-characters
	if r == 0xFFFE || r == 0xFFFF {
		return ErrNonCharacterCodePoint
	}

	// Additional non-characters: U+nFFFE and U+nFFFF for all planes (except U+10FFFF which is the max valid code point)
	if r != 0x10FFFF && ((r&0xFFFF) == 0xFFFE || (r&0xFFFF) == 0xFFFF) {
		return ErrNonCharacterCodePoint
	}

	// U+FDD0 to U+FDEF are also non-characters
	if r >= 0xFDD0 && r <= 0xFDEF {
		return ErrNonCharacterCodePoint
	}

	// Control characters U+0001 to U+001F and U+007F to U+009F
	// According to MQTT spec, these SHOULD NOT be used, but we'll be lenient
	// and only warn about them in strict mode (not implemented yet)
	// Uncomment the following to enforce strict validation:
	/*
		if (r >= 0x0001 && r <= 0x001F) || (r >= 0x007F && r <= 0x009F) {
			return ErrControlCharacter
		}
	*/

	return nil
}

// ValidateUTF8StringStrict performs strict validation including control character checks
func ValidateUTF8StringStrict(data []byte) error {
	// First do standard validation
	if err := ValidateUTF8String(data); err != nil {
		return err
	}

	// Check for control characters
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])

		// Control characters U+0001 to U+001F (excluding tab, newline, carriage return which might be allowed)
		// and U+007F to U+009F
		if (r >= 0x0001 && r <= 0x001F && r != 0x0009 && r != 0x000A && r != 0x000D) ||
			(r >= 0x007F && r <= 0x009F) {
			return ErrControlCharacter
		}

		i += size
	}

	return nil
}

// IsValidUTF8String is a convenience function that returns true if the data is valid
func IsValidUTF8String(data []byte) bool {
	return ValidateUTF8String(data) == nil
}

// IsValidUTF8StringStrict is a convenience function for strict validation
func IsValidUTF8StringStrict(data []byte) bool {
	return ValidateUTF8StringStrict(data) == nil
}
