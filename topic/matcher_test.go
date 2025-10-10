package topic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopicMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		topic     string
		wantMatch bool
	}{
		{
			name:      "exact match",
			filter:    "home/room/temperature",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "no match",
			filter:    "home/room/temperature",
			topic:     "home/room/humidity",
			wantMatch: false,
		},
		{
			name:      "single level wildcard match",
			filter:    "home/+/temperature",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "single level wildcard no match",
			filter:    "home/+/temperature",
			topic:     "home/room/kitchen/temperature",
			wantMatch: false,
		},
		{
			name:      "multi level wildcard match",
			filter:    "home/#",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "multi level wildcard match all",
			filter:    "#",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "multi level wildcard at end",
			filter:    "home/room/#",
			topic:     "home/room/temperature/sensor1",
			wantMatch: true,
		},
		{
			name:      "multiple single level wildcards",
			filter:    "home/+/+/temperature",
			topic:     "home/room/kitchen/temperature",
			wantMatch: true,
		},
		{
			name:      "mixed wildcards",
			filter:    "home/+/sensor/#",
			topic:     "home/room/sensor/temperature/value",
			wantMatch: true,
		},
		{
			name:      "empty topic no match",
			filter:    "home/room",
			topic:     "",
			wantMatch: false,
		},
		{
			name:      "filter longer than topic",
			filter:    "home/room/temperature/sensor",
			topic:     "home/room",
			wantMatch: false,
		},
		{
			name:      "topic longer than filter",
			filter:    "home/room",
			topic:     "home/room/temperature",
			wantMatch: false,
		},
		{
			name:      "single level wildcard only",
			filter:    "+",
			topic:     "home",
			wantMatch: true,
		},
		{
			name:      "single level wildcard only no match",
			filter:    "+",
			topic:     "home/room",
			wantMatch: false,
		},
		{
			name:      "dollar prefix no match with wildcard",
			filter:    "#",
			topic:     "$SYS/broker/clients",
			wantMatch: false,
		},
		{
			name:      "single level at start",
			filter:    "+/room/temperature",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "single level at end",
			filter:    "home/room/+",
			topic:     "home/room/temperature",
			wantMatch: true,
		},
		{
			name:      "trailing slash filter",
			filter:    "home/room/",
			topic:     "home/room/",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewTopicMatcher()
			result := matcher.Match(tt.filter, tt.topic)
			assert.Equal(t, tt.wantMatch, result)
		})
	}
}

func TestMatchTopicFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		topic     string
		wantMatch bool
	}{
		{
			name:      "sports topics",
			filter:    "sport/tennis/+",
			topic:     "sport/tennis/player1",
			wantMatch: true,
		},
		{
			name:      "sports wildcard",
			filter:    "sport/#",
			topic:     "sport/tennis/player1/ranking",
			wantMatch: true,
		},
		{
			name:      "account topics",
			filter:    "account/+/balance",
			topic:     "account/12345/balance",
			wantMatch: true,
		},
		{
			name:      "sensor topics",
			filter:    "sensor/+/+/temperature",
			topic:     "sensor/building1/room2/temperature",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchTopicFilter(tt.filter, tt.topic)
			assert.Equal(t, tt.wantMatch, result)
		})
	}
}

func BenchmarkTopicMatcher_Match(b *testing.B) {
	tests := []struct {
		name   string
		filter string
		topic  string
	}{
		{
			name:   "exact match",
			filter: "home/room/temperature",
			topic:  "home/room/temperature",
		},
		{
			name:   "single level wildcard",
			filter: "home/+/temperature",
			topic:  "home/room/temperature",
		},
		{
			name:   "multi level wildcard",
			filter: "home/#",
			topic:  "home/room/temperature/sensor1",
		},
		{
			name:   "complex filter",
			filter: "home/+/sensor/+/temperature",
			topic:  "home/room/sensor/device1/temperature",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			matcher := NewTopicMatcher()
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				matcher.Match(tt.filter, tt.topic)
			}
		})
	}
}

func BenchmarkMatchTopicFilter(b *testing.B) {
	filter := "home/+/sensor/+/temperature"
	topic := "home/room/sensor/device1/temperature"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		matchTopicFilter(filter, topic)
	}
}
