package hook

import (
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/stretchr/testify/assert"
)

func TestHookBaseID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{
			name: "simple id",
			id:   "test-hook",
		},
		{
			name: "empty id",
			id:   "",
		},
		{
			name: "complex id",
			id:   "my.custom.hook.v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Base{id: tt.id}
			assert.Equal(t, tt.id, h.ID())
		})
	}
}

func TestHookBaseProvides(t *testing.T) {
	h := &Base{id: "test"}

	allEvents := []Event{
		SetOptions, OnSysInfoTick, OnStarted, OnStopped,
		OnConnectAuthenticate, OnACLCheck, OnConnect,
		OnSessionEstablish, OnSessionEstablished, OnDisconnect,
		OnAuthPacket, OnPacketRead, OnPacketEncode,
		OnPacketSent, OnPacketProcessed, OnSubscribe,
		OnSubscribed, OnSelectSubscribers, OnUnsubscribe,
		OnUnsubscribed, OnPublish, OnPublished,
		OnPublishDropped, OnRetainMessage, OnRetainPublished,
		OnQosPublish, OnQosComplete, OnQosDropped,
		OnPacketIDExhausted, OnWill, OnWillSent,
		OnClientExpired, OnRetainedExpired, StoredClients,
		StoredSubscriptions, StoredInflightMessages,
		StoredRetainedMessages, StoredSysInfo,
	}

	for _, event := range allEvents {
		assert.False(t, h.Provides(event))
	}
}

func TestHookBaseInit(t *testing.T) {
	h := &Base{id: "test"}

	err := h.Init(nil)
	assert.NoError(t, err)

	err = h.Init(map[string]interface{}{"key": "value"})
	assert.NoError(t, err)

	err = h.Init("string config")
	assert.NoError(t, err)
}

func TestHookBaseStop(t *testing.T) {
	h := &Base{id: "test"}
	err := h.Stop()
	assert.NoError(t, err)
}

func TestHookBaseSetOptions(t *testing.T) {
	h := &Base{id: "test"}
	opts := &Options{
		Capabilities: &Capabilities{
			MaximumQoS:      2,
			RetainAvailable: true,
		},
	}
	err := h.SetOptions(opts)
	assert.NoError(t, err)
}

func TestHookBaseOnSysInfoTick(t *testing.T) {
	h := &Base{id: "test"}
	info := &SysInfo{
		Time:             time.Now(),
		ClientsConnected: 10,
		MessagesReceived: 100,
	}
	err := h.OnSysInfoTick(info)
	assert.NoError(t, err)
}

func TestHookBaseOnStarted(t *testing.T) {
	h := &Base{id: "test"}
	err := h.OnStarted()
	assert.NoError(t, err)
}

func TestHookBaseOnStopped(t *testing.T) {
	h := &Base{id: "test"}
	err := h.OnStopped(nil)
	assert.NoError(t, err)

	err = h.OnStopped(assert.AnError)
	assert.NoError(t, err)
}

func TestHookBaseOnConnectAuthenticate(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	result := h.OnConnectAuthenticate(client, packet)
	assert.True(t, result)
}

func TestHookBaseOnACLCheck(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	tests := []struct {
		topic  string
		access AccessType
	}{
		{"test/topic", AccessTypeRead},
		{"test/topic", AccessTypeWrite},
		{"test/topic", AccessTypeReadWrite},
	}

	for _, tt := range tests {
		result := h.OnACLCheck(client, tt.topic, tt.access)
		assert.True(t, result)
	}
}

func TestHookBaseOnConnect(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	err := h.OnConnect(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnSessionEstablish(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	state := h.OnSessionEstablish(client, packet)
	assert.Nil(t, state)
}

func TestHookBaseOnSessionEstablished(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	err := h.OnSessionEstablished(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnDisconnect(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnDisconnect(client, nil, false)
	assert.NoError(t, err)

	err = h.OnDisconnect(client, assert.AnError, true)
	assert.NoError(t, err)
}

func TestHookBaseOnAuthPacket(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &AuthPacket{ReasonCode: 0}

	result := h.OnAuthPacket(client, packet)
	assert.True(t, result)
}

func TestHookBaseOnPacketRead(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := []byte{0x01, 0x02, 0x03}

	result, err := h.OnPacketRead(client, packet)
	assert.NoError(t, err)
	assert.Equal(t, packet, result)
}

func TestHookBaseOnPacketEncode(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := []byte{0x01, 0x02, 0x03}

	result := h.OnPacketEncode(client, packet)
	assert.Equal(t, packet, result)
}

func TestHookBaseOnPacketSent(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := []byte{0x01, 0x02, 0x03}

	err := h.OnPacketSent(client, packet, len(packet), nil)
	assert.NoError(t, err)

	err = h.OnPacketSent(client, packet, len(packet), assert.AnError)
	assert.NoError(t, err)
}

func TestHookBaseOnPacketProcessed(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnPacketProcessed(client, encoding.PUBLISH, nil)
	assert.NoError(t, err)

	err = h.OnPacketProcessed(client, encoding.CONNECT, assert.AnError)
	assert.NoError(t, err)
}

func TestHookBaseOnSubscribe(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	sub := &Subscription{
		ClientID:    "client1",
		TopicFilter: "test/#",
		QoS:         1,
	}

	err := h.OnSubscribe(client, sub)
	assert.NoError(t, err)
}

func TestHookBaseOnSubscribed(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	sub := &Subscription{
		ClientID:    "client1",
		TopicFilter: "test/#",
		QoS:         1,
	}

	err := h.OnSubscribed(client, sub)
	assert.NoError(t, err)
}

func TestHookBaseOnSelectSubscribers(t *testing.T) {
	h := &Base{id: "test"}
	subscribers := &Subscribers{
		Subscriptions: []*Subscription{
			{ClientID: "client1", TopicFilter: "test/#"},
		},
	}

	err := h.OnSelectSubscribers(subscribers, "test/topic")
	assert.NoError(t, err)
}

func TestHookBaseOnUnsubscribe(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnUnsubscribe(client, "test/#")
	assert.NoError(t, err)
}

func TestHookBaseOnUnsubscribed(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnUnsubscribed(client, "test/#")
	assert.NoError(t, err)
}

func TestHookBaseOnPublish(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello"),
		QoS:     1,
	}

	err := h.OnPublish(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnPublished(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:   "test/topic",
		Payload: []byte("hello"),
	}

	err := h.OnPublished(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnPublishDropped(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test/topic"}

	allReasons := []DropReason{
		DropReasonQueueFull,
		DropReasonClientDisconnected,
		DropReasonExpired,
		DropReasonInvalidTopic,
		DropReasonACLDenied,
		DropReasonQuotaExceeded,
		DropReasonPacketTooLarge,
		DropReasonInternalError,
	}

	for _, reason := range allReasons {
		err := h.OnPublishDropped(client, packet, reason)
		assert.NoError(t, err)
	}
}

func TestHookBaseOnRetainMessage(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:  "test/topic",
		Retain: true,
	}

	err := h.OnRetainMessage(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnRetainPublished(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic:  "test/topic",
		Retain: true,
	}

	err := h.OnRetainPublished(client, packet)
	assert.NoError(t, err)
}

func TestHookBaseOnQosPublish(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &PublishPacket{
		Topic: "test/topic",
		QoS:   1,
	}

	err := h.OnQosPublish(client, packet, time.Now(), 0)
	assert.NoError(t, err)

	err = h.OnQosPublish(client, packet, time.Now(), 3)
	assert.NoError(t, err)
}

func TestHookBaseOnQosComplete(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	packetTypes := []encoding.PacketType{
		encoding.PUBACK,
		encoding.PUBREC,
		encoding.PUBREL,
		encoding.PUBCOMP,
	}

	for _, pt := range packetTypes {
		err := h.OnQosComplete(client, 1, pt)
		assert.NoError(t, err)
	}
}

func TestHookBaseOnQosDropped(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnQosDropped(client, 1, DropReasonExpired)
	assert.NoError(t, err)
}

func TestHookBaseOnPacketIDExhausted(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}

	err := h.OnPacketIDExhausted(client, encoding.PUBLISH)
	assert.NoError(t, err)
}

func TestHookBaseOnWill(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	will := &WillMessage{
		Topic:   "will/topic",
		Payload: []byte("offline"),
		QoS:     1,
	}

	result := h.OnWill(client, will)
	assert.Equal(t, will, result)
}

func TestHookBaseOnWillSent(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	will := &WillMessage{
		Topic:   "will/topic",
		Payload: []byte("offline"),
	}

	err := h.OnWillSent(client, will)
	assert.NoError(t, err)
}

func TestHookBaseOnClientExpired(t *testing.T) {
	h := &Base{id: "test"}

	err := h.OnClientExpired("client1")
	assert.NoError(t, err)
}

func TestHookBaseOnRetainedExpired(t *testing.T) {
	h := &Base{id: "test"}

	err := h.OnRetainedExpired("test/topic")
	assert.NoError(t, err)
}

func TestHookBaseStoredClients(t *testing.T) {
	h := &Base{id: "test"}

	clients, err := h.StoredClients()
	assert.NoError(t, err)
	assert.Nil(t, clients)
}

func TestHookBaseStoredSubscriptions(t *testing.T) {
	h := &Base{id: "test"}

	subs, err := h.StoredSubscriptions()
	assert.NoError(t, err)
	assert.Nil(t, subs)
}

func TestHookBaseStoredInflightMessages(t *testing.T) {
	h := &Base{id: "test"}

	inflight, err := h.StoredInflightMessages()
	assert.NoError(t, err)
	assert.Nil(t, inflight)
}

func TestHookBaseStoredRetainedMessages(t *testing.T) {
	h := &Base{id: "test"}

	retained, err := h.StoredRetainedMessages()
	assert.NoError(t, err)
	assert.Nil(t, retained)
}

func TestHookBaseStoredSysInfo(t *testing.T) {
	h := &Base{id: "test"}

	sysInfo, err := h.StoredSysInfo()
	assert.NoError(t, err)
	assert.Nil(t, sysInfo)
}

func TestHookBaseNilInputs(t *testing.T) {
	h := &Base{id: "test"}

	err := h.OnConnect(nil, nil)
	assert.NoError(t, err)

	err = h.OnDisconnect(nil, nil, false)
	assert.NoError(t, err)

	err = h.OnPublish(nil, nil)
	assert.NoError(t, err)

	err = h.OnSubscribe(nil, nil)
	assert.NoError(t, err)

	result := h.OnConnectAuthenticate(nil, nil)
	assert.True(t, result)

	state := h.OnSessionEstablish(nil, nil)
	assert.Nil(t, state)

	will := h.OnWill(nil, nil)
	assert.Nil(t, will)
}

func TestHookBaseAllMethodsNoOp(t *testing.T) {
	h := &Base{id: "test"}
	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	err := h.Init(nil)
	assert.NoError(t, err)

	err = h.Stop()
	assert.NoError(t, err)

	err = h.SetOptions(nil)
	assert.NoError(t, err)

	err = h.OnSysInfoTick(nil)
	assert.NoError(t, err)

	err = h.OnStarted()
	assert.NoError(t, err)

	err = h.OnStopped(nil)
	assert.NoError(t, err)

	_ = h.OnConnectAuthenticate(client, packet)
	_ = h.OnACLCheck(client, "topic", AccessTypeRead)

	err = h.OnConnect(client, packet)
	assert.NoError(t, err)

	_ = h.OnSessionEstablish(client, packet)

	err = h.OnSessionEstablished(client, packet)
	assert.NoError(t, err)

	err = h.OnDisconnect(client, nil, false)
	assert.NoError(t, err)

	_ = h.OnAuthPacket(client, nil)

	_, err = h.OnPacketRead(client, []byte{})
	assert.NoError(t, err)

	_ = h.OnPacketEncode(client, []byte{})

	err = h.OnPacketSent(client, []byte{}, 0, nil)
	assert.NoError(t, err)

	err = h.OnPacketProcessed(client, encoding.PUBLISH, nil)
	assert.NoError(t, err)

	err = h.OnSubscribe(client, nil)
	assert.NoError(t, err)

	err = h.OnSubscribed(client, nil)
	assert.NoError(t, err)

	err = h.OnSelectSubscribers(nil, "")
	assert.NoError(t, err)

	err = h.OnUnsubscribe(client, "")
	assert.NoError(t, err)

	err = h.OnUnsubscribed(client, "")
	assert.NoError(t, err)

	err = h.OnPublish(client, nil)
	assert.NoError(t, err)

	err = h.OnPublished(client, nil)
	assert.NoError(t, err)

	err = h.OnPublishDropped(client, nil, DropReasonQueueFull)
	assert.NoError(t, err)

	err = h.OnRetainMessage(client, nil)
	assert.NoError(t, err)

	err = h.OnRetainPublished(client, nil)
	assert.NoError(t, err)

	err = h.OnQosPublish(client, nil, time.Now(), 0)
	assert.NoError(t, err)

	err = h.OnQosComplete(client, 0, encoding.PUBACK)
	assert.NoError(t, err)

	err = h.OnQosDropped(client, 0, DropReasonExpired)
	assert.NoError(t, err)

	err = h.OnPacketIDExhausted(client, encoding.PUBLISH)
	assert.NoError(t, err)

	_ = h.OnWill(client, nil)

	err = h.OnWillSent(client, nil)
	assert.NoError(t, err)

	err = h.OnClientExpired("")
	assert.NoError(t, err)

	err = h.OnRetainedExpired("")
	assert.NoError(t, err)

	_, err = h.StoredClients()
	assert.NoError(t, err)

	_, err = h.StoredSubscriptions()
	assert.NoError(t, err)

	_, err = h.StoredInflightMessages()
	assert.NoError(t, err)

	_, err = h.StoredRetainedMessages()
	assert.NoError(t, err)

	_, err = h.StoredSysInfo()
	assert.NoError(t, err)
}
