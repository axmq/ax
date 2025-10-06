package qos

import (
	"sync"
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name:   "custom config",
			config: DefaultConfig(),
		},
		{
			name: "custom values",
			config: &Config{
				MaxInflight:      100,
				RetryInterval:    2 * time.Second,
				MaxRetries:       3,
				RetryBackoff:     1.5,
				MaxRetryInterval: 30 * time.Second,
				CleanupInterval:  15 * time.Second,
				AckTimeout:       20 * time.Second,
				EnableDedup:      true,
				DedupWindowSize:  500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.config)
			require.NotNil(t, h)
			assert.NotNil(t, h.config)
			assert.NotNil(t, h.qos1Messages)
			assert.NotNil(t, h.qos2Messages)
			assert.NotNil(t, h.qos2Pubrel)
			assert.NotNil(t, h.qos2Received)
			assert.Equal(t, uint16(1), h.nextPacketID)
			assert.Equal(t, 0, h.inflightCount)
			assert.False(t, h.closed)

			err := h.Close()
			assert.NoError(t, err)
		})
	}
}

func TestHandler_HandleQoS0Publish(t *testing.T) {
	tests := []struct {
		name          string
		setupCallback bool
		callbackError error
		wantError     bool
	}{
		{
			name:          "success without callback",
			setupCallback: false,
			callbackError: nil,
			wantError:     false,
		},
		{
			name:          "success with callback",
			setupCallback: true,
			callbackError: nil,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			defer h.Close()

			var callbackCalled bool
			if tt.setupCallback {
				h.SetPublishCallback(func(msg *message.Message) error {
					callbackCalled = true
					return tt.callbackError
				})
			}

			msg := message.NewMessage(0, "test/topic", []byte("payload"), encoding.QoS0, false, nil)
			err := h.HandlePublish(msg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.setupCallback {
				assert.True(t, callbackCalled)
			}
		})
	}
}

func TestHandler_HandleQoS1Publish(t *testing.T) {
	tests := []struct {
		name          string
		setupCallback bool
		enableDedup   bool
		duplicate     bool
		wantError     bool
	}{
		{
			name:          "first publish",
			setupCallback: true,
			enableDedup:   false,
			duplicate:     false,
			wantError:     false,
		},
		{
			name:          "duplicate publish with dedup",
			setupCallback: true,
			enableDedup:   true,
			duplicate:     true,
			wantError:     false,
		},
		{
			name:          "without callback",
			setupCallback: false,
			enableDedup:   false,
			duplicate:     false,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.EnableDedup = tt.enableDedup
			h := NewHandler(config)
			defer h.Close()

			var callbackCount int
			if tt.setupCallback {
				h.SetPublishCallback(func(msg *message.Message) error {
					callbackCount++
					return nil
				})
			}

			var pubackCount int
			h.SetPubackCallback(func(packetID uint16) error {
				pubackCount++
				return nil
			})

			msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)

			if tt.duplicate {
				err := h.HandlePublish(msg)
				assert.NoError(t, err)
			}

			err := h.HandlePublish(msg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Greater(t, pubackCount, 0)
		})
	}
}

func TestHandler_HandleQoS2Publish(t *testing.T) {
	tests := []struct {
		name          string
		setupCallback bool
		enableDedup   bool
		duplicate     bool
		wantError     bool
	}{
		{
			name:          "first publish",
			setupCallback: true,
			enableDedup:   false,
			duplicate:     false,
			wantError:     false,
		},
		{
			name:          "duplicate publish with dedup",
			setupCallback: true,
			enableDedup:   true,
			duplicate:     true,
			wantError:     false,
		},
		{
			name:          "without callback",
			setupCallback: false,
			enableDedup:   false,
			duplicate:     false,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.EnableDedup = tt.enableDedup
			h := NewHandler(config)
			defer h.Close()

			var callbackCount int
			if tt.setupCallback {
				h.SetPublishCallback(func(msg *message.Message) error {
					callbackCount++
					return nil
				})
			}

			var pubrecCount int
			h.SetPubrecCallback(func(packetID uint16) error {
				pubrecCount++
				return nil
			})

			msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS2, false, nil)

			if tt.duplicate {
				err := h.HandlePublish(msg)
				assert.NoError(t, err)
			}

			err := h.HandlePublish(msg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Greater(t, pubrecCount, 0)
		})
	}
}

func TestHandler_PublishQoS1(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		payload       []byte
		retain        bool
		properties    map[string]interface{}
		setupCallback bool
		wantError     bool
	}{
		{
			name:          "simple publish",
			topic:         "test/topic",
			payload:       []byte("payload"),
			retain:        false,
			properties:    nil,
			setupCallback: true,
			wantError:     false,
		},
		{
			name:    "publish with properties",
			topic:   "test/topic",
			payload: []byte("payload"),
			retain:  true,
			properties: map[string]interface{}{
				"MessageExpiryInterval": uint32(60),
			},
			setupCallback: true,
			wantError:     false,
		},
		{
			name:          "empty payload",
			topic:         "test/topic",
			payload:       []byte{},
			retain:        false,
			properties:    nil,
			setupCallback: true,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			defer h.Close()

			var callbackCalled bool
			if tt.setupCallback {
				h.SetPublishCallback(func(msg *message.Message) error {
					callbackCalled = true
					assert.Equal(t, tt.topic, msg.Topic)
					assert.Equal(t, tt.payload, msg.Payload)
					assert.Equal(t, encoding.QoS1, msg.QoS)
					assert.Equal(t, tt.retain, msg.Retain)
					return nil
				})
			}

			packetID, err := h.PublishQoS1(tt.topic, tt.payload, tt.retain, tt.properties)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uint16(0), packetID)
				assert.Equal(t, 1, h.GetInflightCount())
				assert.Equal(t, 1, h.GetPendingQoS1Count())
			}

			if tt.setupCallback {
				assert.True(t, callbackCalled)
			}
		})
	}
}

func TestHandler_PublishQoS2(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		payload       []byte
		retain        bool
		properties    map[string]interface{}
		setupCallback bool
		wantError     bool
	}{
		{
			name:          "simple publish",
			topic:         "test/topic",
			payload:       []byte("payload"),
			retain:        false,
			properties:    nil,
			setupCallback: true,
			wantError:     false,
		},
		{
			name:    "publish with properties",
			topic:   "test/topic",
			payload: []byte("payload"),
			retain:  true,
			properties: map[string]interface{}{
				"MessageExpiryInterval": uint32(60),
			},
			setupCallback: true,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			defer h.Close()

			var callbackCalled bool
			if tt.setupCallback {
				h.SetPublishCallback(func(msg *message.Message) error {
					callbackCalled = true
					assert.Equal(t, tt.topic, msg.Topic)
					assert.Equal(t, tt.payload, msg.Payload)
					assert.Equal(t, encoding.QoS2, msg.QoS)
					assert.Equal(t, tt.retain, msg.Retain)
					return nil
				})
			}

			packetID, err := h.PublishQoS2(tt.topic, tt.payload, tt.retain, tt.properties)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uint16(0), packetID)
				assert.Equal(t, 1, h.GetInflightCount())
				assert.Equal(t, 1, h.GetPendingQoS2Count())
			}

			if tt.setupCallback {
				assert.True(t, callbackCalled)
			}
		})
	}
}

func TestHandler_HandlePuback(t *testing.T) {
	tests := []struct {
		name          string
		setupMessage  bool
		setupCallback bool
		wantError     bool
	}{
		{
			name:          "success",
			setupMessage:  true,
			setupCallback: true,
			wantError:     false,
		},
		{
			name:          "packet not found",
			setupMessage:  false,
			setupCallback: false,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			defer h.Close()

			var callbackCalled bool
			if tt.setupCallback {
				h.SetPubackCallback(func(packetID uint16) error {
					callbackCalled = true
					return nil
				})
			}

			var packetID uint16 = 1

			if tt.setupMessage {
				h.SetPublishCallback(func(msg *message.Message) error { return nil })
				packetID, _ = h.PublishQoS1("test/topic", []byte("payload"), false, nil)
			}

			err := h.HandlePuback(packetID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 0, h.GetInflightCount())
			}

			if tt.setupCallback && !tt.wantError {
				assert.True(t, callbackCalled)
			}
		})
	}
}

func TestHandler_QoS2Flow(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	var publishCalled, pubrecCalled, pubcompCalled bool

	h.SetPublishCallback(func(msg *message.Message) error {
		publishCalled = true
		return nil
	})

	h.SetPubrecCallback(func(packetID uint16) error {
		pubrecCalled = true
		return nil
	})

	h.SetPubrelCallback(func(packetID uint16) error {
		return nil
	})

	h.SetPubcompCallback(func(packetID uint16) error {
		pubcompCalled = true
		return nil
	})

	packetID, err := h.PublishQoS2("test/topic", []byte("payload"), false, nil)
	require.NoError(t, err)
	assert.NotEqual(t, uint16(0), packetID)
	assert.True(t, publishCalled)

	err = h.HandlePubrec(packetID)
	require.NoError(t, err)
	assert.True(t, pubrecCalled)

	err = h.HandlePubcomp(packetID)
	require.NoError(t, err)
	assert.True(t, pubcompCalled)
	assert.Equal(t, 0, h.GetInflightCount())
}

func TestHandler_QoS2InboundFlow(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	var publishCalled, pubrecCalled, pubrelCalled, pubcompCalled bool

	h.SetPublishCallback(func(msg *message.Message) error {
		publishCalled = true
		return nil
	})

	h.SetPubrecCallback(func(packetID uint16) error {
		pubrecCalled = true
		return nil
	})

	h.SetPubrelCallback(func(packetID uint16) error {
		pubrelCalled = true
		return nil
	})

	h.SetPubcompCallback(func(packetID uint16) error {
		pubcompCalled = true
		return nil
	})

	msg := message.NewMessage(100, "test/topic", []byte("payload"), encoding.QoS2, false, nil)

	err := h.HandlePublish(msg)
	require.NoError(t, err)
	assert.True(t, publishCalled)
	assert.True(t, pubrecCalled)

	err = h.HandlePubrel(100)
	require.NoError(t, err)
	assert.True(t, pubrelCalled)
	assert.True(t, pubcompCalled)
}

func TestHandler_HandleExpiredMessage(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(1),
	}

	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)
	msg.CreatedAt = time.Now().Add(-2 * time.Second)

	err := h.HandlePublish(msg)
	assert.Error(t, err)
	assert.Equal(t, ErrMessageExpired, err)
}

func TestHandler_PublishExpiredMessage(t *testing.T) {
	config := DefaultConfig()
	h := NewHandler(config)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(1),
	}

	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)
	msg.CreatedAt = time.Now().Add(-2 * time.Second)

	packetID, err := h.PublishQoS1(msg.Topic, msg.Payload, msg.Retain, properties)
	if err == nil {
		_ = h.HandlePuback(packetID)
	}

	time.Sleep(100 * time.Millisecond)
	h.cleanup()

	assert.Equal(t, 0, h.GetPendingQoS1Count())
}

func TestHandler_MaxInflight(t *testing.T) {
	config := DefaultConfig()
	config.MaxInflight = 2
	h := NewHandler(config)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	_, err := h.PublishQoS1("test/topic1", []byte("payload1"), false, nil)
	require.NoError(t, err)

	_, err = h.PublishQoS1("test/topic2", []byte("payload2"), false, nil)
	require.NoError(t, err)

	_, err = h.PublishQoS1("test/topic3", []byte("payload3"), false, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrQueueFull, err)
}

func TestHandler_PacketIDAllocation(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	usedIDs := make(map[uint16]bool)

	for i := 0; i < 100; i++ {
		packetID, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
		require.NoError(t, err)
		assert.False(t, usedIDs[packetID], "packet ID %d already used", packetID)
		usedIDs[packetID] = true
	}
}

func TestHandler_PacketIDRollover(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.nextPacketID = 65535

	packetID1, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	require.NoError(t, err)
	assert.Equal(t, uint16(65535), packetID1)

	packetID2, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), packetID2)
}

func TestHandler_RetryLogic(t *testing.T) {
	config := DefaultConfig()
	config.RetryInterval = 100 * time.Millisecond
	config.MaxRetries = 2
	h := NewHandler(config)
	defer h.Close()

	var attemptCount int
	var mu sync.Mutex

	h.SetPublishCallback(func(msg *message.Message) error {
		mu.Lock()
		attemptCount++
		mu.Unlock()
		return nil
	})

	_, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := attemptCount
	mu.Unlock()

	assert.GreaterOrEqual(t, count, 2)
}

func TestHandler_MaxRetryCallback(t *testing.T) {
	config := DefaultConfig()
	config.RetryInterval = 50 * time.Millisecond
	config.MaxRetries = 2
	h := NewHandler(config)
	defer h.Close()

	var maxRetryCalled bool
	var mu sync.Mutex

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetMaxRetryCallback(func(msg *message.Message) {
		mu.Lock()
		maxRetryCalled = true
		mu.Unlock()
	})

	_, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	require.NoError(t, err)

	time.Sleep(400 * time.Millisecond)

	mu.Lock()
	called := maxRetryCalled
	mu.Unlock()

	assert.True(t, called)
}

func TestHandler_ExpiredCallback(t *testing.T) {
	config := DefaultConfig()
	config.CleanupInterval = 50 * time.Millisecond
	h := NewHandler(config)
	defer h.Close()

	var expiredCalled bool
	var mu sync.Mutex

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetExpiredCallback(func(msg *message.Message) {
		mu.Lock()
		expiredCalled = true
		mu.Unlock()
	})

	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(1),
	}

	_, err := h.PublishQoS1("test/topic", []byte("payload"), false, properties)
	require.NoError(t, err)

	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	called := expiredCalled
	mu.Unlock()

	assert.True(t, called)
}

func TestHandler_ClosedHandler(t *testing.T) {
	h := NewHandler(nil)
	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	err := h.Close()
	require.NoError(t, err)

	_, err = h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrHandlerClosed, err)

	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)
	err = h.HandlePublish(msg)
	assert.Error(t, err)
	assert.Equal(t, ErrHandlerClosed, err)
}

func TestHandler_DoubleClose(t *testing.T) {
	h := NewHandler(nil)

	err := h.Close()
	assert.NoError(t, err)

	err = h.Close()
	assert.NoError(t, err)
}

func TestHandler_ConcurrentPublish(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.PublishQoS1("test/topic", []byte("payload"), false, nil)
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, h.GetInflightCount())
}

func TestHandler_ConcurrentHandlePuback(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	packetIDs := make([]uint16, 100)
	for i := 0; i < 100; i++ {
		packetID, err := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
		require.NoError(t, err)
		packetIDs[i] = packetID
	}

	var wg sync.WaitGroup
	for _, packetID := range packetIDs {
		wg.Add(1)
		go func(pid uint16) {
			defer wg.Done()
			_ = h.HandlePuback(pid)
		}(packetID)
	}

	wg.Wait()
	assert.Equal(t, 0, h.GetInflightCount())
}

func TestHandler_InvalidQoS(t *testing.T) {
	h := NewHandler(nil)
	defer h.Close()

	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS(3), false, nil)
	err := h.HandlePublish(msg)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidQoS, err)
}

func TestHandler_ExponentialBackoff(t *testing.T) {
	config := DefaultConfig()
	config.RetryInterval = 1 * time.Second
	config.RetryBackoff = 2.0
	config.MaxRetryInterval = 10 * time.Second
	h := NewHandler(config)
	defer h.Close()

	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{attempt: 0, wantMin: 1 * time.Second, wantMax: 1 * time.Second},
		{attempt: 1, wantMin: 1 * time.Second, wantMax: 1 * time.Second},
		{attempt: 2, wantMin: 2 * time.Second, wantMax: 2 * time.Second},
		{attempt: 3, wantMin: 4 * time.Second, wantMax: 4 * time.Second},
		{attempt: 4, wantMin: 8 * time.Second, wantMax: 8 * time.Second},
		{attempt: 5, wantMin: 10 * time.Second, wantMax: 10 * time.Second},
	}

	for _, tt := range tests {
		interval := h.calculateRetryInterval(tt.attempt)
		assert.GreaterOrEqual(t, interval, tt.wantMin)
		assert.LessOrEqual(t, interval, tt.wantMax)
	}
}
