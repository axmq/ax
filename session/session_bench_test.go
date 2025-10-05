package session

import (
	"testing"
	"time"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New("client1", true, 300, 5)
	}
}

func BenchmarkSession_SetActive(b *testing.B) {
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.SetActive()
	}
}

func BenchmarkSession_Touch(b *testing.B) {
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Touch()
	}
}

func BenchmarkSession_IsExpired(b *testing.B) {
	session := New("client1", false, 300, 5)
	session.SetDisconnected()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.IsExpired()
	}
}

func BenchmarkSession_NextPacketID(b *testing.B) {
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.NextPacketID()
	}
}

func BenchmarkSession_AddSubscription(b *testing.B) {
	session := New("client1", true, 300, 5)
	sub := &Subscription{
		TopicFilter: "test/topic",
		QoS:         1,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.AddSubscription(sub)
	}
}

func BenchmarkSession_GetSubscription(b *testing.B) {
	session := New("client1", true, 300, 5)
	session.AddSubscription(&Subscription{
		TopicFilter: "test/topic",
		QoS:         1,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = session.GetSubscription("test/topic")
	}
}

func BenchmarkSession_AddPendingPublish(b *testing.B) {
	session := New("client1", true, 300, 5)
	msg := &PendingMessage{
		PacketID:  1,
		Topic:     "test/topic",
		Payload:   []byte("test payload"),
		QoS:       1,
		Timestamp: time.Now(),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.AddPendingPublish(msg)
	}
}

func BenchmarkSession_GetPendingPublish(b *testing.B) {
	session := New("client1", true, 300, 5)
	session.AddPendingPublish(&PendingMessage{
		PacketID: 1,
		Topic:    "test/topic",
		Payload:  []byte("test payload"),
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = session.GetPendingPublish(1)
	}
}

func BenchmarkSession_SetWillMessage(b *testing.B) {
	session := New("client1", true, 300, 5)
	will := &WillMessage{
		Topic:   "client/status",
		Payload: []byte("offline"),
		QoS:     1,
		Retain:  true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.SetWillMessage(will, 0)
	}
}

func BenchmarkSession_GetWillMessage(b *testing.B) {
	session := New("client1", true, 300, 5)
	session.SetWillMessage(&WillMessage{
		Topic:   "client/status",
		Payload: []byte("offline"),
	}, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.GetWillMessage()
	}
}

func BenchmarkSession_ConcurrentAccess(b *testing.B) {
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			session.Touch()
			_ = session.NextPacketID()
			_ = session.GetState()
		}
	})
}

func BenchmarkSession_AddRemoveSubscription(b *testing.B) {
	session := New("client1", true, 300, 5)
	sub := &Subscription{
		TopicFilter: "test/topic",
		QoS:         1,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.AddSubscription(sub)
		session.RemoveSubscription("test/topic")
	}
}

func BenchmarkSession_MultipleSubscriptions(b *testing.B) {
	session := New("client1", true, 300, 5)
	for i := 0; i < 100; i++ {
		session.AddSubscription(&Subscription{
			TopicFilter: "test/topic",
			QoS:         1,
		})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.GetAllSubscriptions()
	}
}
