package store

import (
	"context"
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
	"github.com/stretchr/testify/assert"
)

type mockTopicMatcher struct{}

func (m *mockTopicMatcher) Match(filter, topic string) bool {
	if filter == topic {
		return true
	}
	if filter == "#" {
		return true
	}
	if filter == "test/+" && (topic == "test/1" || topic == "test/2") {
		return true
	}
	return false
}

func TestRetainedStore_Set(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		msg     *message.Message
		wantErr bool
	}{
		{
			name:  "set retained message",
			topic: "test/topic",
			msg: message.NewMessage(
				1,
				"test/topic",
				[]byte("payload"),
				encoding.QoS1,
				true,
				nil,
			),
			wantErr: false,
		},
		{
			name:  "set message with expiry",
			topic: "test/expiry",
			msg: message.NewMessage(
				2,
				"test/expiry",
				[]byte("expires"),
				encoding.QoS1,
				true,
				map[string]interface{}{"MessageExpiryInterval": uint32(60)},
			),
			wantErr: false,
		},
		{
			name:  "delete retained message with empty payload",
			topic: "test/delete",
			msg: message.NewMessage(
				3,
				"test/delete",
				[]byte{},
				encoding.QoS0,
				true,
				nil,
			),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			ctx := context.Background()
			err := store.Set(ctx, tt.topic, tt.msg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetainedStore_Get(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*RetainedStore)
		topic     string
		wantMsg   bool
		wantErr   bool
		checkData func(*testing.T, *message.Message)
	}{
		{
			name: "get existing message",
			setup: func(s *RetainedStore) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/topic", msg)
			},
			topic:   "test/topic",
			wantMsg: true,
			wantErr: false,
			checkData: func(t *testing.T, msg *message.Message) {
				assert.Equal(t, "test/topic", msg.Topic)
				assert.Equal(t, []byte("data"), msg.Payload)
			},
		},
		{
			name:    "get non-existent message",
			setup:   func(s *RetainedStore) {},
			topic:   "missing/topic",
			wantMsg: false,
			wantErr: true,
		},
		{
			name: "get expired message",
			setup: func(s *RetainedStore) {
				msg := message.NewMessage(
					1,
					"test/expired",
					[]byte("expired"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(1)},
				)
				msg.CreatedAt = time.Now().Add(-2 * time.Second)
				s.Set(context.Background(), "test/expired", msg)
			},
			topic:   "test/expired",
			wantMsg: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			msg, err := store.Get(context.Background(), tt.topic)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantMsg {
				assert.NotNil(t, msg)
				if tt.checkData != nil {
					tt.checkData(t, msg)
				}
			} else {
				assert.Nil(t, msg)
			}
		})
	}
}

func TestRetainedStore_Delete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*RetainedStore)
		topic   string
		wantErr bool
	}{
		{
			name: "delete existing message",
			setup: func(s *RetainedStore) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/topic", msg)
			},
			topic:   "test/topic",
			wantErr: false,
		},
		{
			name:    "delete non-existent message",
			setup:   func(s *RetainedStore) {},
			topic:   "missing/topic",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			err := store.Delete(context.Background(), tt.topic)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			_, err = store.Get(context.Background(), tt.topic)
			assert.Error(t, err)
		})
	}
}

func TestRetainedStore_Match(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*RetainedStore)
		filter     string
		wantCount  int
		wantTopics []string
		wantErr    bool
	}{
		{
			name: "match exact topic",
			setup: func(s *RetainedStore) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/topic", msg)
			},
			filter:     "test/topic",
			wantCount:  1,
			wantTopics: []string{"test/topic"},
			wantErr:    false,
		},
		{
			name: "match wildcard",
			setup: func(s *RetainedStore) {
				msg1 := message.NewMessage(1, "test/1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "test/2", []byte("data2"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/1", msg1)
				s.Set(context.Background(), "test/2", msg2)
			},
			filter:     "test/+",
			wantCount:  2,
			wantTopics: []string{"test/1", "test/2"},
			wantErr:    false,
		},
		{
			name: "match all topics",
			setup: func(s *RetainedStore) {
				msg1 := message.NewMessage(1, "test/1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "test/2", []byte("data2"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/1", msg1)
				s.Set(context.Background(), "test/2", msg2)
			},
			filter:    "#",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "exclude expired messages",
			setup: func(s *RetainedStore) {
				msg1 := message.NewMessage(1, "test/1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(
					2,
					"test/2",
					[]byte("expired"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(1)},
				)
				msg2.CreatedAt = time.Now().Add(-2 * time.Second)
				s.Set(context.Background(), "test/1", msg1)
				s.Set(context.Background(), "test/2", msg2)
			},
			filter:    "#",
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			matcher := &mockTopicMatcher{}
			messages, err := store.Match(context.Background(), tt.filter, matcher)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(messages))

				if len(tt.wantTopics) > 0 {
					topics := make([]string, len(messages))
					for i, msg := range messages {
						topics[i] = msg.Topic
					}
					assert.ElementsMatch(t, tt.wantTopics, topics)
				}
			}
		})
	}
}

func TestRetainedStore_CleanupExpired(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*RetainedStore)
		wantCount int
		wantErr   bool
	}{
		{
			name: "cleanup expired messages",
			setup: func(s *RetainedStore) {
				msg1 := message.NewMessage(
					1,
					"test/expired1",
					[]byte("expired1"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(1)},
				)
				msg1.CreatedAt = time.Now().Add(-2 * time.Second)

				msg2 := message.NewMessage(
					2,
					"test/expired2",
					[]byte("expired2"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(1)},
				)
				msg2.CreatedAt = time.Now().Add(-2 * time.Second)

				msg3 := message.NewMessage(3, "test/valid", []byte("valid"), encoding.QoS1, true, nil)

				s.Set(context.Background(), "test/expired1", msg1)
				s.Set(context.Background(), "test/expired2", msg2)
				s.Set(context.Background(), "test/valid", msg3)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "no expired messages",
			setup: func(s *RetainedStore) {
				msg := message.NewMessage(1, "test/valid", []byte("valid"), encoding.QoS1, true, nil)
				s.Set(context.Background(), "test/valid", msg)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "empty store",
			setup:     func(s *RetainedStore) {},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			count, err := store.CleanupExpired(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}

func TestRetainedStore_Count(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*RetainedStore)
		wantCount int64
		wantErr   bool
	}{
		{
			name: "count messages",
			setup: func(s *RetainedStore) {
				for i := 0; i < 5; i++ {
					msg := message.NewMessage(uint16(i), "test/topic", []byte("data"), encoding.QoS1, true, nil)
					s.Set(context.Background(), "test/topic", msg)
				}
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "empty store",
			setup:     func(s *RetainedStore) {},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			count, err := store.Count(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}

func TestRetainedStore_ContextCancellation(t *testing.T) {
	tests := []struct {
		name string
		op   func(context.Context, *RetainedStore) error
	}{
		{
			name: "set with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				return s.Set(ctx, "test/topic", msg)
			},
		},
		{
			name: "get with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				_, err := s.Get(ctx, "test/topic")
				return err
			},
		},
		{
			name: "delete with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				return s.Delete(ctx, "test/topic")
			},
		},
		{
			name: "match with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				_, err := s.Match(ctx, "#", &mockTopicMatcher{})
				return err
			},
		},
		{
			name: "cleanup with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				_, err := s.CleanupExpired(ctx)
				return err
			},
		},
		{
			name: "count with cancelled context",
			op: func(ctx context.Context, s *RetainedStore) error {
				_, err := s.Count(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			defer store.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := tt.op(ctx, store)
			assert.Error(t, err)
		})
	}
}

func TestRetainedStore_Closed(t *testing.T) {
	tests := []struct {
		name string
		op   func(*RetainedStore) error
	}{
		{
			name: "set on closed store",
			op: func(s *RetainedStore) error {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				return s.Set(context.Background(), "test/topic", msg)
			},
		},
		{
			name: "get on closed store",
			op: func(s *RetainedStore) error {
				_, err := s.Get(context.Background(), "test/topic")
				return err
			},
		},
		{
			name: "delete on closed store",
			op: func(s *RetainedStore) error {
				return s.Delete(context.Background(), "test/topic")
			},
		},
		{
			name: "match on closed store",
			op: func(s *RetainedStore) error {
				_, err := s.Match(context.Background(), "#", &mockTopicMatcher{})
				return err
			},
		},
		{
			name: "cleanup on closed store",
			op: func(s *RetainedStore) error {
				_, err := s.CleanupExpired(context.Background())
				return err
			},
		},
		{
			name: "count on closed store",
			op: func(s *RetainedStore) error {
				_, err := s.Count(context.Background())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRetainedStore()
			store.Close()

			err := tt.op(store)
			assert.ErrorIs(t, err, ErrStoreClosed)
		})
	}
}

func TestRetainedStore_ConcurrentAccess(t *testing.T) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				topic := "test/topic"
				msg := message.NewMessage(uint16(j), topic, []byte("data"), encoding.QoS1, true, nil)

				store.Set(ctx, topic, msg)
				store.Get(ctx, topic)
				store.Match(ctx, "#", &mockTopicMatcher{})
				store.Count(ctx)
				if j%10 == 0 {
					store.Delete(ctx, topic)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestRetainedStore_EmptyPayloadDelete(t *testing.T) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()

	msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
	err := store.Set(ctx, "test/topic", msg)
	assert.NoError(t, err)

	retrieved, err := store.Get(ctx, "test/topic")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	emptyMsg := message.NewMessage(2, "test/topic", []byte{}, encoding.QoS0, true, nil)
	err = store.Set(ctx, "test/topic", emptyMsg)
	assert.NoError(t, err)

	retrieved, err = store.Get(ctx, "test/topic")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}
