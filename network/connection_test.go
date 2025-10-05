package network

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestConnection(t *testing.T) (*Connection, net.Conn, net.Conn) {
	server, client := net.Pipe()
	conn := NewConnection(server, "test-conn-1", nil)
	require.NotNil(t, conn)
	return conn, server, client
}

func TestNewConnection(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		config *ConnectionConfig
	}{
		{
			name:   "with default config",
			id:     "conn-1",
			config: nil,
		},
		{
			name: "with custom config",
			id:   "conn-2",
			config: &ConnectionConfig{
				KeepAlive:     60 * time.Second,
				ReadDeadline:  30 * time.Second,
				WriteDeadline: 30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			conn := NewConnection(server, tt.id, tt.config)
			require.NotNil(t, conn)
			assert.Equal(t, tt.id, conn.ID())
			assert.Equal(t, StateConnected, conn.State())
			assert.False(t, conn.IsTLS())
			assert.NotNil(t, conn.RemoteAddr())
			assert.NotNil(t, conn.LocalAddr())
		})
	}
}

func TestConnectionReadWrite(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "small data",
			data: []byte("hello"),
		},
		{
			name: "large data",
			data: make([]byte, 4096),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, _, client := createTestConnection(t)
			defer conn.Close()
			defer client.Close()

			writeDone := make(chan struct{})
			go func() {
				defer close(writeDone)
				n, err := conn.Write(tt.data)
				require.NoError(t, err)
				assert.Equal(t, len(tt.data), n)
			}()

			buf := make([]byte, len(tt.data)+10)
			n, err := client.Read(buf)
			require.NoError(t, err)
			assert.Equal(t, len(tt.data), n)

			<-writeDone

			assert.Equal(t, uint64(len(tt.data)), conn.BytesWritten())
		})
	}
}

func TestConnectionMetadata(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer conn.Close()
	defer client.Close()

	conn.SetMetadata("key1", "value1")
	val, ok := conn.GetMetadata("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)

	conn.DeleteMetadata("key1")
	_, ok = conn.GetMetadata("key1")
	assert.False(t, ok)
}

func TestConnectionState(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer client.Close()

	assert.Equal(t, StateConnected, conn.State())

	err := conn.Close()
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, conn.State())
}

func TestConnectionActivity(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer conn.Close()
	defer client.Close()

	lastActivity := conn.LastActivity()
	assert.False(t, lastActivity.IsZero())

	time.Sleep(10 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 10)
		_, err := client.Read(buf)
		done <- err
	}()

	_, err := conn.Write([]byte("test"))
	require.NoError(t, err)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out waiting for read")
	}

	newActivity := conn.LastActivity()
	assert.True(t, newActivity.After(lastActivity))
	assert.True(t, conn.IdleDuration() >= 0)
}

func TestConnectionReadAfterClose(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer client.Close()

	err := conn.Close()
	require.NoError(t, err)

	buf := make([]byte, 10)
	_, err = conn.Read(buf)
	assert.Equal(t, ErrConnectionClosed, err)
}

func TestConnectionWriteAfterClose(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer client.Close()

	err := conn.Close()
	require.NoError(t, err)

	_, err = conn.Write([]byte("test"))
	assert.Equal(t, ErrConnectionClosed, err)
}

func TestConnectionBytesRead(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer conn.Close()
	defer client.Close()

	data := []byte("test data")
	go func() {
		client.Write(data)
	}()

	buf := make([]byte, len(data)+10)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, uint64(len(data)), conn.BytesRead())
}

func TestConnectionCloseChan(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer client.Close()

	closeCh := conn.CloseChan()
	assert.NotNil(t, closeCh)

	select {
	case <-closeCh:
		t.Fatal("close channel should not be closed yet")
	default:
	}

	err := conn.Close()
	require.NoError(t, err)

	select {
	case <-closeCh:
	case <-time.After(1 * time.Second):
		t.Fatal("close channel should be closed")
	}
}

func TestConnectionCloseMultipleTimes(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer client.Close()

	err1 := conn.Close()
	assert.NoError(t, err1)

	err2 := conn.Close()
	assert.NoError(t, err2)
}

func TestConnectionTLSConnectionState(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	defer conn.Close()

	_, ok := conn.TLSConnectionState()
	assert.False(t, ok)
}

func TestConnectionSetKeepAlive(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err == nil {
			conn := NewConnection(c, "test", nil)
			defer conn.Close()
			err := conn.SetKeepAlive(30 * time.Second)
			assert.NoError(t, err)
		}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer client.Close()

	<-done
}

func TestConnectionMetadataMultipleKeys(t *testing.T) {
	conn, _, client := createTestConnection(t)
	defer conn.Close()
	defer client.Close()

	conn.SetMetadata("key1", "value1")
	conn.SetMetadata("key2", 123)
	conn.SetMetadata("key3", true)

	val1, ok := conn.GetMetadata("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val1)

	val2, ok := conn.GetMetadata("key2")
	assert.True(t, ok)
	assert.Equal(t, 123, val2)

	val3, ok := conn.GetMetadata("key3")
	assert.True(t, ok)
	assert.Equal(t, true, val3)

	conn.DeleteMetadata("key2")
	_, ok = conn.GetMetadata("key2")
	assert.False(t, ok)

	_, ok = conn.GetMetadata("nonexistent")
	assert.False(t, ok)
}

func TestConnectionWithConfiguredDeadlines(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	cfg := &ConnectionConfig{
		ReadDeadline:  100 * time.Millisecond,
		WriteDeadline: 100 * time.Millisecond,
		KeepAlive:     30 * time.Second,
	}

	conn := NewConnection(server, "test-conn", cfg)
	defer conn.Close()

	assert.Equal(t, 100*time.Millisecond, conn.readDeadline)
	assert.Equal(t, 100*time.Millisecond, conn.writeDeadline)
	assert.Equal(t, 30*time.Second, conn.keepAlive)
}
