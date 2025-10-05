package network

import (
	"context"
	"net"
	"testing"
	"time"
)

var (
	benchConn    *Connection
	benchPool    *Pool
	benchBackoff *Backoff
	benchData    = make([]byte, 1024)
)

func BenchmarkConnectionRead(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	go func() {
		for i := 0; i < b.N; i++ {
			client.Write(benchData)
		}
	}()

	buf := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Read(buf)
	}
}

func BenchmarkConnectionWrite(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	go func() {
		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			client.Read(buf)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Write(benchData)
	}
}

func BenchmarkConnectionState(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = conn.State()
	}
}

func BenchmarkConnectionBytesRead(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = conn.BytesRead()
	}
}

func BenchmarkConnectionMetadataSet(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.SetMetadata("key", "value")
	}
}

func BenchmarkConnectionMetadataGet(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()
	conn.SetMetadata("key", "value")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = conn.GetMetadata("key")
	}
}

func BenchmarkPoolAdd(b *testing.B) {
	config := &PoolConfig{
		MaxConnections:     100000,
		MaxIdleConnections: 10000,
		CleanupInterval:    0,
	}
	pool, _ := NewPool(config)
	defer pool.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		server, client := net.Pipe()
		conn := NewConnection(server, "bench-conn", nil)
		pool.Add(conn)
		server.Close()
		client.Close()
	}
}

func BenchmarkPoolGet(b *testing.B) {
	pool, _ := NewPool(nil)
	defer pool.Close()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	pool.Add(conn)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = pool.Get("bench-conn")
	}
}

func BenchmarkPoolStats(b *testing.B) {
	pool, _ := NewPool(nil)
	defer pool.Close()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	pool.Add(conn)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = pool.Stats()
	}
}

func BenchmarkBackoffNext(b *testing.B) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      10,
		Jitter:          false,
		JitterFactor:    0.2,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		backoff, _ := NewBackoff(config)
		_, _ = backoff.Next()
	}
}

func BenchmarkBackoffNextWithJitter(b *testing.B) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      10,
		Jitter:          true,
		JitterFactor:    0.2,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		backoff, _ := NewBackoff(config)
		_, _ = backoff.Next()
	}
}

func BenchmarkDisconnectManagerHandleDisconnect(b *testing.B) {
	dm := NewDisconnectManager(5 * time.Second)
	dm.OnDisconnect(func(conn *Connection, packet *DisconnectPacket) error {
		return nil
	})

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	packet := &DisconnectPacket{
		ReasonCode: DisconnectNormalDisconnection,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = dm.HandleDisconnect(conn, packet)
	}
}

func BenchmarkKeepAliveOnPong(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	ka := NewKeepAlive(conn, nil)
	defer ka.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ka.OnPong()
	}
}

func BenchmarkRecoveryRetryNoError(b *testing.B) {
	recovery, _ := NewRecovery(&RecoveryConfig{
		BackoffConfig:  DefaultBackoffConfig(),
		EnableRecovery: false,
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = recovery.Retry(context.Background(), func() error {
			return nil
		})
	}
}

func BenchmarkConnectionLastActivity(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = conn.LastActivity()
	}
}

func BenchmarkConnectionIdleDuration(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = conn.IdleDuration()
	}
}

func BenchmarkPoolForEach(b *testing.B) {
	pool, _ := NewPool(nil)
	defer pool.Close()

	for i := 0; i < 100; i++ {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		conn := NewConnection(server, "bench-conn", nil)
		pool.Add(conn)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool.ForEach(func(conn *Connection) bool {
			return true
		})
	}
}

func BenchmarkNewConnection(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		server, client := net.Pipe()
		conn := NewConnection(server, "bench-conn", nil)
		conn.Close()
		client.Close()
	}
}

func BenchmarkConnectionRemoteAddr(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "bench-conn", nil)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = conn.RemoteAddr()
	}
}

func BenchmarkPoolRelease(b *testing.B) {
	config := &PoolConfig{
		MaxConnections:     100000,
		MaxIdleConnections: 10000,
		CleanupInterval:    0,
	}
	pool, _ := NewPool(config)
	defer pool.Close()

	conns := make([]*Connection, b.N)
	for i := 0; i < b.N; i++ {
		server, _ := net.Pipe()
		conn := NewConnection(server, "bench-conn", nil)
		pool.Add(conn)
		conns[i] = conn
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool.Release(conns[i])
	}
}

func BenchmarkBackoffReset(b *testing.B) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      10,
		Jitter:          false,
	}
	backoff, _ := NewBackoff(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		backoff.Reset()
	}
}

func BenchmarkKeepAliveManagerAdd(b *testing.B) {
	kam := NewKeepAliveManager(nil)
	defer kam.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		server, client := net.Pipe()
		conn := NewConnection(server, "bench-conn", nil)
		ka := kam.Add(conn)
		ka.Stop()
		server.Close()
		client.Close()
	}
}
