package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDisconnectManager(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)
	require.NotNil(t, dm)
	assert.Equal(t, 5*time.Second, dm.gracefulTimeout)
}

func TestNewDisconnectManagerDefaultTimeout(t *testing.T) {
	dm := NewDisconnectManager(0)
	require.NotNil(t, dm)
	assert.Equal(t, 5*time.Second, dm.gracefulTimeout)
}

func TestDisconnectManagerOnDisconnect(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)

	callCount := 0
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		callCount++
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectNormalDisconnection,
	}

	err := dm.HandleDisconnect(conn, packet)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDisconnectManagerMultipleHandlers(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)

	call1 := false
	call2 := false

	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		call1 = true
		return nil
	})

	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		call2 = true
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectNormalDisconnection,
	}

	err := dm.HandleDisconnect(conn, packet)
	assert.NoError(t, err)
	assert.True(t, call1)
	assert.True(t, call2)
}

func TestDisconnectManagerHandlerError(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)

	testErr := errors.New("handler error")
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		return testErr
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectNormalDisconnection,
	}

	err := dm.HandleDisconnect(conn, packet)
	assert.Equal(t, testErr, err)
}

func TestDisconnectManagerHandleDisconnect(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)

	received := false
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		received = true
		assert.Equal(t, DisconnectServerBusy, packet.ReasonCode)
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectServerBusy,
	}

	err := dm.HandleDisconnect(conn, packet)
	assert.NoError(t, err)
	assert.True(t, received)
}

func TestDisconnectManagerGracefulDisconnect(t *testing.T) {
	dm := NewDisconnectManager(100 * time.Millisecond)

	server, client := net.Pipe()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	err := dm.GracefulDisconnect(context.Background(), conn, DisconnectNormalDisconnection)
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, conn.State())
}

func TestDisconnectManagerGracefulDisconnectTimeout(t *testing.T) {
	dm := NewDisconnectManager(1 * time.Millisecond)

	server, client := net.Pipe()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	dm.OnDisconnect(func(c *Connection, p *DisconnectPacket) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	err := dm.GracefulDisconnect(context.Background(), conn, DisconnectNormalDisconnection)
	assert.Equal(t, ErrGracefulShutdownTimeout, err)
}

func TestDisconnectManagerSendDisconnect(t *testing.T) {
	dm := NewDisconnectManager(100 * time.Millisecond)

	handlerCalled := false
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		handlerCalled = true
		assert.Equal(t, DisconnectServerShuttingDown, packet.ReasonCode)
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectServerShuttingDown,
	}

	err := dm.SendDisconnect(conn, packet)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestDisconnectManagerSendDisconnectNilPacket(t *testing.T) {
	dm := NewDisconnectManager(100 * time.Millisecond)

	var receivedPacket *DisconnectPacket
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		receivedPacket = packet
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	err := dm.SendDisconnect(conn, nil)
	assert.NoError(t, err)
	assert.NotNil(t, receivedPacket)
	assert.Equal(t, DisconnectNormalDisconnection, receivedPacket.ReasonCode)
}

func TestNewGracefulShutdown(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(5 * time.Second)
	gs := NewGracefulShutdown(pool, dm, 1*time.Second)
	require.NotNil(t, gs)
	assert.Equal(t, 1*time.Second, gs.timeout)
}

func TestNewGracefulShutdownDefaultTimeout(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(5 * time.Second)
	gs := NewGracefulShutdown(pool, dm, 0)
	require.NotNil(t, gs)
	assert.Equal(t, 30*time.Second, gs.timeout)
}

func TestGracefulShutdownShutdown(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(100 * time.Millisecond)
	gs := NewGracefulShutdown(pool, dm, 1*time.Second)

	server, client := net.Pipe()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	err := pool.Add(conn)
	require.NoError(t, err)

	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, gs.IsShutdown())
}

func TestGracefulShutdownIsShutdown(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(100 * time.Millisecond)
	gs := NewGracefulShutdown(pool, dm, 1*time.Second)

	assert.False(t, gs.IsShutdown())

	err := gs.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, gs.IsShutdown())
}

func TestGracefulShutdownMultipleShutdowns(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(100 * time.Millisecond)
	gs := NewGracefulShutdown(pool, dm, 1*time.Second)

	err1 := gs.Shutdown(context.Background())
	assert.NoError(t, err1)

	err2 := gs.Shutdown(context.Background())
	assert.NoError(t, err2)
}

func TestGracefulShutdownMultipleConnections(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	dm := NewDisconnectManager(100 * time.Millisecond)
	gs := NewGracefulShutdown(pool, dm, 1*time.Second)

	for i := 0; i < 5; i++ {
		server, client := net.Pipe()
		defer client.Close()
		conn := NewConnection(server, fmt.Sprintf("conn-%d", i), nil)
		err := pool.Add(conn)
		require.NoError(t, err)
	}

	err := gs.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, gs.IsShutdown())
}

func TestDisconnectPacketWithProperties(t *testing.T) {
	dm := NewDisconnectManager(5 * time.Second)

	var receivedPacket *DisconnectPacket
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		receivedPacket = packet
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)
	sessionExpiry := uint32(3600)
	packet := &DisconnectPacket{
		ReasonCode:      DisconnectNormalDisconnection,
		SessionExpiry:   &sessionExpiry,
		ReasonString:    "Test disconnect",
		ServerReference: "test-server",
	}

	err := dm.HandleDisconnect(conn, packet)
	assert.NoError(t, err)
	assert.NotNil(t, receivedPacket)
	assert.Equal(t, DisconnectNormalDisconnection, receivedPacket.ReasonCode)
	assert.NotNil(t, receivedPacket.SessionExpiry)
	assert.Equal(t, uint32(3600), *receivedPacket.SessionExpiry)
	assert.Equal(t, "Test disconnect", receivedPacket.ReasonString)
	assert.Equal(t, "test-server", receivedPacket.ServerReference)
}
