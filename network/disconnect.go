package network

import (
	"context"
	"sync"
	"time"
)

type DisconnectReason byte

const (
	DisconnectNormalDisconnection  DisconnectReason = 0x00
	DisconnectWithWillMessage      DisconnectReason = 0x04
	DisconnectUnspecifiedError     DisconnectReason = 0x80
	DisconnectMalformedPacket      DisconnectReason = 0x81
	DisconnectProtocolError        DisconnectReason = 0x82
	DisconnectImplementationError  DisconnectReason = 0x83
	DisconnectNotAuthorized        DisconnectReason = 0x87
	DisconnectServerBusy           DisconnectReason = 0x89
	DisconnectServerShuttingDown   DisconnectReason = 0x8B
	DisconnectKeepAliveTimeout     DisconnectReason = 0x8D
	DisconnectSessionTakenOver     DisconnectReason = 0x8E
	DisconnectTopicFilterInvalid   DisconnectReason = 0x8F
	DisconnectPacketTooLarge       DisconnectReason = 0x95
	DisconnectQuotaExceeded        DisconnectReason = 0x97
	DisconnectAdministrativeAction DisconnectReason = 0x98
	DisconnectPayloadFormatInvalid DisconnectReason = 0x99
)

type DisconnectPacket struct {
	ReasonCode      DisconnectReason
	SessionExpiry   *uint32
	ReasonString    string
	ServerReference string
}

type DisconnectHandler func(*Connection, *DisconnectPacket) error

type DisconnectManager struct {
	mu              sync.RWMutex
	handlers        []DisconnectHandler
	gracefulTimeout time.Duration
}

func NewDisconnectManager(gracefulTimeout time.Duration) *DisconnectManager {
	if gracefulTimeout == 0 {
		gracefulTimeout = 5 * time.Second
	}

	return &DisconnectManager{
		handlers:        make([]DisconnectHandler, 0),
		gracefulTimeout: gracefulTimeout,
	}
}

func (dm *DisconnectManager) OnDisconnect(handler DisconnectHandler) {
	dm.mu.Lock()
	dm.handlers = append(dm.handlers, handler)
	dm.mu.Unlock()
}

func (dm *DisconnectManager) HandleDisconnect(conn *Connection, packet *DisconnectPacket) error {
	dm.mu.RLock()
	handlers := make([]DisconnectHandler, len(dm.handlers))
	copy(handlers, dm.handlers)
	dm.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(conn, packet); err != nil {
			return err
		}
	}

	return nil
}

func (dm *DisconnectManager) GracefulDisconnect(ctx context.Context, conn *Connection, reason DisconnectReason) error {
	packet := &DisconnectPacket{
		ReasonCode: reason,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dm.gracefulTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		if err := dm.HandleDisconnect(conn, packet); err != nil {
			done <- err
			return
		}
		done <- conn.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		_ = conn.Close()
		return ErrGracefulShutdownTimeout
	}
}

func (dm *DisconnectManager) SendDisconnect(conn *Connection, packet *DisconnectPacket) error {
	if packet == nil {
		packet = &DisconnectPacket{
			ReasonCode: DisconnectNormalDisconnection,
		}
	}

	return dm.HandleDisconnect(conn, packet)
}

type GracefulShutdown struct {
	pool    *Pool
	dm      *DisconnectManager
	timeout time.Duration

	mu       sync.Mutex
	shutdown bool
}

func NewGracefulShutdown(pool *Pool, dm *DisconnectManager, timeout time.Duration) *GracefulShutdown {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &GracefulShutdown{
		pool:    pool,
		dm:      dm,
		timeout: timeout,
	}
}

func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.mu.Lock()
	if gs.shutdown {
		gs.mu.Unlock()
		return nil
	}
	gs.shutdown = true
	gs.mu.Unlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, gs.timeout)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	gs.pool.ForEach(func(conn *Connection) bool {
		wg.Add(1)
		go func(c *Connection) {
			defer wg.Done()

			packet := &DisconnectPacket{
				ReasonCode: DisconnectServerShuttingDown,
			}

			if err := gs.dm.GracefulDisconnect(timeoutCtx, c, packet.ReasonCode); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(conn)

		return true
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case err := <-errCh:
		return err
	case <-timeoutCtx.Done():
		return ErrGracefulShutdownTimeout
	}
}

func (gs *GracefulShutdown) IsShutdown() bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.shutdown
}
