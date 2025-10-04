package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzEncodeDecodeVariableByteInteger(f *testing.F) {
	// Seed with interesting values
	seeds := []uint32{
		0,
		1,
		127,       // Max 1-byte
		128,       // Min 2-byte
		16383,     // Max 2-byte
		16384,     // Min 3-byte
		2097151,   // Max 3-byte
		2097152,   // Min 4-byte
		268435455, // Max valid value
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value uint32) {
		encoded, err := EncodeVariableByteInteger(value)

		if value > MaxVariableByteInteger {
			require.Error(t, err, "EncodeVariableByteInteger should reject values > 268,435,455")
			assert.ErrorIs(t, err, ErrVariableByteIntegerTooLarge)
			return
		}

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(encoded), 1)
		assert.LessOrEqual(t, len(encoded), MaxVariableByteIntegerBytes)

		// Test DecodeVariableByteIntegerFromBytes
		decoded, bytesRead, err := DecodeVariableByteIntegerFromBytes(encoded)
		require.NoError(t, err)
		assert.Equal(t, value, decoded, "round-trip failed for DecodeVariableByteIntegerFromBytes")
		assert.Equal(t, len(encoded), bytesRead)

		// Test DecodeVariableByteInteger
		r := bytes.NewReader(encoded)
		decoded2, err := DecodeVariableByteInteger(r)
		require.NoError(t, err)
		assert.Equal(t, value, decoded2, "round-trip failed for DecodeVariableByteInteger")

		// Test SizeVariableByteInteger
		size := SizeVariableByteInteger(value)
		assert.Equal(t, len(encoded), size, "SizeVariableByteInteger returned wrong size")
	})
}

func FuzzDecodeVariableByteInteger(f *testing.F) {
	// Seed with valid and edge-case encodings
	seeds := [][]byte{
		{0x00},                         // 0
		{0x7F},                         // 127
		{0x80, 0x01},                   // 128
		{0xFF, 0x7F},                   // 16383
		{0x80, 0x80, 0x01},             // 16384
		{0xFF, 0xFF, 0x7F},             // 2097151
		{0x80, 0x80, 0x80, 0x01},       // 2097152
		{0xFF, 0xFF, 0xFF, 0x7F},       // 268435455 (max)
		{0x80},                         // Incomplete
		{0x80, 0x80},                   // Incomplete
		{0x80, 0x80, 0x80},             // Incomplete
		{0x80, 0x80, 0x80, 0x80},       // Malformed (4 continuation bits)
		{0xFF, 0xFF, 0xFF, 0xFF},       // Malformed
		{0x80, 0x80, 0x80, 0x80, 0x01}, // Too many bytes
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test DecodeVariableByteInteger
		r := bytes.NewReader(data)
		value1, err1 := DecodeVariableByteInteger(r)

		// Test DecodeVariableByteIntegerFromBytes
		value2, bytesRead, err2 := DecodeVariableByteIntegerFromBytes(data)

		// Both decoders should agree on error/success
		assert.Equal(t, err1 == nil, err2 == nil,
			"DecodeVariableByteInteger and DecodeVariableByteIntegerFromBytes disagree on error")

		if err1 == nil && err2 == nil {
			// Both succeeded - they should produce the same value
			assert.Equal(t, value1, value2, "decoders produced different values")
			assert.LessOrEqual(t, value1, MaxVariableByteInteger,
				"decoder produced value exceeding maximum")
			assert.GreaterOrEqual(t, bytesRead, 1)
			assert.LessOrEqual(t, bytesRead, MaxVariableByteIntegerBytes)

			// Verify the value can be re-encoded
			encoded, err := EncodeVariableByteInteger(value1)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(encoded), MaxVariableByteIntegerBytes)
		}
	})
}

func FuzzEncodeVariableByteIntegerTo(f *testing.F) {
	seeds := []struct {
		value  uint32
		offset int
	}{
		{0, 0},
		{127, 0},
		{128, 1},
		{16383, 2},
		{16384, 3},
		{2097151, 0},
		{2097152, 1},
		{268435455, 0},
	}

	for _, seed := range seeds {
		f.Add(seed.value, seed.offset)
	}

	f.Fuzz(func(t *testing.T, value uint32, offset int) {
		// Limit offset to reasonable range
		if offset < 0 || offset > 100 {
			return
		}

		buf := make([]byte, 110)
		n, err := EncodeVariableByteIntegerTo(buf, offset, value)

		if value > MaxVariableByteInteger {
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrVariableByteIntegerTooLarge)
			return
		}

		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, 1)
		assert.LessOrEqual(t, n, MaxVariableByteIntegerBytes)

		// Compare with EncodeVariableByteInteger
		expected, err := EncodeVariableByteInteger(value)
		require.NoError(t, err)
		assert.Equal(t, expected, buf[offset:offset+n])

		// Verify decode
		decoded, bytesRead, err := DecodeVariableByteIntegerFromBytes(buf[offset:])
		require.NoError(t, err)
		assert.Equal(t, value, decoded)
		assert.Equal(t, n, bytesRead)
	})
}
