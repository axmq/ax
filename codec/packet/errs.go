package packet

import "errors"

var (
	ErrInvalidType           = errors.New("invalid packet type")
	ErrInvalidFlags          = errors.New("invalid flags for packet type")
	ErrMalformedRemainingLen = errors.New("malformed remaining length")
	ErrInvalidQoS            = errors.New("invalid QoS level")
	ErrInvalidReservedType   = errors.New("reserved packet type (0) not allowed")
	ErrUnexpectedEOF         = errors.New("unexpected end of input")
)
