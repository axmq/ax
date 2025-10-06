package qos

import "errors"

var (
	ErrInvalidQoS       = errors.New("invalid QoS level")
	ErrPacketIDNotFound = errors.New("packet ID not found")
	ErrMessageExpired   = errors.New("message has expired")
	ErrQueueFull        = errors.New("message queue is full")
	ErrHandlerClosed    = errors.New("handler is closed")
)
