package encoding

import (
	"io"
)

// PropertyID represents MQTT 5.0 property identifiers
type PropertyID byte

const (
	PropPayloadFormatIndicator          PropertyID = 0x01
	PropMessageExpiryInterval           PropertyID = 0x02
	PropContentType                     PropertyID = 0x03
	PropResponseTopic                   PropertyID = 0x08
	PropCorrelationData                 PropertyID = 0x09
	PropSubscriptionIdentifier          PropertyID = 0x0B
	PropSessionExpiryInterval           PropertyID = 0x11
	PropAssignedClientIdentifier        PropertyID = 0x12
	PropServerKeepAlive                 PropertyID = 0x13
	PropAuthenticationMethod            PropertyID = 0x15
	PropAuthenticationData              PropertyID = 0x16
	PropRequestProblemInformation       PropertyID = 0x17
	PropWillDelayInterval               PropertyID = 0x18
	PropRequestResponseInformation      PropertyID = 0x19
	PropResponseInformation             PropertyID = 0x1A
	PropServerReference                 PropertyID = 0x1C
	PropReasonString                    PropertyID = 0x1F
	PropReceiveMaximum                  PropertyID = 0x21
	PropTopicAliasMaximum               PropertyID = 0x22
	PropTopicAlias                      PropertyID = 0x23
	PropMaximumQoS                      PropertyID = 0x24
	PropRetainAvailable                 PropertyID = 0x25
	PropUserProperty                    PropertyID = 0x26
	PropMaximumPacketSize               PropertyID = 0x27
	PropWildcardSubscriptionAvailable   PropertyID = 0x28
	PropSubscriptionIdentifierAvailable PropertyID = 0x29
	PropSharedSubscriptionAvailable     PropertyID = 0x2A
)

// PropertyType represents the data type of a property
type PropertyType byte

const (
	PropertyTypeByte        PropertyType = 1
	PropertyTypeTwoByteInt  PropertyType = 2
	PropertyTypeFourByteInt PropertyType = 3
	PropertyTypeVarInt      PropertyType = 4
	PropertyTypeUTF8String  PropertyType = 5
	PropertyTypeUTF8Pair    PropertyType = 6
	PropertyTypeBinaryData  PropertyType = 7
)

// Property represents a single MQTT 5.0 property
type Property struct {
	ID    PropertyID
	Value interface{}
}

// Properties represents a collection of MQTT 5.0 properties
type Properties struct {
	Properties []Property
	Length     uint32 // Total length in bytes
}

// propertySpec defines the expected type and multiplicity for each property
type propertySpec struct {
	Type     PropertyType
	Multiple bool // Can this property appear multiple times?
}

// propertySpecs maps property IDs to their specifications
var propertySpecs = map[PropertyID]propertySpec{
	PropPayloadFormatIndicator:          {PropertyTypeByte, false},
	PropMessageExpiryInterval:           {PropertyTypeFourByteInt, false},
	PropContentType:                     {PropertyTypeUTF8String, false},
	PropResponseTopic:                   {PropertyTypeUTF8String, false},
	PropCorrelationData:                 {PropertyTypeBinaryData, false},
	PropSubscriptionIdentifier:          {PropertyTypeVarInt, true},
	PropSessionExpiryInterval:           {PropertyTypeFourByteInt, false},
	PropAssignedClientIdentifier:        {PropertyTypeUTF8String, false},
	PropServerKeepAlive:                 {PropertyTypeTwoByteInt, false},
	PropAuthenticationMethod:            {PropertyTypeUTF8String, false},
	PropAuthenticationData:              {PropertyTypeBinaryData, false},
	PropRequestProblemInformation:       {PropertyTypeByte, false},
	PropWillDelayInterval:               {PropertyTypeFourByteInt, false},
	PropRequestResponseInformation:      {PropertyTypeByte, false},
	PropResponseInformation:             {PropertyTypeUTF8String, false},
	PropServerReference:                 {PropertyTypeUTF8String, false},
	PropReasonString:                    {PropertyTypeUTF8String, false},
	PropReceiveMaximum:                  {PropertyTypeTwoByteInt, false},
	PropTopicAliasMaximum:               {PropertyTypeTwoByteInt, false},
	PropTopicAlias:                      {PropertyTypeTwoByteInt, false},
	PropMaximumQoS:                      {PropertyTypeByte, false},
	PropRetainAvailable:                 {PropertyTypeByte, false},
	PropUserProperty:                    {PropertyTypeUTF8Pair, true},
	PropMaximumPacketSize:               {PropertyTypeFourByteInt, false},
	PropWildcardSubscriptionAvailable:   {PropertyTypeByte, false},
	PropSubscriptionIdentifierAvailable: {PropertyTypeByte, false},
	PropSharedSubscriptionAvailable:     {PropertyTypeByte, false},
}

// ParseProperties parses MQTT 5.0 properties from a reader
func ParseProperties(r io.Reader) (*Properties, error) {
	// Read property length (Variable Byte Integer)
	propLength, err := DecodeVariableByteInteger(r)
	if err != nil {
		return nil, err
	}

	props := &Properties{
		Length:     propLength,
		Properties: make([]Property, 0, 4),
	}

	// If no properties, return empty collection
	if propLength == 0 {
		return props, nil
	}

	// Create a limited reader to ensure we don't read beyond property length
	limitedReader := io.LimitedReader{R: r, N: int64(propLength)}

	// Parse individual properties
	for limitedReader.N > 0 {
		prop, err := parseProperty(&limitedReader)
		if err != nil {
			return nil, err
		}
		props.Properties = append(props.Properties, *prop)
	}

	return props, nil
}

// ParsePropertiesFromBytes parses MQTT 5.0 properties from a byte slice
func ParsePropertiesFromBytes(data []byte) (*Properties, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrUnexpectedEOF
	}

	offset := 0

	// Read property length (Variable Byte Integer)
	propLength, bytesRead, err := DecodeVariableByteIntegerFromBytes(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	offset += bytesRead

	props := &Properties{
		Length:     propLength,
		Properties: make([]Property, 0),
	}

	// If no properties, return empty collection
	if propLength == 0 {
		return props, offset, nil
	}

	// Ensure we have enough data
	if len(data[offset:]) < int(propLength) {
		return nil, 0, ErrUnexpectedEOF
	}

	// Parse individual properties
	propEnd := offset + int(propLength)
	for offset < propEnd {
		prop, bytesRead, err := parsePropertyFromBytes(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		props.Properties = append(props.Properties, *prop)
		offset += bytesRead
	}

	return props, offset, nil
}

// parseProperty parses a single property from a reader
func parseProperty(r io.Reader) (*Property, error) {
	// Read property ID
	var idByte [1]byte
	if _, err := io.ReadFull(r, idByte[:]); err != nil {
		if err == io.EOF {
			return nil, ErrUnexpectedEOF
		}
		return nil, err
	}

	propID := PropertyID(idByte[0])
	spec, ok := propertySpecs[propID]
	if !ok {
		return nil, ErrInvalidPropertyID
	}

	prop := &Property{ID: propID}

	// Parse property value based on type
	var err error
	switch spec.Type {
	case PropertyTypeByte:
		prop.Value, err = readByte(r)
	case PropertyTypeTwoByteInt:
		prop.Value, err = readTwoByteInt(r)
	case PropertyTypeFourByteInt:
		prop.Value, err = readFourByteInt(r)
	case PropertyTypeVarInt:
		prop.Value, err = DecodeVariableByteInteger(r)
	case PropertyTypeUTF8String:
		prop.Value, err = readUTF8String(r)
	case PropertyTypeUTF8Pair:
		prop.Value, err = readUTF8Pair(r)
	case PropertyTypeBinaryData:
		prop.Value, err = readBinaryData(r)
	default:
		return nil, ErrInvalidPropertyType
	}

	if err != nil {
		return nil, err
	}

	return prop, nil
}

// parsePropertyFromBytes parses a single property from a byte slice
func parsePropertyFromBytes(data []byte) (*Property, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrUnexpectedEOF
	}

	offset := 0

	// Read property ID
	propID := PropertyID(data[offset])
	offset++

	spec, ok := propertySpecs[propID]
	if !ok {
		return nil, 0, ErrInvalidPropertyID
	}

	prop := &Property{ID: propID}

	// Parse property value based on type
	var err error
	var bytesRead int

	switch spec.Type {
	case PropertyTypeByte:
		prop.Value, bytesRead, err = readByteFromBytes(data[offset:])
	case PropertyTypeTwoByteInt:
		prop.Value, bytesRead, err = readTwoByteIntFromBytes(data[offset:])
	case PropertyTypeFourByteInt:
		prop.Value, bytesRead, err = readFourByteIntFromBytes(data[offset:])
	case PropertyTypeVarInt:
		prop.Value, bytesRead, err = DecodeVariableByteIntegerFromBytes(data[offset:])
	case PropertyTypeUTF8String:
		prop.Value, bytesRead, err = readUTF8StringFromBytes(data[offset:])
	case PropertyTypeUTF8Pair:
		prop.Value, bytesRead, err = readUTF8PairFromBytes(data[offset:])
	case PropertyTypeBinaryData:
		prop.Value, bytesRead, err = readBinaryDataFromBytes(data[offset:])
	default:
		return nil, 0, ErrInvalidPropertyType
	}

	if err != nil {
		return nil, 0, err
	}
	offset += bytesRead

	return prop, offset, nil
}

// EncodeProperties encodes MQTT 5.0 properties to a writer
func (p *Properties) EncodeProperties(w io.Writer) error {
	// Calculate total property length
	length := p.calculateLength()

	// Encode property length
	lengthBytes, err := EncodeVariableByteInteger(length)
	if err != nil {
		return err
	}
	if _, err := w.Write(lengthBytes); err != nil {
		return err
	}

	// If no properties, we're done
	if length == 0 {
		return nil
	}

	// Encode each property
	for _, prop := range p.Properties {
		if err := encodeProperty(w, &prop); err != nil {
			return err
		}
	}

	return nil
}

// EncodePropertiesToBytes encodes MQTT 5.0 properties to a byte slice
func (p *Properties) EncodePropertiesToBytes(buf []byte) (int, error) {
	// Calculate total property length
	length := p.calculateLength()

	offset := 0

	// Encode property length
	bytesWritten, err := EncodeVariableByteIntegerTo(buf, offset, length)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	// If no properties, we're done
	if length == 0 {
		return offset, nil
	}

	// Encode each property
	for _, prop := range p.Properties {
		bytesWritten, err := encodePropertyToBytes(buf[offset:], &prop)
		if err != nil {
			return 0, err
		}
		offset += bytesWritten
	}

	return offset, nil
}

// calculateLength calculates the total byte length of all properties
func (p *Properties) calculateLength() uint32 {
	if len(p.Properties) == 0 {
		return 0
	}

	var length uint32
	for _, prop := range p.Properties {
		length++ // Property ID byte

		spec := propertySpecs[prop.ID]
		switch spec.Type {
		case PropertyTypeByte:
			length += 1
		case PropertyTypeTwoByteInt:
			length += 2
		case PropertyTypeFourByteInt:
			length += 4
		case PropertyTypeVarInt:
			val := prop.Value.(uint32)
			varIntBytes, _ := EncodeVariableByteInteger(val)
			length += uint32(len(varIntBytes))
		case PropertyTypeUTF8String:
			str := prop.Value.(string)
			length += 2 + uint32(len(str))
		case PropertyTypeUTF8Pair:
			pair := prop.Value.(UTF8Pair)
			length += 2 + uint32(len(pair.Key)) + 2 + uint32(len(pair.Value))
		case PropertyTypeBinaryData:
			data := prop.Value.([]byte)
			length += 2 + uint32(len(data))
		}
	}

	return length
}

// encodeProperty encodes a single property to a writer
func encodeProperty(w io.Writer, prop *Property) error {
	// Write property ID
	if _, err := w.Write([]byte{byte(prop.ID)}); err != nil {
		return err
	}

	spec := propertySpecs[prop.ID]

	// Write property value based on type
	switch spec.Type {
	case PropertyTypeByte:
		return writeByte(w, prop.Value.(byte))
	case PropertyTypeTwoByteInt:
		return writeTwoByteInt(w, prop.Value.(uint16))
	case PropertyTypeFourByteInt:
		return writeFourByteInt(w, prop.Value.(uint32))
	case PropertyTypeVarInt:
		bytes, err := EncodeVariableByteInteger(prop.Value.(uint32))
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		return err
	case PropertyTypeUTF8String:
		return writeUTF8String(w, prop.Value.(string))
	case PropertyTypeUTF8Pair:
		return writeUTF8Pair(w, prop.Value.(UTF8Pair))
	case PropertyTypeBinaryData:
		return writeBinaryData(w, prop.Value.([]byte))
	default:
		return ErrInvalidPropertyType
	}
}

// encodePropertyToBytes encodes a single property to a byte slice
func encodePropertyToBytes(buf []byte, prop *Property) (int, error) {
	if len(buf) < 1 {
		return 0, ErrBufferTooSmall
	}

	offset := 0

	// Write property ID
	buf[offset] = byte(prop.ID)
	offset++

	spec := propertySpecs[prop.ID]

	// Write property value based on type
	var bytesWritten int
	var err error

	switch spec.Type {
	case PropertyTypeByte:
		bytesWritten, err = writeByteToBytes(buf[offset:], prop.Value.(byte))
	case PropertyTypeTwoByteInt:
		bytesWritten, err = writeTwoByteIntToBytes(buf[offset:], prop.Value.(uint16))
	case PropertyTypeFourByteInt:
		bytesWritten, err = writeFourByteIntToBytes(buf[offset:], prop.Value.(uint32))
	case PropertyTypeVarInt:
		bytesWritten, err = EncodeVariableByteIntegerTo(buf, offset, prop.Value.(uint32))
	case PropertyTypeUTF8String:
		bytesWritten, err = writeUTF8StringToBytes(buf[offset:], prop.Value.(string))
	case PropertyTypeUTF8Pair:
		bytesWritten, err = writeUTF8PairToBytes(buf[offset:], prop.Value.(UTF8Pair))
	case PropertyTypeBinaryData:
		bytesWritten, err = writeBinaryDataToBytes(buf[offset:], prop.Value.([]byte))
	default:
		return 0, ErrInvalidPropertyType
	}

	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	return offset, nil
}

// UTF8Pair represents a key-value pair for user properties
type UTF8Pair struct {
	Key   string
	Value string
}

// Helper functions for reading/writing different data types

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		if err == io.EOF {
			return 0, ErrUnexpectedEOF
		}
		return 0, err
	}
	return b[0], nil
}

func readByteFromBytes(data []byte) (byte, int, error) {
	if len(data) < 1 {
		return 0, 0, ErrUnexpectedEOF
	}
	return data[0], 1, nil
}

func readTwoByteInt(r io.Reader) (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		if err == io.EOF {
			return 0, ErrUnexpectedEOF
		}
		return 0, err
	}
	return uint16(b[0])<<8 | uint16(b[1]), nil
}

func readTwoByteIntFromBytes(data []byte) (uint16, int, error) {
	if len(data) < 2 {
		return 0, 0, ErrUnexpectedEOF
	}
	value := uint16(data[0])<<8 | uint16(data[1])
	return value, 2, nil
}

func readFourByteInt(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		if err == io.EOF {
			return 0, ErrUnexpectedEOF
		}
		return 0, err
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
}

func readFourByteIntFromBytes(data []byte) (uint32, int, error) {
	if len(data) < 4 {
		return 0, 0, ErrUnexpectedEOF
	}
	value := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	return value, 4, nil
}

func readUTF8String(r io.Reader) (string, error) {
	length, err := readTwoByteInt(r)
	if err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", ErrUnexpectedEOF
	}

	// Validate UTF-8 encoding
	if err := ValidateUTF8String(buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func readUTF8StringFromBytes(data []byte) (string, int, error) {
	if len(data) < 2 {
		return "", 0, ErrUnexpectedEOF
	}

	length := uint16(data[0])<<8 | uint16(data[1])
	offset := 2

	if length == 0 {
		return "", offset, nil
	}

	if len(data[offset:]) < int(length) {
		return "", 0, ErrUnexpectedEOF
	}

	buf := data[offset : offset+int(length)]

	// Validate UTF-8 encoding
	if err := ValidateUTF8String(buf); err != nil {
		return "", 0, err
	}

	str := string(buf)
	offset += int(length)

	return str, offset, nil
}

func readUTF8Pair(r io.Reader) (UTF8Pair, error) {
	key, err := readUTF8String(r)
	if err != nil {
		return UTF8Pair{}, err
	}

	value, err := readUTF8String(r)
	if err != nil {
		return UTF8Pair{}, err
	}

	return UTF8Pair{Key: key, Value: value}, nil
}

func readUTF8PairFromBytes(data []byte) (UTF8Pair, int, error) {
	offset := 0

	key, bytesRead, err := readUTF8StringFromBytes(data[offset:])
	if err != nil {
		return UTF8Pair{}, 0, err
	}
	offset += bytesRead

	value, bytesRead, err := readUTF8StringFromBytes(data[offset:])
	if err != nil {
		return UTF8Pair{}, 0, err
	}
	offset += bytesRead

	return UTF8Pair{Key: key, Value: value}, offset, nil
}

func readBinaryData(r io.Reader) ([]byte, error) {
	length, err := readTwoByteInt(r)
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return []byte{}, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, ErrUnexpectedEOF
	}

	return buf, nil
}

func readBinaryDataFromBytes(data []byte) ([]byte, int, error) {
	if len(data) < 2 {
		return nil, 0, ErrUnexpectedEOF
	}

	length := uint16(data[0])<<8 | uint16(data[1])
	offset := 2

	if length == 0 {
		return []byte{}, offset, nil
	}

	if len(data[offset:]) < int(length) {
		return nil, 0, ErrUnexpectedEOF
	}

	buf := make([]byte, length)
	copy(buf, data[offset:offset+int(length)])
	offset += int(length)

	return buf, offset, nil
}

func writeByte(w io.Writer, value byte) error {
	_, err := w.Write([]byte{value})
	return err
}

func writeByteToBytes(buf []byte, value byte) (int, error) {
	if len(buf) < 1 {
		return 0, ErrBufferTooSmall
	}
	buf[0] = value
	return 1, nil
}

func writeTwoByteInt(w io.Writer, value uint16) error {
	_, err := w.Write([]byte{byte(value >> 8), byte(value)})
	return err
}

func writeTwoByteIntToBytes(buf []byte, value uint16) (int, error) {
	if len(buf) < 2 {
		return 0, ErrBufferTooSmall
	}
	buf[0] = byte(value >> 8)
	buf[1] = byte(value)
	return 2, nil
}

func writeFourByteInt(w io.Writer, value uint32) error {
	_, err := w.Write([]byte{
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	})
	return err
}

func writeFourByteIntToBytes(buf []byte, value uint32) (int, error) {
	if len(buf) < 4 {
		return 0, ErrBufferTooSmall
	}
	buf[0] = byte(value >> 24)
	buf[1] = byte(value >> 16)
	buf[2] = byte(value >> 8)
	buf[3] = byte(value)
	return 4, nil
}

func writeUTF8String(w io.Writer, value string) error {
	length := uint16(len(value))
	if err := writeTwoByteInt(w, length); err != nil {
		return err
	}
	if length > 0 {
		_, err := w.Write([]byte(value))
		return err
	}
	return nil
}

func writeUTF8StringToBytes(buf []byte, value string) (int, error) {
	length := uint16(len(value))
	if len(buf) < int(2+length) {
		return 0, ErrBufferTooSmall
	}

	offset := 0
	bytesWritten, err := writeTwoByteIntToBytes(buf[offset:], length)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	if length > 0 {
		copy(buf[offset:], value)
		offset += int(length)
	}

	return offset, nil
}

func writeUTF8Pair(w io.Writer, value UTF8Pair) error {
	if err := writeUTF8String(w, value.Key); err != nil {
		return err
	}
	return writeUTF8String(w, value.Value)
}

func writeUTF8PairToBytes(buf []byte, value UTF8Pair) (int, error) {
	offset := 0

	bytesWritten, err := writeUTF8StringToBytes(buf[offset:], value.Key)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	bytesWritten, err = writeUTF8StringToBytes(buf[offset:], value.Value)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	return offset, nil
}

func writeBinaryData(w io.Writer, value []byte) error {
	length := uint16(len(value))
	// Write length (2 bytes) + data
	buf := make([]byte, 2+length)
	_, err := writeBinaryDataToBytes(buf, value)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func writeBinaryDataToBytes(buf []byte, value []byte) (int, error) {
	length := uint16(len(value))
	if len(buf) < int(2+length) {
		return 0, ErrBufferTooSmall
	}

	offset := 0
	bytesWritten, err := writeTwoByteIntToBytes(buf[offset:], length)
	if err != nil {
		return 0, err
	}
	offset += bytesWritten

	if length > 0 {
		copy(buf[offset:], value)
		offset += int(length)
	}

	return offset, nil
}

// GetProperty returns the first property with the given ID, or nil if not found
func (p *Properties) GetProperty(id PropertyID) *Property {
	for i := range p.Properties {
		if p.Properties[i].ID == id {
			return &p.Properties[i]
		}
	}
	return nil
}

// GetProperties returns all properties with the given ID
func (p *Properties) GetProperties(id PropertyID) []Property {
	var result []Property
	for _, prop := range p.Properties {
		if prop.ID == id {
			result = append(result, prop)
		}
	}
	return result
}

// AddProperty adds a property to the collection
func (p *Properties) AddProperty(id PropertyID, value interface{}) error {
	spec, ok := propertySpecs[id]
	if !ok {
		return ErrInvalidPropertyID
	}

	// Check if property can appear multiple times
	if !spec.Multiple {
		// Check if property already exists
		if p.GetProperty(id) != nil {
			return ErrDuplicateProperty
		}
	}

	p.Properties = append(p.Properties, Property{
		ID:    id,
		Value: value,
	})

	return nil
}

// String returns human-readable property ID name
func (id PropertyID) String() string {
	names := map[PropertyID]string{
		PropPayloadFormatIndicator:          "PayloadFormatIndicator",
		PropMessageExpiryInterval:           "MessageExpiryInterval",
		PropContentType:                     "ContentType",
		PropResponseTopic:                   "ResponseTopic",
		PropCorrelationData:                 "CorrelationData",
		PropSubscriptionIdentifier:          "SubscriptionIdentifier",
		PropSessionExpiryInterval:           "SessionExpiryInterval",
		PropAssignedClientIdentifier:        "AssignedClientIdentifier",
		PropServerKeepAlive:                 "ServerKeepAlive",
		PropAuthenticationMethod:            "AuthenticationMethod",
		PropAuthenticationData:              "AuthenticationData",
		PropRequestProblemInformation:       "RequestProblemInformation",
		PropWillDelayInterval:               "WillDelayInterval",
		PropRequestResponseInformation:      "RequestResponseInformation",
		PropResponseInformation:             "ResponseInformation",
		PropServerReference:                 "ServerReference",
		PropReasonString:                    "ReasonString",
		PropReceiveMaximum:                  "ReceiveMaximum",
		PropTopicAliasMaximum:               "TopicAliasMaximum",
		PropTopicAlias:                      "TopicAlias",
		PropMaximumQoS:                      "MaximumQoS",
		PropRetainAvailable:                 "RetainAvailable",
		PropUserProperty:                    "UserProperty",
		PropMaximumPacketSize:               "MaximumPacketSize",
		PropWildcardSubscriptionAvailable:   "WildcardSubscriptionAvailable",
		PropSubscriptionIdentifierAvailable: "SubscriptionIdentifierAvailable",
		PropSharedSubscriptionAvailable:     "SharedSubscriptionAvailable",
	}

	if name, ok := names[id]; ok {
		return name
	}
	return "UNKNOWN"
}
