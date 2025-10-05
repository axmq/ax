package topic

import (
	"unicode/utf8"
)

// ValidationError represents a topic validation error
type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

// ValidateTopic validates a topic name according to MQTT 5.0/3.1.1 specification
func ValidateTopic(topic string) error {
	if len(topic) == 0 {
		return &ValidationError{"topic cannot be empty"}
	}

	if len(topic) > 65535 {
		return &ValidationError{"topic exceeds maximum length of 65535 bytes"}
	}

	if !utf8.ValidString(topic) {
		return &ValidationError{"topic contains invalid UTF-8 characters"}
	}

	// Topic names cannot contain wildcards
	for i := 0; i < len(topic); i++ {
		c := topic[i]
		if c == '+' || c == '#' {
			return &ValidationError{"topic name cannot contain wildcard characters"}
		}
		if c == 0 {
			return &ValidationError{"topic cannot contain null characters"}
		}
	}

	return nil
}

// ValidateTopicFilter validates a topic filter according to MQTT 5.0/3.1.1 specification
func ValidateTopicFilter(filter string) error {
	if len(filter) == 0 {
		return &ValidationError{"topic filter cannot be empty"}
	}

	if len(filter) > 65535 {
		return &ValidationError{"topic filter exceeds maximum length of 65535 bytes"}
	}

	if !utf8.ValidString(filter) {
		return &ValidationError{"topic filter contains invalid UTF-8 characters"}
	}

	// Check for null characters
	for i := 0; i < len(filter); i++ {
		if filter[i] == 0 {
			return &ValidationError{"topic filter cannot contain null characters"}
		}
	}

	// Validate wildcard usage
	levels := splitTopicLevels(filter)
	for _, level := range levels {
		if len(level) == 0 {
			continue // Empty level is valid (e.g., "a//b")
		}

		// Multi-level wildcard '#' must be last and alone in its level
		if contains(level, '#') {
			if level != "#" {
				return &ValidationError{"multi-level wildcard '#' must occupy entire level"}
			}
			if level != levels[len(levels)-1] {
				return &ValidationError{"multi-level wildcard '#' must be last level"}
			}
		}

		// Single-level wildcard '+' must be alone in its level
		if contains(level, '+') {
			if level != "+" {
				return &ValidationError{"single-level wildcard '+' must occupy entire level"}
			}
		}
	}

	return nil
}

// ValidateSharedSubscription validates a shared subscription filter
func ValidateSharedSubscription(filter string) (groupName string, topicFilter string, err error) {
	if len(filter) < 9 { // "$share/x/y" minimum length
		return "", "", &ValidationError{"invalid shared subscription format"}
	}

	if filter[:7] != "$share/" {
		return "", "", &ValidationError{"shared subscription must start with $share/"}
	}

	remainder := filter[7:]
	slashIdx := -1
	for i := 0; i < len(remainder); i++ {
		if remainder[i] == '/' {
			slashIdx = i
			break
		}
	}

	if slashIdx == -1 || slashIdx == 0 {
		return "", "", &ValidationError{"shared subscription missing group name"}
	}

	groupName = remainder[:slashIdx]
	if len(groupName) == 0 {
		return "", "", &ValidationError{"shared subscription group name cannot be empty"}
	}

	if slashIdx+1 >= len(remainder) {
		return "", "", &ValidationError{"shared subscription missing topic filter"}
	}

	topicFilter = remainder[slashIdx+1:]
	if err := ValidateTopicFilter(topicFilter); err != nil {
		return "", "", err
	}

	return groupName, topicFilter, nil
}

// IsSharedSubscription checks if a filter is a shared subscription
func IsSharedSubscription(filter string) bool {
	return len(filter) >= 7 && filter[:7] == "$share/"
}

// splitTopicLevels splits a topic into levels by '/'
func splitTopicLevels(topic string) []string {
	if len(topic) == 0 {
		return []string{}
	}

	levels := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(topic); i++ {
		if topic[i] == '/' {
			levels = append(levels, topic[start:i])
			start = i + 1
		}
	}
	levels = append(levels, topic[start:])
	return levels
}

// contains checks if a string contains a byte
func contains(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}
