package hook

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/axmq/ax/encoding"
)

// Manager manages the registration and invocation of hooks
type Manager struct {
	mu       sync.Mutex
	hooksPtr atomic.Pointer[[]Hook]
	index    map[string]int
}

// NewManager creates a new hooks manager
func NewManager() *Manager {
	m := &Manager{
		index: make(map[string]int),
	}
	hooks := make([]Hook, 0)
	m.hooksPtr.Store(&hooks)
	return m
}

// Add adds a hook to the manager
// Returns an error if a hook with the same ID already exists
func (m *Manager) Add(hook Hook) error {
	if hook == nil {
		return ErrEmptyHookID
	}

	id := hook.ID()
	if id == "" {
		return ErrEmptyHookID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.index[id]; exists {
		return ErrHookAlreadyExists
	}

	// Copy-on-write: create new slice with added hook
	oldHooks := *m.hooksPtr.Load()
	newHooks := make([]Hook, len(oldHooks)+1)
	copy(newHooks, oldHooks)
	newHooks[len(oldHooks)] = hook

	m.index[id] = len(oldHooks)
	m.hooksPtr.Store(&newHooks)

	return nil
}

// Remove removes a hook by its ID
// Returns an error if the hook is not found
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.index[id]
	if !exists {
		return ErrHookNotFound
	}

	// Copy-on-write: create new slice without removed hook
	oldHooks := *m.hooksPtr.Load()
	newHooks := make([]Hook, len(oldHooks)-1)
	copy(newHooks[:idx], oldHooks[:idx])
	copy(newHooks[idx:], oldHooks[idx+1:])

	delete(m.index, id)

	// Rebuild index for hooks after removed position
	for i := idx; i < len(newHooks); i++ {
		m.index[newHooks[i].ID()] = i
	}

	m.hooksPtr.Store(&newHooks)

	return nil
}

// Get retrieves a hook by its ID
func (m *Manager) Get(id string) (Hook, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.index[id]
	if !exists {
		return nil, false
	}

	hooks := *m.hooksPtr.Load()
	return hooks[idx], true
}

// List returns a copy of all registered hooks
func (m *Manager) List() []Hook {
	hooks := *m.hooksPtr.Load()
	result := make([]Hook, len(hooks))
	copy(result, hooks)
	return result
}

// Count returns the number of registered hooks
func (m *Manager) Count() int {
	hooks := *m.hooksPtr.Load()
	return len(hooks)
}

// Clear removes all hooks
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldHooks := *m.hooksPtr.Load()
	for _, h := range oldHooks {
		_ = h.Stop()
	}

	newHooks := make([]Hook, 0)
	m.hooksPtr.Store(&newHooks)
	m.index = make(map[string]int)
}

// SetOptions invokes all SetOptions hooks
func (m *Manager) SetOptions(opts *Options) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(SetOptions) {
			if err := hook.SetOptions(opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnSysInfoTick invokes all OnSysInfoTick hooks
func (m *Manager) OnSysInfoTick(info *SysInfo) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnSysInfoTick) {
			_ = hook.OnSysInfoTick(info)
		}
	}
}

// OnStarted invokes all OnStarted hooks
func (m *Manager) OnStarted() {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnStarted) {
			_ = hook.OnStarted()
		}
	}
}

// OnStopped invokes all OnStopped hooks
func (m *Manager) OnStopped(err error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnStopped) {
			_ = hook.OnStopped(err)
		}
	}
}

// OnConnectAuthenticate invokes all OnConnectAuthenticate hooks
func (m *Manager) OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnConnectAuthenticate) {
			if !hook.OnConnectAuthenticate(client, packet) {
				return false
			}
		}
	}
	return true
}

// OnACLCheck invokes all OnACLCheck hooks
func (m *Manager) OnACLCheck(client *Client, topic string, access AccessType) bool {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnACLCheck) {
			if !hook.OnACLCheck(client, topic, access) {
				return false
			}
		}
	}
	return true
}

// OnConnect invokes all OnConnect hooks
func (m *Manager) OnConnect(client *Client, packet *ConnectPacket) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnConnect) {
			if err := hook.OnConnect(client, packet); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnSessionEstablish invokes all OnSessionEstablish hooks
func (m *Manager) OnSessionEstablish(client *Client, packet *ConnectPacket) *SessionState {
	hooks := *m.hooksPtr.Load()

	var state *SessionState
	for _, hook := range hooks {
		if hook.Provides(OnSessionEstablish) {
			if s := hook.OnSessionEstablish(client, packet); s != nil {
				state = s
			}
		}
	}
	return state
}

// OnSessionEstablished invokes all OnSessionEstablished hooks
func (m *Manager) OnSessionEstablished(client *Client, packet *ConnectPacket) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnSessionEstablished) {
			if err := hook.OnSessionEstablished(client, packet); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnDisconnect invokes all OnDisconnect hooks
func (m *Manager) OnDisconnect(client *Client, err error, expire bool) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnDisconnect) {
			_ = hook.OnDisconnect(client, err, expire)
		}
	}
}

// OnAuthPacket invokes all OnAuthPacket hooks
func (m *Manager) OnAuthPacket(client *Client, packet *AuthPacket) bool {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnAuthPacket) {
			if !hook.OnAuthPacket(client, packet) {
				return false
			}
		}
	}
	return true
}

// OnPacketRead invokes all OnPacketRead hooks
func (m *Manager) OnPacketRead(client *Client, packet []byte) ([]byte, error) {
	hooks := *m.hooksPtr.Load()

	var err error
	result := packet
	for _, hook := range hooks {
		if hook.Provides(OnPacketRead) {
			result, err = hook.OnPacketRead(client, result)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// OnPacketEncode invokes all OnPacketEncode hooks
func (m *Manager) OnPacketEncode(client *Client, packet []byte) []byte {
	hooks := *m.hooksPtr.Load()

	result := packet
	for _, hook := range hooks {
		if hook.Provides(OnPacketEncode) {
			result = hook.OnPacketEncode(client, result)
		}
	}
	return result
}

// OnPacketSent invokes all OnPacketSent hooks
func (m *Manager) OnPacketSent(client *Client, packet []byte, count int, err error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPacketSent) {
			_ = hook.OnPacketSent(client, packet, count, err)
		}
	}
}

// OnPacketProcessed invokes all OnPacketProcessed hooks
func (m *Manager) OnPacketProcessed(client *Client, packetType encoding.PacketType, err error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPacketProcessed) {
			_ = hook.OnPacketProcessed(client, packetType, err)
		}
	}
}

// OnSubscribe invokes all OnSubscribe hooks
func (m *Manager) OnSubscribe(client *Client, sub *Subscription) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnSubscribe) {
			if err := hook.OnSubscribe(client, sub); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnSubscribed invokes all OnSubscribed hooks
func (m *Manager) OnSubscribed(client *Client, sub *Subscription) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnSubscribed) {
			_ = hook.OnSubscribed(client, sub)
		}
	}
}

// OnSelectSubscribers invokes all OnSelectSubscribers hooks
func (m *Manager) OnSelectSubscribers(subscribers *Subscribers, topic string) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnSelectSubscribers) {
			_ = hook.OnSelectSubscribers(subscribers, topic)
		}
	}
}

// OnUnsubscribe invokes all OnUnsubscribe hooks
func (m *Manager) OnUnsubscribe(client *Client, topicFilter string) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnUnsubscribe) {
			if err := hook.OnUnsubscribe(client, topicFilter); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnUnsubscribed invokes all OnUnsubscribed hooks
func (m *Manager) OnUnsubscribed(client *Client, topicFilter string) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnUnsubscribed) {
			_ = hook.OnUnsubscribed(client, topicFilter)
		}
	}
}

// OnPublish invokes all OnPublish hooks
func (m *Manager) OnPublish(client *Client, packet *PublishPacket) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPublish) {
			if err := hook.OnPublish(client, packet); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnPublished invokes all OnPublished hooks
func (m *Manager) OnPublished(client *Client, packet *PublishPacket) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPublished) {
			_ = hook.OnPublished(client, packet)
		}
	}
}

// OnPublishDropped invokes all OnPublishDropped hooks
func (m *Manager) OnPublishDropped(client *Client, packet *PublishPacket, reason DropReason) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPublishDropped) {
			_ = hook.OnPublishDropped(client, packet, reason)
		}
	}
}

// OnRetainMessage invokes all OnRetainMessage hooks
func (m *Manager) OnRetainMessage(client *Client, packet *PublishPacket) error {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnRetainMessage) {
			if err := hook.OnRetainMessage(client, packet); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnRetainPublished invokes all OnRetainPublished hooks
func (m *Manager) OnRetainPublished(client *Client, packet *PublishPacket) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnRetainPublished) {
			_ = hook.OnRetainPublished(client, packet)
		}
	}
}

// OnQosPublish invokes all OnQosPublish hooks
func (m *Manager) OnQosPublish(client *Client, packet *PublishPacket, sent time.Time, resend int) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnQosPublish) {
			_ = hook.OnQosPublish(client, packet, sent, resend)
		}
	}
}

// OnQosComplete invokes all OnQosComplete hooks
func (m *Manager) OnQosComplete(client *Client, packetID uint16, packetType encoding.PacketType) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnQosComplete) {
			_ = hook.OnQosComplete(client, packetID, packetType)
		}
	}
}

// OnQosDropped invokes all OnQosDropped hooks
func (m *Manager) OnQosDropped(client *Client, packetID uint16, reason DropReason) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnQosDropped) {
			_ = hook.OnQosDropped(client, packetID, reason)
		}
	}
}

// OnPacketIDExhausted invokes all OnPacketIDExhausted hooks
func (m *Manager) OnPacketIDExhausted(client *Client, packetType encoding.PacketType) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnPacketIDExhausted) {
			_ = hook.OnPacketIDExhausted(client, packetType)
		}
	}
}

// OnWill invokes all OnWill hooks
func (m *Manager) OnWill(client *Client, will *WillMessage) *WillMessage {
	hooks := *m.hooksPtr.Load()

	result := will
	for _, hook := range hooks {
		if hook.Provides(OnWill) {
			if w := hook.OnWill(client, result); w != nil {
				result = w
			}
		}
	}
	return result
}

// OnWillSent invokes all OnWillSent hooks
func (m *Manager) OnWillSent(client *Client, will *WillMessage) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnWillSent) {
			_ = hook.OnWillSent(client, will)
		}
	}
}

// OnClientExpired invokes all OnClientExpired hooks
func (m *Manager) OnClientExpired(clientID string) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnClientExpired) {
			_ = hook.OnClientExpired(clientID)
		}
	}
}

// OnRetainedExpired invokes all OnRetainedExpired hooks
func (m *Manager) OnRetainedExpired(topic string) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(OnRetainedExpired) {
			_ = hook.OnRetainedExpired(topic)
		}
	}
}

// StoredClients invokes all StoredClients hooks
func (m *Manager) StoredClients() ([]*Client, error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(StoredClients) {
			return hook.StoredClients()
		}
	}
	return nil, nil
}

// StoredSubscriptions invokes all StoredSubscriptions hooks
func (m *Manager) StoredSubscriptions() ([]*Subscription, error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(StoredSubscriptions) {
			return hook.StoredSubscriptions()
		}
	}
	return nil, nil
}

// StoredInflightMessages invokes all StoredInflightMessages hooks
func (m *Manager) StoredInflightMessages() ([]*InflightMessage, error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(StoredInflightMessages) {
			return hook.StoredInflightMessages()
		}
	}
	return nil, nil
}

// StoredRetainedMessages invokes all StoredRetainedMessages hooks
func (m *Manager) StoredRetainedMessages() ([]*RetainedMessage, error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(StoredRetainedMessages) {
			return hook.StoredRetainedMessages()
		}
	}
	return nil, nil
}

// StoredSysInfo invokes all StoredSysInfo hooks
func (m *Manager) StoredSysInfo() (*SysInfo, error) {
	hooks := *m.hooksPtr.Load()

	for _, hook := range hooks {
		if hook.Provides(StoredSysInfo) {
			return hook.StoredSysInfo()
		}
	}
	return nil, nil
}
