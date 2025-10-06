package store

import "errors"

var (
	ErrNotFound      = errors.New("key not found")
	ErrAlreadyExists = errors.New("key already exists")
	ErrStoreClosed   = errors.New("store is closed")
)
