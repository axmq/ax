package topic

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouterSubscribe(t *testing.T) {
	t.Run("subscribe to simple topic", func(t *testing.T) {
		router := NewRouter()
		sub := &Subscription{
			ClientID:    "client1",
			TopicFilter: "home/temperature",
			QoS:         1,
		}

		err := router.Subscribe(sub)
		require.NoError(t, err)

		subs := router.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("subscribe with all MQTT 5.0 options", func(t *testing.T) {
		router := NewRouter()
		sub := &Subscription{
			ClientID:               "client1",
			TopicFilter:            "home/+/temperature",
			QoS:                    2,
			NoLocal:                true,
			RetainAsPublished:      true,
			RetainHandling:         1,
			SubscriptionIdentifier: 42,
		}

		err := router.Subscribe(sub)
		require.NoError(t, err)

		subs := router.Match("home/room1/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, byte(2), subs[0].QoS)
		assert.True(t, subs[0].NoLocal)
		assert.True(t, subs[0].RetainAsPublished)
		assert.Equal(t, byte(1), subs[0].RetainHandling)
		assert.Equal(t, uint32(42), subs[0].SubscriptionIdentifier)
	})

	t.Run("subscribe to wildcard topics", func(t *testing.T) {
		router := NewRouter()

		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/+/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/#", QoS: 2})

		subs := router.Match("home/room1/temperature")
		assert.Len(t, subs, 2)
	})

	t.Run("subscribe to invalid topic filter", func(t *testing.T) {
		router := NewRouter()
		sub := &Subscription{
			ClientID:    "client1",
			TopicFilter: "home/room+",
			QoS:         1,
		}

		err := router.Subscribe(sub)
		assert.Error(t, err)
	})

	t.Run("subscribe to shared subscription", func(t *testing.T) {
		router := NewRouter()
		sub := &Subscription{
			ClientID:    "client1",
			TopicFilter: "$share/group1/home/temperature",
			QoS:         1,
		}

		err := router.Subscribe(sub)
		require.NoError(t, err)

		subs := router.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("subscribe multiple clients to shared group", func(t *testing.T) {
		router := NewRouter()

		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "$share/group1/home/temperature", QoS: 1})

		clientIDs := make(map[string]int)
		for i := 0; i < 10; i++ {
			subs := router.Match("home/temperature")
			require.Len(t, subs, 1)
			clientIDs[subs[0].ClientID]++
		}

		assert.True(t, clientIDs["client1"] > 0)
		assert.True(t, clientIDs["client2"] > 0)
	})

	t.Run("subscribe to invalid shared subscription", func(t *testing.T) {
		router := NewRouter()
		sub := &Subscription{
			ClientID:    "client1",
			TopicFilter: "$share/group1/home/room+",
			QoS:         1,
		}

		err := router.Subscribe(sub)
		assert.Error(t, err)
	})
}

func TestRouterUnsubscribe(t *testing.T) {
	t.Run("unsubscribe from simple topic", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})

		found := router.Unsubscribe("client1", "home/temperature")
		assert.True(t, found)

		subs := router.Match("home/temperature")
		assert.Len(t, subs, 0)
	})

	t.Run("unsubscribe one of multiple subscribers", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/temperature", QoS: 1})

		found := router.Unsubscribe("client1", "home/temperature")
		assert.True(t, found)

		subs := router.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})

	t.Run("unsubscribe non-existent subscription", func(t *testing.T) {
		router := NewRouter()

		found := router.Unsubscribe("client1", "home/temperature")
		assert.False(t, found)
	})

	t.Run("unsubscribe from shared subscription", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "$share/group1/home/temperature", QoS: 1})

		found := router.Unsubscribe("client1", "$share/group1/home/temperature")
		assert.True(t, found)

		subs := router.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})

	t.Run("unsubscribe from invalid shared subscription", func(t *testing.T) {
		router := NewRouter()

		found := router.Unsubscribe("client1", "$share//home/temperature")
		assert.False(t, found)
	})
}

func TestRouterUnsubscribeAll(t *testing.T) {
	t.Run("unsubscribe all subscriptions for client", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/humidity", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/pressure", QoS: 1})

		count := router.UnsubscribeAll("client1")
		assert.Equal(t, 3, count)
		assert.Equal(t, 0, router.Count())
	})

	t.Run("unsubscribe all for client with shared subscriptions", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/humidity", QoS: 1})

		count := router.UnsubscribeAll("client1")
		assert.Equal(t, 2, count)
	})

	t.Run("unsubscribe all for non-existent client", func(t *testing.T) {
		router := NewRouter()

		count := router.UnsubscribeAll("client999")
		assert.Equal(t, 0, count)
	})

	t.Run("unsubscribe all does not affect other clients", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/temperature", QoS: 1})

		router.UnsubscribeAll("client1")

		subs := router.Match("home/temperature")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})
}

func TestRouterMatch(t *testing.T) {
	tests := []struct {
		name            string
		subscriptions   []*Subscription
		topic           string
		expectedMatches int
	}{
		{
			name: "exact match",
			subscriptions: []*Subscription{
				{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1},
			},
			topic:           "home/temperature",
			expectedMatches: 1,
		},
		{
			name: "wildcard match",
			subscriptions: []*Subscription{
				{ClientID: "client1", TopicFilter: "home/+/temperature", QoS: 1},
			},
			topic:           "home/room1/temperature",
			expectedMatches: 1,
		},
		{
			name: "multi-level wildcard match",
			subscriptions: []*Subscription{
				{ClientID: "client1", TopicFilter: "home/#", QoS: 1},
			},
			topic:           "home/room1/sensor/temperature",
			expectedMatches: 1,
		},
		{
			name: "multiple matches",
			subscriptions: []*Subscription{
				{ClientID: "client1", TopicFilter: "home/room1/temperature", QoS: 1},
				{ClientID: "client2", TopicFilter: "home/+/temperature", QoS: 1},
				{ClientID: "client3", TopicFilter: "home/#", QoS: 2},
			},
			topic:           "home/room1/temperature",
			expectedMatches: 3,
		},
		{
			name: "no match",
			subscriptions: []*Subscription{
				{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1},
			},
			topic:           "home/humidity",
			expectedMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()

			for _, sub := range tt.subscriptions {
				router.Subscribe(sub)
			}

			subs := router.Match(tt.topic)
			assert.Len(t, subs, tt.expectedMatches)
		})
	}
}

func TestRouterMatchWithPublisher(t *testing.T) {
	t.Run("filter out publisher with NoLocal", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{
			ClientID:    "client1",
			TopicFilter: "home/temperature",
			QoS:         1,
			NoLocal:     true,
		})

		subs := router.MatchWithPublisher("home/temperature", "client1")
		assert.Len(t, subs, 0)
	})

	t.Run("include publisher without NoLocal", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{
			ClientID:    "client1",
			TopicFilter: "home/temperature",
			QoS:         1,
			NoLocal:     false,
		})

		subs := router.MatchWithPublisher("home/temperature", "client1")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})

	t.Run("include other subscribers with NoLocal", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{
			ClientID:    "client1",
			TopicFilter: "home/temperature",
			QoS:         1,
			NoLocal:     true,
		})
		router.Subscribe(&Subscription{
			ClientID:    "client2",
			TopicFilter: "home/temperature",
			QoS:         1,
			NoLocal:     true,
		})

		subs := router.MatchWithPublisher("home/temperature", "client1")
		require.Len(t, subs, 1)
		assert.Equal(t, "client2", subs[0].ClientID)
	})

	t.Run("no publisher specified", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{
			ClientID:    "client1",
			TopicFilter: "home/temperature",
			QoS:         1,
			NoLocal:     true,
		})

		subs := router.MatchWithPublisher("home/temperature", "")
		require.Len(t, subs, 1)
		assert.Equal(t, "client1", subs[0].ClientID)
	})
}

func TestRouterGetSubscription(t *testing.T) {
	t.Run("get existing subscription", func(t *testing.T) {
		router := NewRouter()
		original := &Subscription{
			ClientID:               "client1",
			TopicFilter:            "home/temperature",
			QoS:                    2,
			SubscriptionIdentifier: 42,
		}
		router.Subscribe(original)

		sub, ok := router.GetSubscription("client1", "home/temperature")
		require.True(t, ok)
		assert.Equal(t, "client1", sub.ClientID)
		assert.Equal(t, "home/temperature", sub.TopicFilter)
		assert.Equal(t, byte(2), sub.QoS)
		assert.Equal(t, uint32(42), sub.SubscriptionIdentifier)
	})

	t.Run("get non-existent subscription", func(t *testing.T) {
		router := NewRouter()

		sub, ok := router.GetSubscription("client1", "home/temperature")
		assert.False(t, ok)
		assert.Nil(t, sub)
	})

	t.Run("get subscription for non-existent client", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})

		sub, ok := router.GetSubscription("client999", "home/temperature")
		assert.False(t, ok)
		assert.Nil(t, sub)
	})
}

func TestRouterGetClientSubscriptions(t *testing.T) {
	t.Run("get all subscriptions for client", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/humidity", QoS: 2})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/pressure", QoS: 0})

		subs := router.GetClientSubscriptions("client1")
		assert.Len(t, subs, 2)
	})

	t.Run("get subscriptions for non-existent client", func(t *testing.T) {
		router := NewRouter()

		subs := router.GetClientSubscriptions("client999")
		assert.Nil(t, subs)
	})

	t.Run("get subscriptions includes shared subscriptions", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/humidity", QoS: 1})

		subs := router.GetClientSubscriptions("client1")
		assert.Len(t, subs, 2)
	})
}

func TestRouterCount(t *testing.T) {
	t.Run("count tracks subscriptions", func(t *testing.T) {
		router := NewRouter()

		assert.Equal(t, 0, router.Count())

		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		assert.Equal(t, 1, router.Count())

		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/temperature", QoS: 1})
		assert.Equal(t, 2, router.Count())

		router.Unsubscribe("client1", "home/temperature")
		assert.Equal(t, 1, router.Count())
	})

	t.Run("count includes shared subscriptions", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "$share/group1/home/temperature", QoS: 1})

		assert.Equal(t, 2, router.Count())
	})
}

func TestRouterCountClients(t *testing.T) {
	t.Run("count clients", func(t *testing.T) {
		router := NewRouter()

		assert.Equal(t, 0, router.CountClients())

		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		assert.Equal(t, 1, router.CountClients())

		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/humidity", QoS: 1})
		assert.Equal(t, 1, router.CountClients())

		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/pressure", QoS: 1})
		assert.Equal(t, 2, router.CountClients())
	})

	t.Run("unsubscribe all removes client", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/temperature", QoS: 1})

		router.UnsubscribeAll("client1")
		assert.Equal(t, 1, router.CountClients())
	})
}

func TestRouterClear(t *testing.T) {
	t.Run("clear removes all subscriptions", func(t *testing.T) {
		router := NewRouter()
		router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/humidity", QoS: 1})
		router.Subscribe(&Subscription{ClientID: "client3", TopicFilter: "$share/group1/home/pressure", QoS: 1})

		router.Clear()

		assert.Equal(t, 0, router.Count())
		assert.Equal(t, 0, router.CountClients())
		assert.Len(t, router.Match("home/temperature"), 0)
	})
}

func TestRouterConcurrency(t *testing.T) {
	t.Run("concurrent subscribe and match", func(t *testing.T) {
		router := NewRouter()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sub := &Subscription{
					ClientID:    fmt.Sprintf("client%d", id),
					TopicFilter: fmt.Sprintf("home/room%d/+", id%10),
					QoS:         1,
				}
				router.Subscribe(sub)
			}(i)
		}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				topic := fmt.Sprintf("home/room%d/temperature", id%10)
				router.Match(topic)
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent subscribe and unsubscribe", func(t *testing.T) {
		router := NewRouter()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sub := &Subscription{
					ClientID:    fmt.Sprintf("client%d", id),
					TopicFilter: "home/temperature",
					QoS:         1,
				}
				router.Subscribe(sub)
			}(i)
		}

		wg.Wait()

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				clientID := fmt.Sprintf("client%d", id)
				router.Unsubscribe(clientID, "home/temperature")
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 0, router.Count())
	})

	t.Run("concurrent operations on different clients", func(t *testing.T) {
		router := NewRouter()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				clientID := fmt.Sprintf("client%d", id)
				for j := 0; j < 10; j++ {
					filter := fmt.Sprintf("home/room%d/+", j)
					router.Subscribe(&Subscription{ClientID: clientID, TopicFilter: filter, QoS: 1})
				}
				router.UnsubscribeAll(clientID)
			}(i)
		}

		wg.Wait()
	})
}

func BenchmarkRouterSubscribe(b *testing.B) {
	router := NewRouter()
	sub := &Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Subscribe(sub)
	}
}

func BenchmarkRouterMatch(b *testing.B) {
	router := NewRouter()
	router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/room1/temperature", QoS: 1})
	router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/+/temperature", QoS: 1})
	router.Subscribe(&Subscription{ClientID: "client3", TopicFilter: "home/#", QoS: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Match("home/room1/temperature")
	}
}

func BenchmarkRouterMatchWithPublisher(b *testing.B) {
	router := NewRouter()
	router.Subscribe(&Subscription{ClientID: "client1", TopicFilter: "home/temperature", QoS: 1, NoLocal: true})
	router.Subscribe(&Subscription{ClientID: "client2", TopicFilter: "home/temperature", QoS: 1, NoLocal: true})
	router.Subscribe(&Subscription{ClientID: "client3", TopicFilter: "home/temperature", QoS: 1, NoLocal: false})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.MatchWithPublisher("home/temperature", "client1")
	}
}

func BenchmarkRouterUnsubscribe(b *testing.B) {
	router := NewRouter()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client%d", i)
		router.Subscribe(&Subscription{ClientID: clientID, TopicFilter: "home/temperature", QoS: 1})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client%d", i)
		router.Unsubscribe(clientID, "home/temperature")
	}
}

func BenchmarkRouterSharedSubscribe(b *testing.B) {
	router := NewRouter()
	sub := &Subscription{ClientID: "client1", TopicFilter: "$share/group1/home/temperature", QoS: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Subscribe(sub)
	}
}

func BenchmarkRouterComplexMatching(b *testing.B) {
	router := NewRouter()
	for i := 0; i < 100; i++ {
		filter := fmt.Sprintf("home/room%d/+", i)
		router.Subscribe(&Subscription{ClientID: fmt.Sprintf("client%d", i), TopicFilter: filter, QoS: 1})
	}
	router.Subscribe(&Subscription{ClientID: "clientAll", TopicFilter: "home/#", QoS: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Match("home/room50/temperature")
	}
}
