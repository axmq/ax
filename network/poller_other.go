//go:build !linux && !darwin

package network

import (
	"sync/atomic"
	"time"
)

type FallbackPoller struct {
	state  *pollerState
	config *PollerConfig
	nextFd atomic.Int64
}

func NewPoller(config *PollerConfig) (Poller, error) {
	if config == nil {
		config = DefaultPollerConfig()
	}

	return &FallbackPoller{
		state:  newPollerState(),
		config: config,
	}, nil
}

func (fp *FallbackPoller) Add(conn *Connection, events EventType) error {
	if fp.state.isClosed() {
		return ErrListenerClosed
	}

	fd, err := getConnFd(conn)
	if err != nil {
		fd = int(fp.nextFd.Add(1))
	}

	fp.state.add(fd, conn)
	return nil
}

func (fp *FallbackPoller) Modify(conn *Connection, events EventType) error {
	return nil
}

func (fp *FallbackPoller) Remove(conn *Connection) error {
	fd, _ := getConnFd(conn)
	fp.state.remove(fd)
	return nil
}

func (fp *FallbackPoller) Wait(timeout time.Duration) ([]*Event, error) {
	if fp.state.isClosed() {
		return nil, ErrListenerClosed
	}

	time.Sleep(timeout)
	return nil, nil
}

func (fp *FallbackPoller) Close() error {
	if fp.state.isClosed() {
		return nil
	}

	fp.state.close()
	return nil
}
