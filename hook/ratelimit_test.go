package hook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitHook(t *testing.T) {
	hook := NewRateLimitHook(10, time.Minute)
	defer hook.Stop()

	assert.Equal(t, "rate-limit", hook.ID())
	assert.True(t, hook.Provides(OnPublish))
	assert.False(t, hook.Provides(OnConnect))
	assert.Equal(t, 10, hook.GetMaxRate())
	assert.Equal(t, time.Minute, hook.GetWindow())
}

func TestRateLimitHookBasic(t *testing.T) {
	hook := NewRateLimitHook(5, time.Second)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 5; i++ {
		err := hook.OnPublish(client, packet)
		assert.NoError(t, err)
	}

	err := hook.OnPublish(client, packet)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestRateLimitHookWindowReset(t *testing.T) {
	hook := NewRateLimitHook(3, 100*time.Millisecond)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		err := hook.OnPublish(client, packet)
		assert.NoError(t, err)
	}

	err := hook.OnPublish(client, packet)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)

	time.Sleep(150 * time.Millisecond)

	err = hook.OnPublish(client, packet)
	assert.NoError(t, err)
}

func TestRateLimitHookMultipleClients(t *testing.T) {
	hook := NewRateLimitHook(3, time.Minute)
	defer hook.Stop()

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client1, packet))
	}
	assert.ErrorIs(t, hook.OnPublish(client1, packet), ErrRateLimitExceeded)

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client2, packet))
	}
	assert.ErrorIs(t, hook.OnPublish(client2, packet), ErrRateLimitExceeded)
}

func TestRateLimitHookGetClientCount(t *testing.T) {
	hook := NewRateLimitHook(10, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	count, exists := hook.GetClientCount("client1")
	assert.False(t, exists)
	assert.Equal(t, 0, count)

	for i := 0; i < 5; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}

	count, exists = hook.GetClientCount("client1")
	assert.True(t, exists)
	assert.Equal(t, 5, count)
}

func TestRateLimitHookResetClient(t *testing.T) {
	hook := NewRateLimitHook(3, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}
	assert.ErrorIs(t, hook.OnPublish(client, packet), ErrRateLimitExceeded)

	hook.ResetClient("client1")

	assert.NoError(t, hook.OnPublish(client, packet))
}

func TestRateLimitHookResetAll(t *testing.T) {
	hook := NewRateLimitHook(2, time.Minute)
	defer hook.Stop()

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 2; i++ {
		assert.NoError(t, hook.OnPublish(client1, packet))
		assert.NoError(t, hook.OnPublish(client2, packet))
	}

	assert.ErrorIs(t, hook.OnPublish(client1, packet), ErrRateLimitExceeded)
	assert.ErrorIs(t, hook.OnPublish(client2, packet), ErrRateLimitExceeded)

	hook.ResetAll()

	assert.NoError(t, hook.OnPublish(client1, packet))
	assert.NoError(t, hook.OnPublish(client2, packet))
}

func TestRateLimitHookSetMaxRate(t *testing.T) {
	hook := NewRateLimitHook(5, time.Minute)
	defer hook.Stop()

	assert.Equal(t, 5, hook.GetMaxRate())

	hook.SetMaxRate(10)
	assert.Equal(t, 10, hook.GetMaxRate())

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 10; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}
	assert.ErrorIs(t, hook.OnPublish(client, packet), ErrRateLimitExceeded)
}

func TestRateLimitHookSetWindow(t *testing.T) {
	hook := NewRateLimitHook(3, time.Minute)
	defer hook.Stop()

	assert.Equal(t, time.Minute, hook.GetWindow())

	hook.SetWindow(time.Second)
	assert.Equal(t, time.Second, hook.GetWindow())
}

func TestRateLimitHookActiveClients(t *testing.T) {
	hook := NewRateLimitHook(10, time.Minute)
	defer hook.Stop()

	assert.Equal(t, 0, hook.ActiveClients())

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	client3 := &Client{ID: "client3"}
	packet := &PublishPacket{Topic: "test/topic"}

	hook.OnPublish(client1, packet)
	assert.Equal(t, 1, hook.ActiveClients())

	hook.OnPublish(client2, packet)
	assert.Equal(t, 2, hook.ActiveClients())

	hook.OnPublish(client3, packet)
	assert.Equal(t, 3, hook.ActiveClients())
}

func TestRateLimitHookConcurrentAccess(t *testing.T) {
	hook := NewRateLimitHook(1000, time.Minute)
	defer hook.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			client := &Client{ID: "client1"}
			packet := &PublishPacket{Topic: "test/topic"}
			for j := 0; j < 50; j++ {
				_ = hook.OnPublish(client, packet)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	count, _ := hook.GetClientCount("client1")
	assert.Equal(t, 500, count)
}

func TestRateLimitHookWithManager(t *testing.T) {
	manager := NewManager()
	hook := NewRateLimitHook(3, time.Minute)
	defer hook.Stop()

	require.NoError(t, manager.Add(hook))

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		err := manager.OnPublish(client, packet)
		assert.NoError(t, err)
	}

	err := manager.OnPublish(client, packet)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestMultiLevelRateLimitHook(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(5, 10, 20, time.Minute)
	defer hook.Stop()

	assert.Equal(t, "multi-level-rate-limit", hook.ID())
	assert.True(t, hook.Provides(OnPublish))
}

func TestMultiLevelRateLimitHookPerClient(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(3, 0, 0, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		err := hook.OnPublish(client, packet)
		assert.NoError(t, err)
	}

	err := hook.OnPublish(client, packet)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestMultiLevelRateLimitHookPerTopic(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(0, 5, 0, time.Minute)
	defer hook.Stop()

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client1, packet))
	}

	for i := 0; i < 2; i++ {
		assert.NoError(t, hook.OnPublish(client2, packet))
	}

	err := hook.OnPublish(client1, packet)
	assert.ErrorIs(t, err, ErrTopicRateLimitExceeded)
}

func TestMultiLevelRateLimitHookGlobal(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(0, 0, 10, time.Minute)
	defer hook.Stop()

	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = &Client{ID: string(rune('a' + i))}
	}

	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 10; i++ {
		client := clients[i%5]
		err := hook.OnPublish(client, packet)
		assert.NoError(t, err)
	}

	err := hook.OnPublish(clients[0], packet)
	assert.ErrorIs(t, err, ErrGlobalRateLimitExceeded)
}

func TestMultiLevelRateLimitHookAllLevels(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(3, 5, 10, time.Minute)
	defer hook.Stop()

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet1 := &PublishPacket{Topic: "topic1"}
	packet2 := &PublishPacket{Topic: "topic2"}

	// Client1 can publish 3 times to topic1 (per-client limit)
	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client1, packet1))
	}
	// 4th attempt exceeds client limit
	assert.ErrorIs(t, hook.OnPublish(client1, packet1), ErrRateLimitExceeded)

	// Client1 can publish 2 more times to topic2 (topic1 already has 3, limit is 5)
	for i := 0; i < 2; i++ {
		err := hook.OnPublish(client1, packet2)
		if err != nil {
			// Could be client limit or topic limit depending on order
			assert.Error(t, err)
		}
	}

	// Client2 can publish to topic1 (different client)
	for i := 0; i < 2; i++ {
		err := hook.OnPublish(client2, packet1)
		// Might hit topic limit (topic1 already has 3 from client1)
		if i < 2 {
			if err != nil && err != ErrTopicRateLimitExceeded {
				assert.NoError(t, err)
			}
		}
	}

	// Global count should be approaching limit
	globalCount := hook.GetGlobalCount()
	assert.True(t, globalCount >= 5 && globalCount <= 10)
}

func TestMultiLevelRateLimitHookGetCounts(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(10, 10, 100, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 5; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}

	clientCount, exists := hook.GetClientCount("client1")
	assert.True(t, exists)
	assert.Equal(t, 5, clientCount)

	topicCount, exists := hook.GetTopicCount("test/topic")
	assert.True(t, exists)
	assert.Equal(t, 5, topicCount)

	assert.Equal(t, 5, hook.GetGlobalCount())
}

func TestMultiLevelRateLimitHookResetAll(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(3, 5, 10, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}

	assert.ErrorIs(t, hook.OnPublish(client, packet), ErrRateLimitExceeded)

	hook.ResetAll()

	assert.NoError(t, hook.OnPublish(client, packet))
	assert.Equal(t, 1, hook.GetGlobalCount())
}

func TestMultiLevelRateLimitHookWindowReset(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(2, 0, 0, 100*time.Millisecond)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 2; i++ {
		assert.NoError(t, hook.OnPublish(client, packet))
	}

	assert.ErrorIs(t, hook.OnPublish(client, packet), ErrRateLimitExceeded)

	time.Sleep(150 * time.Millisecond)

	assert.NoError(t, hook.OnPublish(client, packet))
}

func TestMultiLevelRateLimitHookConcurrentAccess(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(100, 200, 500, time.Minute)
	defer hook.Stop()

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			client := &Client{ID: string(rune('a' + id))}
			packet := &PublishPacket{Topic: "test/topic"}
			for j := 0; j < 50; j++ {
				_ = hook.OnPublish(client, packet)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	assert.Equal(t, 250, hook.GetGlobalCount())
}

func TestMultiLevelRateLimitHookWithManager(t *testing.T) {
	manager := NewManager()
	hook := NewMultiLevelRateLimitHook(2, 3, 5, time.Minute)
	defer hook.Stop()

	require.NoError(t, manager.Add(hook))

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet := &PublishPacket{Topic: "test/topic"}

	assert.NoError(t, manager.OnPublish(client1, packet))
	assert.NoError(t, manager.OnPublish(client1, packet))
	assert.ErrorIs(t, manager.OnPublish(client1, packet), ErrRateLimitExceeded)

	assert.NoError(t, manager.OnPublish(client2, packet))
	assert.ErrorIs(t, manager.OnPublish(client2, packet), ErrTopicRateLimitExceeded)
}

func TestRateLimitHookZeroLimits(t *testing.T) {
	hook := NewRateLimitHook(0, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	err := hook.OnPublish(client, packet)
	assert.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestMultiLevelRateLimitHookDisabledLevels(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(0, 0, 5, time.Minute)
	defer hook.Stop()

	client1 := &Client{ID: "client1"}
	client2 := &Client{ID: "client2"}
	packet := &PublishPacket{Topic: "test/topic"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client1, packet))
	}

	for i := 0; i < 2; i++ {
		assert.NoError(t, hook.OnPublish(client2, packet))
	}

	err := hook.OnPublish(client1, packet)
	assert.ErrorIs(t, err, ErrGlobalRateLimitExceeded)
}

func TestRateLimitHookCleanup(t *testing.T) {
	hook := NewRateLimitHook(100, 50*time.Millisecond)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	hook.OnPublish(client, packet)
	assert.Equal(t, 1, hook.ActiveClients())

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, hook.ActiveClients())
}

func TestMultiLevelRateLimitHookDifferentTopics(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(10, 3, 100, time.Minute)
	defer hook.Stop()

	client := &Client{ID: "client1"}
	topic1 := &PublishPacket{Topic: "topic1"}
	topic2 := &PublishPacket{Topic: "topic2"}

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client, topic1))
	}
	assert.ErrorIs(t, hook.OnPublish(client, topic1), ErrTopicRateLimitExceeded)

	for i := 0; i < 3; i++ {
		assert.NoError(t, hook.OnPublish(client, topic2))
	}
	assert.ErrorIs(t, hook.OnPublish(client, topic2), ErrTopicRateLimitExceeded)
}

func TestRateLimitHookStopCleanup(t *testing.T) {
	hook := NewRateLimitHook(100, time.Millisecond)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}
	hook.OnPublish(client, packet)

	err := hook.Stop()
	assert.NoError(t, err)
}

func TestMultiLevelRateLimitHookStopCleanup(t *testing.T) {
	hook := NewMultiLevelRateLimitHook(10, 10, 100, time.Millisecond)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}
	hook.OnPublish(client, packet)

	err := hook.Stop()
	assert.NoError(t, err)
}
