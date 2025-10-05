package session

import "errors"

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrStoreClosed          = errors.New("store is closed")
)
