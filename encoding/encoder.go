package encoding

import (
	"bytes"
	"io"
)

// Encode encodes an MQTT 5.0 CONNECT packet
func (p *ConnectPacket) Encode(w io.Writer) error {
	// Calculate variable header + payload length
	varHeaderLen := 0

	// Protocol name (2 bytes length + "MQTT")
	varHeaderLen += 2 + len(p.ProtocolName)

	// Protocol version (1 byte)
	varHeaderLen += 1

	// Connect flags (1 byte)
	varHeaderLen += 1

	// Keep alive (2 bytes)
	varHeaderLen += 2

	// Properties
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}
	varHeaderLen += len(propsBytes)

	// Payload calculations
	payloadLen := 0

	// Client ID
	payloadLen += 2 + len(p.ClientID)

	// Will properties, topic, and payload
	if p.WillFlag {
		willPropsBytes, err := p.WillProperties.encodeToBytes()
		if err != nil {
			return err
		}
		payloadLen += len(willPropsBytes)
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

	if err := fh.EncodeFixedHeader(w); err != nil {
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
	if p.CleanStart {
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

	// Properties
	if _, err := w.Write(propsBytes); err != nil {
		return err
	}

	// Payload

	// Client ID
	if err := writeUTF8String(w, p.ClientID); err != nil {
		return err
	}

	// Will properties, topic, and payload
	if p.WillFlag {
		willPropsBytes, _ := p.WillProperties.encodeToBytes()
		if _, err := w.Write(willPropsBytes); err != nil {
			return err
		}

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

// Encode encodes an MQTT 5.0 CONNACK packet
func (p *ConnackPacket) Encode(w io.Writer) error {
	// Calculate remaining length
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(1 + 1 + len(propsBytes)) // flags + reason code + properties

	// Encode fixed header
	fh := FixedHeader{
		Type:            CONNACK,
		Flags:           0,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Encode variable header

	// Connect acknowledge flags
	var ackFlags byte
	if p.SessionPresent {
		ackFlags |= 0x01
	}
	if err := writeByte(w, ackFlags); err != nil {
		return err
	}

	// Reason code
	if err := writeByte(w, byte(p.ReasonCode)); err != nil {
		return err
	}

	// Properties
	_, err = w.Write(propsBytes)
	return err
}

// Encode encodes an MQTT 5.0 PUBLISH packet
func (p *PublishPacket) Encode(w io.Writer) error {
	// Calculate remaining length
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(2 + len(p.TopicName) + len(propsBytes) + len(p.Payload))

	// Add packet ID for QoS 1 and 2
	if p.FixedHeader.QoS > QoS0 {
		remainingLength += 2
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            PUBLISH,
		Flags:           p.FixedHeader.BuildPublishFlags(),
		RemainingLength: remainingLength,
		DUP:             p.FixedHeader.DUP,
		QoS:             p.FixedHeader.QoS,
		Retain:          p.FixedHeader.Retain,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Encode variable header

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

	// Properties
	if _, err := w.Write(propsBytes); err != nil {
		return err
	}

	// Payload
	if len(p.Payload) > 0 {
		_, err = w.Write(p.Payload)
	}

	return err
}

// Encode encodes an MQTT 5.0 PUBACK packet
func (p *PubackPacket) Encode(w io.Writer) error {
	return encodeAckPacket(w, PUBACK, p.PacketID, p.ReasonCode, &p.Properties)
}

// Encode encodes an MQTT 5.0 PUBREC packet
func (p *PubrecPacket) Encode(w io.Writer) error {
	return encodeAckPacket(w, PUBREC, p.PacketID, p.ReasonCode, &p.Properties)
}

// Encode encodes an MQTT 5.0 PUBREL packet
func (p *PubrelPacket) Encode(w io.Writer) error {
	return encodeAckPacketWithFlags(w, PUBREL, 0x02, p.PacketID, p.ReasonCode, &p.Properties)
}

// Encode encodes an MQTT 5.0 PUBCOMP packet
func (p *PubcompPacket) Encode(w io.Writer) error {
	return encodeAckPacket(w, PUBCOMP, p.PacketID, p.ReasonCode, &p.Properties)
}

// encodeAckPacket is a helper to encode acknowledgment packets (PUBACK, PUBREC, PUBCOMP)
func encodeAckPacket(w io.Writer, packetType PacketType, packetID uint16, reasonCode ReasonCode, props *Properties) error {
	return encodeAckPacketWithFlags(w, packetType, 0, packetID, reasonCode, props)
}

// encodeAckPacketWithFlags is a helper to encode acknowledgment packets with custom flags
func encodeAckPacketWithFlags(w io.Writer, packetType PacketType, flags byte, packetID uint16, reasonCode ReasonCode, props *Properties) error {
	propsBytes, err := props.encodeToBytes()
	if err != nil {
		return err
	}

	// Calculate remaining length
	remainingLength := uint32(2) // Packet ID

	// Optimize: if reason code is success and no properties, omit them
	if reasonCode != ReasonSuccess || len(propsBytes) > 1 {
		remainingLength += 1 + uint32(len(propsBytes))
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            packetType,
		Flags:           flags,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, packetID); err != nil {
		return err
	}

	// Reason code and properties (if not omitted)
	if reasonCode != ReasonSuccess || len(propsBytes) > 1 {
		if err := writeByte(w, byte(reasonCode)); err != nil {
			return err
		}

		_, err = w.Write(propsBytes)
	}

	return err
}

// writeReasonCodes is a helper to write a slice of reason codes
func writeReasonCodes(w io.Writer, reasonCodes []ReasonCode) error {
	for _, rc := range reasonCodes {
		if err := writeByte(w, byte(rc)); err != nil {
			return err
		}
	}
	return nil
}

// encodeAckPacketWithReasonCodes is a helper to encode acknowledgment packets with reason codes (SUBACK, UNSUBACK)
func encodeAckPacketWithReasonCodes(w io.Writer, packetType PacketType, flags byte, packetID uint16, reasonCodes []ReasonCode, props *Properties) error {
	propsBytes, err := props.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(2 + len(propsBytes) + len(reasonCodes))

	// Encode fixed header
	fh := FixedHeader{
		Type:            packetType,
		Flags:           flags,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, packetID); err != nil {
		return err
	}

	// Properties
	if _, err := w.Write(propsBytes); err != nil {
		return err
	}

	// Reason codes
	return writeReasonCodes(w, reasonCodes)
}

// Encode encodes an MQTT 5.0 SUBSCRIBE packet
func (p *SubscribePacket) Encode(w io.Writer) error {
	// Calculate remaining length
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(2 + len(propsBytes)) // Packet ID + properties

	// Add subscription lengths
	for _, sub := range p.Subscriptions {
		remainingLength += uint32(2 + len(sub.TopicFilter) + 1) // length prefix + topic + options
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            SUBSCRIBE,
		Flags:           0x02, // Reserved flags must be 0010
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, p.PacketID); err != nil {
		return err
	}

	// Properties
	if _, err := w.Write(propsBytes); err != nil {
		return err
	}

	// Subscriptions
	for _, sub := range p.Subscriptions {
		if err := writeUTF8String(w, sub.TopicFilter); err != nil {
			return err
		}

		// Subscription options
		var options byte
		options = byte(sub.QoS & 0x03)
		if sub.NoLocal {
			options |= 0x04
		}
		if sub.RetainAsPublished {
			options |= 0x08
		}
		options |= (sub.RetainHandling & 0x03) << 4

		if err := writeByte(w, options); err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes an MQTT 5.0 SUBACK packet
func (p *SubackPacket) Encode(w io.Writer) error {
	return encodeAckPacketWithReasonCodes(w, SUBACK, 0, p.PacketID, p.ReasonCodes, &p.Properties)
}

// Encode encodes an MQTT 5.0 UNSUBSCRIBE packet
func (p *UnsubscribePacket) Encode(w io.Writer) error {
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(2 + len(propsBytes)) // Packet ID + properties

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

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Packet ID
	if err := writeTwoByteInt(w, p.PacketID); err != nil {
		return err
	}

	// Properties
	if _, err := w.Write(propsBytes); err != nil {
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

// Encode encodes an MQTT 5.0 UNSUBACK packet
func (p *UnsubackPacket) Encode(w io.Writer) error {
	return encodeAckPacketWithReasonCodes(w, UNSUBACK, 0, p.PacketID, p.ReasonCodes, &p.Properties)
}

// Encode encodes an MQTT 5.0 PINGREQ packet
func (p *PingreqPacket) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PINGREQ,
		Flags:           0,
		RemainingLength: 0,
	}
	return fh.EncodeFixedHeader(w)
}

// Encode encodes an MQTT 5.0 PINGRESP packet
func (p *PingrespPacket) Encode(w io.Writer) error {
	fh := FixedHeader{
		Type:            PINGRESP,
		Flags:           0,
		RemainingLength: 0,
	}
	return fh.EncodeFixedHeader(w)
}

// Encode encodes an MQTT 5.0 DISCONNECT packet
func (p *DisconnectPacket) Encode(w io.Writer) error {
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	// Calculate remaining length
	remainingLength := uint32(0)

	// Optimize: if reason code is normal disconnection and no properties, omit them
	if p.ReasonCode != ReasonNormalDisconnection || len(propsBytes) > 1 {
		remainingLength = 1 + uint32(len(propsBytes))
	}

	// Encode fixed header
	fh := FixedHeader{
		Type:            DISCONNECT,
		Flags:           0,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Reason code and properties (if not omitted)
	if remainingLength > 0 {
		if err := writeByte(w, byte(p.ReasonCode)); err != nil {
			return err
		}

		_, err = w.Write(propsBytes)
	}

	return err
}

// Encode encodes an MQTT 5.0 AUTH packet
func (p *AuthPacket) Encode(w io.Writer) error {
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return err
	}

	remainingLength := uint32(1 + len(propsBytes)) // Reason code + properties

	// Encode fixed header
	fh := FixedHeader{
		Type:            AUTH,
		Flags:           0,
		RemainingLength: remainingLength,
	}

	if err := fh.EncodeFixedHeader(w); err != nil {
		return err
	}

	// Reason code
	if err := writeByte(w, byte(p.ReasonCode)); err != nil {
		return err
	}

	// Properties
	_, err = w.Write(propsBytes)
	return err
}

// encodeToBytes is a helper to encode properties to a byte slice
func (p *Properties) encodeToBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := p.EncodeProperties(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeTo encodes a packet to a byte slice with pre-allocated buffer
// This is a zero-allocation optimization when the buffer size is known
func (p *PublishPacket) EncodeTo(buf []byte) (int, error) {
	// Calculate required size
	propsBytes, err := p.Properties.encodeToBytes()
	if err != nil {
		return 0, err
	}

	remainingLength := uint32(2 + len(p.TopicName) + len(propsBytes) + len(p.Payload))
	if p.FixedHeader.QoS > QoS0 {
		remainingLength += 2
	}

	// Create fixed header
	fh := FixedHeader{
		Type:            PUBLISH,
		RemainingLength: remainingLength,
	}

	// Construct flags for PUBLISH
	fh.Flags = 0
	if p.FixedHeader.DUP {
		fh.Flags |= 0x08
	}
	fh.Flags |= byte(p.FixedHeader.QoS) << 1
	if p.FixedHeader.Retain {
		fh.Flags |= 0x01
	}

	offset := 0

	// Encode fixed header
	n, err := fh.EncodeFixedHeaderToBytes(buf)
	if err != nil {
		return 0, err
	}
	offset += n

	// Topic name
	n, err = writeUTF8StringToBytes(buf[offset:], p.TopicName)
	if err != nil {
		return 0, err
	}
	offset += n

	// Packet ID (only for QoS 1 and 2)
	if p.FixedHeader.QoS > QoS0 {
		n, err = writeTwoByteIntToBytes(buf[offset:], p.PacketID)
		if err != nil {
			return 0, err
		}
		offset += n
	}

	// Properties
	copy(buf[offset:], propsBytes)
	offset += len(propsBytes)

	// Payload
	copy(buf[offset:], p.Payload)
	offset += len(p.Payload)

	return offset, nil
}
