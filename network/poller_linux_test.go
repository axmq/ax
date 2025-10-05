//go:build linux

package network

import (
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoller(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		require.NotNil(t, poller)
		defer poller.Close()

		epoll, ok := poller.(*EpollPoller)
		assert.True(t, ok)
		assert.Greater(t, epoll.fd, 0)
		assert.NotNil(t, epoll.state)
		assert.NotNil(t, epoll.config)
		assert.Equal(t, 1024, epoll.config.MaxEvents)
		assert.Equal(t, 100*time.Millisecond, epoll.config.Timeout)
		assert.Len(t, epoll.events, 1024)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &PollerConfig{
			MaxEvents: 512,
			Timeout:   50 * time.Millisecond,
		}
		poller, err := NewPoller(config)
		require.NoError(t, err)
		require.NotNil(t, poller)
		defer poller.Close()

		epoll, ok := poller.(*EpollPoller)
		assert.True(t, ok)
		assert.Equal(t, 512, epoll.config.MaxEvents)
		assert.Equal(t, 50*time.Millisecond, epoll.config.Timeout)
		assert.Len(t, epoll.events, 512)
	})
}

func TestEpollPollerAdd(t *testing.T) {
	t.Run("add tcp connection", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

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

		conn := NewConnection(client, "test-add", nil)
		err = poller.Add(conn, EventRead)
		assert.NoError(t, err)

		<-done
	})

	t.Run("add with read and write events", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

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

		conn := NewConnection(client, "test-add-rw", nil)
		err = poller.Add(conn, EventRead|EventWrite)
		assert.NoError(t, err)

		<-done
	})

	t.Run("add pipe connection fails", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		conn := NewConnection(server, "test-pipe", nil)
		err = poller.Add(conn, EventRead)
		assert.Error(t, err)
	})

	t.Run("add to closed poller", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		poller.Close()

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
		defer client.Close()

		conn := NewConnection(client, "test-closed", nil)
		err = poller.Add(conn, EventRead)
		assert.Equal(t, ErrListenerClosed, err)

		<-done
	})
}

func TestEpollPollerModify(t *testing.T) {
	t.Run("modify tcp connection", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

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

		conn := NewConnection(client, "test-modify", nil)
		err = poller.Add(conn, EventRead)
		require.NoError(t, err)

		err = poller.Modify(conn, EventWrite)
		assert.NoError(t, err)

		err = poller.Modify(conn, EventRead|EventWrite)
		assert.NoError(t, err)

		<-done
	})

	t.Run("modify pipe connection fails", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		conn := NewConnection(server, "test-pipe-mod", nil)
		err = poller.Modify(conn, EventRead)
		assert.Error(t, err)
	})

	t.Run("modify closed poller", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		poller.Close()

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
		defer client.Close()

		conn := NewConnection(client, "test-closed-mod", nil)
		err = poller.Modify(conn, EventRead)
		assert.Equal(t, ErrListenerClosed, err)

		<-done
	})
}

func TestEpollPollerRemove(t *testing.T) {
	t.Run("remove tcp connection", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

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

		conn := NewConnection(client, "test-remove", nil)
		err = poller.Add(conn, EventRead)
		require.NoError(t, err)

		err = poller.Remove(conn)
		assert.NoError(t, err)

		<-done
	})

	t.Run("remove non-existent connection", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

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

		conn := NewConnection(client, "test-not-added", nil)
		err = poller.Remove(conn)
		assert.NoError(t, err)

		<-done
	})

	t.Run("remove pipe connection fails", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		conn := NewConnection(server, "test-pipe-remove", nil)
		err = poller.Remove(conn)
		assert.Error(t, err)
	})
}

func TestEpollPollerWait(t *testing.T) {
	t.Run("wait with timeout no events", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		start := time.Now()
		events, err := poller.Wait(50 * time.Millisecond)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, events, 0)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
	})

	t.Run("wait with data available", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		serverConnCh := make(chan net.Conn)
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				serverConnCh <- conn
			} else {
				close(serverConnCh)
			}
		}()

		client, err := net.Dial("tcp", listener.Addr().String())
		require.NoError(t, err)
		defer client.Close()

		serverConn := <-serverConnCh
		require.NotNil(t, serverConn)
		defer serverConn.Close()

		conn := NewConnection(serverConn, "test-wait-data", nil)
		err = poller.Add(conn, EventRead)
		require.NoError(t, err)

		_, err = client.Write([]byte("hello"))
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		events, err := poller.Wait(100 * time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 1)
		if len(events) > 0 {
			assert.Equal(t, conn, events[0].Conn)
		}
	})

	t.Run("wait with closed connection", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		defer poller.Close()

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		serverConnCh := make(chan net.Conn)
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				serverConnCh <- conn
			} else {
				close(serverConnCh)
			}
		}()

		client, err := net.Dial("tcp", listener.Addr().String())
		require.NoError(t, err)

		serverConn := <-serverConnCh
		require.NotNil(t, serverConn)
		defer serverConn.Close()

		conn := NewConnection(serverConn, "test-wait-closed", nil)
		err = poller.Add(conn, EventRead)
		require.NoError(t, err)

		client.Close()
		time.Sleep(50 * time.Millisecond)

		events, err := poller.Wait(100 * time.Millisecond)
		require.NoError(t, err)
		if len(events) > 0 {
			assert.Equal(t, conn, events[0].Conn)
			if events[0].Error != nil {
				assert.Equal(t, ErrConnectionClosed, events[0].Error)
			}
		}
	})

	t.Run("wait on closed poller", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)
		poller.Close()

		events, err := poller.Wait(50 * time.Millisecond)
		assert.Equal(t, ErrListenerClosed, err)
		assert.Nil(t, events)
	})
}

func TestEpollPollerClose(t *testing.T) {
	t.Run("close poller", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)

		err = poller.Close()
		assert.NoError(t, err)

		epoll := poller.(*EpollPoller)
		assert.True(t, epoll.state.isClosed())
	})

	t.Run("close poller twice", func(t *testing.T) {
		poller, err := NewPoller(nil)
		require.NoError(t, err)

		err = poller.Close()
		assert.NoError(t, err)

		err = poller.Close()
		assert.NoError(t, err)
	})
}

func TestEpollPollerEventsToEpoll(t *testing.T) {
	poller, err := NewPoller(nil)
	require.NoError(t, err)
	defer poller.Close()

	epoll := poller.(*EpollPoller)

	t.Run("read event", func(t *testing.T) {
		events := epoll.eventsToEpoll(EventRead)
		assert.Equal(t, uint32(syscall.EPOLLIN)|(1<<31), events)
	})

	t.Run("write event", func(t *testing.T) {
		events := epoll.eventsToEpoll(EventWrite)
		assert.Equal(t, uint32(syscall.EPOLLOUT)|(1<<31), events)
	})

	t.Run("read and write events", func(t *testing.T) {
		events := epoll.eventsToEpoll(EventRead | EventWrite)
		assert.Equal(t, uint32(syscall.EPOLLIN|syscall.EPOLLOUT)|(1<<31), events)
	})

	t.Run("error event", func(t *testing.T) {
		events := epoll.eventsToEpoll(EventError)
		assert.Equal(t, uint32(1<<31), events)
	})

	t.Run("hup event", func(t *testing.T) {
		events := epoll.eventsToEpoll(EventHup)
		assert.Equal(t, uint32(1<<31), events)
	})
}

func TestEpollPollerMultipleConnections(t *testing.T) {
	poller, err := NewPoller(nil)
	require.NoError(t, err)
	defer poller.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	numConns := 5
	serverConns := make([]net.Conn, numConns)
	clientConns := make([]net.Conn, numConns)
	connections := make([]*Connection, numConns)

	acceptCh := make(chan net.Conn, numConns)
	go func() {
		for i := 0; i < numConns; i++ {
			conn, err := listener.Accept()
			if err == nil {
				acceptCh <- conn
			}
		}
		close(acceptCh)
	}()

	for i := 0; i < numConns; i++ {
		client, err := net.Dial("tcp", listener.Addr().String())
		require.NoError(t, err)
		clientConns[i] = client
	}

	for i := 0; i < numConns; i++ {
		serverConns[i] = <-acceptCh
		require.NotNil(t, serverConns[i])
	}

	for i := 0; i < numConns; i++ {
		connections[i] = NewConnection(serverConns[i], "conn-"+string(rune(i)), nil)
		err = poller.Add(connections[i], EventRead)
		require.NoError(t, err)
	}

	for i := 0; i < numConns; i++ {
		_, err = clientConns[i].Write([]byte("test"))
		require.NoError(t, err)
	}

	time.Sleep(50 * time.Millisecond)

	events, err := poller.Wait(100 * time.Millisecond)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 1)

	for i := 0; i < numConns; i++ {
		err = poller.Remove(connections[i])
		assert.NoError(t, err)
		clientConns[i].Close()
		serverConns[i].Close()
	}
}

func TestEpollPollerAddRemoveAdd(t *testing.T) {
	poller, err := NewPoller(nil)
	require.NoError(t, err)
	defer poller.Close()

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

	conn := NewConnection(client, "test-add-remove-add", nil)

	err = poller.Add(conn, EventRead)
	require.NoError(t, err)

	err = poller.Remove(conn)
	require.NoError(t, err)

	err = poller.Add(conn, EventRead)
	assert.NoError(t, err)

	<-done
}

func TestEpollPollerWaitMultipleTimes(t *testing.T) {
	poller, err := NewPoller(nil)
	require.NoError(t, err)
	defer poller.Close()

	for i := 0; i < 5; i++ {
		events, err := poller.Wait(10 * time.Millisecond)
		assert.NoError(t, err)
		assert.Len(t, events, 0)
	}
}

func TestEpollPollerEdgeTrigger(t *testing.T) {
	poller, err := NewPoller(nil)
	require.NoError(t, err)
	defer poller.Close()

	epoll := poller.(*EpollPoller)
	events := epoll.eventsToEpoll(EventRead)
	assert.NotEqual(t, uint32(0), events&(1<<31))
}
