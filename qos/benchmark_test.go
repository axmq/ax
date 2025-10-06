package qos

import (
	"testing"

	"github.com/axmq/ax/encoding"
	"github.com/axmq/ax/types/message"
)

func BenchmarkNewMessage(b *testing.B) {
	topic := "test/topic"
	payload := []byte("test payload data for benchmarking")
	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(60),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = message.NewMessage(uint16(i), topic, payload, encoding.QoS1, false, properties)
	}
}

func BenchmarkNewMessageNoProperties(b *testing.B) {
	topic := "test/topic"
	payload := []byte("test payload data for benchmarking")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = message.NewMessage(uint16(i), topic, payload, encoding.QoS1, false, nil)
	}
}

func BenchmarkMessage_IsExpired(b *testing.B) {
	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(60),
	}
	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = msg.IsExpired()
	}
}

func BenchmarkMessage_RemainingExpiry(b *testing.B) {
	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(60),
	}
	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, properties)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = msg.RemainingExpiry()
	}
}

func BenchmarkMessage_MarkAttempt(b *testing.B) {
	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg.MarkAttempt()
	}
}

func BenchmarkMessage_Clone(b *testing.B) {
	properties := map[string]interface{}{
		"MessageExpiryInterval": uint32(60),
		"ContentType":           "application/json",
	}
	msg := message.NewMessage(1, "test/topic", []byte("test payload data"), encoding.QoS2, true, properties)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = msg.Clone()
	}
}

func BenchmarkMessage_CloneLargePayload(b *testing.B) {
	payload := make([]byte, 1024*10)
	msg := message.NewMessage(1, "test/topic", payload, encoding.QoS1, false, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = msg.Clone()
	}
}

func BenchmarkHandler_PublishQoS0(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	msg := message.NewMessage(0, "test/topic", []byte("payload"), encoding.QoS0, false, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.HandlePublish(msg)
	}
}

func BenchmarkHandler_PublishQoS1(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	topic := "test/topic"
	payload := []byte("test payload data")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = h.PublishQoS1(topic, payload, false, nil)
	}
}

func BenchmarkHandler_PublishQoS2(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	topic := "test/topic"
	payload := []byte("test payload data")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = h.PublishQoS2(topic, payload, false, nil)
	}
}

func BenchmarkHandler_HandleQoS1Publish(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubackCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := message.NewMessage(uint16(i), "test/topic", []byte("payload"), encoding.QoS1, false, nil)
		_ = h.HandlePublish(msg)
	}
}

func BenchmarkHandler_HandleQoS2Publish(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubrecCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := message.NewMessage(uint16(i), "test/topic", []byte("payload"), encoding.QoS2, false, nil)
		_ = h.HandlePublish(msg)
	}
}

func BenchmarkHandler_HandlePuback(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	for i := 0; i < b.N; i++ {
		_, _ = h.PublishQoS1("test/topic", []byte("payload"), false, nil)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.HandlePuback(uint16(i + 1))
	}
}

func BenchmarkHandler_QoS2Flow(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubrecCallback(func(packetID uint16) error { return nil })
	h.SetPubrelCallback(func(packetID uint16) error { return nil })
	h.SetPubcompCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		packetID, _ := h.PublishQoS2("test/topic", []byte("payload"), false, nil)
		_ = h.HandlePubrec(packetID)
		_ = h.HandlePubcomp(packetID)
	}
}

func BenchmarkHandler_GetInflightCount(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	_, _ = h.PublishQoS1("test/topic", []byte("payload"), false, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.GetInflightCount()
	}
}

func BenchmarkHandler_PacketIDAllocation(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		h.mu.Lock()
		_ = h.allocatePacketID()
		h.mu.Unlock()
	}
}

func BenchmarkDedupCache_Add(b *testing.B) {
	cache := newDedupCache(10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.add(uint16(i))
	}
}

func BenchmarkDedupCache_Exists(b *testing.B) {
	cache := newDedupCache(10000)
	for i := 0; i < 1000; i++ {
		cache.add(uint16(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.exists(uint16(i % 1000))
	}
}

func BenchmarkDedupCache_Remove(b *testing.B) {
	cache := newDedupCache(10000)
	for i := 0; i < b.N; i++ {
		cache.add(uint16(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.remove(uint16(i))
	}
}

func BenchmarkDedupCache_Cleanup(b *testing.B) {
	cache := newDedupCache(10000)
	for i := 0; i < 1000; i++ {
		cache.add(uint16(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.cleanup()
	}
}

func BenchmarkHandler_ConcurrentPublishQoS1(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = h.PublishQoS1("test/topic", []byte("payload"), false, nil)
			i++
		}
	})
}

func BenchmarkHandler_ConcurrentPublishQoS2(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = h.PublishQoS2("test/topic", []byte("payload"), false, nil)
		}
	})
}

func BenchmarkHandler_ConcurrentHandlePuback(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })

	packetIDs := make([]uint16, b.N)
	for i := 0; i < b.N; i++ {
		packetID, _ := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
		packetIDs[i] = packetID
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i < len(packetIDs) {
				_ = h.HandlePuback(packetIDs[i])
				i++
			}
		}
	})
}

func BenchmarkDedupCache_ConcurrentAdd(b *testing.B) {
	cache := newDedupCache(100000)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.add(uint16(i))
			i++
		}
	})
}

func BenchmarkDedupCache_ConcurrentExists(b *testing.B) {
	cache := newDedupCache(100000)
	for i := 0; i < 10000; i++ {
		cache.add(uint16(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = cache.exists(uint16(i % 10000))
			i++
		}
	})
}

func BenchmarkHandler_CalculateRetryInterval(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.calculateRetryInterval(i % 10)
	}
}

func BenchmarkMessage_IsExpiredNoExpiry(b *testing.B) {
	msg := message.NewMessage(1, "test/topic", []byte("payload"), encoding.QoS1, false, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = msg.IsExpired()
	}
}

func BenchmarkHandler_QoS1CompleteFlow(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubackCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		packetID, _ := h.PublishQoS1("test/topic", []byte("payload"), false, nil)
		_ = h.HandlePuback(packetID)
	}
}

func BenchmarkHandler_QoS2CompleteFlow(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubrecCallback(func(packetID uint16) error { return nil })
	h.SetPubrelCallback(func(packetID uint16) error { return nil })
	h.SetPubcompCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		packetID, _ := h.PublishQoS2("test/topic", []byte("payload"), false, nil)
		_ = h.HandlePubrec(packetID)
		_ = h.HandlePubcomp(packetID)
	}
}

func BenchmarkHandler_InboundQoS2Flow(b *testing.B) {
	h := NewHandler(nil)
	defer h.Close()

	h.SetPublishCallback(func(msg *message.Message) error { return nil })
	h.SetPubrecCallback(func(packetID uint16) error { return nil })
	h.SetPubrelCallback(func(packetID uint16) error { return nil })
	h.SetPubcompCallback(func(packetID uint16) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := message.NewMessage(uint16(i), "test/topic", []byte("payload"), encoding.QoS2, false, nil)
		_ = h.HandlePublish(msg)
		_ = h.HandlePubrel(uint16(i))
	}
}

func BenchmarkMessage_SmallPayload(b *testing.B) {
	payload := []byte("x")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = message.NewMessage(uint16(i), "t", payload, encoding.QoS1, false, nil)
	}
}

func BenchmarkMessage_LargePayload(b *testing.B) {
	payload := make([]byte, 1024*256)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = message.NewMessage(uint16(i), "test/topic", payload, encoding.QoS1, false, nil)
	}
}
