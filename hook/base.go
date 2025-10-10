package hook

import (
	"time"

	"github.com/axmq/ax/encoding"
)

// Base provides a default no-op implementation of the Hook interface
// Users can embed this in their custom hooks and override only the methods they need
type Base struct {
	id string
}

// NewHookBase creates a new base hook with the given ID
func NewHookBase(id string) *Base {
	return &Base{id: id}
}

// ID returns the unique identifier for this hook
func (h *Base) ID() string {
	return h.id
}

// Provides determines if the hook provides the given event
func (h *Base) Provides(event Event) bool {
	return false
}

// Init initializes the hook with the given config
func (h *Base) Init(config any) error {
	return nil
}

// Stop stops the hook
func (h *Base) Stop() error {
	return nil
}

// SetOptions sets the options for the hook
func (h *Base) SetOptions(opts *Options) error {
	return nil
}

// OnSysInfoTick is called on sysinfo tick events
func (h *Base) OnSysInfoTick(info *SysInfo) error {
	return nil
}

// OnStarted is called when the hook is started
func (h *Base) OnStarted() error {
	return nil
}

// OnStopped is called when the hook is stopped
func (h *Base) OnStopped(err error) error {
	return nil
}

// OnConnectAuthenticate is called during the connect authenticate phase
func (h *Base) OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool {
	return true
}

// OnACLCheck is called to check ACLs
func (h *Base) OnACLCheck(client *Client, topic string, access AccessType) bool {
	return true
}

// OnConnect is called when a client connects
func (h *Base) OnConnect(client *Client, packet *ConnectPacket) error {
	return nil
}

// OnSessionEstablish is called when a session is being established
func (h *Base) OnSessionEstablish(client *Client, packet *ConnectPacket) *SessionState {
	return nil
}

// OnSessionEstablished is called when a session has been established
func (h *Base) OnSessionEstablished(client *Client, packet *ConnectPacket) error {
	return nil
}

// OnDisconnect is called when a client disconnects
func (h *Base) OnDisconnect(client *Client, err error, expire bool) error {
	return nil
}

// OnAuthPacket is called when an auth packet is received
func (h *Base) OnAuthPacket(client *Client, packet *AuthPacket) bool {
	return true
}

// OnPacketRead is called when a packet is read
func (h *Base) OnPacketRead(client *Client, packet []byte) ([]byte, error) {
	return packet, nil
}

// OnPacketEncode is called to encode a packet
func (h *Base) OnPacketEncode(client *Client, packet []byte) []byte {
	return packet
}

// OnPacketSent is called when a packet is sent
func (h *Base) OnPacketSent(client *Client, packet []byte, count int, err error) error {
	return nil
}

// OnPacketProcessed is called when a packet is processed
func (h *Base) OnPacketProcessed(client *Client, packetType encoding.PacketType, err error) error {
	return nil
}

// OnSubscribe is called when a subscribe packet is received
func (h *Base) OnSubscribe(client *Client, sub *Subscription) error {
	return nil
}

// OnSubscribed is called when a client is subscribed
func (h *Base) OnSubscribed(client *Client, sub *Subscription) error {
	return nil
}

// OnSelectSubscribers is called to select subscribers for a topic
func (h *Base) OnSelectSubscribers(subscribers *Subscribers, topic string) error {
	return nil
}

// OnUnsubscribe is called when an unsubscribe packet is received
func (h *Base) OnUnsubscribe(client *Client, topicFilter string) error {
	return nil
}

// OnUnsubscribed is called when a client is unsubscribed
func (h *Base) OnUnsubscribed(client *Client, topicFilter string) error {
	return nil
}

// OnPublish is called when a publish packet is received
func (h *Base) OnPublish(client *Client, packet *PublishPacket) error {
	return nil
}

// OnPublished is called when a message is published
func (h *Base) OnPublished(client *Client, packet *PublishPacket) error {
	return nil
}

// OnPublishDropped is called when a published message is dropped
func (h *Base) OnPublishDropped(client *Client, packet *PublishPacket, reason DropReason) error {
	return nil
}

// OnRetainMessage is called when a retain message is received
func (h *Base) OnRetainMessage(client *Client, packet *PublishPacket) error {
	return nil
}

// OnRetainPublished is called when a retain message is published
func (h *Base) OnRetainPublished(client *Client, packet *PublishPacket) error {
	return nil
}

// OnQosPublish is called for QoS publish events
func (h *Base) OnQosPublish(client *Client, packet *PublishPacket, sent time.Time, resend int) error {
	return nil
}

// OnQosComplete is called when QoS is complete for a packet
func (h *Base) OnQosComplete(client *Client, packetID uint16, packetType encoding.PacketType) error {
	return nil
}

// OnQosDropped is called when QoS is dropped for a packet
func (h *Base) OnQosDropped(client *Client, packetID uint16, reason DropReason) error {
	return nil
}

// OnPacketIDExhausted is called when the packet ID is exhausted
func (h *Base) OnPacketIDExhausted(client *Client, packetType encoding.PacketType) error {
	return nil
}

// OnWill is called to get the will message for a client
func (h *Base) OnWill(client *Client, will *WillMessage) *WillMessage {
	return will
}

// OnWillSent is called when a will message is sent
func (h *Base) OnWillSent(client *Client, will *WillMessage) error {
	return nil
}

// OnClientExpired is called when a client ID expires
func (h *Base) OnClientExpired(clientID string) error {
	return nil
}

// OnRetainedExpired is called when a retained message expires
func (h *Base) OnRetainedExpired(topic string) error {
	return nil
}

// StoredClients returns the list of stored clients
func (h *Base) StoredClients() ([]*Client, error) {
	return nil, nil
}

// StoredSubscriptions returns the list of stored subscriptions
func (h *Base) StoredSubscriptions() ([]*Subscription, error) {
	return nil, nil
}

// StoredInflightMessages returns the list of stored inflight messages
func (h *Base) StoredInflightMessages() ([]*InflightMessage, error) {
	return nil, nil
}

// StoredRetainedMessages returns the list of stored retained messages
func (h *Base) StoredRetainedMessages() ([]*RetainedMessage, error) {
	return nil, nil
}

// StoredSysInfo returns the stored sysinfo
func (h *Base) StoredSysInfo() (*SysInfo, error) {
	return nil, nil
}
