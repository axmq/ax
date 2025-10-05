package network

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPool(t *testing.T, config *PoolConfig) *Pool {
	pool, err := NewPool(config)
	require.NoError(t, err)
	require.NotNil(t, pool)
	return pool
}

func createTestConn(t *testing.T, id string) (*Connection, net.Conn) {
	server, client := net.Pipe()
	conn := NewConnection(server, id, nil)
	require.NotNil(t, conn)
	return conn, client
}

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 10000, config.MaxConnections)
	assert.Equal(t, 1000, config.MaxIdleConnections)
	assert.Equal(t, 5*time.Minute, config.MaxIdleTime)
	assert.Equal(t, 30*time.Minute, config.MaxLifetime)
	assert.Equal(t, 1*time.Minute, config.CleanupInterval)
}

func TestNewPool(t *testing.T) {
	tests := []struct {
		name      string
		config    *PoolConfig
		expectErr bool
	}{
		{
			name:      "default config",
			config:    nil,
			expectErr: false,
		},
		{
			name:      "custom config",
			config:    &PoolConfig{MaxConnections: 100, MaxIdleConnections: 10, CleanupInterval: 1 * time.Second},
			expectErr: false,
		},
		{
			name:      "invalid config",
			config:    &PoolConfig{MaxConnections: 0},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewPool(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pool)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pool)
				defer pool.Close()
			}
		})
	}
}

func TestNewPoolMaxIdleAdjustment(t *testing.T) {
	config := &PoolConfig{
		MaxConnections:     10,
		MaxIdleConnections: 20,
		CleanupInterval:    0,
	}
	pool, err := NewPool(config)
	require.NoError(t, err)
	defer pool.Close()

	assert.Equal(t, 10, pool.config.MaxIdleConnections)
}

func TestPoolAdd(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{MaxConnections: 2, MaxIdleConnections: 1, CleanupInterval: 0})
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	assert.NoError(t, err)

	stats := pool.Stats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Active)

	conn2, client2 := createTestConn(t, "conn-2")
	defer client2.Close()

	err = pool.Add(conn2)
	assert.NoError(t, err)

	stats = pool.Stats()
	assert.Equal(t, 2, stats.Total)

	conn3, client3 := createTestConn(t, "conn-3")
	defer client3.Close()

	err = pool.Add(conn3)
	assert.Equal(t, ErrConnectionPoolExhausted, err)
}

func TestPoolGet(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	require.NoError(t, err)

	retrieved, ok := pool.Get("conn-1")
	assert.True(t, ok)
	assert.Equal(t, conn1, retrieved)

	_, ok = pool.Get("non-existent")
	assert.False(t, ok)
}

func TestPoolRemove(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	require.NoError(t, err)

	err = pool.Remove("conn-1")
	assert.NoError(t, err)

	_, ok := pool.Get("conn-1")
	assert.False(t, ok)

	err = pool.Remove("non-existent")
	assert.Equal(t, ErrConnectionNotFound, err)
}

func TestPoolRelease(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{MaxConnections: 10, MaxIdleConnections: 5, CleanupInterval: 0})
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	require.NoError(t, err)

	err = pool.Release(conn1)
	assert.NoError(t, err)

	stats := pool.Stats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Active)
	assert.Equal(t, 1, stats.Idle)
}

func TestPoolReleaseExceedingMaxIdle(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{MaxConnections: 10, MaxIdleConnections: 1, CleanupInterval: 0})
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()
	conn2, client2 := createTestConn(t, "conn-2")
	defer client2.Close()

	pool.Add(conn1)
	pool.Add(conn2)

	pool.Release(conn1)
	stats := pool.Stats()
	assert.Equal(t, 1, stats.Idle)

	pool.Release(conn2)
	stats = pool.Stats()
	assert.Equal(t, 1, stats.Idle)
	assert.Equal(t, 1, stats.Total)
}

func TestPoolReleaseClosedConnection(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	pool.Add(conn1)
	conn1.Close()

	err := pool.Release(conn1)
	assert.NoError(t, err)

	_, ok := pool.Get("conn-1")
	assert.False(t, ok)
}

func TestPoolClose(t *testing.T) {
	pool := createTestPool(t, nil)

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	require.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)
	assert.True(t, pool.IsClosed())

	err = pool.Add(conn1)
	assert.Equal(t, ErrPoolClosed, err)
}

func TestPoolCloseMultipleTimes(t *testing.T) {
	pool := createTestPool(t, nil)

	err1 := pool.Close()
	assert.NoError(t, err1)

	err2 := pool.Close()
	assert.NoError(t, err2)
}

func TestPoolForEach(t *testing.T) {
	pool := createTestPool(t, nil)
	defer pool.Close()

	for i := 0; i < 3; i++ {
		conn, client := createTestConn(t, fmt.Sprintf("conn-%d", i))
		defer client.Close()
		pool.Add(conn)
	}

	count := 0
	pool.ForEach(func(conn *Connection) bool {
		count++
		return true
	})

	assert.Equal(t, 3, count)
}

func TestPoolForEachEarlyExit(t *testing.T) {
	pool, err := NewPool(nil)
	require.NoError(t, err)
	defer pool.Close()

	server1, client1 := net.Pipe()
	defer server1.Close()
	defer client1.Close()
	conn1 := NewConnection(server1, "conn1", nil)
	pool.Add(conn1)

	server2, client2 := net.Pipe()
	defer server2.Close()
	defer client2.Close()
	conn2 := NewConnection(server2, "conn2", nil)
	pool.Add(conn2)

	visited := 0
	pool.ForEach(func(conn *Connection) bool {
		visited++
		return false
	})
	assert.Equal(t, 1, visited)
}

func TestPoolManagerCreation(t *testing.T) {
	pm, err := NewPoolManager(nil)
	require.NoError(t, err)
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.Pool())

	err = pm.Close()
	assert.NoError(t, err)
}

func TestPoolManagerWithConfig(t *testing.T) {
	config := &PoolConfig{
		MaxConnections:     10,
		MaxIdleConnections: 5,
		MaxLifetime:        5 * time.Minute,
		MaxIdleTime:        2 * time.Minute,
		CleanupInterval:    1 * time.Minute,
	}

	pm, err := NewPoolManager(config)
	require.NoError(t, err)
	assert.NotNil(t, pm)

	pool := pm.Pool()
	assert.NotNil(t, pool)

	server, client := net.Pipe()
	defer client.Close()
	conn := NewConnection(server, "test", nil)
	_ = pool.Add(conn)

	retrieved, ok := pool.Get("test")
	assert.True(t, ok)
	assert.Equal(t, conn, retrieved)

	err = pm.Close()
	assert.NoError(t, err)
}

func TestPoolManagerClose(t *testing.T) {
	pm, err := NewPoolManager(nil)
	require.NoError(t, err)

	server, client := net.Pipe()
	defer client.Close()
	conn := NewConnection(server, "test", nil)
	pm.Pool().Add(conn)

	err = pm.Close()
	assert.NoError(t, err)
	assert.True(t, pm.Pool().IsClosed())
}

func TestPoolCleanup(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{
		MaxConnections:     10,
		MaxIdleConnections: 5,
		MaxIdleTime:        50 * time.Millisecond,
		CleanupInterval:    100 * time.Millisecond,
	})
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	pool.Add(conn1)
	pool.Release(conn1)

	time.Sleep(200 * time.Millisecond)

	_, ok := pool.Get("conn-1")
	assert.False(t, ok)
}

func TestPoolCleanupMaxLifetime(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{
		MaxConnections:     10,
		MaxIdleConnections: 5,
		MaxLifetime:        50 * time.Millisecond,
		CleanupInterval:    100 * time.Millisecond,
	})
	defer pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	pool.Add(conn1)
	pool.Release(conn1)

	time.Sleep(200 * time.Millisecond)

	_, ok := pool.Get("conn-1")
	assert.False(t, ok)
}

func TestPoolAddToClosed(t *testing.T) {
	pool := createTestPool(t, nil)
	pool.Close()

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	err := pool.Add(conn1)
	assert.Equal(t, ErrPoolClosed, err)
}

func TestPoolReleaseToClosed(t *testing.T) {
	pool := createTestPool(t, nil)

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()

	pool.Add(conn1)
	pool.Close()

	err := pool.Release(conn1)
	assert.NoError(t, err)
}

func TestPoolStats(t *testing.T) {
	pool := createTestPool(t, &PoolConfig{MaxConnections: 10, MaxIdleConnections: 5, CleanupInterval: 0})
	defer pool.Close()

	stats := pool.Stats()
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Active)
	assert.Equal(t, 0, stats.Idle)

	conn1, client1 := createTestConn(t, "conn-1")
	defer client1.Close()
	pool.Add(conn1)

	stats = pool.Stats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Active)
	assert.Equal(t, 0, stats.Idle)

	pool.Release(conn1)

	stats = pool.Stats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Active)
	assert.Equal(t, 1, stats.Idle)
}
