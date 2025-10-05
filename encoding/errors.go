package encoding

import "errors"

var (
	// ErrVariableByteIntegerTooLarge indicates the value exceeds the maximum encodable value (268,435,455)
	ErrVariableByteIntegerTooLarge = errors.New("variable byte integer value exceeds maximum (268,435,455)")

	// ErrMalformedVariableByteInteger indicates invalid variable byte integer encoding
	ErrMalformedVariableByteInteger = errors.New("malformed variable byte integer")

	// ErrUnexpectedEOF indicates unexpected end of input while reading
	ErrUnexpectedEOF = errors.New("unexpected end of input")

	// ErrBufferTooSmall indicates the buffer is too small for the operation
	ErrBufferTooSmall = errors.New("buffer too small")

	ErrInvalidType         = errors.New("invalid packet type")
	ErrInvalidFlags        = errors.New("invalid flags for packet type")
	ErrInvalidQoS          = errors.New("invalid QoS level")
	ErrInvalidReservedType = errors.New("reserved packet type (0) not allowed")

	// Property-related errors
	ErrInvalidPropertyID   = errors.New("invalid property ID")
	ErrInvalidPropertyType = errors.New("invalid property type")
	ErrDuplicateProperty   = errors.New("duplicate property not allowed")

	// Packet-related errors
	ErrInvalidProtocolName    = errors.New("invalid protocol name")
	ErrInvalidProtocolVersion = errors.New("invalid protocol version")
	ErrInvalidPacketID        = errors.New("invalid packet identifier")
	ErrMalformedPacket        = errors.New("malformed packet")

	// UTF-8 validation errors
	ErrInvalidUTF8           = errors.New("invalid UTF-8 encoding")
	ErrNullCharacter         = errors.New("null character (U+0000) not allowed in UTF-8 string")
	ErrInvalidCodePoint      = errors.New("invalid Unicode code point")
	ErrSurrogateCodePoint    = errors.New("UTF-16 surrogate code points (U+D800 to U+DFFF) not allowed")
	ErrNonCharacterCodePoint = errors.New("non-character code points (U+FFFE, U+FFFF) not allowed")
	ErrControlCharacter      = errors.New("control characters (U+0001 to U+001F, U+007F to U+009F) should be avoided")
)
