//go:build darwin

package network

import (
	"fmt"
	"syscall"
	"time"
)

type KqueuePoller struct {
	fd     int
	state  *pollerState
	config *PollerConfig
	events []syscall.Kevent_t
}

func NewPoller(config *PollerConfig) (Poller, error) {
	if config == nil {
		config = DefaultPollerConfig()
	}

	kqfd, err := syscall.Kqueue()
	if err != nil {
		return nil, fmt.Errorf("kqueue failed: %w", err)
	}

	return &KqueuePoller{
		fd:     kqfd,
		state:  newPollerState(),
		config: config,
		events: make([]syscall.Kevent_t, config.MaxEvents),
	}, nil
}

func (kp *KqueuePoller) Add(conn *Connection, events EventType) error {
	if kp.state.isClosed() {
		return ErrListenerClosed
	}

	fd, err := getConnFd(conn)
	if err != nil {
		return err
	}

	changes := make([]syscall.Kevent_t, 0, 2)

	if events&EventRead != 0 {
		changes = append(changes, syscall.Kevent_t{
			Ident:  uint64(fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
		})
	}

	if events&EventWrite != 0 {
		changes = append(changes, syscall.Kevent_t{
			Ident:  uint64(fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
		})
	}

	if len(changes) > 0 {
		_, err := syscall.Kevent(kp.fd, changes, nil, nil)
		if err != nil {
			return fmt.Errorf("kevent add failed: %w", err)
		}
	}

	kp.state.add(fd, conn)
	return nil
}

func (kp *KqueuePoller) Modify(conn *Connection, events EventType) error {
	return kp.Add(conn, events)
}

func (kp *KqueuePoller) Remove(conn *Connection) error {
	fd, err := getConnFd(conn)
	if err != nil {
		return err
	}

	changes := []syscall.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_DELETE,
		},
		{
			Ident:  uint64(fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_DELETE,
		},
	}

	syscall.Kevent(kp.fd, changes, nil, nil)

	kp.state.remove(fd)
	return nil
}

func (kp *KqueuePoller) Wait(timeout time.Duration) ([]*Event, error) {
	if kp.state.isClosed() {
		return nil, ErrListenerClosed
	}

	var ts *syscall.Timespec
	if timeout >= 0 {
		t := syscall.NsecToTimespec(timeout.Nanoseconds())
		ts = &t
	}

	n, err := syscall.Kevent(kp.fd, nil, kp.events, ts)
	if err != nil {
		if err == syscall.EINTR {
			return nil, nil
		}
		return nil, fmt.Errorf("kevent wait failed: %w", err)
	}

	events := make([]*Event, 0, n)
	for i := 0; i < n; i++ {
		fd := int(kp.events[i].Ident)
		conn, ok := kp.state.get(fd)
		if !ok {
			continue
		}

		event := &Event{
			Fd:   fd,
			Conn: conn,
		}

		if kp.events[i].Flags&syscall.EV_EOF != 0 {
			event.Error = ErrConnectionClosed
		}

		events = append(events, event)
	}

	return events, nil
}

func (kp *KqueuePoller) Close() error {
	if kp.state.isClosed() {
		return nil
	}

	kp.state.close()
	return syscall.Close(kp.fd)
}
