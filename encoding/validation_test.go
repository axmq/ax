package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePacketID(t *testing.T) {
	tests := []struct {
		name           string
		packetID       uint16
		requireNonZero bool
		expectError    bool
		expectedErr    error
	}{
		{
			name:           "Valid non-zero packet ID",
			packetID:       1,
			requireNonZero: true,
			expectError:    false,
		},
		{
			name:           "Valid max packet ID",
			packetID:       65535,
			requireNonZero: true,
			expectError:    false,
		},
		{
			name:           "Zero packet ID when not required",
			packetID:       0,
			requireNonZero: false,
			expectError:    false,
		},
		{
			name:           "Zero packet ID when required",
			packetID:       0,
			requireNonZero: true,
			expectError:    true,
			expectedErr:    ErrInvalidPacketIDZero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePacketID(tt.packetID, tt.requireNonZero)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTopicName(t *testing.T) {
	tests := []struct {
		name        string
		topicName   string
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid simple topic",
			topicName:   "sensors/temperature",
			expectError: false,
		},
		{
			name:        "Valid topic with multiple levels",
			topicName:   "home/room1/sensor/temp",
			expectError: false,
		},
		{
			name:        "Valid single level topic",
			topicName:   "temperature",
			expectError: false,
		},
		{
			name:        "Empty topic name",
			topicName:   "",
			expectError: true,
			expectedErr: ErrInvalidTopicName,
		},
		{
			name:        "Topic with single-level wildcard",
			topicName:   "sensors/+/temperature",
			expectError: true,
			expectedErr: ErrInvalidPublishTopicName,
		},
		{
			name:        "Topic with multi-level wildcard",
			topicName:   "sensors/#",
			expectError: true,
			expectedErr: ErrInvalidPublishTopicName,
		},
		{
			name:        "Topic with both wildcards",
			topicName:   "sensors/+/#",
			expectError: true,
			expectedErr: ErrInvalidPublishTopicName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopicName(tt.topicName)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTopicFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid simple filter",
			filter:      "sensors/temperature",
			expectError: false,
		},
		{
			name:        "Valid filter with single-level wildcard",
			filter:      "sensors/+/temperature",
			expectError: false,
		},
		{
			name:        "Valid filter with multi-level wildcard",
			filter:      "sensors/#",
			expectError: false,
		},
		{
			name:        "Valid filter with both wildcards",
			filter:      "sensors/+/room/#",
			expectError: false,
		},
		{
			name:        "Valid single-level wildcard only",
			filter:      "+",
			expectError: false,
		},
		{
			name:        "Valid multi-level wildcard only",
			filter:      "#",
			expectError: false,
		},
		{
			name:        "Empty filter",
			filter:      "",
			expectError: true,
			expectedErr: ErrEmptyTopicFilter,
		},
		{
			name:        "Multi-level wildcard not at end",
			filter:      "sensors/#/temperature",
			expectError: true,
			expectedErr: ErrInvalidTopicFilter,
		},
		{
			name:        "Multi-level wildcard with other characters",
			filter:      "sensors/room#",
			expectError: true,
			expectedErr: ErrInvalidTopicFilter,
		},
		{
			name:        "Single-level wildcard with other characters",
			filter:      "sensors/room+",
			expectError: true,
			expectedErr: ErrInvalidTopicFilter,
		},
		{
			name:        "Multiple multi-level wildcards",
			filter:      "#/#",
			expectError: true,
			expectedErr: ErrInvalidTopicFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopicFilter(tt.filter)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateConnectFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       byte
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid flags: clean start only",
			flags:       0x02,
			expectError: false,
		},
		{
			name:        "Valid flags: clean start + username",
			flags:       0x82,
			expectError: false,
		},
		{
			name:        "Valid flags: clean start + username + password",
			flags:       0xC2,
			expectError: false,
		},
		{
			name:        "Valid flags: with will (QoS 0)",
			flags:       0x06,
			expectError: false,
		},
		{
			name:        "Valid flags: with will (QoS 1)",
			flags:       0x0E,
			expectError: false,
		},
		{
			name:        "Valid flags: with will (QoS 2) and retain",
			flags:       0x36,
			expectError: false,
		},
		{
			name:        "Invalid: reserved bit set",
			flags:       0x01,
			expectError: true,
			expectedErr: ErrInvalidConnectFlags,
		},
		{
			name:        "Invalid: reserved bit set with other flags",
			flags:       0x83,
			expectError: true,
			expectedErr: ErrInvalidConnectFlags,
		},
		{
			name:        "Invalid: will QoS = 3",
			flags:       0x1E,
			expectError: true,
			expectedErr: ErrInvalidWillQoS,
		},
		{
			name:        "Invalid: will retain without will flag",
			flags:       0x20,
			expectError: true,
			expectedErr: ErrWillFlagMismatch,
		},
		{
			name:        "Invalid: will QoS without will flag",
			flags:       0x08,
			expectError: true,
			expectedErr: ErrWillFlagMismatch,
		},
		{
			name:        "Invalid: password without username",
			flags:       0x42,
			expectError: true,
			expectedErr: ErrPasswordWithoutUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConnectFlags(tt.flags)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateSubscriptionOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     byte
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid: QoS 0, all others 0",
			options:     0x00,
			expectError: false,
		},
		{
			name:        "Valid: QoS 1",
			options:     0x01,
			expectError: false,
		},
		{
			name:        "Valid: QoS 2",
			options:     0x02,
			expectError: false,
		},
		{
			name:        "Valid: QoS 1 with No Local",
			options:     0x05,
			expectError: false,
		},
		{
			name:        "Valid: QoS 2 with Retain As Published",
			options:     0x0A,
			expectError: false,
		},
		{
			name:        "Valid: Retain Handling = 1",
			options:     0x10,
			expectError: false,
		},
		{
			name:        "Valid: Retain Handling = 2",
			options:     0x20,
			expectError: false,
		},
		{
			name:        "Valid: All valid options combined",
			options:     0x2E, // QoS=2, NoLocal=1, RetainAsPublished=1, RetainHandling=2
			expectError: false,
		},
		{
			name:        "Invalid: QoS = 3",
			options:     0x03,
			expectError: true,
			expectedErr: ErrInvalidSubscriptionOpts,
		},
		{
			name:        "Invalid: Retain Handling = 3",
			options:     0x30,
			expectError: true,
			expectedErr: ErrInvalidSubscriptionOpts,
		},
		{
			name:        "Invalid: Reserved bit 6 set",
			options:     0x40,
			expectError: true,
			expectedErr: ErrInvalidSubscriptionOpts,
		},
		{
			name:        "Invalid: Reserved bit 7 set",
			options:     0x80,
			expectError: true,
			expectedErr: ErrInvalidSubscriptionOpts,
		},
		{
			name:        "Invalid: Both reserved bits set",
			options:     0xC0,
			expectError: true,
			expectedErr: ErrInvalidSubscriptionOpts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubscriptionOptions(tt.options)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePublishPacket(t *testing.T) {
	tests := []struct {
		name        string
		topicName   string
		qos         QoS
		packetID    uint16
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid QoS 0 packet",
			topicName:   "sensors/temp",
			qos:         QoS0,
			packetID:    0,
			expectError: false,
		},
		{
			name:        "Valid QoS 1 packet",
			topicName:   "sensors/temp",
			qos:         QoS1,
			packetID:    1,
			expectError: false,
		},
		{
			name:        "Valid QoS 2 packet",
			topicName:   "sensors/temp",
			qos:         QoS2,
			packetID:    100,
			expectError: false,
		},
		{
			name:        "Invalid: QoS 1 with zero packet ID",
			topicName:   "sensors/temp",
			qos:         QoS1,
			packetID:    0,
			expectError: true,
			expectedErr: ErrInvalidPacketIDZero,
		},
		{
			name:        "Invalid: QoS 2 with zero packet ID",
			topicName:   "sensors/temp",
			qos:         QoS2,
			packetID:    0,
			expectError: true,
			expectedErr: ErrInvalidPacketIDZero,
		},
		{
			name:        "Invalid: empty topic name",
			topicName:   "",
			qos:         QoS0,
			packetID:    0,
			expectError: true,
			expectedErr: ErrInvalidTopicName,
		},
		{
			name:        "Invalid: topic with wildcard",
			topicName:   "sensors/+",
			qos:         QoS0,
			packetID:    0,
			expectError: true,
			expectedErr: ErrInvalidPublishTopicName,
		},
		{
			name:        "Invalid: QoS = 3",
			topicName:   "sensors/temp",
			qos:         QoS(3),
			packetID:    0,
			expectError: true,
			expectedErr: ErrInvalidQoS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublishPacket(tt.topicName, tt.qos, tt.packetID)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateRemainingLength(t *testing.T) {
	tests := []struct {
		name        string
		length      uint32
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid: zero length",
			length:      0,
			expectError: false,
		},
		{
			name:        "Valid: small length",
			length:      127,
			expectError: false,
		},
		{
			name:        "Valid: medium length",
			length:      16383,
			expectError: false,
		},
		{
			name:        "Valid: large length",
			length:      2097151,
			expectError: false,
		},
		{
			name:        "Valid: maximum allowed length",
			length:      268435455,
			expectError: false,
		},
		{
			name:        "Invalid: exceeds maximum",
			length:      268435456,
			expectError: true,
			expectedErr: ErrInvalidRemainingLength,
		},
		{
			name:        "Invalid: much larger",
			length:      1000000000,
			expectError: true,
			expectedErr: ErrInvalidRemainingLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemainingLength(tt.length)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePropertyLength(t *testing.T) {
	tests := []struct {
		name           string
		propLength     uint32
		remainingBytes uint32
		expectError    bool
		expectedErr    error
	}{
		{
			name:           "Valid: property length within bounds",
			propLength:     10,
			remainingBytes: 20,
			expectError:    false,
		},
		{
			name:           "Valid: property length equals remaining",
			propLength:     20,
			remainingBytes: 20,
			expectError:    false,
		},
		{
			name:           "Valid: zero property length",
			propLength:     0,
			remainingBytes: 10,
			expectError:    false,
		},
		{
			name:           "Invalid: property length exceeds remaining",
			propLength:     30,
			remainingBytes: 20,
			expectError:    true,
			expectedErr:    ErrInvalidPropertyLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePropertyLength(tt.propLength, tt.remainingBytes)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
