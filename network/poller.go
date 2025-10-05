package network

import (
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Event struct {
	Fd    int
	Conn  *Connection
	Error error
}

type EventType int

const (
	EventRead EventType = 1 << iota
	EventWrite
	EventError
	EventHup
)

type Poller interface {
	Add(conn *Connection, events EventType) error
	Modify(conn *Connection, events EventType) error
	Remove(conn *Connection) error
	Wait(timeout time.Duration) ([]*Event, error)
	Close() error
}

type PollerConfig struct {
	MaxEvents int
	Timeout   time.Duration
}

func DefaultPollerConfig() *PollerConfig {
	return &PollerConfig{
		MaxEvents: 1024,
		Timeout:   100 * time.Millisecond,
	}
}

type pollerState struct {
	mu      sync.RWMutex
	connMap map[int]*Connection
	closed  atomic.Bool
}

func newPollerState() *pollerState {
	return &pollerState{
		connMap: make(map[int]*Connection),
	}
}

func (ps *pollerState) add(fd int, conn *Connection) {
	ps.mu.Lock()
	ps.connMap[fd] = conn
	ps.mu.Unlock()
}

func (ps *pollerState) get(fd int) (*Connection, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	conn, ok := ps.connMap[fd]
	return conn, ok
}

func (ps *pollerState) remove(fd int) {
	ps.mu.Lock()
	delete(ps.connMap, fd)
	ps.mu.Unlock()
}

func (ps *pollerState) isClosed() bool {
	return ps.closed.Load()
}

func (ps *pollerState) close() {
	ps.closed.Store(true)
}

func getConnFd(conn *Connection) (int, error) {
	type syscallConn interface {
		SyscallConn() (syscall.RawConn, error)
	}

	if sc, ok := conn.conn.(syscallConn); ok {
		rawConn, err := sc.SyscallConn()
		if err != nil {
			return -1, err
		}

		var fd int
		err = rawConn.Control(func(fdPtr uintptr) {
			fd = int(fdPtr)
		})
		if err != nil {
			return -1, err
		}

		return fd, nil
	}

	return -1, syscall.ENOTSUP
}
