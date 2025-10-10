package topic

import (
	"sync"
	"sync/atomic"

	"github.com/axmq/ax/types/message"
)

// Subscription represents an active subscription with all MQTT 5.0 features
type Subscription struct {
	ClientID               string
	TopicFilter            string
	QoS                    byte
	NoLocal                bool
	RetainAsPublished      bool
	RetainHandling         byte
	SubscriptionIdentifier uint32
	SharedGroup            string // For shared subscriptions ($share/groupname/topic)
}

// RetainedMessage represents a retained message
type RetainedMessage struct {
	Message *message.Message
}

// SubscriberInfo contains subscriber metadata for routing
type SubscriberInfo struct {
	ClientID               string
	QoS                    byte
	NoLocal                bool
	RetainAsPublished      bool
	RetainHandling         byte
	SubscriptionIdentifier uint32
}

// Alias manages topic alias mapping for MQTT 5.0
type Alias struct {
	maxAlias uint16
	aliases  map[uint16]string
}

// NewTopicAlias creates a new topic alias manager
func NewTopicAlias(maxAlias uint16) *Alias {
	return &Alias{
		maxAlias: maxAlias,
		aliases:  make(map[uint16]string),
	}
}

// Set maps an alias to a topic
func (ta *Alias) Set(alias uint16, topic string) bool {
	if alias == 0 || alias > ta.maxAlias {
		return false
	}
	ta.aliases[alias] = topic
	return true
}

// Get retrieves the topic for an alias
func (ta *Alias) Get(alias uint16) (string, bool) {
	topic, ok := ta.aliases[alias]
	return topic, ok
}

// Clear removes all aliases
func (ta *Alias) Clear() {
	ta.aliases = make(map[uint16]string)
}

// SharedSubscriptionGroup manages load balancing for shared subscriptions
type SharedSubscriptionGroup struct {
	groupName   string
	subscribers []SubscriberInfo
	counter     atomic.Uint64 // Round-robin counter
	mu          sync.RWMutex  // Protects subscribers slice
}

// NewSharedSubscriptionGroup creates a new shared subscription group
func NewSharedSubscriptionGroup(groupName string) *SharedSubscriptionGroup {
	return &SharedSubscriptionGroup{
		groupName:   groupName,
		subscribers: make([]SubscriberInfo, 0),
	}
}

// AddSubscriber adds a subscriber to the group
func (g *SharedSubscriptionGroup) AddSubscriber(sub SubscriberInfo) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.subscribers = append(g.subscribers, sub)
}

// RemoveSubscriber removes a subscriber from the group
func (g *SharedSubscriptionGroup) RemoveSubscriber(clientID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, sub := range g.subscribers {
		if sub.ClientID == clientID {
			g.subscribers = append(g.subscribers[:i], g.subscribers[i+1:]...)
			return true
		}
	}
	return false
}

// NextSubscriber returns the next subscriber using round-robin
func (g *SharedSubscriptionGroup) NextSubscriber() (SubscriberInfo, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.subscribers) == 0 {
		return SubscriberInfo{}, false
	}
	idx := g.counter.Add(1) - 1
	return g.subscribers[idx%uint64(len(g.subscribers))], true
}

// Size returns the number of subscribers in the group
func (g *SharedSubscriptionGroup) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.subscribers)
}

// GetSubscribers returns all subscribers in the group
func (g *SharedSubscriptionGroup) GetSubscribers() []SubscriberInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]SubscriberInfo, len(g.subscribers))
	copy(result, g.subscribers)
	return result
}
