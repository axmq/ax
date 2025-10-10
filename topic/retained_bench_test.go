package topic

import (
	"context"
	"fmt"
	"testing"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
)

func BenchmarkRetainedManager_Set(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = rm.Set(ctx, "test/topic", msg)
	}
}

func BenchmarkRetainedManager_Get(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
	rm.Set(ctx, "test/topic", msg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = rm.Get(ctx, "test/topic")
	}
}

func BenchmarkRetainedManager_Delete(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
		rm.Set(ctx, "test/topic", msg)
		b.StartTimer()

		_ = rm.Delete(ctx, "test/topic")
	}
}

func BenchmarkRetainedManager_Match(b *testing.B) {
	sizes := []int{10, 100, 1000}
	matcher := &mockMatcher{}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			rm := NewRetainedManager(nil)
			defer rm.Close()

			ctx := context.Background()

			for i := 0; i < size; i++ {
				topic := fmt.Sprintf("test/topic/%d", i)
				msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
				rm.Set(ctx, topic, msg)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = rm.Match(ctx, "#", matcher)
			}
		})
	}
}

func BenchmarkRetainedManager_Count(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		topic := fmt.Sprintf("test/topic/%d", i)
		msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
		rm.Set(ctx, topic, msg)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = rm.Count(ctx)
	}
}

func BenchmarkRetainedManager_ConcurrentSet(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rm.Set(ctx, "test/topic", msg)
		}
	})
}

func BenchmarkRetainedManager_ConcurrentGet(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	msg := message.NewMessage(1, "test/topic", []byte("benchmark payload"), encoding.QoS1, true, nil)
	rm.Set(ctx, "test/topic", msg)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = rm.Get(ctx, "test/topic")
		}
	})
}

func BenchmarkRetainedManager_ConcurrentMatch(b *testing.B) {
	rm := NewRetainedManager(nil)
	defer rm.Close()

	ctx := context.Background()
	matcher := &mockMatcher{}

	for i := 0; i < 100; i++ {
		topic := fmt.Sprintf("test/topic/%d", i)
		msg := message.NewMessage(uint16(i), topic, []byte("payload"), encoding.QoS1, true, nil)
		rm.Set(ctx, topic, msg)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = rm.Match(ctx, "#", matcher)
		}
	})
}
