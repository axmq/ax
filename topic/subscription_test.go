package topic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscription(t *testing.T) {
	t.Run("create subscription", func(t *testing.T) {
		sub := &Subscription{
			ClientID:               "client1",
			TopicFilter:            "home/+/temperature",
			QoS:                    1,
			NoLocal:                true,
			RetainAsPublished:      true,
			RetainHandling:         2,
			SubscriptionIdentifier: 123,
		}

		assert.Equal(t, "client1", sub.ClientID)
		assert.Equal(t, "home/+/temperature", sub.TopicFilter)
		assert.Equal(t, byte(1), sub.QoS)
		assert.True(t, sub.NoLocal)
		assert.True(t, sub.RetainAsPublished)
		assert.Equal(t, byte(2), sub.RetainHandling)
		assert.Equal(t, uint32(123), sub.SubscriptionIdentifier)
	})
}

func TestTopicAlias(t *testing.T) {
	t.Run("create topic alias", func(t *testing.T) {
		ta := NewTopicAlias(100)
		assert.NotNil(t, ta)
		assert.Equal(t, uint16(100), ta.maxAlias)
	})

	t.Run("set and get alias", func(t *testing.T) {
		ta := NewTopicAlias(100)

		ok := ta.Set(1, "home/temperature")
		require.True(t, ok)

		topic, ok := ta.Get(1)
		require.True(t, ok)
		assert.Equal(t, "home/temperature", topic)
	})

	t.Run("set multiple aliases", func(t *testing.T) {
		ta := NewTopicAlias(100)

		assert.True(t, ta.Set(1, "home/temperature"))
		assert.True(t, ta.Set(2, "home/humidity"))
		assert.True(t, ta.Set(3, "home/pressure"))

		topic, ok := ta.Get(1)
		require.True(t, ok)
		assert.Equal(t, "home/temperature", topic)

		topic, ok = ta.Get(2)
		require.True(t, ok)
		assert.Equal(t, "home/humidity", topic)

		topic, ok = ta.Get(3)
		require.True(t, ok)
		assert.Equal(t, "home/pressure", topic)
	})

	t.Run("overwrite alias", func(t *testing.T) {
		ta := NewTopicAlias(100)

		ta.Set(1, "home/temperature")
		ta.Set(1, "home/humidity")

		topic, ok := ta.Get(1)
		require.True(t, ok)
		assert.Equal(t, "home/humidity", topic)
	})

	t.Run("get non-existent alias", func(t *testing.T) {
		ta := NewTopicAlias(100)

		topic, ok := ta.Get(99)
		assert.False(t, ok)
		assert.Empty(t, topic)
	})

	t.Run("alias zero is invalid", func(t *testing.T) {
		ta := NewTopicAlias(100)

		ok := ta.Set(0, "home/temperature")
		assert.False(t, ok)
	})

	t.Run("alias exceeds max", func(t *testing.T) {
		ta := NewTopicAlias(10)

		ok := ta.Set(11, "home/temperature")
		assert.False(t, ok)
	})

	t.Run("alias at max is valid", func(t *testing.T) {
		ta := NewTopicAlias(10)

		ok := ta.Set(10, "home/temperature")
		assert.True(t, ok)
	})

	t.Run("clear aliases", func(t *testing.T) {
		ta := NewTopicAlias(100)

		ta.Set(1, "home/temperature")
		ta.Set(2, "home/humidity")

		ta.Clear()

		_, ok := ta.Get(1)
		assert.False(t, ok)
		_, ok = ta.Get(2)
		assert.False(t, ok)
	})
}

func TestSharedSubscriptionGroup(t *testing.T) {
	t.Run("create group", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")
		assert.NotNil(t, group)
		assert.Equal(t, "group1", group.groupName)
		assert.Equal(t, 0, group.Size())
	})

	t.Run("add subscriber", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		sub := SubscriberInfo{
			ClientID: "client1",
			QoS:      1,
		}
		group.AddSubscriber(sub)

		assert.Equal(t, 1, group.Size())
	})

	t.Run("add multiple subscribers", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
		group.AddSubscriber(SubscriberInfo{ClientID: "client2", QoS: 2})
		group.AddSubscriber(SubscriberInfo{ClientID: "client3", QoS: 0})

		assert.Equal(t, 3, group.Size())
	})

	t.Run("remove subscriber", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
		group.AddSubscriber(SubscriberInfo{ClientID: "client2", QoS: 2})

		removed := group.RemoveSubscriber("client1")
		assert.True(t, removed)
		assert.Equal(t, 1, group.Size())
	})

	t.Run("remove non-existent subscriber", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})

		removed := group.RemoveSubscriber("client999")
		assert.False(t, removed)
		assert.Equal(t, 1, group.Size())
	})

	t.Run("next subscriber round-robin", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
		group.AddSubscriber(SubscriberInfo{ClientID: "client2", QoS: 1})
		group.AddSubscriber(SubscriberInfo{ClientID: "client3", QoS: 1})

		sub1, ok := group.NextSubscriber()
		require.True(t, ok)
		assert.Equal(t, "client1", sub1.ClientID)

		sub2, ok := group.NextSubscriber()
		require.True(t, ok)
		assert.Equal(t, "client2", sub2.ClientID)

		sub3, ok := group.NextSubscriber()
		require.True(t, ok)
		assert.Equal(t, "client3", sub3.ClientID)

		sub4, ok := group.NextSubscriber()
		require.True(t, ok)
		assert.Equal(t, "client1", sub4.ClientID)
	})

	t.Run("next subscriber empty group", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		_, ok := group.NextSubscriber()
		assert.False(t, ok)
	})

	t.Run("get subscribers", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
		group.AddSubscriber(SubscriberInfo{ClientID: "client2", QoS: 2})

		subs := group.GetSubscribers()
		assert.Len(t, subs, 2)
		assert.Equal(t, "client1", subs[0].ClientID)
		assert.Equal(t, "client2", subs[1].ClientID)
	})

	t.Run("get subscribers returns copy", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})

		subs := group.GetSubscribers()
		subs[0].ClientID = "modified"

		subs2 := group.GetSubscribers()
		assert.Equal(t, "client1", subs2[0].ClientID)
	})

	t.Run("concurrent next subscriber", func(t *testing.T) {
		group := NewSharedSubscriptionGroup("group1")

		for i := 0; i < 10; i++ {
			group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
		}

		seen := make(map[int]bool)
		for i := 0; i < 100; i++ {
			group.NextSubscriber()
			seen[i] = true
		}

		assert.Len(t, seen, 100)
	})
}

func BenchmarkTopicAliasSet(b *testing.B) {
	ta := NewTopicAlias(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.Set(uint16((i%100)+1), "home/temperature")
	}
}

func BenchmarkTopicAliasGet(b *testing.B) {
	ta := NewTopicAlias(100)
	for i := 1; i <= 100; i++ {
		ta.Set(uint16(i), "home/temperature")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.Get(uint16((i % 100) + 1))
	}
}

func BenchmarkSharedGroupNextSubscriber(b *testing.B) {
	group := NewSharedSubscriptionGroup("group1")
	for i := 0; i < 10; i++ {
		group.AddSubscriber(SubscriberInfo{ClientID: "client1", QoS: 1})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group.NextSubscriber()
	}
}

func BenchmarkSharedGroupAddSubscriber(b *testing.B) {
	group := NewSharedSubscriptionGroup("group1")
	sub := SubscriberInfo{ClientID: "client1", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group.AddSubscriber(sub)
	}
}
