//go:build linux

package network

import (
	"errors"
	"fmt"
	"syscall"
	"time"
)

type EpollPoller struct {
	fd     int
	state  *pollerState
	config *PollerConfig
	events []syscall.EpollEvent
}

func NewPoller(config *PollerConfig) (Poller, error) {
	if config == nil {
		config = DefaultPollerConfig()
	}

	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("epoll_create1 failed: %w", err)
	}

	return &EpollPoller{
		fd:     epfd,
		state:  newPollerState(),
		config: config,
		events: make([]syscall.EpollEvent, config.MaxEvents),
	}, nil
}

func (ep *EpollPoller) Add(conn *Connection, events EventType) error {
	if ep.state.isClosed() {
		return ErrListenerClosed
	}

	fd, err := getConnFd(conn)
	if err != nil {
		return err
	}

	event := syscall.EpollEvent{
		Events: ep.eventsToEpoll(events),
		Fd:     int32(fd),
	}

	if err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, fd, &event); err != nil {
		return fmt.Errorf("epoll_ctl add failed: %w", err)
	}

	ep.state.add(fd, conn)
	return nil
}

func (ep *EpollPoller) Modify(conn *Connection, events EventType) error {
	if ep.state.isClosed() {
		return ErrListenerClosed
	}

	fd, err := getConnFd(conn)
	if err != nil {
		return err
	}

	event := syscall.EpollEvent{
		Events: ep.eventsToEpoll(events),
		Fd:     int32(fd),
	}

	if err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_MOD, fd, &event); err != nil {
		return fmt.Errorf("epoll_ctl mod failed: %w", err)
	}

	return nil
}

func (ep *EpollPoller) Remove(conn *Connection) error {
	fd, err := getConnFd(conn)
	if err != nil {
		return err
	}

	if err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_DEL, fd, nil); err != nil && err != syscall.ENOENT {
		return fmt.Errorf("epoll_ctl del failed: %w", err)
	}

	ep.state.remove(fd)
	return nil
}

func (ep *EpollPoller) Wait(timeout time.Duration) ([]*Event, error) {
	if ep.state.isClosed() {
		return nil, ErrListenerClosed
	}

	timeoutMs := int(timeout.Milliseconds())
	if timeoutMs < 0 {
		timeoutMs = -1
	}

	n, err := syscall.EpollWait(ep.fd, ep.events, timeoutMs)
	if err != nil {
		if errors.Is(err, syscall.EINTR) {
			return nil, nil
		}
		return nil, fmt.Errorf("epoll_wait failed: %w", err)
	}

	events := make([]*Event, 0, n)
	for i := 0; i < n; i++ {
		fd := int(ep.events[i].Fd)
		conn, ok := ep.state.get(fd)
		if !ok {
			continue
		}

		event := &Event{
			Fd:   fd,
			Conn: conn,
		}

		if ep.events[i].Events&(syscall.EPOLLERR|syscall.EPOLLHUP) != 0 {
			event.Error = ErrConnectionClosed
		}

		events = append(events, event)
	}

	return events, nil
}

func (ep *EpollPoller) Close() error {
	if ep.state.isClosed() {
		return nil
	}

	ep.state.close()
	return syscall.Close(ep.fd)
}

func (ep *EpollPoller) eventsToEpoll(events EventType) uint32 {
	var epollEvents uint32

	if events&EventRead != 0 {
		epollEvents |= uint32(syscall.EPOLLIN)
	}
	if events&EventWrite != 0 {
		epollEvents |= uint32(syscall.EPOLLOUT)
	}

	epollEvents |= 1 << 31 // EPOLLET edge-triggered flag

	return epollEvents
}
