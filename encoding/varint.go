package encoding

import (
	"errors"
	"io"
)

// Variable Byte Integer encoding/decoding for MQTT 5.0 and MQTT 3.1.1
//
// Per MQTT 5.0 specification section 1.5.5:
// - Variable Byte Integer encodes values from 0 to 268,435,455 (0xFF,0xFF,0xFF,0x7F)
// - Uses continuation bit (bit 7) to indicate if more bytes follow
// - Encodes 7 bits of data per byte
// - Maximum of 4 bytes
//
// This implementation is also compatible with MQTT 3.1.1 specification section 2.2.3
// for remaining length encoding.

const (
	// MaxVariableByteInteger is the maximum value that can be encoded (268,435,455)
	MaxVariableByteInteger uint32 = 268435455 // 0x0FFFFFFF

	// MaxVariableByteIntegerBytes is the maximum number of bytes in a variable byte integer
	MaxVariableByteIntegerBytes = 4
)

// EncodeVariableByteInteger encodes a uint32 as MQTT Variable Byte Integer.
// Returns the encoded bytes and any error.
//
// Per MQTT spec:
// - Values 0-127: 1 byte
// - Values 128-16,383: 2 bytes
// - Values 16,384-2,097,151: 3 bytes
// - Values 2,097,152-268,435,455: 4 bytes
// - Values > 268,435,455: error
func EncodeVariableByteInteger(value uint32) ([]byte, error) {
	// Maximum value is 268,435,455 (0x0FFFFFFF)
	if value > MaxVariableByteInteger {
		return nil, ErrVariableByteIntegerTooLarge
	}

	result := make([]byte, 0, 4)
	for {
		encodedByte := byte(value % 128)
		value = value / 128

		// If there are more data to encode, set the top bit (continuation bit)
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

// EncodeVariableByteIntegerTo encodes a uint32 as MQTT Variable Byte Integer
// and writes it to the provided byte slice starting at offset.
// Returns the number of bytes written and any error.
//
// The caller must ensure the buffer has sufficient space (up to 4 bytes).
func EncodeVariableByteIntegerTo(buf []byte, offset int, value uint32) (int, error) {
	if value > MaxVariableByteInteger {
		return 0, ErrVariableByteIntegerTooLarge
	}

	bytesWritten := 0
	for {
		encodedByte := byte(value % 128)
		value = value / 128

		// If there are more data to encode, set the top bit
		if value > 0 {
			encodedByte |= 0x80
		}

		if offset+bytesWritten >= len(buf) {
			return 0, ErrBufferTooSmall
		}

		buf[offset+bytesWritten] = encodedByte
		bytesWritten++

		if value == 0 {
			break
		}
	}

	return bytesWritten, nil
}

// DecodeVariableByteInteger decodes MQTT Variable Byte Integer from a reader.
// Returns the decoded value and any error.
//
// Per MQTT spec section 1.5.5:
// - Maximum of 4 bytes
// - Each byte encodes 7 bits of data
// - Bit 7 is the continuation bit (1 = more bytes follow, 0 = last byte)
func DecodeVariableByteInteger(r io.Reader) (uint32, error) {
	var value uint32
	var multiplier uint32 = 1
	var buf [1]byte // Stack-allocated for zero heap allocation

	for i := 0; i < MaxVariableByteIntegerBytes; i++ {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
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
		// This check prevents overflow on the next iteration
		if multiplier > 128*128*128 {
			return 0, ErrMalformedVariableByteInteger
		}

		multiplier *= 128
	}

	// If we read 4 bytes and still have continuation bit set, it's malformed
	return 0, ErrMalformedVariableByteInteger
}

// DecodeVariableByteIntegerFromBytes decodes MQTT Variable Byte Integer from a byte slice.
// Returns the decoded value, number of bytes consumed, and any error.
//
// This is a zero-allocation version when you already have the data in memory.
func DecodeVariableByteIntegerFromBytes(data []byte) (uint32, int, error) {
	var value uint32
	var multiplier uint32 = 1

	for i := 0; i < MaxVariableByteIntegerBytes && i < len(data); i++ {
		encodedByte := data[i]

		// Add lower 7 bits to value
		value += uint32(encodedByte&0x7F) * multiplier

		// Check if more bytes follow (bit 7 is continuation bit)
		if (encodedByte & 0x80) == 0 {
			return value, i + 1, nil
		}

		// Check for maximum value exceeded
		if multiplier > 128*128*128 {
			return 0, 0, ErrMalformedVariableByteInteger
		}

		multiplier *= 128
	}

	// Either ran out of data or read 4 bytes with continuation bit still set
	if len(data) < MaxVariableByteIntegerBytes {
		return 0, 0, ErrUnexpectedEOF
	}
	return 0, 0, ErrMalformedVariableByteInteger
}

// SizeVariableByteInteger returns the number of bytes required to encode the given value.
// Returns 0 if the value is too large to encode.
func SizeVariableByteInteger(value uint32) int {
	if value > MaxVariableByteInteger {
		return 0
	}

	if value <= 127 {
		return 1
	} else if value <= 16383 {
		return 2
	} else if value <= 2097151 {
		return 3
	}
	return 4
}
