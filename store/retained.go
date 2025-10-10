package store

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/axmq/ax/types/message"
)

type RetainedMessage struct {
	Message   *message.Message
	ExpiresAt time.Time
}

// retainedTrieNode represents a node in the retained messages trie
type retainedTrieNode struct {
	children map[string]*retainedTrieNode
	message  *RetainedMessage
	mu       sync.RWMutex
}

// newRetainedTrieNode creates a new trie node
func newRetainedTrieNode() *retainedTrieNode {
	return &retainedTrieNode{
		children: make(map[string]*retainedTrieNode),
	}
}

type RetainedStore struct {
	mu     sync.RWMutex
	root   *retainedTrieNode
	count  int64
	closed bool
}

func NewRetainedStore() *RetainedStore {
	return &RetainedStore{
		root: newRetainedTrieNode(),
	}
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

func (r *RetainedStore) Set(ctx context.Context, topic string, msg *message.Message) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	// Delete message if payload is empty
	if len(msg.Payload) == 0 {
		return r.deleteInternal(topic)
	}

	retained := &RetainedMessage{
		Message: msg,
	}

	if msg.MessageExpirySet && msg.ExpiryInterval > 0 {
		retained.ExpiresAt = msg.CreatedAt.Add(time.Duration(msg.ExpiryInterval) * time.Second)
	}

	levels := splitTopicLevels(topic)
	node := r.root

	// Navigate/create path to the topic
	for _, level := range levels {
		node.mu.Lock()
		if node.children[level] == nil {
			node.children[level] = newRetainedTrieNode()
		}
		nextNode := node.children[level]
		node.mu.Unlock()
		node = nextNode
	}

	node.mu.Lock()
	// Increment count only if this is a new message
	if node.message == nil {
		r.count++
	}
	node.message = retained
	node.mu.Unlock()

	return nil
}

func (r *RetainedStore) Get(ctx context.Context, topic string) (*message.Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, ErrStoreClosed
	}

	levels := splitTopicLevels(topic)
	node := r.root

	// Navigate to the topic
	for _, level := range levels {
		node.mu.RLock()
		nextNode := node.children[level]
		node.mu.RUnlock()

		if nextNode == nil {
			return nil, ErrNotFound
		}
		node = nextNode
	}

	node.mu.RLock()
	retained := node.message
	node.mu.RUnlock()

	if retained == nil {
		return nil, ErrNotFound
	}

	if !retained.ExpiresAt.IsZero() && time.Now().After(retained.ExpiresAt) {
		return nil, ErrNotFound
	}

	return retained.Message, nil
}

func (r *RetainedStore) Delete(ctx context.Context, topic string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	return r.deleteInternal(topic)
}

// deleteInternal removes a retained message from the trie
// Caller must hold r.mu lock
func (r *RetainedStore) deleteInternal(topic string) error {
	levels := splitTopicLevels(topic)
	if len(levels) == 0 {
		return nil
	}

	// Navigate to parent and track path for cleanup
	path := make([]*retainedTrieNode, 0, len(levels)+1)
	path = append(path, r.root)
	node := r.root

	for _, level := range levels {
		node.mu.RLock()
		nextNode := node.children[level]
		node.mu.RUnlock()

		if nextNode == nil {
			return nil
		}
		path = append(path, nextNode)
		node = nextNode
	}

	// Remove message at the leaf
	node.mu.Lock()
	if node.message != nil {
		node.message = nil
		r.count--
	}
	node.mu.Unlock()

	// Prune empty nodes from leaf to root
	for i := len(path) - 1; i > 0; i-- {
		current := path[i]
		parent := path[i-1]

		current.mu.RLock()
		isEmpty := current.message == nil && len(current.children) == 0
		current.mu.RUnlock()

		if !isEmpty {
			break
		}

		// Remove from parent
		parent.mu.Lock()
		for key, child := range parent.children {
			if child == current {
				delete(parent.children, key)
				break
			}
		}
		parent.mu.Unlock()
	}

	return nil
}

func (r *RetainedStore) Match(ctx context.Context, topicFilter string, matcher TopicMatcher) ([]*message.Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, ErrStoreClosed
	}

	// For system topics ($SYS/...), don't match wildcards
	if strings.HasPrefix(topicFilter, "$") {
		if strings.Contains(topicFilter, "#") || strings.Contains(topicFilter, "+") {
			return nil, nil
		}
	}

	filterLevels := splitTopicLevels(topicFilter)
	var matched []*message.Message
	now := time.Now()

	r.matchRecursive(r.root, filterLevels, 0, "", &matched, now)

	return matched, nil
}

// matchRecursive performs trie-based recursive matching
func (r *RetainedStore) matchRecursive(node *retainedTrieNode, filterLevels []string, depth int, currentTopic string, matched *[]*message.Message, now time.Time) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	// If we've consumed all filter levels
	if depth == len(filterLevels) {
		if node.message != nil {
			// Check if message is expired
			if node.message.ExpiresAt.IsZero() || now.Before(node.message.ExpiresAt) {
				*matched = append(*matched, node.message.Message)
			}
		}
		return
	}

	filterLevel := filterLevels[depth]

	// Multi-level wildcard '#' matches everything from this point
	if filterLevel == "#" {
		r.collectAllMessages(node, matched, now)
		return
	}

	// Single-level wildcard '+' matches any single level
	if filterLevel == "+" {
		for levelName, child := range node.children {
			// Skip system topics if filter doesn't start with $
			if depth == 0 && strings.HasPrefix(levelName, "$") {
				continue
			}
			r.matchRecursive(child, filterLevels, depth+1, currentTopic+levelName+"/", matched, now)
		}
		return
	}

	// Exact match
	if child := node.children[filterLevel]; child != nil {
		r.matchRecursive(child, filterLevels, depth+1, currentTopic+filterLevel+"/", matched, now)
	}
}

// collectAllMessages recursively collects all messages from a node and its descendants
func (r *RetainedStore) collectAllMessages(node *retainedTrieNode, matched *[]*message.Message, now time.Time) {
	// Note: node.mu should already be held by caller

	if node.message != nil {
		// Check if message is expired
		if node.message.ExpiresAt.IsZero() || now.Before(node.message.ExpiresAt) {
			*matched = append(*matched, node.message.Message)
		}
	}

	for _, child := range node.children {
		child.mu.RLock()
		r.collectAllMessages(child, matched, now)
		child.mu.RUnlock()
	}
}

func (r *RetainedStore) CleanupExpired(ctx context.Context) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, ErrStoreClosed
	}

	count := 0
	now := time.Now()

	r.cleanupExpiredRecursive(r.root, now, &count)

	return count, nil
}

// cleanupExpiredRecursive recursively removes expired messages
func (r *RetainedStore) cleanupExpiredRecursive(node *retainedTrieNode, now time.Time, count *int) {
	node.mu.Lock()

	if node.message != nil && !node.message.ExpiresAt.IsZero() && now.After(node.message.ExpiresAt) {
		node.message = nil
		*count++
		r.count--
	}

	children := make([]*retainedTrieNode, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}
	node.mu.Unlock()

	for _, child := range children {
		r.cleanupExpiredRecursive(child, now, count)
	}
}

func (r *RetainedStore) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return 0, ErrStoreClosed
	}

	return r.count, nil
}

func (r *RetainedStore) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	r.closed = true
	r.root = nil
	r.count = 0
	return nil
}

type TopicMatcher interface {
	Match(filter, topic string) bool
}
