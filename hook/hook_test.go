package hook

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientStructure(t *testing.T) {
	now := time.Now()
	client := &Client{
		ID:              "test-client",
		RemoteAddr:      &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1883},
		LocalAddr:       &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1883},
		Username:        "testuser",
		CleanStart:      true,
		ProtocolVersion: 5,
		KeepAlive:       60,
		SessionPresent:  false,
		Properties:      Properties{"key": "value"},
		Will:            &WillMessage{Topic: "will/topic"},
		ConnectedAt:     now,
		DisconnectedAt:  now,
		State:           ClientStateConnected,
	}

	assert.Equal(t, "test-client", client.ID)
	assert.Equal(t, "testuser", client.Username)
	assert.True(t, client.CleanStart)
	assert.Equal(t, byte(5), client.ProtocolVersion)
	assert.Equal(t, uint16(60), client.KeepAlive)
	assert.Equal(t, ClientStateConnected, client.State)
}

func TestConnectPacketStructure(t *testing.T) {
	packet := &ConnectPacket{
		ProtocolName:    "MQTT",
		ProtocolVersion: 5,
		CleanStart:      true,
		KeepAlive:       60,
		ClientID:        "client1",
		Username:        "user",
		Password:        []byte("pass"),
		Will: &WillMessage{
			Topic:   "will/topic",
			Payload: []byte("offline"),
			QoS:     1,
			Retain:  true,
		},
		Properties:     Properties{"key": "value"},
		SessionPresent: false,
	}

	assert.Equal(t, "MQTT", packet.ProtocolName)
	assert.Equal(t, byte(5), packet.ProtocolVersion)
	assert.True(t, packet.CleanStart)
	assert.Equal(t, "client1", packet.ClientID)
	assert.NotNil(t, packet.Will)
}

func TestAuthPacketStructure(t *testing.T) {
	packet := &AuthPacket{
		ReasonCode: 0,
		Properties: Properties{"AuthMethod": "SCRAM"},
		AuthMethod: "SCRAM",
		AuthData:   []byte("authdata"),
	}

	assert.Equal(t, byte(0), packet.ReasonCode)
	assert.Equal(t, "SCRAM", packet.AuthMethod)
	assert.NotNil(t, packet.AuthData)
}

func TestPublishPacketStructure(t *testing.T) {
	now := time.Now()
	packet := &PublishPacket{
		PacketID:        1,
		Topic:           "test/topic",
		Payload:         []byte("hello world"),
		QoS:             1,
		Retain:          true,
		Duplicate:       false,
		Properties:      Properties{"ContentType": "text/plain"},
		ProtocolVersion: 5,
		Created:         now,
		Origin:          "client1",
	}

	assert.Equal(t, uint16(1), packet.PacketID)
	assert.Equal(t, "test/topic", packet.Topic)
	assert.Equal(t, []byte("hello world"), packet.Payload)
	assert.Equal(t, byte(1), packet.QoS)
	assert.True(t, packet.Retain)
	assert.False(t, packet.Duplicate)
}

func TestSubscriptionStructure(t *testing.T) {
	now := time.Now()
	sub := &Subscription{
		ClientID:               "client1",
		TopicFilter:            "test/#",
		QoS:                    2,
		NoLocal:                true,
		RetainAsPublished:      true,
		RetainHandling:         1,
		SubscriptionIdentifier: 123,
		SubscribedAt:           now,
	}

	assert.Equal(t, "client1", sub.ClientID)
	assert.Equal(t, "test/#", sub.TopicFilter)
	assert.Equal(t, byte(2), sub.QoS)
	assert.True(t, sub.NoLocal)
	assert.True(t, sub.RetainAsPublished)
	assert.Equal(t, byte(1), sub.RetainHandling)
	assert.Equal(t, uint32(123), sub.SubscriptionIdentifier)
}

func TestWillMessageStructure(t *testing.T) {
	will := &WillMessage{
		Topic:             "will/topic",
		Payload:           []byte("client offline"),
		QoS:               1,
		Retain:            true,
		Properties:        Properties{"key": "value"},
		WillDelayInterval: 30,
	}

	assert.Equal(t, "will/topic", will.Topic)
	assert.Equal(t, []byte("client offline"), will.Payload)
	assert.Equal(t, byte(1), will.QoS)
	assert.True(t, will.Retain)
	assert.Equal(t, uint32(30), will.WillDelayInterval)
}

func TestSessionStateStructure(t *testing.T) {
	state := &SessionState{
		ClientID:       "client1",
		CleanStart:     false,
		SessionPresent: true,
		ExpiryInterval: 3600,
		Subscriptions: map[string]*Subscription{
			"test/#": {ClientID: "client1", TopicFilter: "test/#"},
		},
		PendingMessages: []*InflightMessage{
			{PacketID: 1, Topic: "test/topic"},
		},
		NextPacketID: 2,
	}

	assert.Equal(t, "client1", state.ClientID)
	assert.False(t, state.CleanStart)
	assert.True(t, state.SessionPresent)
	assert.Equal(t, uint32(3600), state.ExpiryInterval)
	assert.Len(t, state.Subscriptions, 1)
	assert.Len(t, state.PendingMessages, 1)
	assert.Equal(t, uint16(2), state.NextPacketID)
}

func TestInflightMessageStructure(t *testing.T) {
	now := time.Now()
	msg := &InflightMessage{
		PacketID:    1,
		ClientID:    "client1",
		Topic:       "test/topic",
		Payload:     []byte("data"),
		QoS:         1,
		Retain:      false,
		Duplicate:   false,
		Properties:  Properties{"key": "value"},
		Sent:        now,
		ResendCount: 0,
	}

	assert.Equal(t, uint16(1), msg.PacketID)
	assert.Equal(t, "client1", msg.ClientID)
	assert.Equal(t, "test/topic", msg.Topic)
	assert.Equal(t, byte(1), msg.QoS)
	assert.Equal(t, 0, msg.ResendCount)
}

func TestRetainedMessageStructure(t *testing.T) {
	now := time.Now()
	msg := &RetainedMessage{
		Topic:      "test/topic",
		Payload:    []byte("retained data"),
		QoS:        1,
		Properties: Properties{"key": "value"},
		Timestamp:  now,
	}

	assert.Equal(t, "test/topic", msg.Topic)
	assert.Equal(t, []byte("retained data"), msg.Payload)
	assert.Equal(t, byte(1), msg.QoS)
	assert.NotNil(t, msg.Properties)
}

func TestOptionsStructure(t *testing.T) {
	opts := &Options{
		Capabilities: &Capabilities{
			MaximumSessionExpiryInterval: 86400,
			MaximumMessageExpiryInterval: 3600,
			ReceiveMaximum:               100,
			MaximumQoS:                   2,
			RetainAvailable:              true,
			MaximumPacketSize:            268435456,
			MaximumTopicAlias:            10,
			WildcardSubAvailable:         true,
			SubIDAvailable:               true,
			SharedSubAvailable:           true,
		},
		Config: map[string]any{
			"key": "value",
		},
	}

	assert.Equal(t, uint32(86400), opts.Capabilities.MaximumSessionExpiryInterval)
	assert.Equal(t, uint16(100), opts.Capabilities.ReceiveMaximum)
	assert.Equal(t, byte(2), opts.Capabilities.MaximumQoS)
	assert.True(t, opts.Capabilities.RetainAvailable)
}

func TestSysInfoStructure(t *testing.T) {
	now := time.Now()
	started := now.Add(-time.Hour)
	info := &SysInfo{
		Uptime:              3600,
		Version:             "1.0.0",
		Started:             started,
		Time:                now,
		ClientsConnected:    100,
		ClientsTotal:        1000,
		ClientsMaximum:      150,
		ClientsDisconnected: 50,
		MessagesReceived:    10000,
		MessagesSent:        9500,
		MessagesDropped:     50,
		Subscriptions:       500,
		Retained:            100,
		Inflight:            20,
		MemoryAlloc:         1048576,
		Threads:             8,
	}

	assert.Equal(t, int64(3600), info.Uptime)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, int64(100), info.ClientsConnected)
	assert.Equal(t, int64(10000), info.MessagesReceived)
}

func TestPropertiesType(t *testing.T) {
	props := Properties{
		"key1":   "value1",
		"key2":   123,
		"key3":   true,
		"nested": map[string]interface{}{"inner": "value"},
	}

	assert.Equal(t, "value1", props["key1"])
	assert.Equal(t, 123, props["key2"])
	assert.Equal(t, true, props["key3"])
	assert.NotNil(t, props["nested"])
}

func TestSubscribersOperations(t *testing.T) {
	subs := &Subscribers{
		Subscriptions: []*Subscription{
			{ClientID: "client1", TopicFilter: "test/#"},
			{ClientID: "client2", TopicFilter: "test/+"},
		},
	}

	assert.Len(t, subs.Subscriptions, 2)

	subs.Add(&Subscription{ClientID: "client3", TopicFilter: "test/topic"})
	assert.Len(t, subs.Subscriptions, 3)

	subs.Remove("client2")
	assert.Len(t, subs.Subscriptions, 2)

	found := false
	for _, sub := range subs.Subscriptions {
		if sub.ClientID == "client2" {
			found = true
		}
	}
	assert.False(t, found)

	subs.Clear()
	assert.Len(t, subs.Subscriptions, 0)
}

func TestClientStateValues(t *testing.T) {
	states := []ClientState{
		ClientStateConnecting,
		ClientStateConnected,
		ClientStateDisconnecting,
		ClientStateDisconnected,
	}

	for i, state := range states {
		assert.Equal(t, ClientState(i), state)
	}
}

func TestAccessTypeValues(t *testing.T) {
	types := []AccessType{
		AccessTypeRead,
		AccessTypeWrite,
		AccessTypeReadWrite,
	}

	for i, accessType := range types {
		assert.Equal(t, AccessType(i), accessType)
	}
}

func TestDropReasonValues(t *testing.T) {
	reasons := []DropReason{
		DropReasonQueueFull,
		DropReasonClientDisconnected,
		DropReasonExpired,
		DropReasonInvalidTopic,
		DropReasonACLDenied,
		DropReasonQuotaExceeded,
		DropReasonPacketTooLarge,
		DropReasonInternalError,
	}

	for i, reason := range reasons {
		assert.Equal(t, DropReason(i), reason)
	}
}

func TestEventValues(t *testing.T) {
	events := []Event{
		SetOptions,
		OnSysInfoTick,
		OnStarted,
		OnStopped,
		OnConnectAuthenticate,
		OnACLCheck,
		OnConnect,
		OnSessionEstablish,
		OnSessionEstablished,
		OnDisconnect,
		OnAuthPacket,
		OnPacketRead,
		OnPacketEncode,
		OnPacketSent,
		OnPacketProcessed,
		OnSubscribe,
		OnSubscribed,
		OnSelectSubscribers,
		OnUnsubscribe,
		OnUnsubscribed,
		OnPublish,
		OnPublished,
		OnPublishDropped,
		OnRetainMessage,
		OnRetainPublished,
		OnQosPublish,
		OnQosComplete,
		OnQosDropped,
		OnPacketIDExhausted,
		OnWill,
		OnWillSent,
		OnClientExpired,
		OnRetainedExpired,
		StoredClients,
		StoredSubscriptions,
		StoredInflightMessages,
		StoredRetainedMessages,
		StoredSysInfo,
	}

	for i, event := range events {
		assert.Equal(t, Event(i), event)
	}
}

func TestEmptyStructures(t *testing.T) {
	client := &Client{}
	assert.Equal(t, "", client.ID)

	packet := &ConnectPacket{}
	assert.Equal(t, "", packet.ClientID)

	sub := &Subscription{}
	assert.Equal(t, "", sub.ClientID)

	will := &WillMessage{}
	assert.Equal(t, "", will.Topic)

	props := Properties{}
	assert.Len(t, props, 0)
}

func TestNilHandling(t *testing.T) {
	var client *Client
	assert.Nil(t, client)

	var packet *ConnectPacket
	assert.Nil(t, packet)

	var sub *Subscription
	assert.Nil(t, sub)

	var will *WillMessage
	assert.Nil(t, will)

	var state *SessionState
	assert.Nil(t, state)
}

func TestPropertiesNilSafe(t *testing.T) {
	var props Properties
	assert.Nil(t, props)

	props = make(Properties)
	assert.NotNil(t, props)
	assert.Len(t, props, 0)
}

func TestSubscribersNilSafe(t *testing.T) {
	subs := &Subscribers{}
	assert.NotNil(t, subs)
	assert.Nil(t, subs.Subscriptions)

	subs.Add(&Subscription{ClientID: "test"})
	assert.NotNil(t, subs.Subscriptions)
}

func TestComplexScenario(t *testing.T) {
	client := &Client{
		ID:              "mqtt-client-123",
		RemoteAddr:      &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 54321},
		LocalAddr:       &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1883},
		Username:        "user@example.com",
		CleanStart:      false,
		ProtocolVersion: 5,
		KeepAlive:       300,
		SessionPresent:  true,
		Properties: Properties{
			"SessionExpiryInterval": uint32(3600),
			"ReceiveMaximum":        uint16(100),
		},
		Will: &WillMessage{
			Topic:             "device/mqtt-client-123/status",
			Payload:           []byte(`{"status":"offline","timestamp":1234567890}`),
			QoS:               1,
			Retain:            true,
			WillDelayInterval: 10,
			Properties: Properties{
				"MessageExpiryInterval": uint32(300),
			},
		},
		ConnectedAt: time.Now(),
		State:       ClientStateConnected,
	}

	assert.NotNil(t, client)
	assert.Equal(t, "mqtt-client-123", client.ID)
	assert.Equal(t, byte(5), client.ProtocolVersion)
	assert.NotNil(t, client.Will)
	assert.Equal(t, "device/mqtt-client-123/status", client.Will.Topic)
	assert.Equal(t, ClientStateConnected, client.State)
}
