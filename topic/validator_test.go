package topic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		wantErr bool
	}{
		{
			name:    "valid simple topic",
			topic:   "sensor/temperature",
			wantErr: false,
		},
		{
			name:    "valid topic with multiple levels",
			topic:   "home/room1/sensor/temperature",
			wantErr: false,
		},
		{
			name:    "valid topic with numbers",
			topic:   "device/123/status",
			wantErr: false,
		},
		{
			name:    "valid topic with special chars",
			topic:   "home/room-1/sensor_temp",
			wantErr: false,
		},
		{
			name:    "valid topic with unicode",
			topic:   "home/комната/температура",
			wantErr: false,
		},
		{
			name:    "valid topic with emoji",
			topic:   "home/room/🌡️",
			wantErr: false,
		},
		{
			name:    "valid single level topic",
			topic:   "temperature",
			wantErr: false,
		},
		{
			name:    "valid topic with trailing slash",
			topic:   "home/room/",
			wantErr: false,
		},
		{
			name:    "valid topic with leading slash",
			topic:   "/home/room",
			wantErr: false,
		},
		{
			name:    "empty topic",
			topic:   "",
			wantErr: true,
		},
		{
			name:    "topic with single-level wildcard",
			topic:   "home/+/temperature",
			wantErr: true,
		},
		{
			name:    "topic with multi-level wildcard",
			topic:   "home/#",
			wantErr: true,
		},
		{
			name:    "topic with null character",
			topic:   "home/\x00/temperature",
			wantErr: true,
		},
		{
			name:    "topic exceeding max length",
			topic:   strings.Repeat("a", 65536),
			wantErr: true,
		},
		{
			name:    "topic with invalid UTF-8",
			topic:   "home/\xff\xfe/temperature",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopic(tt.topic)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTopicFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{
			name:    "valid simple filter",
			filter:  "sensor/temperature",
			wantErr: false,
		},
		{
			name:    "valid filter with single-level wildcard",
			filter:  "home/+/temperature",
			wantErr: false,
		},
		{
			name:    "valid filter with multi-level wildcard",
			filter:  "home/#",
			wantErr: false,
		},
		{
			name:    "valid filter with both wildcards",
			filter:  "home/+/sensor/#",
			wantErr: false,
		},
		{
			name:    "valid filter with multiple single-level wildcards",
			filter:  "+/+/temperature",
			wantErr: false,
		},
		{
			name:    "valid filter single level wildcard only",
			filter:  "+",
			wantErr: false,
		},
		{
			name:    "valid filter multi level wildcard only",
			filter:  "#",
			wantErr: false,
		},
		{
			name:    "valid filter with leading slash",
			filter:  "/home/+/temperature",
			wantErr: false,
		},
		{
			name:    "valid filter with trailing slash before wildcard",
			filter:  "home/room/#",
			wantErr: false,
		},
		{
			name:    "empty filter",
			filter:  "",
			wantErr: true,
		},
		{
			name:    "filter with invalid single-level wildcard",
			filter:  "home/room+/temperature",
			wantErr: true,
		},
		{
			name:    "filter with invalid multi-level wildcard not at end",
			filter:  "home/#/temperature",
			wantErr: true,
		},
		{
			name:    "filter with invalid multi-level wildcard with text",
			filter:  "home/room#",
			wantErr: true,
		},
		{
			name:    "filter with null character",
			filter:  "home/+/\x00",
			wantErr: true,
		},
		{
			name:    "filter exceeding max length",
			filter:  strings.Repeat("a", 65536),
			wantErr: true,
		},
		{
			name:    "filter with invalid UTF-8",
			filter:  "home/\xff\xfe/+",
			wantErr: true,
		},
		{
			name:    "filter with plus in middle of level",
			filter:  "home/te+mp/sensor",
			wantErr: true,
		},
		{
			name:    "filter with hash in middle of level",
			filter:  "home/te#mp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopicFilter(tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSharedSubscription(t *testing.T) {
	tests := []struct {
		name            string
		filter          string
		wantGroup       string
		wantTopicFilter string
		wantErr         bool
	}{
		{
			name:            "valid shared subscription",
			filter:          "$share/group1/sensor/temperature",
			wantGroup:       "group1",
			wantTopicFilter: "sensor/temperature",
			wantErr:         false,
		},
		{
			name:            "valid shared subscription with wildcard",
			filter:          "$share/group1/sensor/#",
			wantGroup:       "group1",
			wantTopicFilter: "sensor/#",
			wantErr:         false,
		},
		{
			name:            "valid shared subscription with plus wildcard",
			filter:          "$share/mygroup/home/+/temp",
			wantGroup:       "mygroup",
			wantTopicFilter: "home/+/temp",
			wantErr:         false,
		},
		{
			name:            "valid shared subscription single char group",
			filter:          "$share/g/topic",
			wantGroup:       "g",
			wantTopicFilter: "topic",
			wantErr:         false,
		},
		{
			name:    "missing prefix",
			filter:  "share/group1/sensor/temperature",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			filter:  "$shared/group1/sensor/temperature",
			wantErr: true,
		},
		{
			name:    "missing group name",
			filter:  "$share//sensor/temperature",
			wantErr: true,
		},
		{
			name:    "missing topic filter",
			filter:  "$share/group1/",
			wantErr: true,
		},
		{
			name:    "missing topic filter no slash",
			filter:  "$share/group1",
			wantErr: true,
		},
		{
			name:    "too short",
			filter:  "$share/",
			wantErr: true,
		},
		{
			name:    "empty string",
			filter:  "",
			wantErr: true,
		},
		{
			name:    "invalid topic filter in shared subscription",
			filter:  "$share/group1/sensor#",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, topicFilter, err := ValidateSharedSubscription(tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantGroup, group)
				assert.Equal(t, tt.wantTopicFilter, topicFilter)
			}
		})
	}
}

func TestIsSharedSubscription(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{
			name:   "shared subscription",
			filter: "$share/group1/sensor/temperature",
			want:   true,
		},
		{
			name:   "regular subscription",
			filter: "sensor/temperature",
			want:   false,
		},
		{
			name:   "shared prefix only",
			filter: "$share/",
			want:   true,
		},
		{
			name:   "partial shared prefix",
			filter: "$shar",
			want:   false,
		},
		{
			name:   "empty string",
			filter: "",
			want:   false,
		},
		{
			name:   "wrong case",
			filter: "$SHARE/group/topic",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSharedSubscription(tt.filter)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSplitTopicLevels(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		want  []string
	}{
		{
			name:  "simple topic",
			topic: "sensor/temperature",
			want:  []string{"sensor", "temperature"},
		},
		{
			name:  "multiple levels",
			topic: "home/room1/sensor/temperature",
			want:  []string{"home", "room1", "sensor", "temperature"},
		},
		{
			name:  "single level",
			topic: "temperature",
			want:  []string{"temperature"},
		},
		{
			name:  "empty string",
			topic: "",
			want:  []string{},
		},
		{
			name:  "leading slash",
			topic: "/home/room",
			want:  []string{"", "home", "room"},
		},
		{
			name:  "trailing slash",
			topic: "home/room/",
			want:  []string{"home", "room", ""},
		},
		{
			name:  "double slash",
			topic: "home//room",
			want:  []string{"home", "", "room"},
		},
		{
			name:  "single slash",
			topic: "/",
			want:  []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitTopicLevels(tt.topic)
			assert.Equal(t, tt.want, result)
		})
	}
}

func BenchmarkValidateTopic(b *testing.B) {
	topic := "home/room1/sensor/temperature/value"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateTopic(topic)
	}
}

func BenchmarkValidateTopicFilter(b *testing.B) {
	filter := "home/+/sensor/#"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateTopicFilter(filter)
	}
}

func BenchmarkValidateSharedSubscription(b *testing.B) {
	filter := "$share/group1/home/+/sensor/#"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ValidateSharedSubscription(filter)
	}
}

func BenchmarkSplitTopicLevels(b *testing.B) {
	topic := "home/room1/sensor/temperature/value"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitTopicLevels(topic)
	}
}
