package packet

import (
	"io"
)

// Type represents MQTT control packet types
type Type byte

const (
	Reserved    Type = 0
	CONNECT     Type = 1
	CONNACK     Type = 2
	PUBLISH     Type = 3
	PUBACK      Type = 4
	PUBREC      Type = 5
	PUBREL      Type = 6
	PUBCOMP     Type = 7
	SUBSCRIBE   Type = 8
	SUBACK      Type = 9
	UNSUBSCRIBE Type = 10
	UNSUBACK    Type = 11
	PINGREQ     Type = 12
	PINGRESP    Type = 13
	DISCONNECT  Type = 14
	AUTH        Type = 15
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
	Type            Type
	Flags           byte
	RemainingLength uint32

	// PUBLISH-specific flags (decoded from Flags field)
	DUP    bool
	QoS    QoS
	Retain bool
}

// ParseFixedHeader parses the MQTT 5.0 fixed header from a reader
// This function aims for zero allocations by reusing a single-byte buffer internally
func ParseFixedHeader(r io.Reader) (*FixedHeader, error) {
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
	header.Type = Type(firstByte[0] >> 4)

	// Validate packet type - Reserved (0) is invalid per MQTT spec
	if header.Type == Reserved {
		return nil, ErrInvalidReservedType
	}
	if header.Type > AUTH {
		return nil, ErrInvalidType
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
		if err := validateFlags(header.Type, header.Flags); err != nil {
			return nil, err
		}
	}

	// Parse remaining length (Variable Byte Integer)
	remainingLength, err := decodeVariableByteInteger(r)
	if err != nil {
		return nil, err
	}
	header.RemainingLength = remainingLength

	return header, nil
}

// ParseFixedHeaderFromBytes parses the MQTT 5.0 fixed header from a byte slice
// This is a zero-allocation version when you already have the data in memory
func ParseFixedHeaderFromBytes(data []byte) (*FixedHeader, int, error) {
	if len(data) < 2 {
		return nil, 0, ErrUnexpectedEOF
	}

	header := &FixedHeader{}
	offset := 0

	// Extract packet type (bits 7-4)
	header.Type = Type(data[offset] >> 4)

	// Validate packet type
	if header.Type == Reserved {
		return nil, 0, ErrInvalidReservedType
	}
	if header.Type > AUTH {
		return nil, 0, ErrInvalidType
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
		if err := validateFlags(header.Type, header.Flags); err != nil {
			return nil, 0, err
		}
	}

	// Parse remaining length (Variable Byte Integer)
	remainingLength, bytesRead, err := decodeVariableByteIntegerFromBytes(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	header.RemainingLength = remainingLength
	offset += bytesRead

	return header, offset, nil
}

// decodeVariableByteInteger decodes MQTT Variable Byte Integer from a reader
// Maximum of 4 bytes, each byte encodes 7 bits of data
func decodeVariableByteInteger(r io.Reader) (uint32, error) {
	var value uint32
	var multiplier uint32 = 1
	var buf [1]byte // Stack-allocated for zero heap allocation

	for i := 0; i < 4; i++ {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			if err == io.EOF {
				return 0, ErrUnexpectedEOF
			}
			return 0, err
		}

		encodedByte := buf[0]

		// Add lower 7 bits to value
		value += uint32(encodedByte&0x7F) * multiplier

		// Check if more bytes follow (bit 7 is continuation bit)
		if (encodedByte & 0x80) == 0 {
			return value, nil
		}

		// Check for maximum value exceeded (268,435,455)
		if multiplier > 128*128*128 {
			return 0, ErrMalformedRemainingLen
		}

		multiplier *= 128
	}

	// If we read 4 bytes and still have continuation bit, it's malformed
	return 0, ErrMalformedRemainingLen
}

// decodeVariableByteIntegerFromBytes decodes MQTT Variable Byte Integer from a byte slice
// Returns the decoded value, number of bytes consumed, and any error
func decodeVariableByteIntegerFromBytes(data []byte) (uint32, int, error) {
	var value uint32
	var multiplier uint32 = 1

	for i := 0; i < 4 && i < len(data); i++ {
		encodedByte := data[i]

		// Add lower 7 bits to value
		value += uint32(encodedByte&0x7F) * multiplier

		// Check if more bytes follow (bit 7 is continuation bit)
		if (encodedByte & 0x80) == 0 {
			return value, i + 1, nil
		}

		// Check for maximum value exceeded
		if multiplier > 128*128*128 {
			return 0, 0, ErrMalformedRemainingLen
		}

		multiplier *= 128
	}

	// Either ran out of data or read 4 bytes with continuation bit still set
	if len(data) < 4 {
		return 0, 0, ErrUnexpectedEOF
	}
	return 0, 0, ErrMalformedRemainingLen
}

// EncodeVariableByteInteger encodes a uint32 as MQTT Variable Byte Integer
// Returns the encoded bytes and any error
func EncodeVariableByteInteger(value uint32) ([]byte, error) {
	// Maximum value is 268,435,455 (0x0FFFFFFF)
	if value > 268435455 {
		return nil, ErrMalformedRemainingLen
	}

	result := make([]byte, 0, 4)
	for {
		encodedByte := byte(value % 128)
		value = value / 128

		// If there are more data to encode, set the top bit
		if value > 0 {
			encodedByte |= 0x80
		}

		result = append(result, encodedByte)

		if value == 0 {
			break
		}
	}

	return result, nil
}

// validateFlags checks if flags are valid for the given packet type
// Per MQTT 5.0 specification section 2.1.3
func validateFlags(tp Type, flags byte) error {
	expectedFlags := map[Type]byte{
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
		AUTH:        0x00,
	}

	if expected, ok := expectedFlags[tp]; ok {
		if flags != expected {
			return ErrInvalidFlags
		}
	}

	return nil
}

// String returns human-readable packet type name
func (t Type) String() string {
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
