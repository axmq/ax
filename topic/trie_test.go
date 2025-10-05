package topic

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrieSubscribe(t *testing.T) {
	t.Run("subscribe to simple topic", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		err := trie.Subscribe("home/temperature", sub)
		require.NoError(t, err)

		subs := trie.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("subscribe to wildcard topic", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		err := trie.Subscribe("home/+/temperature", sub)
		require.NoError(t, err)

		subs := trie.Match("home/room1/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("subscribe to multi-level wildcard", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		err := trie.Subscribe("home/#", sub)
		require.NoError(t, err)

		subs := trie.Match("home/room1/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("subscribe multiple clients to same topic", func(t *testing.T) {
		trie := NewTrie()

		err := trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		require.NoError(t, err)
		err = trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client2", QoS: 2})
		require.NoError(t, err)

		subs := trie.Match("home/temperature")
		require.Len(t, subs, 2)
	})

	t.Run("subscribe to invalid topic filter", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		err := trie.Subscribe("home/room+", sub)
		assert.Error(t, err)
	})
}

func TestTrieUnsubscribe(t *testing.T) {
	t.Run("unsubscribe from simple topic", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		trie.Subscribe("home/temperature", sub)
		found := trie.Unsubscribe("home/temperature", "client1")
		assert.True(t, found)

		subs := trie.Match("home/temperature")
		assert.Len(t, subs, 0)
	})

	t.Run("unsubscribe one of multiple subscribers", func(t *testing.T) {
		trie := NewTrie()

		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client2", QoS: 2})

		found := trie.Unsubscribe("home/temperature", "client1")
		assert.True(t, found)

		subs := trie.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})

	t.Run("unsubscribe non-existent subscription", func(t *testing.T) {
		trie := NewTrie()

		found := trie.Unsubscribe("home/temperature", "client999")
		assert.False(t, found)
	})

	t.Run("unsubscribe removes empty nodes", func(t *testing.T) {
		trie := NewTrie()
		sub := SubscriberInfo{ClientID: "client1", QoS: 1}

		trie.Subscribe("home/room/temperature", sub)
		trie.Unsubscribe("home/room/temperature", "client1")

		assert.Equal(t, 0, trie.Count())
	})
}

func TestTrieMatch(t *testing.T) {
	tests := []struct {
		name            string
		subscriptions   []string
		topic           string
		expectedMatches int
	}{
		{
			name:            "exact match",
			subscriptions:   []string{"home/temperature"},
			topic:           "home/temperature",
			expectedMatches: 1,
		},
		{
			name:            "no match",
			subscriptions:   []string{"home/temperature"},
			topic:           "home/humidity",
			expectedMatches: 0,
		},
		{
			name:            "single-level wildcard match",
			subscriptions:   []string{"home/+/temperature"},
			topic:           "home/room1/temperature",
			expectedMatches: 1,
		},
		{
			name:            "multi-level wildcard match",
			subscriptions:   []string{"home/#"},
			topic:           "home/room1/temperature",
			expectedMatches: 1,
		},
		{
			name:            "multi-level wildcard matches all levels",
			subscriptions:   []string{"home/#"},
			topic:           "home/room1/sensor/temperature",
			expectedMatches: 1,
		},
		{
			name:            "multiple subscriptions match",
			subscriptions:   []string{"home/+/temperature", "home/room1/#", "home/room1/temperature"},
			topic:           "home/room1/temperature",
			expectedMatches: 3,
		},
		{
			name:            "wildcard only matches one level",
			subscriptions:   []string{"home/+/temperature"},
			topic:           "home/room1/sensor/temperature",
			expectedMatches: 0,
		},
		{
			name:            "multiple single-level wildcards",
			subscriptions:   []string{"+/+/temperature"},
			topic:           "home/room1/temperature",
			expectedMatches: 1,
		},
		{
			name:            "root level subscription",
			subscriptions:   []string{"#"},
			topic:           "home/room1/temperature",
			expectedMatches: 1,
		},
		{
			name:            "single-level wildcard at root",
			subscriptions:   []string{"+"},
			topic:           "home",
			expectedMatches: 1,
		},
		{
			name:            "topic with leading slash",
			subscriptions:   []string{"/home/temperature"},
			topic:           "/home/temperature",
			expectedMatches: 1,
		},
		{
			name:            "subscription with leading slash matches",
			subscriptions:   []string{"/+/temperature"},
			topic:           "/home/temperature",
			expectedMatches: 1,
		},
		{
			name:            "empty level matching",
			subscriptions:   []string{"home//temperature"},
			topic:           "home//temperature",
			expectedMatches: 1,
		},
		{
			name:            "wildcard matches empty level",
			subscriptions:   []string{"home/+/temperature"},
			topic:           "home//temperature",
			expectedMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewTrie()

			for i, sub := range tt.subscriptions {
				clientID := fmt.Sprintf("client%d", i+1)
				trie.Subscribe(sub, SubscriberInfo{ClientID: clientID, QoS: 1})
			}

			subs := trie.Match(tt.topic)
			assert.Len(t, subs, tt.expectedMatches)
		})
	}
}

func TestTrieSharedSubscriptions(t *testing.T) {
	t.Run("subscribe to shared group", func(t *testing.T) {
		trie := NewTrie()

		err := trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		require.NoError(t, err)

		subs := trie.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("multiple subscribers in same group round-robin", func(t *testing.T) {
		trie := NewTrie()

		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})

		clientIDs := make(map[string]int)
		for i := 0; i < 10; i++ {
			subs := trie.Match("home/temperature")
			require.Len(t, subs, 1)
			clientIDs[subs[0].ClientID]++
		}

		assert.True(t, clientIDs["client1"] > 0)
		assert.True(t, clientIDs["client2"] > 0)
	})

	t.Run("different shared groups independent", func(t *testing.T) {
		trie := NewTrie()

		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.SubscribeShared("group2", "home/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})

		subs := trie.Match("home/temperature")
		assert.Len(t, subs, 2)
	})

	t.Run("unsubscribe from shared group", func(t *testing.T) {
		trie := NewTrie()

		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})

		found := trie.UnsubscribeShared("group1", "home/temperature", "client1")
		assert.True(t, found)

		subs := trie.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})

	t.Run("unsubscribe last member removes group", func(t *testing.T) {
		trie := NewTrie()

		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.UnsubscribeShared("group1", "home/temperature", "client1")

		subs := trie.Match("home/temperature")
		assert.Len(t, subs, 0)
	})
}

func TestTrieClear(t *testing.T) {
	t.Run("clear removes all subscriptions", func(t *testing.T) {
		trie := NewTrie()

		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.Subscribe("home/humidity", SubscriberInfo{ClientID: "client2", QoS: 1})
		trie.SubscribeShared("group1", "home/pressure", SubscriberInfo{ClientID: "client3", QoS: 1})

		trie.Clear()

		assert.Equal(t, 0, trie.Count())
		assert.Len(t, trie.Match("home/temperature"), 0)
		assert.Len(t, trie.Match("home/humidity"), 0)
		assert.Len(t, trie.Match("home/pressure"), 0)
	})
}

func TestTrieCount(t *testing.T) {
	t.Run("count tracks subscriptions", func(t *testing.T) {
		trie := NewTrie()

		assert.Equal(t, 0, trie.Count())

		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		assert.Equal(t, 1, trie.Count())

		trie.Subscribe("home/humidity", SubscriberInfo{ClientID: "client2", QoS: 1})
		assert.Equal(t, 2, trie.Count())

		trie.Unsubscribe("home/temperature", "client1")
		assert.Equal(t, 1, trie.Count())
	})

	t.Run("count includes shared subscriptions", func(t *testing.T) {
		trie := NewTrie()

		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
		trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})

		assert.Equal(t, 2, trie.Count())
	})
}

func TestTrieConcurrency(t *testing.T) {
	t.Run("concurrent subscribe and match", func(t *testing.T) {
		trie := NewTrie()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				filter := fmt.Sprintf("home/room%d/+", id%10)
				clientID := fmt.Sprintf("client%d", id)
				trie.Subscribe(filter, SubscriberInfo{ClientID: clientID, QoS: 1})
			}(i)
		}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				topic := fmt.Sprintf("home/room%d/temperature", id%10)
				trie.Match(topic)
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent subscribe and unsubscribe", func(t *testing.T) {
		trie := NewTrie()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				clientID := fmt.Sprintf("client%d", id)
				trie.Subscribe("home/temperature", SubscriberInfo{ClientID: clientID, QoS: 1})
			}(i)
		}

		wg.Wait()

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				clientID := fmt.Sprintf("client%d", id)
				trie.Unsubscribe("home/temperature", clientID)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 0, trie.Count())
	})
}

func TestTrieMatchInvalidTopic(t *testing.T) {
	t.Run("match with wildcard in topic returns nil", func(t *testing.T) {
		trie := NewTrie()
		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})

		subs := trie.Match("home/+")
		assert.Nil(t, subs)
	})

	t.Run("match with empty topic returns nil", func(t *testing.T) {
		trie := NewTrie()
		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})

		subs := trie.Match("")
		assert.Nil(t, subs)
	})
}

func BenchmarkTrieSubscribe(b *testing.B) {
	trie := NewTrie()
	sub := SubscriberInfo{ClientID: "client1", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Subscribe("home/room1/temperature", sub)
	}
}

func BenchmarkTrieSubscribeWildcard(b *testing.B) {
	trie := NewTrie()
	sub := SubscriberInfo{ClientID: "client1", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Subscribe("home/+/temperature", sub)
	}
}

func BenchmarkTrieMatch(b *testing.B) {
	trie := NewTrie()
	trie.Subscribe("home/room1/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
	trie.Subscribe("home/+/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})
	trie.Subscribe("home/#", SubscriberInfo{ClientID: "client3", QoS: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Match("home/room1/temperature")
	}
}

func BenchmarkTrieMatchComplex(b *testing.B) {
	trie := NewTrie()
	for i := 0; i < 100; i++ {
		filter := fmt.Sprintf("home/room%d/+", i)
		trie.Subscribe(filter, SubscriberInfo{ClientID: fmt.Sprintf("client%d", i), QoS: 1})
	}
	trie.Subscribe("home/#", SubscriberInfo{ClientID: "clientAll", QoS: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Match("home/room50/temperature")
	}
}

func BenchmarkTrieUnsubscribe(b *testing.B) {
	trie := NewTrie()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client%d", i)
		trie.Subscribe("home/temperature", SubscriberInfo{ClientID: clientID, QoS: 1})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client%d", i)
		trie.Unsubscribe("home/temperature", clientID)
	}
}

func BenchmarkTrieSharedSubscribe(b *testing.B) {
	trie := NewTrie()
	sub := SubscriberInfo{ClientID: "client1", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.SubscribeShared("group1", "home/temperature", sub)
	}
}

func BenchmarkTrieMatchShared(b *testing.B) {
	trie := NewTrie()
	trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client1", QoS: 1})
	trie.SubscribeShared("group1", "home/temperature", SubscriberInfo{ClientID: "client2", QoS: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Match("home/temperature")
	}
}
