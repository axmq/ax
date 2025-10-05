package session

import (
	"context"
	"testing"
)

func BenchmarkMemoryStore_Save(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Save(ctx, session)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	session := New("client1", true, 300, 5)
	_ = store.Save(ctx, session)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Load(ctx, "client1")
	}
}

func BenchmarkMemoryStore_Delete(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_ = store.Save(ctx, New("client1", true, 300, 5))
		b.StartTimer()
		_ = store.Delete(ctx, "client1")
	}
}

func BenchmarkMemoryStore_Exists(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	_ = store.Save(ctx, New("client1", true, 300, 5))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Exists(ctx, "client1")
	}
}

func BenchmarkMemoryStore_List(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_ = store.Save(ctx, New("client1", true, 300, 5))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.List(ctx)
	}
}

func BenchmarkMemoryStore_Count(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_ = store.Save(ctx, New("client1", true, 300, 5))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Count(ctx)
	}
}

func BenchmarkMemoryStore_CountByState(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		s := New("client1", true, 300, 5)
		s.SetActive()
		_ = store.Save(ctx, s)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.CountByState(ctx, StateActive)
	}
}

func BenchmarkMemoryStore_ConcurrentSave(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	session := New("client1", true, 300, 5)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.Save(ctx, session)
		}
	})
}

func BenchmarkMemoryStore_ConcurrentLoad(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	_ = store.Save(ctx, New("client1", true, 300, 5))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.Load(ctx, "client1")
		}
	})
}

func BenchmarkMemoryStore_ConcurrentMixed(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	session := New("client1", true, 300, 5)
	_ = store.Save(ctx, session)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				_ = store.Save(ctx, session)
			case 1:
				_, _ = store.Load(ctx, "client1")
			case 2:
				_, _ = store.Exists(ctx, "client1")
			case 3:
				_, _ = store.Count(ctx)
			}
			i++
		}
	})
}

func BenchmarkMemoryStore_SaveLoad_1000Sessions(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	sessions := make([]*Session, 1000)
	for i := 0; i < 1000; i++ {
		sessions[i] = New("client1", true, 300, 5)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range sessions {
			_ = store.Save(ctx, s)
		}
		for _, s := range sessions {
			_, _ = store.Load(ctx, s.ClientID)
		}
	}
}
