package topic

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
	"github.com/stretchr/testify/assert"
)

type mockMatcher struct{}

func (m *mockMatcher) Match(filter, topic string) bool {
	if filter == "#" {
		return true
	}
	return filter == topic
}

func TestNewRetainedManager(t *testing.T) {
	tests := []struct {
		name   string
		config *RetainedConfig
	}{
		{
			name:   "with default config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &RetainedConfig{
				CleanupInterval: 1 * time.Minute,
			},
		},
		{
			name: "with zero cleanup interval",
			config: &RetainedConfig{
				CleanupInterval: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRetainedManager(tt.config)
			assert.NotNil(t, rm)
			assert.NotNil(t, rm.store)
			assert.NotNil(t, rm.cleanupTicker)
			rm.Close()
		})
	}
}

func TestRetainedManager_Set(t *testing.T) {
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
			name:  "set with expiry",
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
			name:  "delete with empty payload",
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
			rm := NewRetainedManager(nil)
			defer rm.Close()

			ctx := context.Background()
			err := rm.Set(ctx, tt.topic, tt.msg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetainedManager_Get(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*RetainedManager)
		topic   string
		wantMsg bool
		wantErr bool
	}{
		{
			name: "get existing message",
			setup: func(rm *RetainedManager) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/topic", msg)
			},
			topic:   "test/topic",
			wantMsg: true,
			wantErr: false,
		},
		{
			name:    "get non-existent message",
			setup:   func(rm *RetainedManager) {},
			topic:   "missing/topic",
			wantMsg: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRetainedManager(nil)
			defer rm.Close()

			if tt.setup != nil {
				tt.setup(rm)
			}

			msg, err := rm.Get(context.Background(), tt.topic)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantMsg {
				assert.NotNil(t, msg)
			} else {
				assert.Nil(t, msg)
			}
		})
	}
}

func TestRetainedManager_Delete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*RetainedManager)
		topic   string
		wantErr bool
	}{
		{
			name: "delete existing message",
			setup: func(rm *RetainedManager) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/topic", msg)
			},
			topic:   "test/topic",
			wantErr: false,
		},
		{
			name:    "delete non-existent message",
			setup:   func(rm *RetainedManager) {},
			topic:   "missing/topic",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRetainedManager(nil)
			defer rm.Close()

			if tt.setup != nil {
				tt.setup(rm)
			}

			err := rm.Delete(context.Background(), tt.topic)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetainedManager_Match(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*RetainedManager)
		filter    string
		wantCount int
		wantErr   bool
	}{
		{
			name: "match exact topic",
			setup: func(rm *RetainedManager) {
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/topic", msg)
			},
			filter:    "test/topic",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "match all topics",
			setup: func(rm *RetainedManager) {
				msg1 := message.NewMessage(1, "test/1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "test/2", []byte("data2"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/1", msg1)
				rm.Set(context.Background(), "test/2", msg2)
			},
			filter:    "#",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no matches",
			setup:     func(rm *RetainedManager) {},
			filter:    "test/topic",
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRetainedManager(nil)
			defer rm.Close()

			if tt.setup != nil {
				tt.setup(rm)
			}

			matcher := &mockMatcher{}
			messages, err := rm.Match(context.Background(), tt.filter, matcher)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(messages))
			}
		})
	}
}

func TestRetainedManager_Count(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*RetainedManager)
		wantCount int64
		wantErr   bool
	}{
		{
			name: "count messages",
			setup: func(rm *RetainedManager) {
				msg1 := message.NewMessage(1, "test/1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "test/2", []byte("data2"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/1", msg1)
				rm.Set(context.Background(), "test/2", msg2)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "empty store",
			setup:     func(rm *RetainedManager) {},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRetainedManager(nil)
			defer rm.Close()

			if tt.setup != nil {
				tt.setup(rm)
			}

			count, err := rm.Count(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}

func TestRetainedManager_CleanupLoop(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(*RetainedManager)
		cleanupInterval time.Duration
		waitTime        time.Duration
		wantCleanup     bool
	}{
		{
			name: "cleanup expired messages",
			setup: func(rm *RetainedManager) {
				msg := message.NewMessage(
					1,
					"test/expired",
					[]byte("expired"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(1)},
				)
				msg.CreatedAt = time.Now().Add(-2 * time.Second)
				rm.Set(context.Background(), "test/expired", msg)
			},
			cleanupInterval: 100 * time.Millisecond,
			waitTime:        200 * time.Millisecond,
			wantCleanup:     true,
		},
		{
			name: "no expired messages",
			setup: func(rm *RetainedManager) {
				msg := message.NewMessage(1, "test/valid", []byte("valid"), encoding.QoS1, true, nil)
				rm.Set(context.Background(), "test/valid", msg)
			},
			cleanupInterval: 100 * time.Millisecond,
			waitTime:        200 * time.Millisecond,
			wantCleanup:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanupCount atomic.Int32

			config := &RetainedConfig{
				CleanupInterval: tt.cleanupInterval,
				OnCleanup: func(count int) {
					cleanupCount.Add(int32(count))
				},
			}

			rm := NewRetainedManager(config)
			defer rm.Close()

			if tt.setup != nil {
				tt.setup(rm)
			}

			time.Sleep(tt.waitTime)

			if tt.wantCleanup {
				assert.Greater(t, cleanupCount.Load(), int32(0))
			}
		})
	}
}

func TestRetainedManager_ConcurrentOperations(t *testing.T) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				topic := "test/topic"
				msg := message.NewMessage(uint16(j), topic, []byte("data"), encoding.QoS1, true, nil)

				rm.Set(ctx, topic, msg)
				rm.Get(ctx, topic)
				rm.Match(ctx, "#", &mockMatcher{})
				rm.Count(ctx)
				if j%10 == 0 {
					rm.Delete(ctx, topic)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestRetainedManager_Close(t *testing.T) {
	rm := NewRetainedManager(nil)

	msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
	err := rm.Set(context.Background(), "test/topic", msg)
	assert.NoError(t, err)

	err = rm.Close()
	assert.NoError(t, err)
}
