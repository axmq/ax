package encoding

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacketError(t *testing.T) {
	t.Run("Error method with message", func(t *testing.T) {
		pktErr := &PacketError{
			Err:        ErrMalformedPacket,
			ReasonCode: ReasonMalformedPacket,
			Message:    "invalid variable byte integer",
		}
		expected := "malformed packet: invalid variable byte integer"
		assert.Equal(t, expected, pktErr.Error())
	})

	t.Run("Error method without message", func(t *testing.T) {
		pktErr := &PacketError{
			Err:        ErrMalformedPacket,
			ReasonCode: ReasonMalformedPacket,
		}
		assert.Equal(t, "malformed packet", pktErr.Error())
	})

	t.Run("Unwrap method", func(t *testing.T) {
		pktErr := &PacketError{
			Err:        ErrMalformedPacket,
			ReasonCode: ReasonMalformedPacket,
			Message:    "test",
		}
		assert.Equal(t, ErrMalformedPacket, pktErr.Unwrap())
	})
}

func TestNewMalformedPacketError(t *testing.T) {
	err := NewMalformedPacketError(ErrInvalidQoS, "QoS value is 3")

	require.NotNil(t, err)
	assert.Equal(t, ReasonMalformedPacket, err.ReasonCode)
	assert.Equal(t, ErrInvalidQoS, err.Err)
	assert.Equal(t, "QoS value is 3", err.Message)
	assert.Contains(t, err.Error(), "invalid QoS level")
	assert.Contains(t, err.Error(), "QoS value is 3")
}

func TestNewProtocolError(t *testing.T) {
	err := NewProtocolError(ErrInvalidFlags, "PUBREL flags must be 0x02")

	require.NotNil(t, err)
	assert.Equal(t, ReasonProtocolError, err.ReasonCode)
	assert.Equal(t, ErrInvalidFlags, err.Err)
	assert.Equal(t, "PUBREL flags must be 0x02", err.Message)
}

func TestGetReasonCode(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedReasonCode ReasonCode
	}{
		{
			name:               "PacketError with malformed packet",
			err:                NewMalformedPacketError(ErrInvalidQoS, "test"),
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "PacketError with protocol error",
			err:                NewProtocolError(ErrInvalidFlags, "test"),
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrMalformedPacket",
			err:                ErrMalformedPacket,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrMalformedVariableByteInteger",
			err:                ErrMalformedVariableByteInteger,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrInvalidConnectFlags",
			err:                ErrInvalidConnectFlags,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrInvalidWillQoS",
			err:                ErrInvalidWillQoS,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrInvalidQoS",
			err:                ErrInvalidQoS,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrInvalidRemainingLength",
			err:                ErrInvalidRemainingLength,
			expectedReasonCode: ReasonMalformedPacket,
		},
		{
			name:               "ErrInvalidType",
			err:                ErrInvalidType,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrInvalidFlags",
			err:                ErrInvalidFlags,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrInvalidReservedType",
			err:                ErrInvalidReservedType,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrWillFlagMismatch",
			err:                ErrWillFlagMismatch,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrInvalidPacketID",
			err:                ErrInvalidPacketID,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrInvalidPacketIDZero",
			err:                ErrInvalidPacketIDZero,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrMissingPacketID",
			err:                ErrMissingPacketID,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrEmptySubscriptionList",
			err:                ErrEmptySubscriptionList,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrEmptyUnsubscribeList",
			err:                ErrEmptyUnsubscribeList,
			expectedReasonCode: ReasonProtocolError,
		},
		{
			name:               "ErrInvalidProtocolVersion",
			err:                ErrInvalidProtocolVersion,
			expectedReasonCode: ReasonUnsupportedProtocolVersion,
		},
		{
			name:               "ErrInvalidTopicFilter",
			err:                ErrInvalidTopicFilter,
			expectedReasonCode: ReasonTopicFilterInvalid,
		},
		{
			name:               "ErrEmptyTopicFilter",
			err:                ErrEmptyTopicFilter,
			expectedReasonCode: ReasonTopicFilterInvalid,
		},
		{
			name:               "ErrInvalidTopicName",
			err:                ErrInvalidTopicName,
			expectedReasonCode: ReasonTopicNameInvalid,
		},
		{
			name:               "ErrInvalidPublishTopicName",
			err:                ErrInvalidPublishTopicName,
			expectedReasonCode: ReasonTopicNameInvalid,
		},
		{
			name:               "ErrPayloadTooLarge",
			err:                ErrPayloadTooLarge,
			expectedReasonCode: ReasonPacketTooLarge,
		},
		{
			name:               "Unknown error",
			err:                errors.New("unknown error"),
			expectedReasonCode: ReasonUnspecifiedError,
		},
		{
			name:               "ErrUnexpectedEOF (unspecified)",
			err:                ErrUnexpectedEOF,
			expectedReasonCode: ReasonUnspecifiedError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reasonCode := GetReasonCode(tt.err)
			assert.Equal(t, tt.expectedReasonCode, reasonCode)
		})
	}
}

func TestGetReasonCode_WrappedErrors(t *testing.T) {
	t.Run("Wrapped PacketError", func(t *testing.T) {
		pktErr := NewMalformedPacketError(ErrInvalidQoS, "test")

		// Should still extract reason code from wrapped PacketError using errors.As
		var targetErr *PacketError
		if errors.As(pktErr, &targetErr) {
			assert.Equal(t, ReasonMalformedPacket, targetErr.ReasonCode)
		}
	})

	t.Run("Wrapped standard error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped: " + ErrInvalidQoS.Error())
		reasonCode := GetReasonCode(wrappedErr)
		// Should return unspecified since it can't unwrap the error
		assert.Equal(t, ReasonUnspecifiedError, reasonCode)
	})
}

func TestErrorPropagation(t *testing.T) {
	t.Run("Error chain with Is", func(t *testing.T) {
		pktErr := NewMalformedPacketError(ErrInvalidQoS, "test")
		assert.True(t, errors.Is(pktErr, ErrInvalidQoS))
	})

	t.Run("Error chain with As", func(t *testing.T) {
		pktErr := NewProtocolError(ErrInvalidFlags, "test")
		var target *PacketError
		assert.True(t, errors.As(pktErr, &target))
		assert.Equal(t, ReasonProtocolError, target.ReasonCode)
	})
}

func TestMalformedPacketErrors(t *testing.T) {
	// Test that all new error types are properly defined
	assert.NotNil(t, ErrInvalidConnectFlags)
	assert.NotNil(t, ErrInvalidWillQoS)
	assert.NotNil(t, ErrWillFlagMismatch)
	assert.NotNil(t, ErrMissingPacketID)
	assert.NotNil(t, ErrInvalidPacketIDZero)
	assert.NotNil(t, ErrInvalidRemainingLength)
	assert.NotNil(t, ErrInvalidTopicName)
	assert.NotNil(t, ErrInvalidTopicFilter)
	assert.NotNil(t, ErrEmptyTopicFilter)
	assert.NotNil(t, ErrInvalidSubscriptionOpts)
	assert.NotNil(t, ErrEmptySubscriptionList)
	assert.NotNil(t, ErrEmptyUnsubscribeList)
	assert.NotNil(t, ErrInvalidPropertyLength)
	assert.NotNil(t, ErrPropertyTooLarge)
	assert.NotNil(t, ErrInvalidReasonCode)
	assert.NotNil(t, ErrPayloadTooLarge)
	assert.NotNil(t, ErrInvalidPublishTopicName)
	assert.NotNil(t, ErrUsernameWithoutFlag)
	assert.NotNil(t, ErrPasswordWithoutFlag)
	assert.NotNil(t, ErrPasswordWithoutUsername)
	assert.NotNil(t, ErrWillPropsWithoutWillFlag)
}

func TestReasonCodeMapping(t *testing.T) {
	// Test that reason codes are correctly mapped
	tests := []struct {
		reasonCode ReasonCode
		value      byte
	}{
		{ReasonSuccess, 0x00},
		{ReasonMalformedPacket, 0x81},
		{ReasonProtocolError, 0x82},
		{ReasonImplementationSpecificError, 0x83},
		{ReasonUnsupportedProtocolVersion, 0x84},
		{ReasonTopicFilterInvalid, 0x8F},
		{ReasonTopicNameInvalid, 0x90},
		{ReasonPacketTooLarge, 0x95},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.value, byte(tt.reasonCode))
	}
}
