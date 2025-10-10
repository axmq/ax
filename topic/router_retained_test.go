package topic

import (
	"context"
	"testing"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
	"github.com/stretchr/testify/assert"
)

func TestRouter_RetainedMessages(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Router)
		test    func(*testing.T, *Router)
		wantErr bool
	}{
		{
			name:  "set and get retained message",
			setup: func(r *Router) {},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()
				msg := message.NewMessage(1, "test/topic", []byte("retained data"), encoding.QoS1, true, nil)
				retained := &RetainedMessage{Topic: "test/topic", Message: msg}

				err := r.SetRetainedMessage(ctx, retained)
				assert.NoError(t, err)

				messages, err := r.GetRetainedMessages(ctx, "test/topic")
				assert.NoError(t, err)
				assert.Len(t, messages, 1)
				assert.Equal(t, "test/topic", messages[0].Topic)
				assert.Equal(t, []byte("retained data"), messages[0].Message.Payload)
			},
		},
		{
			name: "get retained messages with wildcard filter",
			setup: func(r *Router) {
				ctx := context.Background()
				msg1 := message.NewMessage(1, "home/room1/temp", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "home/room2/temp", []byte("data2"), encoding.QoS1, true, nil)
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "home/room1/temp", Message: msg1})
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "home/room2/temp", Message: msg2})
			},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()
				messages, err := r.GetRetainedMessages(ctx, "home/+/temp")
				assert.NoError(t, err)
				assert.Len(t, messages, 2)
			},
		},
		{
			name: "delete retained message with empty payload",
			setup: func(r *Router) {
				ctx := context.Background()
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/topic", Message: msg})
			},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()

				messages, err := r.GetRetainedMessages(ctx, "test/topic")
				assert.NoError(t, err)
				assert.Len(t, messages, 1)

				emptyMsg := message.NewMessage(2, "test/topic", []byte{}, encoding.QoS0, true, nil)
				err = r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/topic", Message: emptyMsg})
				assert.NoError(t, err)

				messages, err = r.GetRetainedMessages(ctx, "test/topic")
				assert.NoError(t, err)
				assert.Len(t, messages, 0)
			},
		},
		{
			name: "delete retained message explicitly",
			setup: func(r *Router) {
				ctx := context.Background()
				msg := message.NewMessage(1, "test/topic", []byte("data"), encoding.QoS1, true, nil)
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/topic", Message: msg})
			},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()

				err := r.DeleteRetainedMessage(ctx, "test/topic")
				assert.NoError(t, err)

				messages, err := r.GetRetainedMessages(ctx, "test/topic")
				assert.NoError(t, err)
				assert.Len(t, messages, 0)
			},
		},
		{
			name: "count retained messages",
			setup: func(r *Router) {
				ctx := context.Background()
				for i := 0; i < 5; i++ {
					msg := message.NewMessage(uint16(i), "test/topic", []byte("data"), encoding.QoS1, true, nil)
					r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/topic", Message: msg})
				}
			},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()
				count, err := r.RetainedMessageCount(ctx)
				assert.NoError(t, err)
				assert.Equal(t, int64(1), count)
			},
		},
		{
			name: "multiple topics retained messages",
			setup: func(r *Router) {
				ctx := context.Background()
				msg1 := message.NewMessage(1, "topic1", []byte("data1"), encoding.QoS1, true, nil)
				msg2 := message.NewMessage(2, "topic2", []byte("data2"), encoding.QoS1, true, nil)
				msg3 := message.NewMessage(3, "topic3", []byte("data3"), encoding.QoS1, true, nil)
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "topic1", Message: msg1})
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "topic2", Message: msg2})
				r.SetRetainedMessage(ctx, &RetainedMessage{Topic: "topic3", Message: msg3})
			},
			test: func(t *testing.T, r *Router) {
				ctx := context.Background()
				count, err := r.RetainedMessageCount(ctx)
				assert.NoError(t, err)
				assert.Equal(t, int64(3), count)

				messages, err := r.GetRetainedMessages(ctx, "#")
				assert.NoError(t, err)
				assert.Len(t, messages, 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			defer router.Close()

			if tt.setup != nil {
				tt.setup(router)
			}

			if tt.test != nil {
				tt.test(t, router)
			}
		})
	}
}

func TestRouter_RetainedMessagesWithSubscription(t *testing.T) {
	router := NewRouter()
	defer router.Close()

	ctx := context.Background()

	msg := message.NewMessage(1, "home/temperature", []byte("25.5"), encoding.QoS1, true, nil)
	err := router.SetRetainedMessage(ctx, &RetainedMessage{Topic: "home/temperature", Message: msg})
	assert.NoError(t, err)

	sub := &Subscription{
		ClientID:          "client1",
		TopicFilter:       "home/+",
		QoS:               1,
		RetainHandling:    0,
		RetainAsPublished: true,
	}
	err = router.Subscribe(sub)
	assert.NoError(t, err)

	messages, err := router.GetRetainedMessages(ctx, "home/+")
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "home/temperature", messages[0].Topic)
}

func TestRouter_RetainedMessagesWithExpiry(t *testing.T) {
	router := NewRouter()
	defer router.Close()

	ctx := context.Background()

	msg := message.NewMessage(
		1,
		"test/expiry",
		[]byte("expires soon"),
		encoding.QoS1,
		true,
		map[string]interface{}{"MessageExpiryInterval": uint32(60)},
	)
	err := router.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/expiry", Message: msg})
	assert.NoError(t, err)

	messages, err := router.GetRetainedMessages(ctx, "test/expiry")
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.True(t, messages[0].Message.MessageExpirySet)
	assert.Equal(t, uint32(60), messages[0].Message.ExpiryInterval)
}

func TestRouter_ConcurrentRetainedOperations(t *testing.T) {
	router := NewRouter()
	defer router.Close()

	ctx := context.Background()
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				msg := message.NewMessage(uint16(j), "test/topic", []byte("data"), encoding.QoS1, true, nil)
				router.SetRetainedMessage(ctx, &RetainedMessage{Topic: "test/topic", Message: msg})
				router.GetRetainedMessages(ctx, "test/topic")
				router.RetainedMessageCount(ctx)
				if j%10 == 0 {
					router.DeleteRetainedMessage(ctx, "test/topic")
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
