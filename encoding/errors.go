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

	// Additional malformed packet detection errors
	ErrInvalidConnectFlags      = errors.New("invalid CONNECT flags: reserved bit must be 0")
	ErrInvalidWillQoS           = errors.New("invalid Will QoS level")
	ErrWillFlagMismatch         = errors.New("Will flag inconsistent with Will QoS or Will Retain")
	ErrMissingPacketID          = errors.New("missing packet identifier for QoS > 0")
	ErrInvalidPacketIDZero      = errors.New("packet identifier cannot be 0 for QoS > 0")
	ErrInvalidRemainingLength   = errors.New("remaining length exceeds maximum or packet bounds")
	ErrInvalidTopicName         = errors.New("invalid topic name")
	ErrInvalidTopicFilter       = errors.New("invalid topic filter")
	ErrEmptyTopicFilter         = errors.New("empty topic filter not allowed")
	ErrInvalidSubscriptionOpts  = errors.New("invalid subscription options")
	ErrEmptySubscriptionList    = errors.New("SUBSCRIBE packet must contain at least one subscription")
	ErrEmptyUnsubscribeList     = errors.New("UNSUBSCRIBE packet must contain at least one topic filter")
	ErrInvalidPropertyLength    = errors.New("invalid property length")
	ErrPropertyTooLarge         = errors.New("property value exceeds maximum size")
	ErrInvalidReasonCode        = errors.New("invalid reason code for packet type")
	ErrPayloadTooLarge          = errors.New("payload exceeds maximum size")
	ErrInvalidPublishTopicName  = errors.New("PUBLISH topic name cannot contain wildcards")
	ErrUsernameWithoutFlag      = errors.New("username present but username flag not set")
	ErrPasswordWithoutFlag      = errors.New("password present but password flag not set")
	ErrPasswordWithoutUsername  = errors.New("password flag set without username flag")
	ErrWillPropsWithoutWillFlag = errors.New("will properties present but will flag not set")
)

// PacketError represents a packet parsing error with associated protocol reason code
type PacketError struct {
	Err        error      // The underlying error
	ReasonCode ReasonCode // MQTT 5.0 reason code (0x81 for malformed, 0x82 for protocol error, etc.)
	Message    string     // Additional context message
}

func (e *PacketError) Error() string {
	if e.Message != "" {
		return e.Err.Error() + ": " + e.Message
	}
	return e.Err.Error()
}

func (e *PacketError) Unwrap() error {
	return e.Err
}

// NewMalformedPacketError creates a PacketError for malformed packets
func NewMalformedPacketError(err error, message string) *PacketError {
	return &PacketError{
		Err:        err,
		ReasonCode: ReasonMalformedPacket,
		Message:    message,
	}
}

// NewProtocolError creates a PacketError for protocol violations
func NewProtocolError(err error, message string) *PacketError {
	return &PacketError{
		Err:        err,
		ReasonCode: ReasonProtocolError,
		Message:    message,
	}
}

// GetReasonCode extracts the reason code from an error if it's a PacketError
func GetReasonCode(err error) ReasonCode {
	var pktErr *PacketError
	if errors.As(err, &pktErr) {
		return pktErr.ReasonCode
	}

	// Map common errors to reason codes
	switch {
	case errors.Is(err, ErrMalformedPacket),
		errors.Is(err, ErrMalformedVariableByteInteger),
		errors.Is(err, ErrInvalidConnectFlags),
		errors.Is(err, ErrInvalidWillQoS),
		errors.Is(err, ErrInvalidQoS),
		errors.Is(err, ErrInvalidRemainingLength):
		return ReasonMalformedPacket
	case errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrInvalidFlags),
		errors.Is(err, ErrInvalidReservedType),
		errors.Is(err, ErrWillFlagMismatch),
		errors.Is(err, ErrInvalidPacketID),
		errors.Is(err, ErrInvalidPacketIDZero),
		errors.Is(err, ErrMissingPacketID),
		errors.Is(err, ErrEmptySubscriptionList),
		errors.Is(err, ErrEmptyUnsubscribeList):
		return ReasonProtocolError
	case errors.Is(err, ErrInvalidProtocolVersion):
		return ReasonUnsupportedProtocolVersion
	case errors.Is(err, ErrInvalidTopicFilter),
		errors.Is(err, ErrEmptyTopicFilter):
		return ReasonTopicFilterInvalid
	case errors.Is(err, ErrInvalidTopicName),
		errors.Is(err, ErrInvalidPublishTopicName):
		return ReasonTopicNameInvalid
	case errors.Is(err, ErrPayloadTooLarge):
		return ReasonPacketTooLarge
	default:
		return ReasonUnspecifiedError
	}
}
