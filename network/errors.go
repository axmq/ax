package network

import "errors"

var (
	ErrConnectionClosed        = errors.New("connection closed")
	ErrConnectionPoolExhausted = errors.New("connection pool exhausted")
	ErrInvalidTLSConfig        = errors.New("invalid TLS configuration")
	ErrKeepAliveTimeout        = errors.New("keep-alive timeout")
	ErrInvalidAddress          = errors.New("invalid address")
	ErrListenerClosed          = errors.New("listener closed")
	ErrMaxRetriesExceeded      = errors.New("max retries exceeded")
	ErrInvalidBackoffConfig    = errors.New("invalid backoff configuration")
	ErrConnectionNotFound      = errors.New("connection not found")
	ErrInvalidPoolConfig       = errors.New("invalid pool configuration")
	ErrPoolClosed              = errors.New("pool closed")
	ErrCertificateVerification = errors.New("certificate verification failed")
	ErrGracefulShutdownTimeout = errors.New("graceful shutdown timeout")
)
