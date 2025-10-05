package network

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultListenerConfig(t *testing.T) {
	config := DefaultListenerConfig("localhost:8080")
	assert.NotNil(t, config)
	assert.Equal(t, "localhost:8080", config.Address)
	assert.Equal(t, 30*time.Second, config.TCPKeepAlive)
	assert.Equal(t, 5*time.Second, config.AcceptTimeout)
	assert.Equal(t, 10000, config.MaxConnections)
	assert.Equal(t, 4096, config.ReadBufferSize)
	assert.Equal(t, 4096, config.WriteBufferSize)
	assert.True(t, config.ReusePort)
}

func TestNewListener(t *testing.T) {
	config := &ListenerConfig{
		Address:      "localhost:0",
		TCPKeepAlive: 30 * time.Second,
	}

	listener, err := NewListener(config, nil)
	assert.NoError(t, err)
	assert.NotNil(t, listener)
}

func TestNewListenerNilConfig(t *testing.T) {
	listener, err := NewListener(nil, nil)
	assert.Error(t, err)
	assert.Nil(t, listener)
}

func TestListenerStartStop(t *testing.T) {
	config := &ListenerConfig{
		Address:      "127.0.0.1:0",
		TCPKeepAlive: 10 * time.Second,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	err = listener.Start()
	require.NoError(t, err)

	addr := listener.Addr()
	require.NotNil(t, addr)

	err = listener.Close()
	assert.NoError(t, err)
}

func TestListenerAcceptConnection(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	connected := make(chan struct{})
	listener.OnConnection(func(conn *Connection) error {
		assert.NotNil(t, conn)
		close(connected)
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			defer conn.Close()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("connection not accepted")
	}

	stats := listener.Stats()
	assert.Equal(t, uint64(1), stats.Accepted)
}

func TestListenerMultipleConnections(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	var connCount int32
	listener.OnConnection(func(conn *Connection) error {
		atomic.AddInt32(&connCount, 1)
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	for i := 0; i < 3; i++ {
		go func() {
			conn, err := net.Dial("tcp", addr)
			if err == nil {
				defer conn.Close()
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)

	stats := listener.Stats()
	assert.Equal(t, uint64(3), stats.Accepted)
}

func TestListenerMaxConnections(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 2,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	listener.OnConnection(func(conn *Connection) error {
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	// Keep connections open so they count against the limit
	var conns []net.Conn
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	// Connect more than max connections
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conns = append(conns, conn)
		}
		time.Sleep(50 * time.Millisecond) // Give time for connection to be processed
	}

	time.Sleep(100 * time.Millisecond)

	stats := listener.Stats()
	assert.True(t, stats.Rejected > 0, "Expected at least one rejected connection")
}

func TestListenerOnConnectionError(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	listener.OnConnection(func(conn *Connection) error {
		return fmt.Errorf("handler error")
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	conn, err := net.Dial("tcp", addr)
	if err == nil {
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)
}

func TestListenerCloseTwice(t *testing.T) {
	config := &ListenerConfig{
		Address:      "127.0.0.1:0",
		TCPKeepAlive: 10 * time.Second,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	err = listener.Start()
	require.NoError(t, err)

	err1 := listener.Close()
	assert.NoError(t, err1)

	err2 := listener.Close()
	assert.NoError(t, err2)
}

func TestListenerStartAfterClose(t *testing.T) {
	config := &ListenerConfig{
		Address:      "127.0.0.1:0",
		TCPKeepAlive: 10 * time.Second,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	err = listener.Start()
	require.NoError(t, err)

	listener.Close()

	err = listener.Start()
	assert.Equal(t, ErrListenerClosed, err)
}

func TestListenerAddr(t *testing.T) {
	config := &ListenerConfig{
		Address:      "127.0.0.1:0",
		TCPKeepAlive: 10 * time.Second,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	assert.Nil(t, listener.Addr())

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	assert.NotNil(t, listener.Addr())
}

func TestListenerStats(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	listener.OnConnection(func(conn *Connection) error {
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	conn, err := net.Dial("tcp", addr)
	if err == nil {
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	stats := listener.Stats()
	assert.Equal(t, uint64(1), stats.Accepted)
}

func TestListenerMultipleHandlers(t *testing.T) {
	config := &ListenerConfig{
		Address:        "127.0.0.1:0",
		TCPKeepAlive:   10 * time.Second,
		MaxConnections: 10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	var mu sync.Mutex
	handler1Called := false
	handler2Called := false

	listener.OnConnection(func(conn *Connection) error {
		mu.Lock()
		handler1Called = true
		mu.Unlock()
		return nil
	})

	listener.OnConnection(func(conn *Connection) error {
		mu.Lock()
		handler2Called = true
		mu.Unlock()
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	conn, err := net.Dial("tcp", addr)
	if err == nil {
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
	mu.Unlock()
}

func TestListenerWithCustomPool(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{
		MaxConnections:     5,
		MaxIdleConnections: 2,
		CleanupInterval:    0,
	})
	defer pool.Close()

	config := &ListenerConfig{
		Address:      "127.0.0.1:0",
		TCPKeepAlive: 10 * time.Second,
	}

	listener, err := NewListener(config, pool)
	require.NoError(t, err)

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()
}

func TestListenerBufferSizes(t *testing.T) {
	config := &ListenerConfig{
		Address:         "127.0.0.1:0",
		TCPKeepAlive:    10 * time.Second,
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
		MaxConnections:  10,
	}

	listener, err := NewListener(config, nil)
	require.NoError(t, err)

	listener.OnConnection(func(conn *Connection) error {
		return nil
	})

	err = listener.Start()
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	conn, err := net.Dial("tcp", addr)
	if err == nil {
		defer conn.Close()
		time.Sleep(50 * time.Millisecond)
	}
}
