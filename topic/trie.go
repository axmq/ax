package topic

import "sync"

// trieNode represents a node in the topic trie
type trieNode struct {
	children       map[string]*trieNode
	subscribers    []SubscriberInfo
	sharedGroups   map[string]*SharedSubscriptionGroup
	hasMultiLevel  bool // Has '#' wildcard
	hasSingleLevel bool // Has '+' wildcard
	mu             sync.RWMutex
}

// newTrieNode creates a new trie node
func newTrieNode() *trieNode {
	return &trieNode{
		children:     make(map[string]*trieNode),
		subscribers:  make([]SubscriberInfo, 0),
		sharedGroups: make(map[string]*SharedSubscriptionGroup),
	}
}

// Trie implements a trie-based topic filter matcher
type Trie struct {
	root *trieNode
	mu   sync.RWMutex
}

// NewTrie creates a new topic trie
func NewTrie() *Trie {
	return &Trie{
		root: newTrieNode(),
	}
}

// Subscribe adds a subscription to the trie
func (t *Trie) Subscribe(filter string, sub SubscriberInfo) error {
	if err := ValidateTopicFilter(filter); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	node := t.navigateToNode(filter)

	node.mu.Lock()
	node.subscribers = append(node.subscribers, sub)
	node.mu.Unlock()

	return nil
}

// SubscribeShared adds a shared subscription to the trie
func (t *Trie) SubscribeShared(groupName, filter string, sub SubscriberInfo) error {
	if err := ValidateTopicFilter(filter); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	node := t.navigateToNode(filter)

	node.mu.Lock()
	if node.sharedGroups[groupName] == nil {
		node.sharedGroups[groupName] = NewSharedSubscriptionGroup(groupName)
	}
	node.sharedGroups[groupName].AddSubscriber(sub)
	node.mu.Unlock()

	return nil
}

// navigateToNode traverses the trie to find or create the node for a filter
// Caller must hold t.mu lock
func (t *Trie) navigateToNode(filter string) *trieNode {
	levels := splitTopicLevels(filter)
	node := t.root

	for _, level := range levels {
		node.mu.Lock()
		if node.children[level] == nil {
			node.children[level] = newTrieNode()
		}
		nextNode := node.children[level]

		if level == "+" {
			node.hasSingleLevel = true
		} else if level == "#" {
			node.hasMultiLevel = true
		}
		node.mu.Unlock()

		node = nextNode
	}

	return node
}

// Unsubscribe removes a subscription from the trie
func (t *Trie) Unsubscribe(filter, clientID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	levels := splitTopicLevels(filter)
	return t.unsubscribeRecursive(t.root, levels, clientID, 0)
}

// unsubscribeRecursive removes a subscription recursively
func (t *Trie) unsubscribeRecursive(node *trieNode, levels []string, clientID string, depth int) bool {
	if depth == len(levels) {
		node.mu.Lock()
		defer node.mu.Unlock()

		for i, sub := range node.subscribers {
			if sub.ClientID == clientID {
				node.subscribers = append(node.subscribers[:i], node.subscribers[i+1:]...)
				return true
			}
		}
		return false
	}

	level := levels[depth]
	node.mu.RLock()
	child := node.children[level]
	node.mu.RUnlock()

	if child == nil {
		return false
	}

	found := t.unsubscribeRecursive(child, levels, clientID, depth+1)

	if found && t.shouldPruneNode(child) {
		node.mu.Lock()
		delete(node.children, level)
		node.mu.Unlock()
	}

	return found
}

// UnsubscribeShared removes a shared subscription from the trie
func (t *Trie) UnsubscribeShared(groupName, filter, clientID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	levels := splitTopicLevels(filter)
	return t.unsubscribeSharedRecursive(t.root, levels, groupName, clientID, 0)
}

// unsubscribeSharedRecursive removes a shared subscription recursively
func (t *Trie) unsubscribeSharedRecursive(node *trieNode, levels []string, groupName, clientID string, depth int) bool {
	if depth == len(levels) {
		node.mu.Lock()
		defer node.mu.Unlock()

		group, ok := node.sharedGroups[groupName]
		if !ok {
			return false
		}

		removed := group.RemoveSubscriber(clientID)
		if group.Size() == 0 {
			delete(node.sharedGroups, groupName)
		}
		return removed
	}

	level := levels[depth]
	node.mu.RLock()
	child := node.children[level]
	node.mu.RUnlock()

	if child == nil {
		return false
	}

	found := t.unsubscribeSharedRecursive(child, levels, groupName, clientID, depth+1)

	if found && t.shouldPruneNode(child) {
		node.mu.Lock()
		delete(node.children, level)
		node.mu.Unlock()
	}

	return found
}

// Match finds all subscribers matching a topic
func (t *Trie) Match(topic string) []SubscriberInfo {
	if err := ValidateTopic(topic); err != nil {
		return nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	levels := splitTopicLevels(topic)
	subscribers := make([]SubscriberInfo, 0, 16)
	t.matchRecursive(t.root, levels, 0, &subscribers)
	return subscribers
}

// matchRecursive recursively matches subscribers
func (t *Trie) matchRecursive(node *trieNode, levels []string, depth int, subscribers *[]SubscriberInfo) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	// Check for multi-level wildcard '#'
	if multiNode := node.children["#"]; multiNode != nil {
		multiNode.mu.RLock()
		*subscribers = append(*subscribers, multiNode.subscribers...)
		for _, group := range multiNode.sharedGroups {
			if sub, ok := group.NextSubscriber(); ok {
				*subscribers = append(*subscribers, sub)
			}
		}
		multiNode.mu.RUnlock()
	}

	// If we've consumed all levels, add subscribers at this node
	if depth == len(levels) {
		*subscribers = append(*subscribers, node.subscribers...)
		for _, group := range node.sharedGroups {
			if sub, ok := group.NextSubscriber(); ok {
				*subscribers = append(*subscribers, sub)
			}
		}
		return
	}

	level := levels[depth]

	// Match exact level
	if exactNode := node.children[level]; exactNode != nil {
		t.matchRecursive(exactNode, levels, depth+1, subscribers)
	}

	// Match single-level wildcard '+'
	if plusNode := node.children["+"]; plusNode != nil {
		t.matchRecursive(plusNode, levels, depth+1, subscribers)
	}
}

// shouldPruneNode checks if a node should be removed (has no subscribers or children)
func (t *Trie) shouldPruneNode(node *trieNode) bool {
	node.mu.RLock()
	defer node.mu.RUnlock()

	return len(node.subscribers) == 0 && len(node.children) == 0 && len(node.sharedGroups) == 0
}

// Clear removes all subscriptions from the trie
func (t *Trie) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.root = newTrieNode()
}

// Count returns the total number of subscriptions
func (t *Trie) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.countRecursive(t.root)
}

// countRecursive recursively counts subscriptions
func (t *Trie) countRecursive(node *trieNode) int {
	node.mu.RLock()
	defer node.mu.RUnlock()

	count := len(node.subscribers)
	for _, group := range node.sharedGroups {
		count += group.Size()
	}

	for _, child := range node.children {
		count += t.countRecursive(child)
	}

	return count
}
