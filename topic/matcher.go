package topic

import "strings"

type TopicMatcher struct{}

func NewTopicMatcher() *TopicMatcher {
	return &TopicMatcher{}
}

func (tm *TopicMatcher) Match(filter, topic string) bool {
	return matchTopicFilter(filter, topic)
}

func matchTopicFilter(filter, topic string) bool {
	if strings.HasPrefix(topic, "$") &&
		(strings.Contains(filter, "#") ||
			strings.Contains(filter, "+")) {
		return false
	}

	if filter == topic {
		return true
	}

	filterLevels := splitTopicLevels(filter)
	topicLevels := splitTopicLevels(topic)

	return matchLevels(filterLevels, topicLevels)
}

func matchLevels(filterLevels, topicLevels []string) bool {
	filterLen := len(filterLevels)
	topicLen := len(topicLevels)

	fi := 0
	ti := 0

	for fi < filterLen && ti < topicLen {
		filterLevel := filterLevels[fi]
		topicLevel := topicLevels[ti]

		if filterLevel == "#" {
			return true
		}

		if filterLevel == "+" {
			fi++
			ti++
			continue
		}

		if filterLevel != topicLevel {
			return false
		}

		fi++
		ti++
	}

	if fi < filterLen {
		return filterLen-fi == 1 && filterLevels[fi] == "#"
	}

	return ti == topicLen
}
