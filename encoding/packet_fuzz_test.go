package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzParseFixedHeader(f *testing.F) {
	seeds := [][]byte{
		{0x10, 0x00},
		{0x20, 0x02},
		{0x30, 0x00},
		{0x32, 0x05},
		{0x34, 0x07},
		{0x3D, 0x08},
		{0x40, 0x02},
		{0x50, 0x02},
		{0x62, 0x02},
		{0x70, 0x02},
		{0x82, 0x05},
		{0x90, 0x03},
		{0xA2, 0x04},
		{0xB0, 0x02},
		{0xC0, 0x00},
		{0xD0, 0x00},
		{0xE0, 0x00},
		{0xF0, 0x00},
		{0x10, 0x7F},
		{0x10, 0x80, 0x01},
		{0x10, 0xFF, 0x7F},
		{0x10, 0x80, 0x80, 0x01},
		{0x10, 0xFF, 0xFF, 0x7F},
		{0x10, 0x80, 0x80, 0x80, 0x01},
		{0x10, 0xFF, 0xFF, 0xFF, 0x7F},
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		header1, err1 := ParseFixedHeader(r)

		header2, _, err2 := ParseFixedHeaderFromBytes(data)

		assert.Equal(t, err1 == nil, err2 == nil, "ParseFixedHeader and ParseFixedHeaderFromBytes disagree on error")

		if err1 == nil && err2 == nil {
			assert.Equal(t, header1.Type, header2.Type, "PacketType mismatch")
			assert.Equal(t, header1.Flags, header2.Flags, "Flags mismatch")
			assert.Equal(t, header1.RemainingLength, header2.RemainingLength, "RemainingLength mismatch")
			if header1.Type == PUBLISH {
				assert.Equal(t, header1.DUP, header2.DUP, "DUP mismatch")
				assert.Equal(t, header1.QoS, header2.QoS, "QoS mismatch")
				assert.Equal(t, header1.Retain, header2.Retain, "Retain mismatch")
			}
		}

		if err1 == nil {
			assert.True(t, header1.Type != Reserved && header1.Type <= AUTH, "Parser accepted invalid type")

			if header1.Type == PUBLISH {
				assert.True(t, header1.QoS.IsValid(), "Parser accepted invalid QoS")
			}

			assert.LessOrEqual(t, header1.RemainingLength, uint32(268435455), "Parser accepted invalid remaining length")
		}
	})
}
