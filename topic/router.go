package topic

import "sync"

// Router manages topic subscriptions and routes messages to subscribers
type Router struct {
	trie          *Trie
	subscriptions map[string]map[string]*Subscription // clientID -> filter -> Subscription
	mu            sync.RWMutex
}

// NewRouter creates a new topic router
func NewRouter() *Router {
	return &Router{
		trie:          NewTrie(),
		subscriptions: make(map[string]map[string]*Subscription),
	}
}

// Subscribe adds a subscription to the router
func (r *Router) Subscribe(sub *Subscription) error {
	// Check if this is a shared subscription
	if IsSharedSubscription(sub.TopicFilter) {
		groupName, topicFilter, err := ValidateSharedSubscription(sub.TopicFilter)
		if err != nil {
			return err
		}

		subInfo := SubscriberInfo{
			ClientID:               sub.ClientID,
			QoS:                    sub.QoS,
			NoLocal:                sub.NoLocal,
			RetainAsPublished:      sub.RetainAsPublished,
			RetainHandling:         sub.RetainHandling,
			SubscriptionIdentifier: sub.SubscriptionIdentifier,
		}

		if err := r.trie.SubscribeShared(groupName, topicFilter, subInfo); err != nil {
			return err
		}

		// Store subscription metadata
		r.mu.Lock()
		if r.subscriptions[sub.ClientID] == nil {
			r.subscriptions[sub.ClientID] = make(map[string]*Subscription)
		}
		r.subscriptions[sub.ClientID][sub.TopicFilter] = sub
		r.mu.Unlock()

		return nil
	}

	// Regular subscription
	if err := ValidateTopicFilter(sub.TopicFilter); err != nil {
		return err
	}

	subInfo := SubscriberInfo{
		ClientID:               sub.ClientID,
		QoS:                    sub.QoS,
		NoLocal:                sub.NoLocal,
		RetainAsPublished:      sub.RetainAsPublished,
		RetainHandling:         sub.RetainHandling,
		SubscriptionIdentifier: sub.SubscriptionIdentifier,
	}

	if err := r.trie.Subscribe(sub.TopicFilter, subInfo); err != nil {
		return err
	}

	// Store subscription metadata
	r.mu.Lock()
	if r.subscriptions[sub.ClientID] == nil {
		r.subscriptions[sub.ClientID] = make(map[string]*Subscription)
	}
	r.subscriptions[sub.ClientID][sub.TopicFilter] = sub
	r.mu.Unlock()

	return nil
}

// Unsubscribe removes a subscription from the router
func (r *Router) Unsubscribe(clientID, filter string) bool {
	// Check if this is a shared subscription
	if IsSharedSubscription(filter) {
		groupName, topicFilter, err := ValidateSharedSubscription(filter)
		if err != nil {
			return false
		}

		found := r.trie.UnsubscribeShared(groupName, topicFilter, clientID)

		r.mu.Lock()
		if clientSubs, ok := r.subscriptions[clientID]; ok {
			delete(clientSubs, filter)
			if len(clientSubs) == 0 {
				delete(r.subscriptions, clientID)
			}
		}
		r.mu.Unlock()

		return found
	}

	// Regular unsubscribe
	found := r.trie.Unsubscribe(filter, clientID)

	r.mu.Lock()
	if clientSubs, ok := r.subscriptions[clientID]; ok {
		delete(clientSubs, filter)
		if len(clientSubs) == 0 {
			delete(r.subscriptions, clientID)
		}
	}
	r.mu.Unlock()

	return found
}

// UnsubscribeAll removes all subscriptions for a client
func (r *Router) UnsubscribeAll(clientID string) int {
	r.mu.Lock()
	clientSubs, ok := r.subscriptions[clientID]
	if !ok {
		r.mu.Unlock()
		return 0
	}

	filters := make([]string, 0, len(clientSubs))
	for filter := range clientSubs {
		filters = append(filters, filter)
	}
	delete(r.subscriptions, clientID)
	r.mu.Unlock()

	count := 0
	for _, filter := range filters {
		if r.Unsubscribe(clientID, filter) {
			count++
		}
	}

	return count
}

// Match finds all subscribers for a topic
func (r *Router) Match(topic string) []SubscriberInfo {
	return r.trie.Match(topic)
}

// MatchWithPublisher finds all subscribers for a topic, excluding the publisher if NoLocal is set
func (r *Router) MatchWithPublisher(topic, publisherClientID string) []SubscriberInfo {
	allSubs := r.trie.Match(topic)
	if publisherClientID == "" {
		return allSubs
	}

	// Filter out publisher for subscriptions with NoLocal=true
	filtered := make([]SubscriberInfo, 0, len(allSubs))
	for _, sub := range allSubs {
		if sub.NoLocal && sub.ClientID == publisherClientID {
			continue
		}
		filtered = append(filtered, sub)
	}

	return filtered
}

// GetSubscription retrieves a specific subscription
func (r *Router) GetSubscription(clientID, filter string) (*Subscription, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if clientSubs, ok := r.subscriptions[clientID]; ok {
		sub, ok := clientSubs[filter]
		return sub, ok
	}
	return nil, false
}

// GetClientSubscriptions retrieves all subscriptions for a client
func (r *Router) GetClientSubscriptions(clientID string) []*Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clientSubs, ok := r.subscriptions[clientID]
	if !ok {
		return nil
	}

	result := make([]*Subscription, 0, len(clientSubs))
	for _, sub := range clientSubs {
		result = append(result, sub)
	}
	return result
}

// Count returns the total number of subscriptions
func (r *Router) Count() int {
	return r.trie.Count()
}

// CountClients returns the number of clients with subscriptions
func (r *Router) CountClients() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.subscriptions)
}

// Clear removes all subscriptions
func (r *Router) Clear() {
	r.mu.Lock()
	r.subscriptions = make(map[string]map[string]*Subscription)
	r.mu.Unlock()
	r.trie.Clear()
}
