package session

import (
	"context"
	"testing"
)

func BenchmarkManager_CreateSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	}
}

func BenchmarkManager_GetSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetSession(ctx, "client1")
	}
}

func BenchmarkManager_DisconnectSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
		b.StartTimer()
		_ = manager.DisconnectSession(ctx, "client1", false)
	}
}

func BenchmarkManager_GenerateClientID(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateClientID(ctx)
	}
}

func BenchmarkManager_TakeoverSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.TakeoverSession(ctx, "client1")
	}
}

func BenchmarkManager_ConcurrentCreateSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = manager.CreateSession(ctx, "client1", false, 300, 5)
		}
	})
}

func BenchmarkManager_ConcurrentGetSession(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = manager.GetSession(ctx, "client1")
		}
	})
}

func BenchmarkManager_SessionLifecycle(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, _, _ := manager.CreateSession(ctx, "client1", false, 300, 5)
		s.AddSubscription(&Subscription{TopicFilter: "test/topic", QoS: 1})
		_ = manager.DisconnectSession(ctx, "client1", false)
		_, _, _ = manager.CreateSession(ctx, "client1", false, 300, 5)
		_ = manager.RemoveSession(ctx, "client1")
	}
}

func BenchmarkManager_1000ActiveSessions(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetActiveSessionCount()
	}
}

func BenchmarkManager_GetAllActiveSessions(b *testing.B) {
	manager := NewManager(ManagerConfig{Store: NewMemoryStore()})
	defer manager.Close()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_, _, _ = manager.CreateSession(ctx, "client1", true, 300, 5)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetAllActiveSessions()
	}
}
