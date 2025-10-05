package network

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ConnectionState int32

const (
	StateConnecting ConnectionState = iota
	StateConnected
	StateClosing
	StateClosed
)

type Connection struct {
	conn          net.Conn
	id            string
	state         atomic.Int32
	lastActivity  atomic.Int64
	keepAlive     time.Duration
	readDeadline  time.Duration
	writeDeadline time.Duration

	tlsConn *tls.Conn
	isTLS   bool

	mu       sync.RWMutex
	metadata map[string]interface{}

	closeOnce sync.Once
	closeCh   chan struct{}

	bytesRead    atomic.Uint64
	bytesWritten atomic.Uint64
}

type ConnectionConfig struct {
	KeepAlive     time.Duration
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
	TLSConfig     *tls.Config
}

func NewConnection(conn net.Conn, id string, cfg *ConnectionConfig) *Connection {
	if cfg == nil {
		cfg = &ConnectionConfig{
			KeepAlive:     30 * time.Second,
			ReadDeadline:  60 * time.Second,
			WriteDeadline: 30 * time.Second,
		}
	}

	c := &Connection{
		conn:          conn,
		id:            id,
		keepAlive:     cfg.KeepAlive,
		readDeadline:  cfg.ReadDeadline,
		writeDeadline: cfg.WriteDeadline,
		metadata:      make(map[string]interface{}),
		closeCh:       make(chan struct{}),
	}

	c.state.Store(int32(StateConnected))
	c.updateActivity()

	if tlsConn, ok := conn.(*tls.Conn); ok {
		c.tlsConn = tlsConn
		c.isTLS = true
	}

	if cfg.KeepAlive > 0 {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(cfg.KeepAlive)
		}
	}

	return c
}

func (c *Connection) ID() string {
	return c.id
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Connection) State() ConnectionState {
	return ConnectionState(c.state.Load())
}

func (c *Connection) IsTLS() bool {
	return c.isTLS
}

func (c *Connection) Read(b []byte) (int, error) {
	if c.State() != StateConnected {
		return 0, ErrConnectionClosed
	}

	if c.readDeadline > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readDeadline))
	}

	n, err := c.conn.Read(b)
	if n > 0 {
		c.bytesRead.Add(uint64(n))
		c.updateActivity()
	}

	return n, err
}

func (c *Connection) Write(b []byte) (int, error) {
	if c.State() != StateConnected {
		return 0, ErrConnectionClosed
	}

	if c.writeDeadline > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
	}

	n, err := c.conn.Write(b)
	if n > 0 {
		c.bytesWritten.Add(uint64(n))
		c.updateActivity()
	}

	return n, err
}

func (c *Connection) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.state.Store(int32(StateClosing))
		close(c.closeCh)
		err = c.conn.Close()
		c.state.Store(int32(StateClosed))
	})
	return err
}

func (c *Connection) CloseChan() <-chan struct{} {
	return c.closeCh
}

func (c *Connection) updateActivity() {
	c.lastActivity.Store(time.Now().UnixNano())
}

func (c *Connection) LastActivity() time.Time {
	return time.Unix(0, c.lastActivity.Load())
}

func (c *Connection) IdleDuration() time.Duration {
	return time.Since(c.LastActivity())
}

func (c *Connection) BytesRead() uint64 {
	return c.bytesRead.Load()
}

func (c *Connection) BytesWritten() uint64 {
	return c.bytesWritten.Load()
}

func (c *Connection) SetMetadata(key string, value interface{}) {
	c.mu.Lock()
	c.metadata[key] = value
	c.mu.Unlock()
}

func (c *Connection) GetMetadata(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.metadata[key]
	return val, ok
}

func (c *Connection) DeleteMetadata(key string) {
	c.mu.Lock()
	delete(c.metadata, key)
	c.mu.Unlock()
}

func (c *Connection) TLSConnectionState() (tls.ConnectionState, bool) {
	if c.tlsConn != nil {
		return c.tlsConn.ConnectionState(), true
	}
	return tls.ConnectionState{}, false
}

func (c *Connection) SetKeepAlive(d time.Duration) error {
	c.keepAlive = d
	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(d > 0); err != nil {
			return err
		}
		if d > 0 {
			return tcpConn.SetKeepAlivePeriod(d)
		}
	}
	return nil
}

var _ io.ReadWriter = (*Connection)(nil)
