package encoding

import (
	"io"
)

// MQTT 3.1.1 Packet Encoders
// These encoders are optimized for MQTT 3.1.1 protocol version

// ConnectPacket311 represents an MQTT 3.1.1 CONNECT packet
type ConnectPacket311 struct {
	FixedHeader     FixedHeader
	ProtocolName    string
	ProtocolVersion ProtocolVersion
	CleanSession    bool
	WillFlag        bool
	WillQoS         QoS
	WillRetain      bool
	PasswordFlag    bool
	UsernameFlag    bool
	KeepAlive       uint16
	ClientID        string
	WillTopic       string
	WillPayload     []byte
	Username        string
	Password        []byte
}

// ConnackPacket311 represents an MQTT 3.1.1 CONNACK packet
type ConnackPacket311 struct {
	FixedHeader    FixedHeader
	SessionPresent bool
	ReturnCode     byte // 3.1.1 uses return codes instead of reason codes
}

// PublishPacket311 represents an MQTT 3.1.1 PUBLISH packet
type PublishPacket311 struct {
	FixedHeader FixedHeader
	TopicName   string
	PacketID    uint16
	Payload     []byte
}

// SubscribePacket311 represents an MQTT 3.1.1 SUBSCRIBE packet
type SubscribePacket311 struct {
	FixedHeader   FixedHeader
	PacketID      uint16
	Subscriptions []Subscription311
}

// Subscription311 represents a single subscription in MQTT 3.1.1
type Subscription311 struct {
	TopicFilter string
	QoS         QoS
}

// SubackPacket311 represents an MQTT 3.1.1 SUBACK packet
type SubackPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
	ReturnCodes []byte
}

// UnsubscribePacket311 represents an MQTT 3.1.1 UNSUBSCRIBE packet
type UnsubscribePacket311 struct {
	FixedHeader  FixedHeader
	PacketID     uint16
	TopicFilters []string
}

// UnsubackPacket311 represents an MQTT 3.1.1 UNSUBACK packet
type UnsubackPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
}

// DisconnectPacket311 represents an MQTT 3.1.1 DISCONNECT packet
type DisconnectPacket311 struct {
	FixedHeader FixedHeader
}

// PubackPacket311 represents an MQTT 3.1.1 PUBACK packet
type PubackPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
}

// PubrecPacket311 represents an MQTT 3.1.1 PUBREC packet
type PubrecPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
}

// PubrelPacket311 represents an MQTT 3.1.1 PUBREL packet
type PubrelPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
}

// PubcompPacket311 represents an MQTT 3.1.1 PUBCOMP packet
type PubcompPacket311 struct {
	FixedHeader FixedHeader
	PacketID    uint16
}

// Encode encodes an MQTT 3.1.1 CONNECT packet
func (p *ConnectPacket311) Encode(w io.Writer) error {
	// Calculate variable header + payload length
	varHeaderLen := 0

	// Protocol name (2 bytes length + "MQTT" for 3.1.1, or "MQIsdp" for 3.0)
	varHeaderLen += 2 + len(p.ProtocolName)

	// Protocol version (1 byte)
	varHeaderLen += 1

	// Connect flags (1 byte)
	varHeaderLen += 1

	// Keep alive (2 bytes)
	varHeaderLen += 2

	// Payload calculations
	payloadLen := 0

	// Client ID
	payloadLen += 2 + len(p.ClientID)

	// Will topic and payload
	if p.WillFlag {
		payloadLen += 2 + len(p.WillTopic)
		payloadLen += 2 + len(p.WillPayload)
	}

	// Username
	if p.UsernameFlag {
		payloadLen += 2 + len(p.Username)
	}

	// Password
	if p.PasswordFlag {
		payloadLen += 2 + len(p.Password)
	}

	remainingLength := uint32(varHeaderLen + payloadLen)

	// Encode fixed header
	fh := FixedHeader{
		Type:            CONNECT,
		Flags:           0,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Encode variable header

	// Protocol name
	if err := writeUTF8String(w, p.ProtocolName); err != nil {
		return err
	}

	// Protocol version
	if err := writeByte(w, byte(p.ProtocolVersion)); err != nil {
		return err
	}

	// Connect flags
	var connectFlags byte
	if p.CleanSession {
		connectFlags |= 0x02
	}
	if p.WillFlag {
		connectFlags |= 0x04
		connectFlags |= byte(p.WillQoS << 3)
		if p.WillRetain {
			connectFlags |= 0x20
		}
	}
	if p.PasswordFlag {
		connectFlags |= 0x40
	}
	if p.UsernameFlag {
		connectFlags |= 0x80
	}

	if err := writeByte(w, connectFlags); err != nil {
		return err
	}

	// Keep alive
	if err := writeTwoByteInt(w, p.KeepAlive); err != nil {
		return err
	}

	// Payload

	// Client ID
	if err := writeUTF8String(w, p.ClientID); err != nil {
		return err
	}

	// Will topic and payload
	if p.WillFlag {
		if err := writeUTF8String(w, p.WillTopic); err != nil {
			return err
		}

		if err := writeBinaryData(w, p.WillPayload); err != nil {
			return err
		}
	}

	// Username
	if p.UsernameFlag {
		if err := writeUTF8String(w, p.Username); err != nil {
			return err
		}
	}

	// Password
	if p.PasswordFlag {
		if err := writeBinaryData(w, p.Password); err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes an MQTT 3.1.1 CONNACK packet
func (p *ConnackPacket311) Encode(w io.Writer) error {
	// Encode fixed header
	fh := FixedHeader{
		Type:            CONNACK,
		Flags:           0,
		RemainingLength: 2, // flags + return code
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Connect acknowledge flags
	var ackFlags byte
	if p.SessionPresent {
		ackFlags |= 0x01
	}
	if err := writeByte(w, ackFlags); err != nil {
		return err
	}

	// Return code
	return writeByte(w, p.ReturnCode)
}

// Encode encodes an MQTT 3.1.1 PUBLISH packet
func (p *PublishPacket311) Encode(w io.Writer) error {
	remainingLength := uint32(2 + len(p.TopicName) + len(p.Payload))

	// Add packet ID for QoS 1 and 2
	if p.FixedHeader.QoS > QoS0 {
		remainingLength += 2
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            PUBLISH,
		Flags:           p.FixedHeader.Flags,
		RemainingLength: remainingLength,
		DUP:             p.FixedHeader.DUP,
		QoS:             p.FixedHeader.QoS,
		Retain:          p.FixedHeader.Retain,
	}

	// Construct flags for PUBLISH
	fh.Flags = 0
	if fh.DUP {
		fh.Flags |= 0x08
	}
	fh.Flags |= byte(fh.QoS) << 1
	if fh.Retain {
		fh.Flags |= 0x01
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Topic name
	if err := writeUTF8String(w, p.TopicName); err != nil {
		return err
	}

	// Packet ID (only for QoS 1 and 2)
	if p.FixedHeader.QoS > QoS0 {
		if err := writeTwoByteInt(w, p.PacketID); err != nil {
			return err
		}
	}

	// Payload
	if len(p.Payload) > 0 {
		_, err := w.Write(p.Payload)
		return err
	}

	return nil
}

// Encode encodes an MQTT 3.1.1 PUBACK packet
func (p *PubackPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PUBACK,
		Flags:           0,
		RemainingLength: 2,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	return writeTwoByteInt(w, p.PacketID)
}

// Encode encodes an MQTT 3.1.1 PUBREC packet
func (p *PubrecPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PUBREC,
		Flags:           0,
		RemainingLength: 2,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	return writeTwoByteInt(w, p.PacketID)
}

// Encode encodes an MQTT 3.1.1 PUBREL packet
func (p *PubrelPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PUBREL,
		Flags:           0x02, // Reserved flags must be 0010
		RemainingLength: 2,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	return writeTwoByteInt(w, p.PacketID)
}

// Encode encodes an MQTT 3.1.1 PUBCOMP packet
func (p *PubcompPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PUBCOMP,
		Flags:           0,
		RemainingLength: 2,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	return writeTwoByteInt(w, p.PacketID)
}

// Encode encodes an MQTT 3.1.1 SUBSCRIBE packet
func (p *SubscribePacket311) Encode(w io.Writer) error {
	remainingLength := uint32(2) // Packet ID

	// Add subscription lengths
	for _, sub := range p.Subscriptions {
		remainingLength += uint32(2 + len(sub.TopicFilter) + 1) // length prefix + topic + QoS byte
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            SUBSCRIBE,
		Flags:           0x02, // Reserved flags must be 0010
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, p.PacketID); err != nil {
		return err
	}

	// Subscriptions
	for _, sub := range p.Subscriptions {
		if err := writeUTF8String(w, sub.TopicFilter); err != nil {
			return err
		}

		if err := writeByte(w, byte(sub.QoS)); err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes an MQTT 3.1.1 SUBACK packet
func (p *SubackPacket311) Encode(w io.Writer) error {
	remainingLength := uint32(2 + len(p.ReturnCodes))

	// Encode fixed header
	fh := FixedHeader{
		Type:            SUBACK,
		Flags:           0,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, p.PacketID); err != nil {
		return err
	}

	// Return codes
	_, err := w.Write(p.ReturnCodes)
	return err
}

// Encode encodes an MQTT 3.1.1 UNSUBSCRIBE packet
func (p *UnsubscribePacket311) Encode(w io.Writer) error {
	remainingLength := uint32(2) // Packet ID

	// Add topic filter lengths
	for _, topic := range p.TopicFilters {
		remainingLength += uint32(2 + len(topic))
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            UNSUBSCRIBE,
		Flags:           0x02, // Reserved flags must be 0010
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, p.PacketID); err != nil {
		return err
	}

	// Topic filters
	for _, topic := range p.TopicFilters {
		if err := writeUTF8String(w, topic); err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes an MQTT 3.1.1 UNSUBACK packet
func (p *UnsubackPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            UNSUBACK,
		Flags:           0,
		RemainingLength: 2,
	}

	if err := fh.EncodeFixedHeader311(w); err != nil {
		return err
	}

	return writeTwoByteInt(w, p.PacketID)
}

// Encode encodes an MQTT 3.1.1 DISCONNECT packet
func (p *DisconnectPacket311) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            DISCONNECT,
		Flags:           0,
		RemainingLength: 0,
	}
	return fh.EncodeFixedHeader311(w)
}

// MQTT 3.1.1 Return Codes
const (
	ConnectAccepted311                    byte = 0x00
	ConnectRefusedUnacceptableProtocol311 byte = 0x01
	ConnectRefusedIdentifierRejected311   byte = 0x02
	ConnectRefusedServerUnavailable311    byte = 0x03
	ConnectRefusedBadUsernamePassword311  byte = 0x04
	ConnectRefusedNotAuthorized311        byte = 0x05
)
