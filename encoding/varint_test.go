package encoding

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeVariableByteInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected []byte
		wantErr  error
	}{
		// Valid single-byte encodings (0-127)
		{
			name:     "zero",
			input:    0,
			expected: []byte{0x00},
		},
		{
			name:     "one",
			input:    1,
			expected: []byte{0x01},
		},
		{
			name:     "max_single_byte",
			input:    127,
			expected: []byte{0x7F},
		},
		// Valid two-byte encodings (128-16,383)
		{
			name:     "min_two_byte",
			input:    128,
			expected: []byte{0x80, 0x01},
		},
		{
			name:     "mid_two_byte",
			input:    8192,
			expected: []byte{0x80, 0x40},
		},
		{
			name:     "max_two_byte",
			input:    16383,
			expected: []byte{0xFF, 0x7F},
		},
		// Valid three-byte encodings (16,384-2,097,151)
		{
			name:     "min_three_byte",
			input:    16384,
			expected: []byte{0x80, 0x80, 0x01},
		},
		{
			name:     "mid_three_byte",
			input:    1048576,
			expected: []byte{0x80, 0x80, 0x40},
		},
		{
			name:     "max_three_byte",
			input:    2097151,
			expected: []byte{0xFF, 0xFF, 0x7F},
		},
		// Valid four-byte encodings (2,097,152-268,435,455)
		{
			name:     "min_four_byte",
			input:    2097152,
			expected: []byte{0x80, 0x80, 0x80, 0x01},
		},
		{
			name:     "mid_four_byte",
			input:    134217728,
			expected: []byte{0x80, 0x80, 0x80, 0x40},
		},
		{
			name:     "max_four_byte_max_value",
			input:    268435455,
			expected: []byte{0xFF, 0xFF, 0xFF, 0x7F},
		},
		// Invalid: too large
		{
			name:    "exceeds_maximum",
			input:   268435456,
			wantErr: ErrVariableByteIntegerTooLarge,
		},
		{
			name:    "far_exceeds_maximum",
			input:   0xFFFFFFFF,
			wantErr: ErrVariableByteIntegerTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeVariableByteInteger(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Verify round-trip
			decoded, bytesRead, err := DecodeVariableByteIntegerFromBytes(result)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded, "round-trip decode failed")
			assert.Equal(t, len(result), bytesRead)
		})
	}
}

func TestEncodeVariableByteIntegerTo(t *testing.T) {
	tests := []struct {
		name          string
		bufSize       int
		offset        int
		input         uint32
		expectedBytes int
		wantErr       error
	}{
		{
			name:          "single_byte_to_buffer",
			bufSize:       10,
			offset:        0,
			input:         127,
			expectedBytes: 1,
		},
		{
			name:          "two_byte_to_buffer",
			bufSize:       10,
			offset:        5,
			input:         16383,
			expectedBytes: 2,
		},
		{
			name:          "four_byte_to_buffer",
			bufSize:       10,
			offset:        3,
			input:         268435455,
			expectedBytes: 4,
		},
		{
			name:    "buffer_too_small",
			bufSize: 2,
			offset:  0,
			input:   268435455,
			wantErr: ErrBufferTooSmall,
		},
		{
			name:    "offset_too_large",
			bufSize: 5,
			offset:  5,
			input:   1,
			wantErr: ErrBufferTooSmall,
		},
		{
			name:    "value_too_large",
			bufSize: 10,
			offset:  0,
			input:   268435456,
			wantErr: ErrVariableByteIntegerTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n, err := EncodeVariableByteIntegerTo(buf, tt.offset, tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedBytes, n)

			// Verify the encoded bytes match EncodeVariableByteInteger
			expected, err := EncodeVariableByteInteger(tt.input)
			require.NoError(t, err)
			assert.Equal(t, expected, buf[tt.offset:tt.offset+n])
		})
	}
}

func TestDecodeVariableByteInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint32
		wantErr  error
	}{
		// Valid encodings
		{
			name:     "zero",
			input:    []byte{0x00},
			expected: 0,
		},
		{
			name:     "one_byte_127",
			input:    []byte{0x7F},
			expected: 127,
		},
		{
			name:     "two_bytes_128",
			input:    []byte{0x80, 0x01},
			expected: 128,
		},
		{
			name:     "two_bytes_16383",
			input:    []byte{0xFF, 0x7F},
			expected: 16383,
		},
		{
			name:     "three_bytes_16384",
			input:    []byte{0x80, 0x80, 0x01},
			expected: 16384,
		},
		{
			name:     "three_bytes_2097151",
			input:    []byte{0xFF, 0xFF, 0x7F},
			expected: 2097151,
		},
		{
			name:     "four_bytes_2097152",
			input:    []byte{0x80, 0x80, 0x80, 0x01},
			expected: 2097152,
		},
		{
			name:     "four_bytes_max",
			input:    []byte{0xFF, 0xFF, 0xFF, 0x7F},
			expected: 268435455,
		},
		// Invalid encodings
		{
			name:    "empty_input",
			input:   []byte{},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "incomplete_two_bytes",
			input:   []byte{0x80},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "incomplete_three_bytes",
			input:   []byte{0x80, 0x80},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "incomplete_four_bytes",
			input:   []byte{0x80, 0x80, 0x80},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "five_bytes_malformed",
			input:   []byte{0x80, 0x80, 0x80, 0x80, 0x01},
			wantErr: ErrMalformedVariableByteInteger,
		},
		{
			name:    "four_bytes_all_continuation_bits",
			input:   []byte{0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: ErrMalformedVariableByteInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			result, err := DecodeVariableByteInteger(r)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeVariableByteIntegerFromBytes(t *testing.T) {
	tests := []struct {
		name              string
		input             []byte
		expectedValue     uint32
		expectedBytesRead int
		wantErr           error
	}{
		// Valid encodings
		{
			name:              "zero",
			input:             []byte{0x00},
			expectedValue:     0,
			expectedBytesRead: 1,
		},
		{
			name:              "one_byte_with_extra_data",
			input:             []byte{0x7F, 0xFF, 0xFF},
			expectedValue:     127,
			expectedBytesRead: 1,
		},
		{
			name:              "two_bytes",
			input:             []byte{0x80, 0x01},
			expectedValue:     128,
			expectedBytesRead: 2,
		},
		{
			name:              "three_bytes",
			input:             []byte{0x80, 0x80, 0x01},
			expectedValue:     16384,
			expectedBytesRead: 3,
		},
		{
			name:              "four_bytes",
			input:             []byte{0xFF, 0xFF, 0xFF, 0x7F},
			expectedValue:     268435455,
			expectedBytesRead: 4,
		},
		{
			name:              "four_bytes_with_trailing_data",
			input:             []byte{0xFF, 0xFF, 0xFF, 0x7F, 0x99, 0x88},
			expectedValue:     268435455,
			expectedBytesRead: 4,
		},
		// Invalid encodings
		{
			name:    "empty",
			input:   []byte{},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "incomplete",
			input:   []byte{0x80},
			wantErr: ErrUnexpectedEOF,
		},
		{
			name:    "malformed_all_continuation",
			input:   []byte{0x80, 0x80, 0x80, 0x80},
			wantErr: ErrMalformedVariableByteInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, bytesRead, err := DecodeVariableByteIntegerFromBytes(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedBytesRead, bytesRead)
		})
	}
}

func TestSizeVariableByteInteger(t *testing.T) {
	tests := []struct {
		name         string
		value        uint32
		expectedSize int
	}{
		{"zero", 0, 1},
		{"one", 1, 1},
		{"max_1_byte", 127, 1},
		{"min_2_bytes", 128, 2},
		{"mid_2_bytes", 8192, 2},
		{"max_2_bytes", 16383, 2},
		{"min_3_bytes", 16384, 3},
		{"mid_3_bytes", 1048576, 3},
		{"max_3_bytes", 2097151, 3},
		{"min_4_bytes", 2097152, 4},
		{"mid_4_bytes", 134217728, 4},
		{"max_4_bytes", 268435455, 4},
		{"too_large", 268435456, 0},
		{"way_too_large", 0xFFFFFFFF, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := SizeVariableByteInteger(tt.value)
			assert.Equal(t, tt.expectedSize, size)

			// Verify consistency with actual encoding (if valid)
			if tt.expectedSize > 0 {
				encoded, err := EncodeVariableByteInteger(tt.value)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSize, len(encoded))
			}
		})
	}
}

func TestVariableByteInteger_IoReaderConsistency(t *testing.T) {
	// Test that DecodeVariableByteInteger and DecodeVariableByteIntegerFromBytes
	// produce the same results
	testValues := []uint32{
		0, 1, 127, 128, 16383, 16384, 2097151, 2097152, 268435455,
	}

	for _, value := range testValues {
		t.Run("", func(t *testing.T) {
			encoded, err := EncodeVariableByteInteger(value)
			require.NoError(t, err)

			// Test io.Reader version
			r := bytes.NewReader(encoded)
			decoded1, err1 := DecodeVariableByteInteger(r)
			require.NoError(t, err1)

			// Test byte slice version
			decoded2, bytesRead, err2 := DecodeVariableByteIntegerFromBytes(encoded)
			require.NoError(t, err2)

			// Both should produce same result
			assert.Equal(t, decoded1, decoded2)
			assert.Equal(t, value, decoded1)
			assert.Equal(t, len(encoded), bytesRead)
		})
	}
}

func TestVariableByteInteger_EdgeCases(t *testing.T) {
	t.Run("boundary_127_128", func(t *testing.T) {
		// 127 should be 1 byte
		enc127, _ := EncodeVariableByteInteger(127)
		assert.Len(t, enc127, 1)
		assert.Equal(t, byte(0x7F), enc127[0])

		// 128 should be 2 bytes
		enc128, _ := EncodeVariableByteInteger(128)
		assert.Len(t, enc128, 2)
		assert.Equal(t, []byte{0x80, 0x01}, enc128)
	})

	t.Run("boundary_16383_16384", func(t *testing.T) {
		// 16383 should be 2 bytes
		enc16383, _ := EncodeVariableByteInteger(16383)
		assert.Len(t, enc16383, 2)

		// 16384 should be 3 bytes
		enc16384, _ := EncodeVariableByteInteger(16384)
		assert.Len(t, enc16384, 3)
	})

	t.Run("boundary_2097151_2097152", func(t *testing.T) {
		// 2097151 should be 3 bytes
		enc2097151, _ := EncodeVariableByteInteger(2097151)
		assert.Len(t, enc2097151, 3)

		// 2097152 should be 4 bytes
		enc2097152, _ := EncodeVariableByteInteger(2097152)
		assert.Len(t, enc2097152, 4)
	})

	t.Run("max_valid_value", func(t *testing.T) {
		enc, err := EncodeVariableByteInteger(268435455)
		require.NoError(t, err)
		assert.Equal(t, []byte{0xFF, 0xFF, 0xFF, 0x7F}, enc)

		dec, _, err := DecodeVariableByteIntegerFromBytes(enc)
		require.NoError(t, err)
		assert.Equal(t, uint32(268435455), dec)
	})

	t.Run("first_invalid_value", func(t *testing.T) {
		_, err := EncodeVariableByteInteger(268435456)
		assert.ErrorIs(t, err, ErrVariableByteIntegerTooLarge)
	})
}

func TestVariableByteInteger_IOReaderError(t *testing.T) {
	t.Run("io_error_propagation", func(t *testing.T) {
		// Create a reader that returns an error
		errReader := &errorReader{err: io.ErrUnexpectedEOF}
		_, err := DecodeVariableByteInteger(errReader)
		assert.Error(t, err)
	})

	t.Run("eof_error_conversion", func(t *testing.T) {
		// EOF should be converted to ErrUnexpectedEOF
		eofReader := &errorReader{err: io.EOF}
		_, err := DecodeVariableByteInteger(eofReader)
		assert.ErrorIs(t, err, ErrUnexpectedEOF)
	})
}

// errorReader is a test helper that always returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
