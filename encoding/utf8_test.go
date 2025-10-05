package encoding

import (
	"testing"
)

func TestValidateUTF8String(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			name:    "valid ASCII string",
			input:   []byte("Hello, World!"),
			wantErr: nil,
		},
		{
			name:    "valid UTF-8 with emoji",
			input:   []byte("Hello 🌍 World"),
			wantErr: nil,
		},
		{
			name:    "valid UTF-8 with Chinese characters",
			input:   []byte("你好世界"),
			wantErr: nil,
		},
		{
			name:    "valid UTF-8 with Arabic",
			input:   []byte("مرحبا بالعالم"),
			wantErr: nil,
		},
		{
			name:    "valid UTF-8 with mixed scripts",
			input:   []byte("Hello мир 世界 🌍"),
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   []byte(""),
			wantErr: nil,
		},
		{
			name:    "null character at start",
			input:   []byte("\x00Hello"),
			wantErr: ErrNullCharacter,
		},
		{
			name:    "null character in middle",
			input:   []byte("Hello\x00World"),
			wantErr: ErrNullCharacter,
		},
		{
			name:    "null character at end",
			input:   []byte("Hello\x00"),
			wantErr: ErrNullCharacter,
		},
		{
			name:    "only null character",
			input:   []byte("\x00"),
			wantErr: ErrNullCharacter,
		},
		{
			name:    "invalid UTF-8 sequence",
			input:   []byte{0xFF, 0xFE, 0xFD},
			wantErr: ErrInvalidUTF8,
		},
		{
			name:    "invalid UTF-8 continuation byte",
			input:   []byte{0x80},
			wantErr: ErrInvalidUTF8,
		},
		{
			name:    "incomplete UTF-8 sequence",
			input:   []byte{0xE0, 0x80},
			wantErr: ErrInvalidUTF8,
		},
		{
			name:    "overlong encoding",
			input:   []byte{0xC0, 0x80}, // Overlong encoding of null
			wantErr: ErrInvalidUTF8,
		},
		{
			name:    "surrogate pair U+D800",
			input:   []byte{0xED, 0xA0, 0x80},
			wantErr: ErrInvalidUTF8, // Invalid UTF-8 encoding
		},
		{
			name:    "surrogate pair U+DFFF",
			input:   []byte{0xED, 0xBF, 0xBF},
			wantErr: ErrInvalidUTF8, // Invalid UTF-8 encoding
		},
		{
			name:    "non-character U+FFFE",
			input:   []byte{0xEF, 0xBF, 0xBE},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FFFF",
			input:   []byte{0xEF, 0xBF, 0xBF},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+1FFFE",
			input:   []byte{0xF0, 0x9F, 0xBF, 0xBE},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+1FFFF",
			input:   []byte{0xF0, 0x9F, 0xBF, 0xBF},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FDD0",
			input:   []byte{0xEF, 0xB7, 0x90},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FDEF",
			input:   []byte{0xEF, 0xB7, 0xAF},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "valid tab character",
			input:   []byte("Hello\tWorld"),
			wantErr: nil,
		},
		{
			name:    "valid newline character",
			input:   []byte("Hello\nWorld"),
			wantErr: nil,
		},
		{
			name:    "valid carriage return",
			input:   []byte("Hello\rWorld"),
			wantErr: nil,
		},
		{
			name:    "control character U+0001 (allowed in non-strict mode)",
			input:   []byte{0x01},
			wantErr: nil, // Allowed in non-strict mode
		},
		{
			name:    "control character U+001F (allowed in non-strict mode)",
			input:   []byte{0x1F},
			wantErr: nil, // Allowed in non-strict mode
		},
		{
			name:    "valid space character",
			input:   []byte(" "),
			wantErr: nil,
		},
		{
			name:    "maximum valid 2-byte UTF-8",
			input:   []byte{0xDF, 0xBF}, // U+07FF
			wantErr: nil,
		},
		{
			name:    "maximum valid 3-byte UTF-8",
			input:   []byte{0xEF, 0xBF, 0xBD}, // U+FFFD (replacement character)
			wantErr: nil,
		},
		{
			name:    "maximum valid 4-byte UTF-8",
			input:   []byte{0xF4, 0x8F, 0xBF, 0xBF}, // U+10FFFF
			wantErr: nil,
		},
		{
			name:    "beyond valid Unicode range",
			input:   []byte{0xF4, 0x90, 0x80, 0x80}, // Beyond U+10FFFF
			wantErr: ErrInvalidUTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUTF8String(tt.input)
			if err != tt.wantErr {
				t.Errorf("ValidateUTF8String() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUTF8StringStrict(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			name:    "valid ASCII string",
			input:   []byte("Hello, World!"),
			wantErr: nil,
		},
		{
			name:    "tab is allowed in strict mode",
			input:   []byte("Hello\tWorld"),
			wantErr: nil,
		},
		{
			name:    "newline is allowed in strict mode",
			input:   []byte("Hello\nWorld"),
			wantErr: nil,
		},
		{
			name:    "carriage return is allowed in strict mode",
			input:   []byte("Hello\rWorld"),
			wantErr: nil,
		},
		{
			name:    "control character U+0001",
			input:   []byte{0x01},
			wantErr: ErrControlCharacter,
		},
		{
			name:    "control character U+001F",
			input:   []byte{0x1F},
			wantErr: ErrControlCharacter,
		},
		{
			name:    "control character U+007F (DEL)",
			input:   []byte{0x7F},
			wantErr: ErrControlCharacter,
		},
		{
			name:    "control character U+0080",
			input:   []byte{0xC2, 0x80},
			wantErr: ErrControlCharacter,
		},
		{
			name:    "control character U+009F",
			input:   []byte{0xC2, 0x9F},
			wantErr: ErrControlCharacter,
		},
		{
			name:    "U+00A0 (non-breaking space) is allowed",
			input:   []byte{0xC2, 0xA0},
			wantErr: nil,
		},
		{
			name:    "null character",
			input:   []byte("\x00"),
			wantErr: ErrNullCharacter,
		},
		{
			name:    "non-character U+FFFE",
			input:   []byte{0xEF, 0xBF, 0xBE},
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "mixed valid and control character",
			input:   []byte("Hello\x01World"),
			wantErr: ErrControlCharacter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUTF8StringStrict(tt.input)
			if err != tt.wantErr {
				t.Errorf("ValidateUTF8StringStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidUTF8String(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{
			name:  "valid string",
			input: []byte("Hello, World!"),
			want:  true,
		},
		{
			name:  "invalid string with null",
			input: []byte("Hello\x00World"),
			want:  false,
		},
		{
			name:  "invalid UTF-8",
			input: []byte{0xFF, 0xFE},
			want:  false,
		},
		{
			name:  "non-character",
			input: []byte{0xEF, 0xBF, 0xBE},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUTF8String(tt.input); got != tt.want {
				t.Errorf("IsValidUTF8String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidUTF8StringStrict(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{
			name:  "valid string",
			input: []byte("Hello, World!"),
			want:  true,
		},
		{
			name:  "control character",
			input: []byte{0x01},
			want:  false,
		},
		{
			name:  "tab is valid",
			input: []byte("Hello\tWorld"),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUTF8StringStrict(tt.input); got != tt.want {
				t.Errorf("IsValidUTF8StringStrict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCodePoint(t *testing.T) {
	tests := []struct {
		name    string
		r       rune
		wantErr error
	}{
		{
			name:    "valid ASCII",
			r:       'A',
			wantErr: nil,
		},
		{
			name:    "valid emoji",
			r:       '🌍',
			wantErr: nil,
		},
		{
			name:    "null character",
			r:       0x0000,
			wantErr: ErrNullCharacter,
		},
		{
			name:    "surrogate start U+D800",
			r:       0xD800,
			wantErr: ErrSurrogateCodePoint,
		},
		{
			name:    "surrogate end U+DFFF",
			r:       0xDFFF,
			wantErr: ErrSurrogateCodePoint,
		},
		{
			name:    "surrogate middle U+DC00",
			r:       0xDC00,
			wantErr: ErrSurrogateCodePoint,
		},
		{
			name:    "non-character U+FFFE",
			r:       0xFFFE,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FFFF",
			r:       0xFFFF,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+1FFFE",
			r:       0x1FFFE,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+1FFFF",
			r:       0x1FFFF,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FDD0",
			r:       0xFDD0,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "non-character U+FDEF",
			r:       0xFDEF,
			wantErr: ErrNonCharacterCodePoint,
		},
		{
			name:    "valid character after non-character range",
			r:       0xFDF0,
			wantErr: nil,
		},
		{
			name:    "valid character before non-character range",
			r:       0xFDCF,
			wantErr: nil,
		},
		{
			name:    "tab character",
			r:       '\t',
			wantErr: nil,
		},
		{
			name:    "newline character",
			r:       '\n',
			wantErr: nil,
		},
		{
			name:    "space character",
			r:       ' ',
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCodePoint(tt.r)
			if err != tt.wantErr {
				t.Errorf("validateCodePoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateUTF8String(b *testing.B) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "ASCII short",
			data: []byte("Hello, World!"),
		},
		{
			name: "ASCII long",
			data: []byte("The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog."),
		},
		{
			name: "UTF-8 with emoji",
			data: []byte("Hello 🌍 World 🚀 Testing 💻 Validation ✅"),
		},
		{
			name: "UTF-8 mixed scripts",
			data: []byte("Hello мир 世界 مرحبا Γειά σου שלום สวัสดี"),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = ValidateUTF8String(tc.data)
			}
		})
	}
}

func BenchmarkValidateUTF8StringStrict(b *testing.B) {
	data := []byte("Hello, World! This is a test string for benchmarking.")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ValidateUTF8StringStrict(data)
	}
}

func BenchmarkIsValidUTF8String(b *testing.B) {
	data := []byte("Hello, World! This is a test string for benchmarking.")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IsValidUTF8String(data)
	}
}
