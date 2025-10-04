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
)
