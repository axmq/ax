package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ListenerConfig struct {
	Address         string
	TLSConfig       *tls.Config
	TCPKeepAlive    time.Duration
	AcceptTimeout   time.Duration
	MaxConnections  int
	ReadBufferSize  int
	WriteBufferSize int
	ReusePort       bool
}

func DefaultListenerConfig(address string) *ListenerConfig {
	return &ListenerConfig{
		Address:         address,
		TCPKeepAlive:    30 * time.Second,
		AcceptTimeout:   5 * time.Second,
		MaxConnections:  10000,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		ReusePort:       true,
	}
}

type Listener struct {
	config   *ListenerConfig
	listener net.Listener
	pool     *Pool

	connSeq  atomic.Uint64
	accepted atomic.Uint64
	rejected atomic.Uint64

	mu       sync.RWMutex
	handlers []ConnectionHandler

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	closed    atomic.Bool
	closeOnce sync.Once
}

type ConnectionHandler func(*Connection) error

func NewListener(config *ListenerConfig, pool *Pool) (*Listener, error) {
	if config == nil {
		return nil, ErrInvalidAddress
	}

	if pool == nil {
		var err error
		pool, err = NewPool(DefaultPoolConfig())
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Listener{
		config:   config,
		pool:     pool,
		handlers: make([]ConnectionHandler, 0),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (l *Listener) Start() error {
	if l.closed.Load() {
		return ErrListenerClosed
	}

	var err error
	if l.config.TLSConfig != nil {
		l.listener, err = tls.Listen("tcp", l.config.Address, l.config.TLSConfig)
	} else {
		l.listener, err = net.Listen("tcp", l.config.Address)
	}

	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	l.wg.Add(1)
	go l.acceptLoop()

	return nil
}

func (l *Listener) acceptLoop() {
	defer l.wg.Done()

	for {
		select {
		case <-l.ctx.Done():
			return
		default:
		}

		if l.config.AcceptTimeout > 0 {
			if tcpListener, ok := l.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(l.config.AcceptTimeout))
			}
		}

		netConn, err := l.listener.Accept()
		if err != nil {
			if l.closed.Load() {
				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}

			continue
		}

		if l.config.MaxConnections > 0 && int(l.pool.total.Load()) >= l.config.MaxConnections {
			_ = netConn.Close()
			l.rejected.Add(1)
			continue
		}

		l.wg.Add(1)
		go l.handleConnection(netConn)
	}
}

func (l *Listener) handleConnection(netConn net.Conn) {
	defer l.wg.Done()

	if tcpConn, ok := netConn.(*net.TCPConn); ok {
		if l.config.TCPKeepAlive > 0 {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(l.config.TCPKeepAlive)
		}

		if l.config.ReadBufferSize > 0 {
			tcpConn.SetReadBuffer(l.config.ReadBufferSize)
		}

		if l.config.WriteBufferSize > 0 {
			tcpConn.SetWriteBuffer(l.config.WriteBufferSize)
		}
	}

	connID := l.generateConnectionID()
	conn := NewConnection(netConn, connID, &ConnectionConfig{
		KeepAlive:     l.config.TCPKeepAlive,
		ReadDeadline:  0,
		WriteDeadline: 0,
		TLSConfig:     l.config.TLSConfig,
	})

	if err := l.pool.Add(conn); err != nil {
		conn.Close()
		l.rejected.Add(1)
		return
	}

	l.accepted.Add(1)

	l.mu.RLock()
	handlers := make([]ConnectionHandler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(conn); err != nil {
			l.pool.Remove(conn.ID())
			return
		}
	}
}

func (l *Listener) generateConnectionID() string {
	seq := l.connSeq.Add(1)
	return fmt.Sprintf("conn-%d-%d", time.Now().UnixNano(), seq)
}

func (l *Listener) OnConnection(handler ConnectionHandler) {
	l.mu.Lock()
	l.handlers = append(l.handlers, handler)
	l.mu.Unlock()
}

func (l *Listener) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return nil
	}

	var err error
	l.closeOnce.Do(func() {
		l.cancel()

		if l.listener != nil {
			err = l.listener.Close()
		}

		l.wg.Wait()
	})

	return err
}

func (l *Listener) Addr() net.Addr {
	if l.listener != nil {
		return l.listener.Addr()
	}
	return nil
}

func (l *Listener) Stats() ListenerStats {
	return ListenerStats{
		Accepted: l.accepted.Load(),
		Rejected: l.rejected.Load(),
		Active:   uint64(l.pool.active.Load()),
	}
}

type ListenerStats struct {
	Accepted uint64
	Rejected uint64
	Active   uint64
}
