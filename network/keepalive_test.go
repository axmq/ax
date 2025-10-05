package network

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultKeepAliveConfig(t *testing.T) {
	config := DefaultKeepAliveConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.Interval)
	assert.Equal(t, 10*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestNewKeepAlive(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	ka := NewKeepAlive(conn, nil)
	assert.NotNil(t, ka)
	defer ka.Stop()
}

func TestKeepAliveWithCustomConfig(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   50 * time.Millisecond,
		Timeout:    10 * time.Millisecond,
		MaxRetries: 5,
	}
	ka := NewKeepAlive(conn, config)
	assert.NotNil(t, ka)
	assert.Equal(t, config.Interval, ka.config.Interval)
	assert.Equal(t, config.Timeout, ka.config.Timeout)
	assert.Equal(t, config.MaxRetries, ka.config.MaxRetries)
	defer ka.Stop()
}

func TestKeepAliveStartStop(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   50 * time.Millisecond,
		Timeout:    10 * time.Millisecond,
		MaxRetries: 3,
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	ka.Start()
	time.Sleep(20 * time.Millisecond)
	ka.Stop()
}

func TestKeepAliveOnPong(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	ka := NewKeepAlive(conn, nil)
	require.NotNil(t, ka)

	lastPong1 := ka.LastPong()
	time.Sleep(10 * time.Millisecond)

	ka.OnPong()
	lastPong2 := ka.LastPong()

	assert.True(t, lastPong2.After(lastPong1))
	assert.Equal(t, 0, ka.MissedPings())
}

func TestKeepAliveMissedPings(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   20 * time.Millisecond,
		Timeout:    5 * time.Millisecond,
		MaxRetries: 5,
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	ka.Start()
	time.Sleep(100 * time.Millisecond)
	ka.Stop()
}

func TestKeepAliveWithPingHandler(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	pingCalled := false
	config := &KeepAliveConfig{
		Interval:   30 * time.Millisecond,
		Timeout:    10 * time.Millisecond,
		MaxRetries: 3,
		PingHandler: func(c *Connection) error {
			pingCalled = true
			return nil
		},
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	ka.Start()
	time.Sleep(50 * time.Millisecond)
	ka.Stop()

	assert.True(t, pingCalled)
}

func TestKeepAliveLastPing(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   30 * time.Millisecond,
		Timeout:    10 * time.Millisecond,
		MaxRetries: 3,
		PingHandler: func(c *Connection) error {
			return nil
		},
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	initialPing := ka.LastPing()
	assert.True(t, initialPing.IsZero())

	ka.Start()
	time.Sleep(50 * time.Millisecond)
	ka.Stop()

	laterPing := ka.LastPing()
	assert.True(t, laterPing.After(initialPing))
}

func TestNewKeepAliveManager(t *testing.T) {
	kam := NewKeepAliveManager(nil)
	assert.NotNil(t, kam)
	defer kam.Close()
}

func TestKeepAliveManagerWithConfig(t *testing.T) {
	config := &KeepAliveConfig{
		Interval:   60 * time.Second,
		Timeout:    20 * time.Second,
		MaxRetries: 5,
	}
	kam := NewKeepAliveManager(config)
	assert.NotNil(t, kam)
	assert.Equal(t, config, kam.config)
	defer kam.Close()
}

func TestKeepAliveManagerAdd(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	kam := NewKeepAliveManager(nil)
	defer kam.Close()

	ka := kam.Add(conn)
	assert.NotNil(t, ka)

	retrieved, ok := kam.Get(conn.ID())
	assert.True(t, ok)
	assert.Equal(t, ka, retrieved)
}

func TestKeepAliveManagerRemove(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	kam := NewKeepAliveManager(nil)
	defer kam.Close()

	ka := kam.Add(conn)
	assert.NotNil(t, ka)

	kam.Remove(conn.ID())

	_, ok := kam.Get(conn.ID())
	assert.False(t, ok)
}

func TestKeepAliveManagerGetNonExistent(t *testing.T) {
	kam := NewKeepAliveManager(nil)
	defer kam.Close()

	_, ok := kam.Get("non-existent")
	assert.False(t, ok)
}

func TestKeepAliveManagerClose(t *testing.T) {
	kam := NewKeepAliveManager(nil)

	for i := 0; i < 3; i++ {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		conn := NewConnection(server, fmt.Sprintf("conn-%d", i), nil)
		kam.Add(conn)
	}

	kam.Close()

	_, ok := kam.Get("conn-0")
	assert.False(t, ok)
}

func TestKeepAliveConnectionClose(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   30 * time.Millisecond,
		Timeout:    10 * time.Millisecond,
		MaxRetries: 3,
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	ka.Start()
	time.Sleep(20 * time.Millisecond)
	conn.Close()
	time.Sleep(20 * time.Millisecond)
	ka.Stop()
}

func TestKeepAlivePongResetsMissedPings(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	config := &KeepAliveConfig{
		Interval:   20 * time.Millisecond,
		Timeout:    5 * time.Millisecond,
		MaxRetries: 5,
		PingHandler: func(c *Connection) error {
			return nil
		},
	}

	ka := NewKeepAlive(conn, config)
	require.NotNil(t, ka)

	ka.Start()
	time.Sleep(60 * time.Millisecond)

	ka.OnPong()
	assert.Equal(t, 0, ka.MissedPings())

	ka.Stop()
}
