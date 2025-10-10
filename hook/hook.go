package hook

import (
	"net"
	"time"

	"github.com/axmq/ax/encoding"
)

// Event represents hook event types
type Event byte

const (
	SetOptions Event = iota
	OnSysInfoTick
	OnStarted
	OnStopped
	OnConnectAuthenticate
	OnACLCheck
	OnConnect
	OnSessionEstablish
	OnSessionEstablished
	OnDisconnect
	OnAuthPacket
	OnPacketRead
	OnPacketEncode
	OnPacketSent
	OnPacketProcessed
	OnSubscribe
	OnSubscribed
	OnSelectSubscribers
	OnUnsubscribe
	OnUnsubscribed
	OnPublish
	OnPublished
	OnPublishDropped
	OnRetainMessage
	OnRetainPublished
	OnQosPublish
	OnQosComplete
	OnQosDropped
	OnPacketIDExhausted
	OnWill
	OnWillSent
	OnClientExpired
	OnRetainedExpired
	StoredClients
	StoredSubscriptions
	StoredInflightMessages
	StoredRetainedMessages
	StoredSysInfo
)

// String returns the string representation of the event
func (e Event) String() string {
	names := [...]string{
		"SetOptions",
		"OnSysInfoTick",
		"OnStarted",
		"OnStopped",
		"OnConnectAuthenticate",
		"OnACLCheck",
		"OnConnect",
		"OnSessionEstablish",
		"OnSessionEstablished",
		"OnDisconnect",
		"OnAuthPacket",
		"OnPacketRead",
		"OnPacketEncode",
		"OnPacketSent",
		"OnPacketProcessed",
		"OnSubscribe",
		"OnSubscribed",
		"OnSelectSubscribers",
		"OnUnsubscribe",
		"OnUnsubscribed",
		"OnPublish",
		"OnPublished",
		"OnPublishDropped",
		"OnRetainMessage",
		"OnRetainPublished",
		"OnQosPublish",
		"OnQosComplete",
		"OnQosDropped",
		"OnPacketIDExhausted",
		"OnWill",
		"OnWillSent",
		"OnClientExpired",
		"OnRetainedExpired",
		"StoredClients",
		"StoredSubscriptions",
		"StoredInflightMessages",
		"StoredRetainedMessages",
		"StoredSysInfo",
	}
	if e < Event(len(names)) {
		return names[e]
	}
	return "Unknown"
}

// Hook defines the interface that all hooks must implement
// Hooks can intercept and modify broker behavior at various lifecycle points
type Hook interface {
	// ID returns a unique identifier for this hook
	ID() string

	// Provides indicates if the hook provides implementation for the given event
	Provides(event Event) bool

	// Init initializes the hook with the given configuration
	Init(config any) error

	// Stop stops the hook
	Stop() error

	// SetOptions is called when broker options are being configured
	SetOptions(opts *Options) error

	// OnSysInfoTick is called on system info timer tick
	OnSysInfoTick(info *SysInfo) error

	// OnStarted is called when the broker has started
	OnStarted() error

	// OnStopped is called when the broker has stopped
	OnStopped(err error) error

	// OnConnectAuthenticate is called to authenticate a client connection
	OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool

	// OnACLCheck is called to check access control for topic operations
	OnACLCheck(client *Client, topic string, access AccessType) bool

	// OnConnect is called when a client connects
	OnConnect(client *Client, packet *ConnectPacket) error

	// OnSessionEstablish is called before establishing a session
	OnSessionEstablish(client *Client, packet *ConnectPacket) *SessionState

	// OnSessionEstablished is called after a session is established
	OnSessionEstablished(client *Client, packet *ConnectPacket) error

	// OnDisconnect is called when a client disconnects
	OnDisconnect(client *Client, err error, expire bool) error

	// OnAuthPacket is called when an AUTH packet is received (MQTT 5.0)
	OnAuthPacket(client *Client, packet *AuthPacket) bool

	// OnPacketRead is called when a packet is read from the network
	OnPacketRead(client *Client, packet []byte) ([]byte, error)

	// OnPacketEncode is called before encoding a packet
	OnPacketEncode(client *Client, packet []byte) []byte

	// OnPacketSent is called after a packet is sent
	OnPacketSent(client *Client, packet []byte, count int, err error) error

	// OnPacketProcessed is called after a packet is processed
	OnPacketProcessed(client *Client, packetType encoding.PacketType, err error) error

	// OnSubscribe is called before processing a subscription
	OnSubscribe(client *Client, sub *Subscription) error

	// OnSubscribed is called after a subscription is completed
	OnSubscribed(client *Client, sub *Subscription) error

	// OnSelectSubscribers is called to filter/modify subscribers for a publish
	OnSelectSubscribers(subscribers *Subscribers, topic string) error

	// OnUnsubscribe is called before processing an unsubscription
	OnUnsubscribe(client *Client, topicFilter string) error

	// OnUnsubscribed is called after an unsubscription is completed
	OnUnsubscribed(client *Client, topicFilter string) error

	// OnPublish is called before publishing a message
	OnPublish(client *Client, packet *PublishPacket) error

	// OnPublished is called after a message is published
	OnPublished(client *Client, packet *PublishPacket) error

	// OnPublishDropped is called when a publish is dropped
	OnPublishDropped(client *Client, packet *PublishPacket, reason DropReason) error

	// OnRetainMessage is called before retaining a message
	OnRetainMessage(client *Client, packet *PublishPacket) error

	// OnRetainPublished is called when a retained message is published to a subscriber
	OnRetainPublished(client *Client, packet *PublishPacket) error

	// OnQosPublish is called when a QoS message is published
	OnQosPublish(client *Client, packet *PublishPacket, sent time.Time, resend int) error

	// OnQosComplete is called when a QoS flow is completed
	OnQosComplete(client *Client, packetID uint16, packetType encoding.PacketType) error

	// OnQosDropped is called when a QoS message is dropped
	OnQosDropped(client *Client, packetID uint16, reason DropReason) error

	// OnPacketIDExhausted is called when packet IDs are exhausted
	OnPacketIDExhausted(client *Client, packetType encoding.PacketType) error

	// OnWill is called before processing a will message
	OnWill(client *Client, will *WillMessage) *WillMessage

	// OnWillSent is called after a will message is sent
	OnWillSent(client *Client, will *WillMessage) error

	// OnClientExpired is called when a client session expires
	OnClientExpired(clientID string) error

	// OnRetainedExpired is called when a retained message expires
	OnRetainedExpired(topic string) error

	// StoredClients is called to store/load client data
	StoredClients() ([]*Client, error)

	// StoredSubscriptions is called to store/load subscription data
	StoredSubscriptions() ([]*Subscription, error)

	// StoredInflightMessages is called to store/load inflight messages
	StoredInflightMessages() ([]*InflightMessage, error)

	// StoredRetainedMessages is called to store/load retained messages
	StoredRetainedMessages() ([]*RetainedMessage, error)

	// StoredSysInfo is called to store/load system info
	StoredSysInfo() (*SysInfo, error)
}

// Options holds the configuration options for the broker
type Options struct {
	Capabilities *Capabilities
	Config       map[string]any
}

// Capabilities defines the supported capabilities of the broker
type Capabilities struct {
	MaximumSessionExpiryInterval uint32
	MaximumMessageExpiryInterval uint32
	ReceiveMaximum               uint16
	MaximumQoS                   byte
	RetainAvailable              bool
	MaximumPacketSize            uint32
	MaximumTopicAlias            uint16
	WildcardSubAvailable         bool
	SubIDAvailable               bool
	SharedSubAvailable           bool
}

// SysInfo holds system information for the broker
type SysInfo struct {
	Uptime              int64
	Version             string
	Started             time.Time
	Time                time.Time
	ClientsConnected    int64
	ClientsTotal        int64
	ClientsMaximum      int64
	ClientsDisconnected int64
	MessagesReceived    int64
	MessagesSent        int64
	MessagesDropped     int64
	Subscriptions       int64
	Retained            int64
	Inflight            int64
	MemoryAlloc         uint64
	Threads             int
}

// Client represents a connected client
type Client struct {
	ID              string
	RemoteAddr      net.Addr
	LocalAddr       net.Addr
	Username        string
	CleanStart      bool
	ProtocolVersion byte
	KeepAlive       uint16
	SessionPresent  bool
	Properties      Properties
	Will            *WillMessage
	ConnectedAt     time.Time
	DisconnectedAt  time.Time
	State           ClientState
}

// ClientState represents the state of a client
type ClientState byte

const (
	ClientStateConnecting ClientState = iota
	ClientStateConnected
	ClientStateDisconnecting
	ClientStateDisconnected
)

// ConnectPacket holds the information for a CONNECT packet
type ConnectPacket struct {
	ProtocolName    string
	ProtocolVersion byte
	CleanStart      bool
	KeepAlive       uint16
	ClientID        string
	Username        string
	Password        []byte
	Will            *WillMessage
	Properties      Properties
	SessionPresent  bool
}

// AuthPacket holds AUTH packet information
type AuthPacket struct {
	ReasonCode byte
	Properties Properties
	AuthMethod string
	AuthData   []byte
}

// PublishPacket holds publish information
type PublishPacket struct {
	PacketID        uint16
	Topic           string
	Payload         []byte
	QoS             byte
	Retain          bool
	Duplicate       bool
	Properties      Properties
	ProtocolVersion byte
	Created         time.Time
	Origin          string
}

// Subscription represents a client's subscription to a topic
type Subscription struct {
	ClientID               string
	TopicFilter            string
	QoS                    byte
	NoLocal                bool
	RetainAsPublished      bool
	RetainHandling         byte
	SubscriptionIdentifier uint32
	SubscribedAt           time.Time
}

// Subscribers holds a list of subscriptions for a topic
type Subscribers struct {
	Subscriptions []*Subscription
}

// Add adds a subscription to the list
func (s *Subscribers) Add(sub *Subscription) {
	s.Subscriptions = append(s.Subscriptions, sub)
}

// Remove removes a subscription from the list by client ID
func (s *Subscribers) Remove(clientID string) {
	n := 0
	for _, sub := range s.Subscriptions {
		if sub.ClientID != clientID {
			s.Subscriptions[n] = sub
			n++
		}
	}
	// Nil out the rest of the slice to prevent memory leaks
	for i := n; i < len(s.Subscriptions); i++ {
		s.Subscriptions[i] = nil
	}
	s.Subscriptions = s.Subscriptions[:n]
}

// Clear clears the list of subscriptions
func (s *Subscribers) Clear() {
	s.Subscriptions = s.Subscriptions[:0]
}

// WillMessage represents a will message for a client
type WillMessage struct {
	Topic             string
	Payload           []byte
	QoS               byte
	Retain            bool
	Properties        Properties
	WillDelayInterval uint32
}

// SessionState holds the state of a session
type SessionState struct {
	ClientID        string
	CleanStart      bool
	SessionPresent  bool
	ExpiryInterval  uint32
	Subscriptions   map[string]*Subscription
	PendingMessages []*InflightMessage
	NextPacketID    uint16
}

// InflightMessage represents a message that is in flight
type InflightMessage struct {
	PacketID    uint16
	ClientID    string
	Topic       string
	Payload     []byte
	QoS         byte
	Retain      bool
	Duplicate   bool
	Properties  Properties
	Sent        time.Time
	ResendCount int
}

// RetainedMessage represents a retained message
type RetainedMessage struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Properties Properties
	Timestamp  time.Time
}

// Properties is a map of key-value pairs for message properties
type Properties map[string]any

// AccessType represents the type of access for ACL checks
type AccessType byte

const (
	AccessTypeRead AccessType = iota
	AccessTypeWrite
	AccessTypeReadWrite
)

// DropReason represents the reason for dropping a message
type DropReason byte

const (
	DropReasonQueueFull DropReason = iota
	DropReasonClientDisconnected
	DropReasonExpired
	DropReasonInvalidTopic
	DropReasonACLDenied
	DropReasonQuotaExceeded
	DropReasonPacketTooLarge
	DropReasonInternalError
)

// String returns the string representation of the drop reason
func (d DropReason) String() string {
	switch d {
	case DropReasonQueueFull:
		return "queue_full"
	case DropReasonClientDisconnected:
		return "client_disconnected"
	case DropReasonExpired:
		return "expired"
	case DropReasonInvalidTopic:
		return "invalid_topic"
	case DropReasonACLDenied:
		return "acl_denied"
	case DropReasonQuotaExceeded:
		return "quota_exceeded"
	case DropReasonPacketTooLarge:
		return "packet_too_large"
	case DropReasonInternalError:
		return "internal_error"
	default:
		return "unknown"
	}
}
