package store

import (
	"context"
	"sync"
	"time"

	"github.com/axmq/ax/types/message"
)

type RetainedMessage struct {
	Message   *message.Message
	ExpiresAt time.Time
}

type RetainedStore struct {
	mu       sync.RWMutex
	messages map[string]*RetainedMessage
	closed   bool
}

func NewRetainedStore() *RetainedStore {
	return &RetainedStore{
		messages: make(map[string]*RetainedMessage),
	}
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

	if len(msg.Payload) == 0 {
		delete(r.messages, topic)
		return nil
	}

	retained := &RetainedMessage{
		Message: msg,
	}

	if msg.MessageExpirySet && msg.ExpiryInterval > 0 {
		retained.ExpiresAt = msg.CreatedAt.Add(time.Duration(msg.ExpiryInterval) * time.Second)
	}

	r.messages[topic] = retained
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

	retained, ok := r.messages[topic]
	if !ok {
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

	delete(r.messages, topic)
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

	var matched []*message.Message
	now := time.Now()

	for topic, retained := range r.messages {
		if !retained.ExpiresAt.IsZero() && now.After(retained.ExpiresAt) {
			continue
		}

		if matcher.Match(topicFilter, topic) {
			matched = append(matched, retained.Message)
		}
	}

	return matched, nil
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

	for topic, retained := range r.messages {
		if !retained.ExpiresAt.IsZero() && now.After(retained.ExpiresAt) {
			delete(r.messages, topic)
			count++
		}
	}

	return count, nil
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

	return int64(len(r.messages)), nil
}

func (r *RetainedStore) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStoreClosed
	}

	r.closed = true
	r.messages = nil
	return nil
}

type TopicMatcher interface {
	Match(filter, topic string) bool
}
