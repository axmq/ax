package hook

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/axmq/ax/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHook struct {
	*Base
	events       map[Event]bool
	authResult   bool
	aclResult    bool
	initCalled   int
	stopCalled   int
	mu           sync.Mutex
	callCounts   map[string]int
	returnError  bool
	modifyPacket bool
}

func newTestHook(id string, events ...Event) *testHook {
	h := &testHook{
		Base:       &Base{id: id},
		events:     make(map[Event]bool),
		authResult: true,
		aclResult:  true,
		callCounts: make(map[string]int),
	}
	for _, e := range events {
		h.events[e] = true
	}
	return h
}

func (h *testHook) Provides(event Event) bool {
	return h.events[event]
}

func (h *testHook) Init(config any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.initCalled++
	if h.returnError {
		return errors.New("init error")
	}
	return nil
}

func (h *testHook) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopCalled++
	if h.returnError {
		return errors.New("stop error")
	}
	return nil
}

func (h *testHook) incrementCall(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callCounts[name]++
}

func (h *testHook) getCallCount(name string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callCounts[name]
}

func (h *testHook) OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool {
	h.incrementCall("OnConnectAuthenticate")
	return h.authResult
}

func (h *testHook) OnACLCheck(client *Client, topic string, access AccessType) bool {
	h.incrementCall("OnACLCheck")
	return h.aclResult
}

func (h *testHook) OnConnect(client *Client, packet *ConnectPacket) error {
	h.incrementCall("OnConnect")
	if h.returnError {
		return errors.New("connect error")
	}
	return nil
}

func (h *testHook) OnSessionEstablish(client *Client, packet *ConnectPacket) *SessionState {
	h.incrementCall("OnSessionEstablish")
	return &SessionState{ClientID: client.ID}
}

func (h *testHook) OnSessionEstablished(client *Client, packet *ConnectPacket) error {
	h.incrementCall("OnSessionEstablished")
	if h.returnError {
		return errors.New("session error")
	}
	return nil
}

func (h *testHook) OnDisconnect(client *Client, err error, expire bool) error {
	h.incrementCall("OnDisconnect")
	return nil
}

func (h *testHook) OnPacketRead(client *Client, packet []byte) ([]byte, error) {
	h.incrementCall("OnPacketRead")
	if h.returnError {
		return nil, errors.New("packet read error")
	}
	if h.modifyPacket {
		modified := make([]byte, len(packet)+1)
		copy(modified, packet)
		modified[len(packet)] = 0xFF
		return modified, nil
	}
	return packet, nil
}

func (h *testHook) OnPacketEncode(client *Client, packet []byte) []byte {
	h.incrementCall("OnPacketEncode")
	if h.modifyPacket {
		modified := make([]byte, len(packet)+1)
		copy(modified, packet)
		modified[len(packet)] = 0xEE
		return modified
	}
	return packet
}

func (h *testHook) OnPublish(client *Client, packet *PublishPacket) error {
	h.incrementCall("OnPublish")
	if h.returnError {
		return errors.New("publish error")
	}
	return nil
}

func (h *testHook) OnPublished(client *Client, packet *PublishPacket) error {
	h.incrementCall("OnPublished")
	return nil
}

func (h *testHook) OnSubscribe(client *Client, sub *Subscription) error {
	h.incrementCall("OnSubscribe")
	if h.returnError {
		return errors.New("subscribe error")
	}
	return nil
}

func (h *testHook) OnSubscribed(client *Client, sub *Subscription) error {
	h.incrementCall("OnSubscribed")
	return nil
}

func (h *testHook) OnUnsubscribe(client *Client, topicFilter string) error {
	h.incrementCall("OnUnsubscribe")
	if h.returnError {
		return errors.New("unsubscribe error")
	}
	return nil
}

func (h *testHook) OnUnsubscribed(client *Client, topicFilter string) error {
	h.incrementCall("OnUnsubscribed")
	return nil
}

func (h *testHook) OnPublishDropped(client *Client, packet *PublishPacket, reason DropReason) error {
	h.incrementCall("OnPublishDropped")
	return nil
}

func (h *testHook) OnRetainMessage(client *Client, packet *PublishPacket) error {
	h.incrementCall("OnRetainMessage")
	if h.returnError {
		return errors.New("retain error")
	}
	return nil
}

func (h *testHook) OnRetainPublished(client *Client, packet *PublishPacket) error {
	h.incrementCall("OnRetainPublished")
	return nil
}

func (h *testHook) OnQosPublish(client *Client, packet *PublishPacket, sent time.Time, resend int) error {
	h.incrementCall("OnQosPublish")
	return nil
}

func (h *testHook) OnQosComplete(client *Client, packetID uint16, packetType encoding.PacketType) error {
	h.incrementCall("OnQosComplete")
	return nil
}

func (h *testHook) OnQosDropped(client *Client, packetID uint16, reason DropReason) error {
	h.incrementCall("OnQosDropped")
	return nil
}

func (h *testHook) OnPacketSent(client *Client, packet []byte, count int, err error) error {
	h.incrementCall("OnPacketSent")
	return nil
}

func (h *testHook) OnPacketProcessed(client *Client, packetType encoding.PacketType, err error) error {
	h.incrementCall("OnPacketProcessed")
	return nil
}

func (h *testHook) OnStarted() error {
	h.incrementCall("OnStarted")
	return nil
}

func (h *testHook) OnStopped(err error) error {
	h.incrementCall("OnStopped")
	return nil
}

func (h *testHook) OnSysInfoTick(info *SysInfo) error {
	h.incrementCall("OnSysInfoTick")
	return nil
}

func (h *testHook) OnAuthPacket(client *Client, packet *AuthPacket) bool {
	h.incrementCall("OnAuthPacket")
	return h.authResult
}

func (h *testHook) OnSelectSubscribers(subscribers *Subscribers, topic string) error {
	h.incrementCall("OnSelectSubscribers")
	return nil
}

func (h *testHook) OnClientExpired(clientID string) error {
	h.incrementCall("OnClientExpired")
	return nil
}

func (h *testHook) OnRetainedExpired(topic string) error {
	h.incrementCall("OnRetainedExpired")
	return nil
}

func (h *testHook) OnPacketIDExhausted(client *Client, packetType encoding.PacketType) error {
	h.incrementCall("OnPacketIDExhausted")
	return nil
}

func (h *testHook) OnWillSent(client *Client, will *WillMessage) error {
	h.incrementCall("OnWillSent")
	return nil
}

func (h *testHook) OnWill(client *Client, will *WillMessage) *WillMessage {
	h.incrementCall("OnWill")
	if h.modifyPacket && will != nil {
		modified := *will
		modified.Topic = will.Topic + "/modified"
		return &modified
	}
	return will
}

func TestManagerAddHook(t *testing.T) {
	tests := []struct {
		name      string
		hook      Hook
		expectErr error
	}{
		{
			name:      "add valid hook",
			hook:      newTestHook("test1"),
			expectErr: nil,
		},
		{
			name:      "add nil hook",
			hook:      nil,
			expectErr: ErrEmptyHookID,
		},
		{
			name:      "add hook with empty id",
			hook:      &Base{id: ""},
			expectErr: ErrEmptyHookID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			err := m.Add(tt.hook)
			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, m.Count())
			}
		})
	}
}

func TestManagerAddDuplicateHook(t *testing.T) {
	m := NewManager()
	h1 := newTestHook("duplicate")
	h2 := newTestHook("duplicate")

	err := m.Add(h1)
	require.NoError(t, err)

	err = m.Add(h2)
	assert.ErrorIs(t, err, ErrHookAlreadyExists)
	assert.Equal(t, 1, m.Count())
}

func TestManagerRemoveHook(t *testing.T) {
	m := NewManager()
	h1 := newTestHook("hook1")
	h2 := newTestHook("hook2")
	h3 := newTestHook("hook3")

	require.NoError(t, m.Add(h1))
	require.NoError(t, m.Add(h2))
	require.NoError(t, m.Add(h3))
	assert.Equal(t, 3, m.Count())

	err := m.Remove("hook2")
	assert.NoError(t, err)
	assert.Equal(t, 2, m.Count())

	_, exists := m.Get("hook2")
	assert.False(t, exists)

	_, exists = m.Get("hook1")
	assert.True(t, exists)

	_, exists = m.Get("hook3")
	assert.True(t, exists)
}

func TestManagerRemoveNonExistentHook(t *testing.T) {
	m := NewManager()
	err := m.Remove("nonexistent")
	assert.ErrorIs(t, err, ErrHookNotFound)
}

func TestManagerGetHook(t *testing.T) {
	m := NewManager()
	h := newTestHook("test")
	require.NoError(t, m.Add(h))

	retrieved, exists := m.Get("test")
	assert.True(t, exists)
	assert.Equal(t, h, retrieved)

	_, exists = m.Get("nonexistent")
	assert.False(t, exists)
}

func TestManagerList(t *testing.T) {
	m := NewManager()
	h1 := newTestHook("hook1")
	h2 := newTestHook("hook2")

	require.NoError(t, m.Add(h1))
	require.NoError(t, m.Add(h2))

	list := m.List()
	assert.Len(t, list, 2)
	assert.Contains(t, list, h1)
	assert.Contains(t, list, h2)
}

func TestManagerClear(t *testing.T) {
	m := NewManager()
	h1 := newTestHook("hook1")
	h2 := newTestHook("hook2")

	require.NoError(t, m.Add(h1))
	require.NoError(t, m.Add(h2))
	assert.Equal(t, 2, m.Count())

	m.Clear()
	assert.Equal(t, 0, m.Count())
	assert.Equal(t, 1, h1.stopCalled)
	assert.Equal(t, 1, h2.stopCalled)
}

func TestManagerOnConnectAuthenticate(t *testing.T) {
	tests := []struct {
		name       string
		hooks      []*testHook
		expectAuth bool
	}{
		{
			name: "single hook allows",
			hooks: []*testHook{
				newTestHook("auth1", OnConnectAuthenticate),
			},
			expectAuth: true,
		},
		{
			name: "single hook denies",
			hooks: []*testHook{
				func() *testHook {
					h := newTestHook("auth1", OnConnectAuthenticate)
					h.authResult = false
					return h
				}(),
			},
			expectAuth: false,
		},
		{
			name: "multiple hooks all allow",
			hooks: []*testHook{
				newTestHook("auth1", OnConnectAuthenticate),
				newTestHook("auth2", OnConnectAuthenticate),
			},
			expectAuth: true,
		},
		{
			name: "multiple hooks one denies",
			hooks: []*testHook{
				newTestHook("auth1", OnConnectAuthenticate),
				func() *testHook {
					h := newTestHook("auth2", OnConnectAuthenticate)
					h.authResult = false
					return h
				}(),
			},
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			for _, h := range tt.hooks {
				require.NoError(t, m.Add(h))
			}

			client := &Client{ID: "client1"}
			packet := &ConnectPacket{ClientID: "client1"}

			result := m.OnConnectAuthenticate(client, packet)
			assert.Equal(t, tt.expectAuth, result)

			for _, h := range tt.hooks {
				if h.Provides(OnConnectAuthenticate) {
					assert.Equal(t, 1, h.getCallCount("OnConnectAuthenticate"))
				}
			}
		})
	}
}

func TestManagerOnACLCheck(t *testing.T) {
	tests := []struct {
		name      string
		hooks     []*testHook
		expectACL bool
	}{
		{
			name: "acl allows",
			hooks: []*testHook{
				newTestHook("acl1", OnACLCheck),
			},
			expectACL: true,
		},
		{
			name: "acl denies",
			hooks: []*testHook{
				func() *testHook {
					h := newTestHook("acl1", OnACLCheck)
					h.aclResult = false
					return h
				}(),
			},
			expectACL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			for _, h := range tt.hooks {
				require.NoError(t, m.Add(h))
			}

			client := &Client{ID: "client1"}
			result := m.OnACLCheck(client, "test/topic", AccessTypeWrite)
			assert.Equal(t, tt.expectACL, result)
		})
	}
}

func TestManagerOnConnect(t *testing.T) {
	m := NewManager()
	h := newTestHook("connect1", OnConnect)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	err := m.OnConnect(client, packet)
	assert.NoError(t, err)
	assert.Equal(t, 1, h.getCallCount("OnConnect"))
}

func TestManagerOnConnectError(t *testing.T) {
	m := NewManager()
	h := newTestHook("connect1", OnConnect)
	h.returnError = true
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	err := m.OnConnect(client, packet)
	assert.Error(t, err)
}

func TestManagerOnSessionEstablish(t *testing.T) {
	m := NewManager()
	h := newTestHook("session1", OnSessionEstablish)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	state := m.OnSessionEstablish(client, packet)
	assert.NotNil(t, state)
	assert.Equal(t, "client1", state.ClientID)
	assert.Equal(t, 1, h.getCallCount("OnSessionEstablish"))
}

func TestManagerOnDisconnect(t *testing.T) {
	m := NewManager()
	h := newTestHook("disconnect1", OnDisconnect)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}

	m.OnDisconnect(client, nil, false)
	assert.Equal(t, 1, h.getCallCount("OnDisconnect"))
}

func TestManagerOnPacketRead(t *testing.T) {
	tests := []struct {
		name         string
		hooks        []*testHook
		inputPacket  []byte
		expectPacket []byte
		expectError  bool
	}{
		{
			name: "single hook no modification",
			hooks: []*testHook{
				newTestHook("packet1", OnPacketRead),
			},
			inputPacket:  []byte{0x01, 0x02, 0x03},
			expectPacket: []byte{0x01, 0x02, 0x03},
			expectError:  false,
		},
		{
			name: "single hook with modification",
			hooks: []*testHook{
				func() *testHook {
					h := newTestHook("packet1", OnPacketRead)
					h.modifyPacket = true
					return h
				}(),
			},
			inputPacket:  []byte{0x01, 0x02, 0x03},
			expectPacket: []byte{0x01, 0x02, 0x03, 0xFF},
			expectError:  false,
		},
		{
			name: "multiple hooks with modifications",
			hooks: []*testHook{
				func() *testHook {
					h := newTestHook("packet1", OnPacketRead)
					h.modifyPacket = true
					return h
				}(),
				func() *testHook {
					h := newTestHook("packet2", OnPacketRead)
					h.modifyPacket = true
					return h
				}(),
			},
			inputPacket:  []byte{0x01},
			expectPacket: []byte{0x01, 0xFF, 0xFF},
			expectError:  false,
		},
		{
			name: "hook returns error",
			hooks: []*testHook{
				func() *testHook {
					h := newTestHook("packet1", OnPacketRead)
					h.returnError = true
					return h
				}(),
			},
			inputPacket:  []byte{0x01, 0x02},
			expectPacket: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			for _, h := range tt.hooks {
				require.NoError(t, m.Add(h))
			}

			client := &Client{ID: "client1"}
			result, err := m.OnPacketRead(client, tt.inputPacket)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectPacket, result)
			}
		})
	}
}

func TestManagerOnPacketEncode(t *testing.T) {
	m := NewManager()
	h := newTestHook("encode1", OnPacketEncode)
	h.modifyPacket = true
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	input := []byte{0x01, 0x02}
	result := m.OnPacketEncode(client, input)

	assert.Equal(t, []byte{0x01, 0x02, 0xEE}, result)
	assert.Equal(t, 1, h.getCallCount("OnPacketEncode"))
}

func TestManagerOnPublish(t *testing.T) {
	tests := []struct {
		name        string
		returnError bool
		expectError bool
	}{
		{
			name:        "publish success",
			returnError: false,
			expectError: false,
		},
		{
			name:        "publish error",
			returnError: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			h := newTestHook("publish1", OnPublish)
			h.returnError = tt.returnError
			require.NoError(t, m.Add(h))

			client := &Client{ID: "client1"}
			packet := &PublishPacket{Topic: "test/topic"}

			err := m.OnPublish(client, packet)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, h.getCallCount("OnPublish"))
		})
	}
}

func TestManagerOnSubscribe(t *testing.T) {
	m := NewManager()
	h := newTestHook("sub1", OnSubscribe)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	sub := &Subscription{ClientID: "client1", TopicFilter: "test/#"}

	err := m.OnSubscribe(client, sub)
	assert.NoError(t, err)
	assert.Equal(t, 1, h.getCallCount("OnSubscribe"))
}

func TestManagerOnWill(t *testing.T) {
	m := NewManager()
	h := newTestHook("will1", OnWill)
	h.modifyPacket = true
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	will := &WillMessage{Topic: "test/will"}

	result := m.OnWill(client, will)
	assert.NotNil(t, result)
	assert.Equal(t, "test/will/modified", result.Topic)
	assert.Equal(t, 1, h.getCallCount("OnWill"))
}

func TestManagerHookOrdering(t *testing.T) {
	m := NewManager()
	h1 := newTestHook("hook1", OnPublish)
	h2 := newTestHook("hook2", OnPublish)
	h3 := newTestHook("hook3", OnPublish)

	require.NoError(t, m.Add(h1))
	require.NoError(t, m.Add(h2))
	require.NoError(t, m.Add(h3))

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test"}

	err := m.OnPublish(client, packet)
	assert.NoError(t, err)

	assert.Equal(t, 1, h1.getCallCount("OnPublish"))
	assert.Equal(t, 1, h2.getCallCount("OnPublish"))
	assert.Equal(t, 1, h3.getCallCount("OnPublish"))
}

func TestManagerConcurrentAccess(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup

	numGoroutines := 100
	numOperations := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				hookID := string(rune('a' + (id % 26)))
				h := newTestHook(hookID, OnPublish)
				_ = m.Add(h)

				client := &Client{ID: "client1"}
				packet := &PublishPacket{Topic: "test"}
				_ = m.OnPublish(client, packet)

				_ = m.Remove(hookID)
			}
		}(i)
	}

	wg.Wait()
}

func TestManagerMultipleEventTypes(t *testing.T) {
	m := NewManager()
	h := newTestHook("multi", OnConnect, OnDisconnect, OnPublish, OnSubscribe)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	connectPacket := &ConnectPacket{ClientID: "client1"}
	publishPacket := &PublishPacket{Topic: "test"}
	sub := &Subscription{ClientID: "client1", TopicFilter: "test/#"}

	err := m.OnConnect(client, connectPacket)
	assert.NoError(t, err)

	err = m.OnPublish(client, publishPacket)
	assert.NoError(t, err)

	err = m.OnSubscribe(client, sub)
	assert.NoError(t, err)

	m.OnDisconnect(client, nil, false)

	assert.Equal(t, 1, h.getCallCount("OnConnect"))
	assert.Equal(t, 1, h.getCallCount("OnPublish"))
	assert.Equal(t, 1, h.getCallCount("OnSubscribe"))
	assert.Equal(t, 1, h.getCallCount("OnDisconnect"))
}

func TestManagerQosEvents(t *testing.T) {
	m := NewManager()
	h := newTestHook("qos", OnQosPublish, OnQosComplete, OnQosDropped)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test", QoS: 1}

	m.OnQosPublish(client, packet, time.Now(), 0)
	assert.Equal(t, 1, h.getCallCount("OnQosPublish"))

	m.OnQosComplete(client, 1, encoding.PUBACK)
	assert.Equal(t, 1, h.getCallCount("OnQosComplete"))

	m.OnQosDropped(client, 2, DropReasonQueueFull)
	assert.Equal(t, 1, h.getCallCount("OnQosDropped"))
}

func TestManagerStorageHooks(t *testing.T) {
	m := NewManager()

	clients, err := m.StoredClients()
	assert.NoError(t, err)
	assert.Nil(t, clients)

	subs, err := m.StoredSubscriptions()
	assert.NoError(t, err)
	assert.Nil(t, subs)

	inflight, err := m.StoredInflightMessages()
	assert.NoError(t, err)
	assert.Nil(t, inflight)

	retained, err := m.StoredRetainedMessages()
	assert.NoError(t, err)
	assert.Nil(t, retained)

	sysInfo, err := m.StoredSysInfo()
	assert.NoError(t, err)
	assert.Nil(t, sysInfo)
}

func TestManagerLifecycleHooks(t *testing.T) {
	m := NewManager()
	h := newTestHook("lifecycle", OnStarted, OnStopped, OnSysInfoTick)
	require.NoError(t, m.Add(h))

	m.OnStarted()
	assert.Equal(t, 1, h.getCallCount("OnStarted"))

	info := &SysInfo{Time: time.Now()}
	m.OnSysInfoTick(info)
	assert.Equal(t, 1, h.getCallCount("OnSysInfoTick"))

	m.OnStopped(nil)
	assert.Equal(t, 1, h.getCallCount("OnStopped"))
}

func TestManagerPacketEvents(t *testing.T) {
	m := NewManager()
	h := newTestHook("packet", OnPacketSent, OnPacketProcessed)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := []byte{0x01, 0x02}

	m.OnPacketSent(client, packet, len(packet), nil)
	assert.Equal(t, 1, h.getCallCount("OnPacketSent"))

	m.OnPacketProcessed(client, encoding.PUBLISH, nil)
	assert.Equal(t, 1, h.getCallCount("OnPacketProcessed"))
}

func TestManagerExpiryHooks(t *testing.T) {
	m := NewManager()
	h := newTestHook("expiry", OnClientExpired, OnRetainedExpired)
	require.NoError(t, m.Add(h))

	m.OnClientExpired("client1")
	assert.Equal(t, 1, h.getCallCount("OnClientExpired"))

	m.OnRetainedExpired("test/topic")
	assert.Equal(t, 1, h.getCallCount("OnRetainedExpired"))
}

func TestManagerEmptyHookList(t *testing.T) {
	m := NewManager()

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{ClientID: "client1"}

	result := m.OnConnectAuthenticate(client, packet)
	assert.True(t, result)

	err := m.OnConnect(client, packet)
	assert.NoError(t, err)

	m.OnDisconnect(client, nil, false)
}

func TestManagerSetOptions(t *testing.T) {
	m := NewManager()
	h := newTestHook("opts", SetOptions)
	require.NoError(t, m.Add(h))

	opts := &Options{
		Capabilities: &Capabilities{
			MaximumQoS: 2,
		},
	}

	err := m.SetOptions(opts)
	assert.NoError(t, err)
}

func TestManagerSubscriberSelection(t *testing.T) {
	m := NewManager()
	h := newTestHook("select", OnSelectSubscribers)
	require.NoError(t, m.Add(h))

	subscribers := &Subscribers{
		Subscriptions: []*Subscription{
			{ClientID: "client1", TopicFilter: "test/#"},
			{ClientID: "client2", TopicFilter: "test/+"},
		},
	}

	m.OnSelectSubscribers(subscribers, "test/topic")
	assert.Equal(t, 1, h.getCallCount("OnSelectSubscribers"))
}

func TestManagerRetainHooks(t *testing.T) {
	m := NewManager()
	h := newTestHook("retain", OnRetainMessage, OnRetainPublished)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test", Retain: true}

	err := m.OnRetainMessage(client, packet)
	assert.NoError(t, err)
	assert.Equal(t, 1, h.getCallCount("OnRetainMessage"))

	m.OnRetainPublished(client, packet)
	assert.Equal(t, 1, h.getCallCount("OnRetainPublished"))
}

func TestManagerWillHooks(t *testing.T) {
	m := NewManager()
	h := newTestHook("will", OnWill, OnWillSent)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	will := &WillMessage{Topic: "will/topic"}

	result := m.OnWill(client, will)
	assert.NotNil(t, result)
	assert.Equal(t, 1, h.getCallCount("OnWill"))

	m.OnWillSent(client, will)
	assert.Equal(t, 1, h.getCallCount("OnWillSent"))
}

func TestManagerAuthPacket(t *testing.T) {
	m := NewManager()
	h := newTestHook("auth", OnAuthPacket)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	authPacket := &AuthPacket{ReasonCode: 0}

	result := m.OnAuthPacket(client, authPacket)
	assert.True(t, result)
	assert.Equal(t, 1, h.getCallCount("OnAuthPacket"))
}

func TestManagerUnsubscribe(t *testing.T) {
	m := NewManager()
	h := newTestHook("unsub", OnUnsubscribe, OnUnsubscribed)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}

	err := m.OnUnsubscribe(client, "test/#")
	assert.NoError(t, err)
	assert.Equal(t, 1, h.getCallCount("OnUnsubscribe"))

	m.OnUnsubscribed(client, "test/#")
	assert.Equal(t, 1, h.getCallCount("OnUnsubscribed"))
}

func TestManagerPublishDropped(t *testing.T) {
	m := NewManager()
	h := newTestHook("drop", OnPublishDropped)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}
	packet := &PublishPacket{Topic: "test"}

	m.OnPublishDropped(client, packet, DropReasonQueueFull)
	assert.Equal(t, 1, h.getCallCount("OnPublishDropped"))
}

func TestManagerPacketIDExhausted(t *testing.T) {
	m := NewManager()
	h := newTestHook("exhausted", OnPacketIDExhausted)
	require.NoError(t, m.Add(h))

	client := &Client{ID: "client1"}

	m.OnPacketIDExhausted(client, encoding.PUBLISH)
	assert.Equal(t, 1, h.getCallCount("OnPacketIDExhausted"))
}

func TestClientStateConstant(t *testing.T) {
	assert.Equal(t, ClientState(0), ClientStateConnecting)
	assert.Equal(t, ClientState(1), ClientStateConnected)
	assert.Equal(t, ClientState(2), ClientStateDisconnecting)
	assert.Equal(t, ClientState(3), ClientStateDisconnected)
}

func TestAccessTypeConstant(t *testing.T) {
	assert.Equal(t, AccessType(0), AccessTypeRead)
	assert.Equal(t, AccessType(1), AccessTypeWrite)
	assert.Equal(t, AccessType(2), AccessTypeReadWrite)
}

func TestDropReasonString(t *testing.T) {
	tests := []struct {
		reason   DropReason
		expected string
	}{
		{DropReasonQueueFull, "queue_full"},
		{DropReasonClientDisconnected, "client_disconnected"},
		{DropReasonExpired, "expired"},
		{DropReasonInvalidTopic, "invalid_topic"},
		{DropReasonACLDenied, "acl_denied"},
		{DropReasonQuotaExceeded, "quota_exceeded"},
		{DropReasonPacketTooLarge, "packet_too_large"},
		{DropReasonInternalError, "internal_error"},
		{DropReason(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.reason.String())
		})
	}
}

func TestEventString(t *testing.T) {
	assert.Equal(t, "SetOptions", SetOptions.String())
	assert.Equal(t, "OnStarted", OnStarted.String())
	assert.Equal(t, "OnConnectAuthenticate", OnConnectAuthenticate.String())
	assert.Equal(t, "OnPublish", OnPublish.String())
	assert.Equal(t, "StoredSysInfo", StoredSysInfo.String())
	assert.Equal(t, "Unknown", Event(99).String())
}

func TestSubscribersHelpers(t *testing.T) {
	subs := &Subscribers{}

	sub1 := &Subscription{ClientID: "client1", TopicFilter: "test/#"}
	sub2 := &Subscription{ClientID: "client2", TopicFilter: "test/+"}
	sub3 := &Subscription{ClientID: "client3", TopicFilter: "test/topic"}

	subs.Add(sub1)
	subs.Add(sub2)
	subs.Add(sub3)
	assert.Len(t, subs.Subscriptions, 3)

	subs.Remove("client2")
	assert.Len(t, subs.Subscriptions, 2)
	assert.NotContains(t, subs.Subscriptions, sub2)

	subs.Clear()
	assert.Len(t, subs.Subscriptions, 0)
}

func TestManagerWithRealNetAddr(t *testing.T) {
	m := NewManager()
	h := newTestHook("test", OnConnect)
	require.NoError(t, m.Add(h))

	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 1883,
	}

	client := &Client{
		ID:         "client1",
		RemoteAddr: addr,
		LocalAddr:  addr,
	}

	packet := &ConnectPacket{ClientID: "client1"}
	err := m.OnConnect(client, packet)
	assert.NoError(t, err)
}
