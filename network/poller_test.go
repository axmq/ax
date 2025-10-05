package network

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPollerConfig(t *testing.T) {
	config := DefaultPollerConfig()
	require.NotNil(t, config)
	assert.Equal(t, 1024, config.MaxEvents)
	assert.Equal(t, 100*time.Millisecond, config.Timeout)
}

func TestNewPollerState(t *testing.T) {
	ps := newPollerState()
	require.NotNil(t, ps)
	assert.NotNil(t, ps.connMap)
	assert.False(t, ps.isClosed())
}

func TestPollerStateAdd(t *testing.T) {
	ps := newPollerState()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-1", nil)
	ps.add(10, conn)

	retrieved, ok := ps.get(10)
	assert.True(t, ok)
	assert.Equal(t, conn, retrieved)
}

func TestPollerStateGet(t *testing.T) {
	ps := newPollerState()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-1", nil)
	ps.add(10, conn)

	retrieved, ok := ps.get(10)
	assert.True(t, ok)
	assert.Equal(t, conn, retrieved)

	_, ok = ps.get(999)
	assert.False(t, ok)
}

func TestPollerStateRemove(t *testing.T) {
	ps := newPollerState()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-1", nil)
	ps.add(10, conn)

	_, ok := ps.get(10)
	assert.True(t, ok)

	ps.remove(10)

	_, ok = ps.get(999)
	assert.False(t, ok)
}

func TestPollerStateClose(t *testing.T) {
	ps := newPollerState()
	assert.False(t, ps.isClosed())

	ps.close()
	assert.True(t, ps.isClosed())
}

func TestPollerStateConcurrentAccess(t *testing.T) {
	ps := newPollerState()
	server1, client1 := net.Pipe()
	defer server1.Close()
	defer client1.Close()
	server2, client2 := net.Pipe()
	defer server2.Close()
	defer client2.Close()

	conn1 := NewConnection(server1, "test-1", nil)
	conn2 := NewConnection(server2, "test-2", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			ps.add(i, conn1)
			ps.get(i)
			ps.remove(i)
		}
	}()

	for i := 100; i < 200; i++ {
		ps.add(i, conn2)
		ps.get(i)
		ps.remove(i)
	}

	<-done
}

func TestGetConnFd(t *testing.T) {
	t.Run("tcp connection", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			conn, err := listener.Accept()
			if err == nil {
				defer conn.Close()
				time.Sleep(100 * time.Millisecond)
			}
		}()

		client, err := net.Dial("tcp", listener.Addr().String())
		require.NoError(t, err)
		defer client.Close()

		conn := NewConnection(client, "test-tcp", nil)
		fd, err := getConnFd(conn)
		require.NoError(t, err)
		assert.Greater(t, fd, 0)

		<-done
	})

	t.Run("pipe connection", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		conn := NewConnection(server, "test-pipe", nil)
		fd, err := getConnFd(conn)
		assert.Error(t, err)
		assert.Equal(t, -1, fd)
	})
}

func TestEventType(t *testing.T) {
	assert.NotEqual(t, EventRead, EventWrite)
	assert.NotEqual(t, EventRead, EventError)
	assert.NotEqual(t, EventRead, EventHup)
	assert.NotEqual(t, EventWrite, EventError)
	assert.NotEqual(t, EventWrite, EventHup)
	assert.NotEqual(t, EventError, EventHup)

	combined := EventRead | EventWrite
	assert.NotEqual(t, EventRead, combined)
	assert.NotEqual(t, EventWrite, combined)
	assert.Equal(t, EventRead, combined&EventRead)
	assert.Equal(t, EventWrite, combined&EventWrite)
}

func TestEvent(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-event", nil)
	event := &Event{
		Fd:    10,
		Conn:  conn,
		Error: nil,
	}

	assert.Equal(t, 10, event.Fd)
	assert.Equal(t, conn, event.Conn)
	assert.Nil(t, event.Error)

	event.Error = ErrConnectionClosed
	assert.Equal(t, ErrConnectionClosed, event.Error)
}

func TestPollerConfig(t *testing.T) {
	config := &PollerConfig{
		MaxEvents: 2048,
		Timeout:   200 * time.Millisecond,
	}

	assert.Equal(t, 2048, config.MaxEvents)
	assert.Equal(t, 200*time.Millisecond, config.Timeout)
}

func TestGetConnFdWithClosedConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	conn := NewConnection(client, "test-closed", nil)
	fd, err := getConnFd(conn)
	require.NoError(t, err)
	assert.Greater(t, fd, 0)

	client.Close()

	_, err = getConnFd(conn)
	assert.Error(t, err)

	<-done
}

func TestPollerStateMultipleConnections(t *testing.T) {
	ps := newPollerState()
	connections := make([]*Connection, 10)
	servers := make([]net.Conn, 10)
	clients := make([]net.Conn, 10)

	for i := 0; i < 10; i++ {
		server, client := net.Pipe()
		servers[i] = server
		clients[i] = client
		connections[i] = NewConnection(server, "test-"+string(rune(i)), nil)
		ps.add(i, connections[i])
	}

	for i := 0; i < 10; i++ {
		conn, ok := ps.get(i)
		assert.True(t, ok)
		assert.Equal(t, connections[i], conn)
	}

	for i := 0; i < 10; i++ {
		ps.remove(i)
		_, ok := ps.get(i)
		assert.False(t, ok)
	}

	for i := 0; i < 10; i++ {
		servers[i].Close()
		clients[i].Close()
	}
}

func TestPollerStateIsClosed(t *testing.T) {
	ps := newPollerState()

	assert.False(t, ps.isClosed())
	assert.False(t, ps.isClosed())

	ps.close()

	assert.True(t, ps.isClosed())
	assert.True(t, ps.isClosed())

	ps.close()
	assert.True(t, ps.isClosed())
}
