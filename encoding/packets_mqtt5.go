package encoding

import (
	"io"
)

// ReasonCode represents MQTT 5.0 reason codes
type ReasonCode byte

const (
	// Success and normal disconnection codes
	ReasonSuccess                   ReasonCode = 0x00
	ReasonNormalDisconnection       ReasonCode = 0x00
	ReasonGrantedQoS0               ReasonCode = 0x00
	ReasonGrantedQoS1               ReasonCode = 0x01
	ReasonGrantedQoS2               ReasonCode = 0x02
	ReasonDisconnectWithWillMessage ReasonCode = 0x04
	ReasonNoMatchingSubscribers     ReasonCode = 0x10
	ReasonNoSubscriptionExisted     ReasonCode = 0x11
	ReasonContinueAuthentication    ReasonCode = 0x18
	ReasonReAuthenticate            ReasonCode = 0x19

	// Error codes
	ReasonUnspecifiedError                    ReasonCode = 0x80
	ReasonMalformedPacket                     ReasonCode = 0x81
	ReasonProtocolError                       ReasonCode = 0x82
	ReasonImplementationSpecificError         ReasonCode = 0x83
	ReasonUnsupportedProtocolVersion          ReasonCode = 0x84
	ReasonClientIdentifierNotValid            ReasonCode = 0x85
	ReasonBadUsernameOrPassword               ReasonCode = 0x86
	ReasonNotAuthorized                       ReasonCode = 0x87
	ReasonServerUnavailable                   ReasonCode = 0x88
	ReasonServerBusy                          ReasonCode = 0x89
	ReasonBanned                              ReasonCode = 0x8A
	ReasonServerShuttingDown                  ReasonCode = 0x8B
	ReasonBadAuthenticationMethod             ReasonCode = 0x8C
	ReasonKeepAliveTimeout                    ReasonCode = 0x8D
	ReasonSessionTakenOver                    ReasonCode = 0x8E
	ReasonTopicFilterInvalid                  ReasonCode = 0x8F
	ReasonTopicNameInvalid                    ReasonCode = 0x90
	ReasonPacketIdentifierInUse               ReasonCode = 0x91
	ReasonPacketIdentifierNotFound            ReasonCode = 0x92
	ReasonReceiveMaximumExceeded              ReasonCode = 0x93
	ReasonTopicAliasInvalid                   ReasonCode = 0x94
	ReasonPacketTooLarge                      ReasonCode = 0x95
	ReasonMessageRateTooHigh                  ReasonCode = 0x96
	ReasonQuotaExceeded                       ReasonCode = 0x97
	ReasonAdministrativeAction                ReasonCode = 0x98
	ReasonPayloadFormatInvalid                ReasonCode = 0x99
	ReasonRetainNotSupported                  ReasonCode = 0x9A
	ReasonQoSNotSupported                     ReasonCode = 0x9B
	ReasonUseAnotherServer                    ReasonCode = 0x9C
	ReasonServerMoved                         ReasonCode = 0x9D
	ReasonSharedSubscriptionsNotSupported     ReasonCode = 0x9E
	ReasonConnectionRateExceeded              ReasonCode = 0x9F
	ReasonMaximumConnectTime                  ReasonCode = 0xA0
	ReasonSubscriptionIdentifiersNotSupported ReasonCode = 0xA1
	ReasonWildcardSubscriptionsNotSupported   ReasonCode = 0xA2
)

// ConnectPacket represents an MQTT 5.0 CONNECT packet
type ConnectPacket struct {
	FixedHeader     FixedHeader
	ProtocolName    string
	ProtocolVersion ProtocolVersion
	CleanStart      bool
	WillFlag        bool
	WillQoS         QoS
	WillRetain      bool
	PasswordFlag    bool
	UsernameFlag    bool
	KeepAlive       uint16
	Properties      Properties
	ClientID        string
	WillProperties  Properties
	WillTopic       string
	WillPayload     []byte
	Username        string
	Password        []byte
}

// ConnackPacket represents an MQTT 5.0 CONNACK packet
type ConnackPacket struct {
	FixedHeader    FixedHeader
	SessionPresent bool
	ReasonCode     ReasonCode
	Properties     Properties
}

// PublishPacket represents an MQTT 5.0 PUBLISH packet
type PublishPacket struct {
	FixedHeader FixedHeader
	TopicName   string
	PacketID    uint16 // Only for QoS 1 and 2
	Properties  Properties
	Payload     []byte
}

// PubackPacket represents an MQTT 5.0 PUBACK packet
type PubackPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	ReasonCode  ReasonCode
	Properties  Properties
}

// PubrecPacket represents an MQTT 5.0 PUBREC packet
type PubrecPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	ReasonCode  ReasonCode
	Properties  Properties
}

// PubrelPacket represents an MQTT 5.0 PUBREL packet
type PubrelPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	ReasonCode  ReasonCode
	Properties  Properties
}

// PubcompPacket represents an MQTT 5.0 PUBCOMP packet
type PubcompPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	ReasonCode  ReasonCode
	Properties  Properties
}

// Subscription represents a single subscription in SUBSCRIBE packet
type Subscription struct {
	TopicFilter            string
	QoS                    QoS
	NoLocal                bool
	RetainAsPublished      bool
	RetainHandling         byte
	SubscriptionIdentifier uint32 // From properties
}

// SubscribePacket represents an MQTT 5.0 SUBSCRIBE packet
type SubscribePacket struct {
	FixedHeader   FixedHeader
	PacketID      uint16
	Properties    Properties
	Subscriptions []Subscription
}

// SubackPacket represents an MQTT 5.0 SUBACK packet
type SubackPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	Properties  Properties
	ReasonCodes []ReasonCode
}

// UnsubscribePacket represents an MQTT 5.0 UNSUBSCRIBE packet
type UnsubscribePacket struct {
	FixedHeader  FixedHeader
	PacketID     uint16
	Properties   Properties
	TopicFilters []string
}

// UnsubackPacket represents an MQTT 5.0 UNSUBACK packet
type UnsubackPacket struct {
	FixedHeader FixedHeader
	PacketID    uint16
	Properties  Properties
	ReasonCodes []ReasonCode
}

// PingreqPacket represents an MQTT 5.0 PINGREQ packet
type PingreqPacket struct {
	FixedHeader FixedHeader
}

// PingrespPacket represents an MQTT 5.0 PINGRESP packet
type PingrespPacket struct {
	FixedHeader FixedHeader
}

// DisconnectPacket represents an MQTT 5.0 DISCONNECT packet
type DisconnectPacket struct {
	FixedHeader FixedHeader
	ReasonCode  ReasonCode
	Properties  Properties
}

// AuthPacket represents an MQTT 5.0 AUTH packet
type AuthPacket struct {
	FixedHeader FixedHeader
	ReasonCode  ReasonCode
	Properties  Properties
}

// ParseConnectPacket parses an MQTT 5.0 CONNECT packet
func ParseConnectPacket(r io.Reader, fh *FixedHeader) (*ConnectPacket, error) {
	pkt := &ConnectPacket{FixedHeader: *fh}

	// Read protocol name
	protocolName, err := readUTF8String(r)
	if err != nil {
		return nil, err
	}
	pkt.ProtocolName = protocolName

	// Validate protocol name
	if protocolName != "MQTT" {
		return nil, ErrInvalidProtocolName
	}

	// Read protocol version
	version, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ProtocolVersion = ProtocolVersion(version)

	if pkt.ProtocolVersion != ProtocolVersion50 {
		return nil, ErrInvalidProtocolVersion
	}

	// Read connect flags
	flags, err := readByte(r)
	if err != nil {
		return nil, err
	}

	pkt.CleanStart = (flags & 0x02) != 0
	pkt.WillFlag = (flags & 0x04) != 0
	pkt.WillQoS = QoS((flags & 0x18) >> 3)
	pkt.WillRetain = (flags & 0x20) != 0
	pkt.PasswordFlag = (flags & 0x40) != 0
	pkt.UsernameFlag = (flags & 0x80) != 0

	// Validate reserved bit (bit 0) must be 0
	if (flags & 0x01) != 0 {
		return nil, ErrMalformedPacket
	}

	// Read keep alive
	keepAlive, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.KeepAlive = keepAlive

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	// Read client ID
	clientID, err := readUTF8String(r)
	if err != nil {
		return nil, err
	}
	pkt.ClientID = clientID

	// Read Will properties and topic/payload if Will flag is set
	if pkt.WillFlag {
		willProps, err := ParseProperties(r)
		if err != nil {
			return nil, err
		}
		pkt.WillProperties = *willProps

		willTopic, err := readUTF8String(r)
		if err != nil {
			return nil, err
		}
		pkt.WillTopic = willTopic

		willPayload, err := readBinaryData(r)
		if err != nil {
			return nil, err
		}
		pkt.WillPayload = willPayload
	}

	// Read username if flag is set
	if pkt.UsernameFlag {
		username, err := readUTF8String(r)
		if err != nil {
			return nil, err
		}
		pkt.Username = username
	}

	// Read password if flag is set
	if pkt.PasswordFlag {
		password, err := readBinaryData(r)
		if err != nil {
			return nil, err
		}
		pkt.Password = password
	}

	return pkt, nil
}

// ParseConnackPacket parses an MQTT 5.0 CONNACK packet
func ParseConnackPacket(r io.Reader, fh *FixedHeader) (*ConnackPacket, error) {
	pkt := &ConnackPacket{FixedHeader: *fh}

	// Read connect acknowledge flags
	flags, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.SessionPresent = (flags & 0x01) != 0

	// Reserved bits (bits 7-1) must be 0
	if (flags & 0xFE) != 0 {
		return nil, ErrMalformedPacket
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParsePublishPacket parses an MQTT 5.0 PUBLISH packet
func ParsePublishPacket(r io.Reader, fh *FixedHeader) (*PublishPacket, error) {
	pkt := &PublishPacket{FixedHeader: *fh}

	// Read topic name
	topicName, err := readUTF8String(r)
	if err != nil {
		return nil, err
	}
	pkt.TopicName = topicName

	// Read packet ID for QoS 1 and 2
	if fh.QoS > QoS0 {
		packetID, err := readTwoByteInt(r)
		if err != nil {
			return nil, err
		}
		if packetID == 0 {
			return nil, ErrInvalidPacketID
		}
		pkt.PacketID = packetID
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	// Calculate payload length
	headerSize := 2 + len(topicName) // Topic name length prefix + topic name
	if fh.QoS > QoS0 {
		headerSize += 2 // Packet ID
	}
	headerSize += int(props.Length) + len(EncodeVariableByteIntegerMust(props.Length))

	payloadLength := int(fh.RemainingLength) - headerSize
	if payloadLength > 0 {
		payload := make([]byte, payloadLength)
		if _, err := io.ReadFull(r, payload); err != nil {
			if err == io.EOF {
				return nil, ErrUnexpectedEOF
			}
			return nil, err
		}
		pkt.Payload = payload
	}

	return pkt, nil
}

// ParsePubackPacket parses an MQTT 5.0 PUBACK packet
func ParsePubackPacket(r io.Reader, fh *FixedHeader) (*PubackPacket, error) {
	pkt := &PubackPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	// Remaining length of 2 means no reason code or properties
	if fh.RemainingLength == 2 {
		pkt.ReasonCode = ReasonSuccess
		return pkt, nil
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	// Remaining length of 3 means no properties
	if fh.RemainingLength == 3 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParsePubrecPacket parses an MQTT 5.0 PUBREC packet
func ParsePubrecPacket(r io.Reader, fh *FixedHeader) (*PubrecPacket, error) {
	pkt := &PubrecPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	if fh.RemainingLength == 2 {
		pkt.ReasonCode = ReasonSuccess
		return pkt, nil
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	if fh.RemainingLength == 3 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParsePubrelPacket parses an MQTT 5.0 PUBREL packet
func ParsePubrelPacket(r io.Reader, fh *FixedHeader) (*PubrelPacket, error) {
	pkt := &PubrelPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	if fh.RemainingLength == 2 {
		pkt.ReasonCode = ReasonSuccess
		return pkt, nil
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	if fh.RemainingLength == 3 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParsePubcompPacket parses an MQTT 5.0 PUBCOMP packet
func ParsePubcompPacket(r io.Reader, fh *FixedHeader) (*PubcompPacket, error) {
	pkt := &PubcompPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	if fh.RemainingLength == 2 {
		pkt.ReasonCode = ReasonSuccess
		return pkt, nil
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	if fh.RemainingLength == 3 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParseSubscribePacket parses an MQTT 5.0 SUBSCRIBE packet
func ParseSubscribePacket(r io.Reader, fh *FixedHeader) (*SubscribePacket, error) {
	pkt := &SubscribePacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	pkt.Subscriptions = make([]Subscription, 0, 2)

	// Calculate how many bytes we've read
	bytesRead := 2 + int(props.Length) + len(EncodeVariableByteIntegerMust(props.Length))

	for bytesRead < int(fh.RemainingLength) {
		// Read topic filter
		topicFilter, err := readUTF8String(r)
		if err != nil {
			return nil, err
		}
		bytesRead += 2 + len(topicFilter)

		// Read subscription options
		options, err := readByte(r)
		if err != nil {
			return nil, err
		}
		bytesRead++

		sub := Subscription{
			TopicFilter:       topicFilter,
			QoS:               QoS(options & 0x03),
			NoLocal:           (options & 0x04) != 0,
			RetainAsPublished: (options & 0x08) != 0,
			RetainHandling:    (options & 0x30) >> 4,
		}

		// Reserved bits (bits 7, 6) must be 0
		if (options & 0xC0) != 0 {
			return nil, ErrMalformedPacket
		}

		pkt.Subscriptions = append(pkt.Subscriptions, sub)
	}

	return pkt, nil
}

// ParseSubackPacket parses an MQTT 5.0 SUBACK packet
func ParseSubackPacket(r io.Reader, fh *FixedHeader) (*SubackPacket, error) {
	pkt := &SubackPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	// Read reason codes
	bytesRead := 2 + int(props.Length) + len(EncodeVariableByteIntegerMust(props.Length))
	reasonCodeCount := int(fh.RemainingLength) - bytesRead

	pkt.ReasonCodes = make([]ReasonCode, reasonCodeCount)
	for i := 0; i < reasonCodeCount; i++ {
		rc, err := readByte(r)
		if err != nil {
			return nil, err
		}
		pkt.ReasonCodes[i] = ReasonCode(rc)
	}

	return pkt, nil
}

// ParseUnsubscribePacket parses an MQTT 5.0 UNSUBSCRIBE packet
func ParseUnsubscribePacket(r io.Reader, fh *FixedHeader) (*UnsubscribePacket, error) {
	pkt := &UnsubscribePacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	// Read topic filters
	pkt.TopicFilters = make([]string, 0)

	bytesRead := 2 + int(props.Length) + len(EncodeVariableByteIntegerMust(props.Length))

	for bytesRead < int(fh.RemainingLength) {
		topicFilter, err := readUTF8String(r)
		if err != nil {
			return nil, err
		}
		bytesRead += 2 + len(topicFilter)
		pkt.TopicFilters = append(pkt.TopicFilters, topicFilter)
	}

	return pkt, nil
}

// ParseUnsubackPacket parses an MQTT 5.0 UNSUBACK packet
func ParseUnsubackPacket(r io.Reader, fh *FixedHeader) (*UnsubackPacket, error) {
	pkt := &UnsubackPacket{FixedHeader: *fh}

	// Read packet ID
	packetID, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}
	pkt.PacketID = packetID

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	// Read reason codes
	bytesRead := 2 + int(props.Length) + len(EncodeVariableByteIntegerMust(props.Length))
	reasonCodeCount := int(fh.RemainingLength) - bytesRead

	pkt.ReasonCodes = make([]ReasonCode, reasonCodeCount)
	for i := 0; i < reasonCodeCount; i++ {
		rc, err := readByte(r)
		if err != nil {
			return nil, err
		}
		pkt.ReasonCodes[i] = ReasonCode(rc)
	}

	return pkt, nil
}

// ParseDisconnectPacket parses an MQTT 5.0 DISCONNECT packet
func ParseDisconnectPacket(r io.Reader, fh *FixedHeader) (*DisconnectPacket, error) {
	pkt := &DisconnectPacket{FixedHeader: *fh}

	// Remaining length of 0 means normal disconnection
	if fh.RemainingLength == 0 {
		pkt.ReasonCode = ReasonNormalDisconnection
		return pkt, nil
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	if fh.RemainingLength == 1 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParseAuthPacket parses an MQTT 5.0 AUTH packet
func ParseAuthPacket(r io.Reader, fh *FixedHeader) (*AuthPacket, error) {
	pkt := &AuthPacket{FixedHeader: *fh}

	// Remaining length of 0 is not valid for AUTH
	if fh.RemainingLength == 0 {
		return nil, ErrMalformedPacket
	}

	// Read reason code
	reasonCode, err := readByte(r)
	if err != nil {
		return nil, err
	}
	pkt.ReasonCode = ReasonCode(reasonCode)

	if fh.RemainingLength == 1 {
		return pkt, nil
	}

	// Read properties
	props, err := ParseProperties(r)
	if err != nil {
		return nil, err
	}
	pkt.Properties = *props

	return pkt, nil
}

// ParsePingreqPacket parses an MQTT 5.0 PINGREQ packet
func ParsePingreqPacket(fh *FixedHeader) (*PingreqPacket, error) {
	if fh.RemainingLength != 0 {
		return nil, ErrMalformedPacket
	}
	return &PingreqPacket{FixedHeader: *fh}, nil
}

// ParsePingrespPacket parses an MQTT 5.0 PINGRESP packet
func ParsePingrespPacket(fh *FixedHeader) (*PingrespPacket, error) {
	if fh.RemainingLength != 0 {
		return nil, ErrMalformedPacket
	}
	return &PingrespPacket{FixedHeader: *fh}, nil
}

// EncodeVariableByteIntegerMust is a helper that panics on error (for internal use)
func EncodeVariableByteIntegerMust(value uint32) []byte {
	bytes, err := EncodeVariableByteInteger(value)
	if err != nil {
		panic(err)
	}
	return bytes
}

// String returns human-readable reason code name
func (rc ReasonCode) String() string {
	names := map[ReasonCode]string{
		ReasonSuccess:                             "Success",
		ReasonGrantedQoS1:                         "GrantedQoS1",
		ReasonGrantedQoS2:                         "GrantedQoS2",
		ReasonDisconnectWithWillMessage:           "DisconnectWithWillMessage",
		ReasonNoMatchingSubscribers:               "NoMatchingSubscribers",
		ReasonNoSubscriptionExisted:               "NoSubscriptionExisted",
		ReasonContinueAuthentication:              "ContinueAuthentication",
		ReasonReAuthenticate:                      "ReAuthenticate",
		ReasonUnspecifiedError:                    "UnspecifiedError",
		ReasonMalformedPacket:                     "MalformedPacket",
		ReasonProtocolError:                       "ProtocolError",
		ReasonImplementationSpecificError:         "ImplementationSpecificError",
		ReasonUnsupportedProtocolVersion:          "UnsupportedProtocolVersion",
		ReasonClientIdentifierNotValid:            "ClientIdentifierNotValid",
		ReasonBadUsernameOrPassword:               "BadUsernameOrPassword",
		ReasonNotAuthorized:                       "NotAuthorized",
		ReasonServerUnavailable:                   "ServerUnavailable",
		ReasonServerBusy:                          "ServerBusy",
		ReasonBanned:                              "Banned",
		ReasonServerShuttingDown:                  "ServerShuttingDown",
		ReasonBadAuthenticationMethod:             "BadAuthenticationMethod",
		ReasonKeepAliveTimeout:                    "KeepAliveTimeout",
		ReasonSessionTakenOver:                    "SessionTakenOver",
		ReasonTopicFilterInvalid:                  "TopicFilterInvalid",
		ReasonTopicNameInvalid:                    "TopicNameInvalid",
		ReasonPacketIdentifierInUse:               "PacketIdentifierInUse",
		ReasonPacketIdentifierNotFound:            "PacketIdentifierNotFound",
		ReasonReceiveMaximumExceeded:              "ReceiveMaximumExceeded",
		ReasonTopicAliasInvalid:                   "TopicAliasInvalid",
		ReasonPacketTooLarge:                      "PacketTooLarge",
		ReasonMessageRateTooHigh:                  "MessageRateTooHigh",
		ReasonQuotaExceeded:                       "QuotaExceeded",
		ReasonAdministrativeAction:                "AdministrativeAction",
		ReasonPayloadFormatInvalid:                "PayloadFormatInvalid",
		ReasonRetainNotSupported:                  "RetainNotSupported",
		ReasonQoSNotSupported:                     "QoSNotSupported",
		ReasonUseAnotherServer:                    "UseAnotherServer",
		ReasonServerMoved:                         "ServerMoved",
		ReasonSharedSubscriptionsNotSupported:     "SharedSubscriptionsNotSupported",
		ReasonConnectionRateExceeded:              "ConnectionRateExceeded",
		ReasonMaximumConnectTime:                  "MaximumConnectTime",
		ReasonSubscriptionIdentifiersNotSupported: "SubscriptionIdentifiersNotSupported",
		ReasonWildcardSubscriptionsNotSupported:   "WildcardSubscriptionsNotSupported",
	}

	if name, ok := names[rc]; ok {
		return name
	}
	return "UNKNOWN"
}
