package hook

import (
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
)

func BenchmarkManagerAdd(b *testing.B) {
	m := NewManager()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		h := &Base{id: string(rune(i))}
		_ = m.Add(h)
	}
}

func BenchmarkManagerRemove(b *testing.B) {
	m := NewManager()
	for i := 0; i < 1000; i++ {
		h := &Base{id: string(rune(i))}
		_ = m.Add(h)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		id := string(rune(i % 1000))
		_ = m.Remove(id)
	}
}

func BenchmarkManagerGet(b *testing.B) {
	m := NewManager()
	for i := 0; i < 100; i++ {
		h := &Base{id: string(rune(i))}
		_ = m.Add(h)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		id := string(rune(i % 100))
		_, _ = m.Get(id)
	}
}

func BenchmarkManagerOnConnect(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnConnect)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnConnect(client, packet)
	}
}

func BenchmarkManagerOnConnectMultipleHooks(b *testing.B) {
	m := NewManager()
	for i := 0; i < 10; i++ {
		h := newTestHook(string(rune('a'+i)), OnConnect)
		_ = m.Add(h)
	}

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnConnect(client, packet)
	}
}

func BenchmarkManagerOnConnectAuthenticate(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnConnectAuthenticate)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnConnectAuthenticate(client, packet)
	}
}

func BenchmarkManagerOnACLCheck(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnACLCheck)
	_ = m.Add(h)

	client := &Client{ID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnACLCheck(client, "test/topic", AccessTypeWrite)
	}
}

func BenchmarkManagerOnPublish(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPublish)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello world"),
		QoS:     1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnPublish(client, packet)
	}
}

func BenchmarkManagerOnPublishMultipleHooks(b *testing.B) {
	m := NewManager()
	for i := 0; i < 5; i++ {
		h := newTestHook(string(rune('a'+i)), OnPublish)
		_ = m.Add(h)
	}

	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello world"),
		QoS:     1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnPublish(client, packet)
	}
}

func BenchmarkManagerOnSubscribe(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnSubscribe)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	sub := &Subscription{
		ClientID:    "client1",
		TopicFilter: "test/#",
		QoS:         1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnSubscribe(client, sub)
	}
}

func BenchmarkManagerOnDisconnect(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnDisconnect)
	_ = m.Add(h)

	client := &Client{ID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.OnDisconnect(client, nil, false)
	}
}

func BenchmarkManagerOnPacketRead(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPacketRead)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := []byte{0x30, 0x0d, 0x00, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x68, 0x65, 0x6c, 0x6c, 0x6f}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = m.OnPacketRead(client, packet)
	}
}

func BenchmarkManagerOnPacketReadWithModification(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPacketRead)
	h.modifyPacket = true
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := []byte{0x30, 0x0d, 0x00, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = m.OnPacketRead(client, packet)
	}
}

func BenchmarkManagerOnPacketEncode(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPacketEncode)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := []byte{0x30, 0x0d, 0x00, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnPacketEncode(client, packet)
	}
}

func BenchmarkManagerOnQosPublish(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnQosPublish)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("data"),
		QoS:     1,
	}
	sent := time.Now()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.OnQosPublish(client, packet, sent, 0)
	}
}

func BenchmarkManagerOnQosComplete(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnQosComplete)
	_ = m.Add(h)

	client := &Client{ID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.OnQosComplete(client, 1, encoding.PUBACK)
	}
}

func BenchmarkManagerOnWill(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnWill)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	will := &WillMessage{
		Topic:   "will/topic",
		Payload: []byte("offline"),
		QoS:     1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnWill(client, will)
	}
}

func BenchmarkManagerOnSelectSubscribers(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnSelectSubscribers)
	_ = m.Add(h)

	subscribers := &Subscribers{
		Subscriptions: []*Subscription{
			{ClientID: "client1", TopicFilter: "test/#"},
			{ClientID: "client2", TopicFilter: "test/+"},
			{ClientID: "client3", TopicFilter: "test/topic"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.OnSelectSubscribers(subscribers, "test/topic")
	}
}

func BenchmarkManagerNoHooks(b *testing.B) {
	m := NewManager()

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnConnect(client, packet)
	}
}

func BenchmarkManagerMixedOperations(b *testing.B) {
	m := NewManager()
	for i := 0; i < 5; i++ {
		h := newTestHook(string(rune('a'+i)), OnConnect, OnPublish, OnSubscribe, OnDisconnect)
		_ = m.Add(h)
	}

	client := &Client{ID: "client1"}
	connectPacket := &ConnectPacket{ClientID: "client1"}
	publishPacket := &PublishPacket{Topic: "test", Payload: []byte("data")}
	sub := &Subscription{ClientID: "client1", TopicFilter: "test/#"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnConnect(client, connectPacket)
		_ = m.OnPublish(client, publishPacket)
		_ = m.OnSubscribe(client, sub)
		m.OnDisconnect(client, nil, false)
	}
}

func BenchmarkManagerParallelOnConnect(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnConnect)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = m.OnConnect(client, packet)
		}
	})
}

func BenchmarkManagerParallelOnPublish(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPublish)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello"),
		QoS:     1,
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = m.OnPublish(client, packet)
		}
	})
}

func BenchmarkManagerParallelOnACLCheck(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnACLCheck)
	_ = m.Add(h)

	client := &Client{ID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = m.OnACLCheck(client, "test/topic", AccessTypeWrite)
		}
	})
}

func BenchmarkManagerParallelAddRemove(b *testing.B) {
	m := NewManager()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := string(rune('a' + (i % 26)))
			h := &Base{id: id}
			_ = m.Add(h)
			_ = m.Remove(id)
			i++
		}
	})
}

func BenchmarkHookBaseOnConnect(b *testing.B) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.OnConnect(client, packet)
	}
}

func BenchmarkHookBaseOnPublish(b *testing.B) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello"),
		QoS:     1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.OnPublish(client, packet)
	}
}

func BenchmarkHookBaseOnPacketRead(b *testing.B) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := []byte{0x30, 0x0d, 0x00, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = h.OnPacketRead(client, packet)
	}
}

func BenchmarkHookBaseProvides(b *testing.B) {
	h := &Base{id: "test"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = h.Provides(OnConnect)
	}
}

func BenchmarkSubscribersAdd(b *testing.B) {
	subs := &Subscribers{}
	sub := &Subscription{
		ClientID:    "client1",
		TopicFilter: "test/#",
		QoS:         1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		subs.Add(sub)
		subs.Clear()
	}
}

func BenchmarkSubscribersRemove(b *testing.B) {
	subs := &Subscribers{}
	for i := 0; i < 100; i++ {
		sub := &Subscription{
			ClientID:    string(rune('a' + (i % 26))),
			TopicFilter: "test/#",
		}
		subs.Add(sub)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		clientID := string(rune('a' + (i % 26)))
		subs.Remove(clientID)
	}
}

func BenchmarkDropReasonString(b *testing.B) {
	reason := DropReasonQueueFull

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = reason.String()
	}
}

func BenchmarkEventString(b *testing.B) {
	event := OnPublish

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = event.String()
	}
}

func BenchmarkManagerList(b *testing.B) {
	m := NewManager()
	for i := 0; i < 10; i++ {
		h := &Base{id: string(rune('a' + i))}
		_ = m.Add(h)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.List()
	}
}

func BenchmarkManagerCount(b *testing.B) {
	m := NewManager()
	for i := 0; i < 10; i++ {
		h := &Base{id: string(rune('a' + i))}
		_ = m.Add(h)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.Count()
	}
}

func BenchmarkManagerClear(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m := NewManager()
		for j := 0; j < 10; j++ {
			h := &Base{id: string(rune('a' + j))}
			_ = m.Add(h)
		}
		m.Clear()
	}
}

func BenchmarkManagerOnSessionEstablish(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnSessionEstablish)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnSessionEstablish(client, packet)
	}
}

func BenchmarkManagerOnRetainMessage(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnRetainMessage)
	_ = m.Add(h)

	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:  "test/topic",
		Retain: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.OnRetainMessage(client, packet)
	}
}

func BenchmarkManagerOnPacketIDExhausted(b *testing.B) {
	m := NewManager()
	h := newTestHook("test", OnPacketIDExhausted)
	_ = m.Add(h)

	client := &Client{ID: "client1"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.OnPacketIDExhausted(client, encoding.PUBLISH)
	}
}

func BenchmarkManagerStorageOperations(b *testing.B) {
	m := NewManager()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = m.StoredClients()
		_, _ = m.StoredSubscriptions()
		_, _ = m.StoredInflightMessages()
		_, _ = m.StoredRetainedMessages()
		_, _ = m.StoredSysInfo()
	}
}
