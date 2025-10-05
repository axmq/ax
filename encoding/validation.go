package encoding

import (
	"strings"
)

// ValidatePacketID checks if a packet ID is valid for packets requiring one
func ValidatePacketID(packetID uint16, requireNonZero bool) error {
	if requireNonZero && packetID == 0 {
		return ErrInvalidPacketIDZero
	}
	return nil
}

// ValidateTopicName validates an MQTT topic name (used in PUBLISH)
// Topic names must not contain wildcards and follow MQTT spec rules
func ValidateTopicName(topic string) error {
	if topic == "" {
		return ErrInvalidTopicName
	}

	// Topic names cannot contain wildcard characters
	if strings.ContainsAny(topic, "+#") {
		return ErrInvalidPublishTopicName
	}

	// Check for valid UTF-8 (already done during string reading, but double-check)
	if !isValidMQTTString(topic) {
		return ErrInvalidTopicName
	}

	return nil
}

// ValidateTopicFilter validates an MQTT topic filter (used in SUBSCRIBE/UNSUBSCRIBE)
func ValidateTopicFilter(filter string) error {
	if filter == "" {
		return ErrEmptyTopicFilter
	}

	// Split by '/' to validate each level
	levels := strings.Split(filter, "/")

	for i, level := range levels {
		// Multi-level wildcard '#' must be last and alone in its level
		if strings.Contains(level, "#") {
			if level != "#" || i != len(levels)-1 {
				return ErrInvalidTopicFilter
			}
		}

		// Single-level wildcard '+' must be alone in its level
		if strings.Contains(level, "+") {
			if level != "+" {
				return ErrInvalidTopicFilter
			}
		}

		// Check for valid UTF-8
		if !isValidMQTTString(level) {
			return ErrInvalidTopicFilter
		}
	}

	return nil
}

// isValidMQTTString checks if a string is valid for MQTT
func isValidMQTTString(s string) bool {
	// Must not contain null characters (U+0000)
	// Must not contain characters U+D800 to U+DFFF (UTF-16 surrogates)
	for _, r := range s {
		if r == 0x0000 {
			return false
		}
		if r >= 0xD800 && r <= 0xDFFF {
			return false
		}
	}
	return true
}

// ValidateConnectFlags validates the CONNECT packet flags
func ValidateConnectFlags(flags byte) error {
	// Reserved bit (bit 0) must be 0 per MQTT spec
	if (flags & 0x01) != 0 {
		return ErrInvalidConnectFlags
	}

	willFlag := (flags & 0x04) != 0
	willQoS := QoS((flags & 0x18) >> 3)
	willRetain := (flags & 0x20) != 0
	passwordFlag := (flags & 0x40) != 0
	usernameFlag := (flags & 0x80) != 0

	// Validate Will QoS
	if !willQoS.IsValid() {
		return ErrInvalidWillQoS
	}

	// If Will flag is 0, Will QoS and Will Retain must be 0
	if !willFlag && (willQoS != QoS0 || willRetain) {
		return ErrWillFlagMismatch
	}

	// Password flag requires Username flag (MQTT 5.0 [MQTT-3.1.2-22])
	if passwordFlag && !usernameFlag {
		return ErrPasswordWithoutUsername
	}

	return nil
}

// ValidateSubscriptionOptions validates subscription options byte
func ValidateSubscriptionOptions(options byte) error {
	// Extract QoS (bits 0-1)
	qos := QoS(options & 0x03)
	if !qos.IsValid() {
		return ErrInvalidSubscriptionOpts
	}

	// Bits 2-3: No Local, Retain As Published (valid values 0 or 1)
	// Bits 4-5: Retain Handling (valid values 0, 1, 2)
	retainHandling := (options & 0x30) >> 4
	if retainHandling > 2 {
		return ErrInvalidSubscriptionOpts
	}

	// Bits 6-7: Reserved, must be 0
	if (options & 0xC0) != 0 {
		return ErrInvalidSubscriptionOpts
	}

	return nil
}

// ValidatePublishPacket validates PUBLISH packet structure
func ValidatePublishPacket(topicName string, qos QoS, packetID uint16) error {
	if err := ValidateTopicName(topicName); err != nil {
		return err
	}

	if !qos.IsValid() {
		return ErrInvalidQoS
	}

	// QoS 1 and 2 require non-zero packet ID
	if qos > QoS0 {
		if err := ValidatePacketID(packetID, true); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRemainingLength checks if remaining length is within bounds
// MQTT spec allows maximum 268,435,455 bytes (0xFF, 0xFF, 0xFF, 0x7F)
func ValidateRemainingLength(length uint32) error {
	const maxRemainingLength uint32 = 268435455 // (0xFF, 0xFF, 0xFF, 0x7F)
	if length > maxRemainingLength {
		return ErrInvalidRemainingLength
	}
	return nil
}

// ValidateReasonCodeForPacket validates that a reason code is appropriate for a packet type
func ValidateReasonCodeForPacket(packetType PacketType, reasonCode ReasonCode) error {
	// Define valid reason codes for each packet type
	// This is a simplified validation - full spec has more detailed rules
	switch packetType {
	case CONNACK:
		// CONNACK can use various success and error codes
		return nil
	case PUBACK, PUBREC, PUBREL, PUBCOMP:
		// QoS acknowledgment packets
		return nil
	case SUBACK:
		// SUBACK can use QoS grant codes and error codes
		return nil
	case UNSUBACK:
		// UNSUBACK uses specific reason codes
		return nil
	case DISCONNECT, AUTH:
		// DISCONNECT and AUTH have their own reason codes
		return nil
	default:
		// Other packets don't typically have reason codes
		if reasonCode != 0 {
			return ErrInvalidReasonCode
		}
	}
	return nil
}

// ValidatePropertyLength validates that property length doesn't exceed packet bounds
func ValidatePropertyLength(propLength uint32, remainingBytes uint32) error {
	if propLength > remainingBytes {
		return ErrInvalidPropertyLength
	}
	return nil
}
