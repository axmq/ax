package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
)

func BenchmarkRetainedStore_Set(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.Set(ctx, "test/topic", msg)
	}
}

func BenchmarkRetainedStore_Get(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
	store.Set(ctx, "test/topic", msg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = store.Get(ctx, "test/topic")
	}
}

func BenchmarkRetainedStore_Delete(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
		store.Set(ctx, "test/topic", msg)
		b.StartTimer()

		_ = store.Delete(ctx, "test/topic")
	}
}

func BenchmarkRetainedStore_Match(b *testing.B) {
	sizes := []int{10, 100, 1000}
	matcher := &mockTopicMatcher{}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			store := NewRetainedStore()
			defer store.Close()

			ctx := context.Background()

			for i := 0; i < size; i++ {
				topic := fmt.Sprintf("test/topic/%d", i)
				msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
				store.Set(ctx, topic, msg)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = store.Match(ctx, "#", matcher)
			}
		})
	}
}

func BenchmarkRetainedStore_CleanupExpired(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			store := NewRetainedStore()
			defer store.Close()

			ctx := context.Background()

			for i := 0; i < size; i++ {
				topic := fmt.Sprintf("test/topic/%d", i)
				msg := message.NewMessage(
					uint16(i),
					topic,
					[]byte("payload"),
					encoding.QoS1,
					true,
					map[string]interface{}{"MessageExpiryInterval": uint32(3600)},
				)
				store.Set(ctx, topic, msg)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = store.CleanupExpired(ctx)
			}
		})
	}
}

func BenchmarkRetainedStore_Count(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		topic := fmt.Sprintf("test/topic/%d", i)
		msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
		store.Set(ctx, topic, msg)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = store.Count(ctx)
	}
}

func BenchmarkRetainedStore_ConcurrentSet(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.Set(ctx, "test/topic", msg)
		}
	})
}

func BenchmarkRetainedStore_ConcurrentGet(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
	store.Set(ctx, "test/topic", msg)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.Get(ctx, "test/topic")
		}
	})
}

func BenchmarkRetainedStore_ConcurrentMatch(b *testing.B) {
	store := NewRetainedStore()
	defer store.Close()

	ctx := context.Background()
	matcher := &mockTopicMatcher{}

	for i := 0; i < 100; i++ {
		topic := fmt.Sprintf("test/topic/%d", i)
		msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
		store.Set(ctx, topic, msg)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.Match(ctx, "#", matcher)
		}
	})
}
