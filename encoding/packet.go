package encoding

import (
	"io"
)

// ProtocolVersion represents the MQTT protocol version
type ProtocolVersion byte

const (
	// ProtocolVersion311 represents MQTT 3.1.1
	ProtocolVersion311 ProtocolVersion = 4
	// ProtocolVersion50 represents MQTT 5.0
	ProtocolVersion50 ProtocolVersion = 5
)

// PacketType represents MQTT control packet types
type PacketType byte

const (
	// Reserved control packet type reserved by the spec; receiving this is a protocol error.
	Reserved PacketType = 0
	// CONNECT client→server request to establish a connection; first packet a client sends.
	CONNECT PacketType = 1
	// CONNACK server→client acknowledgment of CONNECT; conveys session state and reason code.
	CONNACK PacketType = 2
	// PUBLISH application message transport; flags encode DUP, QoS, and RETAIN.
	PUBLISH PacketType = 3
	// PUBACK QoS 1 publish acknowledgment; completes the QoS 1 flow.
	PUBACK PacketType = 4
	// PUBREC QoS 2 publish received; step 1 acknowledging PUBLISH.
	PUBREC PacketType = 5
	// PUBREL QoS 2 publish release; step 2. Must be sent with flags 0010.
	PUBREL PacketType = 6
	// PUBCOMP QoS 2 publish complete; final step completing QoS 2 handshake.
	PUBCOMP PacketType = 7
	// SUBSCRIBE client→server request to add topic filters. Must be sent with flags 0010.
	SUBSCRIBE PacketType = 8
	// SUBACK server→client acknowledgment of SUBSCRIBE; returns per-subscription reason codes.
	SUBACK PacketType = 9
	// UNSUBSCRIBE client→server request to remove topic filters. Must be sent with flags 0010.
	UNSUBSCRIBE PacketType = 10
	// UNSUBACK server→client acknowledgment of UNSUBSCRIBE; may include reason codes (MQTT 5.0).
	UNSUBACK PacketType = 11
	// PINGREQ client→server ping to test the network and keep the connection alive.
	PINGREQ PacketType = 12
	// PINGRESP server→client response to PINGREQ.
	PINGRESP PacketType = 13
	// DISCONNECT indicates disconnection; may carry reason code and session expiry (MQTT 5.0).
	DISCONNECT PacketType = 14
	// AUTH authentication exchange for enhanced authentication mechanisms (MQTT 5.0 only).
	AUTH PacketType = 15
)

// QoS levels
type QoS byte

const (
	QoS0 QoS = 0 // At most once
	QoS1 QoS = 1 // At least once
	QoS2 QoS = 2 // Exactly once
)

// IsValid returns true if the QoS level is valid (0, 1, or 2)
func (q QoS) IsValid() bool {
	return q <= QoS2
}

// FixedHeader represents the MQTT fixed header
type FixedHeader struct {
	Type            PacketType
	Flags           byte
	RemainingLength uint32

	// PUBLISH-specific flags (decoded from Flags field)
	DUP    bool
	QoS    QoS
	Retain bool
}

// ParseFixedHeader parses the MQTT fixed header from a reader for MQTT 5.0
// This function aims for zero allocations by reusing a single-byte buffer internally
func ParseFixedHeader(r io.Reader) (*FixedHeader, error) {
	return ParseFixedHeaderWithVersion(r, ProtocolVersion50)
}

// ParseFixedHeader311 parses the MQTT 3.1.1 fixed header from a reader
func ParseFixedHeader311(r io.Reader) (*FixedHeader, error) {
	return ParseFixedHeaderWithVersion(r, ProtocolVersion311)
}

// ParseFixedHeaderWithVersion parses the MQTT fixed header from a reader with version-specific validation
// This function aims for zero allocations by reusing a single-byte buffer internally
func ParseFixedHeaderWithVersion(r io.Reader, version ProtocolVersion) (*FixedHeader, error) {
	header := &FixedHeader{}

	// Read first byte (packet type + flags)
	var firstByte [1]byte // Stack-allocated, zero heap allocation
	if _, err := io.ReadFull(r, firstByte[:]); err != nil {
		if err == io.EOF {
			return nil, ErrUnexpectedEOF
		}
		return nil, err
	}

	// Extract packet type (bits 7-4)
	header.Type = PacketType(firstByte[0] >> 4)

	// Validate packet type - Reserved (0) is invalid per MQTT spec
	if header.Type == Reserved {
		return nil, ErrInvalidReservedType
	}

	// Version-specific validation
	if version == ProtocolVersion311 {
		// MQTT 3.1.1: AUTH packet (15) doesn't exist
		if header.Type > DISCONNECT {
			return nil, ErrInvalidType
		}
	} else {
		// MQTT 5.0 and later
		if header.Type > AUTH {
			return nil, ErrInvalidType
		}
	}

	// Extract flags (bits 3-0)
	header.Flags = firstByte[0] & 0x0F

	// Decode PUBLISH-specific flags
	if header.Type == PUBLISH {
		header.DUP = (header.Flags & 0x08) != 0
		header.QoS = QoS((header.Flags & 0x06) >> 1)
		header.Retain = (header.Flags & 0x01) != 0

		// Validate QoS level (must be 0, 1, or 2)
		if !header.QoS.IsValid() {
			return nil, ErrInvalidQoS
		}
	} else {
		// Validate reserved flags for non-PUBLISH packets
		if err := validateFlagsWithVersion(header.Type, header.Flags, version); err != nil {
			return nil, err
		}
	}

	// Parse remaining length (Variable Byte Integer)
	remainingLength, err := DecodeVariableByteInteger(r)
	if err != nil {
		return nil, err
	}
	header.RemainingLength = remainingLength

	return header, nil
}

// ParseFixedHeaderFromBytes parses the MQTT 5.0 fixed header from a byte slice
// This is a zero-allocation version when you already have the data in memory
func ParseFixedHeaderFromBytes(data []byte) (*FixedHeader, int, error) {
	return ParseFixedHeaderFromBytesWithVersion(data, ProtocolVersion50)
}

// ParseFixedHeaderFromBytes311 parses the MQTT 3.1.1 fixed header from a byte slice
func ParseFixedHeaderFromBytes311(data []byte) (*FixedHeader, int, error) {
	return ParseFixedHeaderFromBytesWithVersion(data, ProtocolVersion311)
}

// ParseFixedHeaderFromBytesWithVersion parses the MQTT fixed header from a byte slice with version-specific validation
// This is a zero-allocation version when you already have the data in memory
func ParseFixedHeaderFromBytesWithVersion(data []byte, version ProtocolVersion) (*FixedHeader, int, error) {
	if len(data) < 2 {
		return nil, 0, ErrUnexpectedEOF
	}

	header := &FixedHeader{}
	offset := 0

	// Extract packet type (bits 7-4)
	header.Type = PacketType(data[offset] >> 4)

	// Validate packet type
	if header.Type == Reserved {
		return nil, 0, ErrInvalidReservedType
	}

	// Version-specific validation
	if version == ProtocolVersion311 {
		// MQTT 3.1.1: AUTH packet (15) doesn't exist
		if header.Type > DISCONNECT {
			return nil, 0, ErrInvalidType
		}
	} else {
		// MQTT 5.0 and later
		if header.Type > AUTH {
			return nil, 0, ErrInvalidType
		}
	}

	// Extract flags (bits 3-0)
	header.Flags = data[offset] & 0x0F
	offset++

	// Decode PUBLISH-specific flags
	if header.Type == PUBLISH {
		header.DUP = (header.Flags & 0x08) != 0
		header.QoS = QoS((header.Flags & 0x06) >> 1)
		header.Retain = (header.Flags & 0x01) != 0

		// Validate QoS level
		if !header.QoS.IsValid() {
			return nil, 0, ErrInvalidQoS
		}
	} else {
		// Validate reserved flags for non-PUBLISH packets
		if err := validateFlagsWithVersion(header.Type, header.Flags, version); err != nil {
			return nil, 0, err
		}
	}

	// Parse remaining length (Variable Byte Integer)
	remainingLength, bytesRead, err := DecodeVariableByteIntegerFromBytes(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	header.RemainingLength = remainingLength
	offset += bytesRead

	return header, offset, nil
}

// EncodeFixedHeader encodes the MQTT fixed header to a writer
func (h *FixedHeader) EncodeFixedHeader(w io.Writer) error {
	return h.EncodeFixedHeaderWithVersion(w, ProtocolVersion50)
}

// EncodeFixedHeader311 encodes the MQTT 3.1.1 fixed header to a writer
func (h *FixedHeader) EncodeFixedHeader311(w io.Writer) error {
	return h.EncodeFixedHeaderWithVersion(w, ProtocolVersion311)
}

// EncodeFixedHeaderWithVersion encodes the MQTT fixed header to a writer with version-specific validation
func (h *FixedHeader) EncodeFixedHeaderWithVersion(w io.Writer, version ProtocolVersion) error {
	// Validate packet type
	if h.Type == Reserved {
		return ErrInvalidReservedType
	}

	// Version-specific validation
	if version == ProtocolVersion311 {
		if h.Type > DISCONNECT {
			return ErrInvalidType
		}
	} else {
		if h.Type > AUTH {
			return ErrInvalidType
		}
	}

	// Validate flags for non-PUBLISH packets
	if h.Type != PUBLISH {
		if err := validateFlagsWithVersion(h.Type, h.Flags, version); err != nil {
			return err
		}
	} else {
		// Validate QoS for PUBLISH packets
		if !h.QoS.IsValid() {
			return ErrInvalidQoS
		}
	}

	// Construct first byte: packet type (bits 7-4) | flags (bits 3-0)
	firstByte := byte(h.Type<<4) | h.Flags

	// Write first byte
	if _, err := w.Write([]byte{firstByte}); err != nil {
		return err
	}

	// Write remaining length as Variable Byte Integer
	remainingLengthBytes, err := EncodeVariableByteInteger(h.RemainingLength)
	if err != nil {
		return err
	}

	_, err = w.Write(remainingLengthBytes)
	return err
}

// EncodeFixedHeaderToBytes encodes the MQTT 5.0 fixed header to a byte slice
func (h *FixedHeader) EncodeFixedHeaderToBytes(buf []byte) (int, error) {
	return h.EncodeFixedHeaderToBytesWithVersion(buf, ProtocolVersion50)
}

// EncodeFixedHeaderToBytes311 encodes the MQTT 3.1.1 fixed header to a byte slice
func (h *FixedHeader) EncodeFixedHeaderToBytes311(buf []byte) (int, error) {
	return h.EncodeFixedHeaderToBytesWithVersion(buf, ProtocolVersion311)
}

// EncodeFixedHeaderToBytesWithVersion encodes the MQTT fixed header to a byte slice with version-specific validation
func (h *FixedHeader) EncodeFixedHeaderToBytesWithVersion(buf []byte, version ProtocolVersion) (int, error) {
	if len(buf) < 2 {
		return 0, ErrBufferTooSmall
	}

	// Validate packet type
	if h.Type == Reserved {
		return 0, ErrInvalidReservedType
	}

	// Version-specific validation
	if version == ProtocolVersion311 {
		if h.Type > DISCONNECT {
			return 0, ErrInvalidType
		}
	} else {
		if h.Type > AUTH {
			return 0, ErrInvalidType
		}
	}

	// Validate flags for non-PUBLISH packets
	if h.Type != PUBLISH {
		if err := validateFlagsWithVersion(h.Type, h.Flags, version); err != nil {
			return 0, err
		}
	} else {
		// Validate QoS for PUBLISH packets
		if !h.QoS.IsValid() {
			return 0, ErrInvalidQoS
		}
	}

	offset := 0

	// Construct first byte: packet type (bits 7-4) | flags (bits 3-0)
	buf[offset] = byte(h.Type<<4) | h.Flags
	offset++

	// Write remaining length as Variable Byte Integer
	bytesWritten, err := EncodeVariableByteIntegerTo(buf, offset, h.RemainingLength)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	return offset, nil
}

// validateFlags checks if flags are valid for the given packet type
// Per MQTT 5.0 specification section 2.1.3
func validateFlags(tp PacketType, flags byte) error {
	return validateFlagsWithVersion(tp, flags, ProtocolVersion50)
}

// validateFlagsWithVersion checks if flags are valid for the given packet type and protocol version
// Per MQTT 3.1.1 specification section 2.2 and MQTT 5.0 specification section 2.1.3
func validateFlagsWithVersion(tp PacketType, flags byte, version ProtocolVersion) error {
	// Expected flags are the same for both MQTT 3.1.1 and 5.0
	expectedFlags := map[PacketType]byte{
		CONNECT:     0x00,
		CONNACK:     0x00,
		PUBACK:      0x00,
		PUBREC:      0x00,
		PUBREL:      0x02, // Reserved bits must be 0010
		PUBCOMP:     0x00,
		SUBSCRIBE:   0x02, // Reserved bits must be 0010
		SUBACK:      0x00,
		UNSUBSCRIBE: 0x02, // Reserved bits must be 0010
		UNSUBACK:    0x00,
		PINGREQ:     0x00,
		PINGRESP:    0x00,
		DISCONNECT:  0x00,
	}

	// AUTH is only valid in MQTT 5.0
	if version == ProtocolVersion50 {
		expectedFlags[AUTH] = 0x00
	}

	if expected, ok := expectedFlags[tp]; ok {
		if flags != expected {
			return ErrInvalidFlags
		}
	}

	return nil
}

// String returns human-readable packet type name
func (t PacketType) String() string {
	names := [16]string{
		Reserved:    "RESERVED",
		CONNECT:     "CONNECT",
		CONNACK:     "CONNACK",
		PUBLISH:     "PUBLISH",
		PUBACK:      "PUBACK",
		PUBREC:      "PUBREC",
		PUBREL:      "PUBREL",
		PUBCOMP:     "PUBCOMP",
		SUBSCRIBE:   "SUBSCRIBE",
		SUBACK:      "SUBACK",
		UNSUBSCRIBE: "UNSUBSCRIBE",
		UNSUBACK:    "UNSUBACK",
		PINGREQ:     "PINGREQ",
		PINGRESP:    "PINGRESP",
		DISCONNECT:  "DISCONNECT",
		AUTH:        "AUTH",
	}

	if t <= AUTH {
		return names[t]
	}
	return "UNKNOWN"
}

// String returns human-readable QoS level
func (q QoS) String() string {
	switch q {
	case QoS0:
		return "QoS0"
	case QoS1:
		return "QoS1"
	case QoS2:
		return "QoS2"
	default:
		return "INVALID"
	}
}
