package network

import (
	"context"
	"sync"
	"time"
)

type KeepAliveConfig struct {
	Interval    time.Duration
	Timeout     time.Duration
	MaxRetries  int
	PingHandler func(*Connection) error
	PongHandler func(*Connection) error
}

func DefaultKeepAliveConfig() *KeepAliveConfig {
	return &KeepAliveConfig{
		Interval:   30 * time.Second,
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}
}

type KeepAlive struct {
	config *KeepAliveConfig
	conn   *Connection

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	lastPing time.Time
	lastPong time.Time
	mu       sync.RWMutex

	missedPings int
}

func NewKeepAlive(conn *Connection, config *KeepAliveConfig) *KeepAlive {
	if config == nil {
		config = DefaultKeepAliveConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	ka := &KeepAlive{
		config:   config,
		conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
		lastPong: time.Now(),
	}

	return ka
}

func (ka *KeepAlive) Start() {
	ka.wg.Add(1)
	go ka.keepAliveLoop()
}

func (ka *KeepAlive) keepAliveLoop() {
	defer ka.wg.Done()

	ticker := time.NewTicker(ka.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ka.sendPing(); err != nil {
				ka.conn.Close()
				return
			}
		case <-ka.ctx.Done():
			return
		case <-ka.conn.CloseChan():
			return
		}
	}
}

func (ka *KeepAlive) sendPing() error {
	ka.mu.Lock()
	defer ka.mu.Unlock()

	if time.Since(ka.lastPong) > ka.config.Interval+ka.config.Timeout {
		ka.missedPings++
		if ka.missedPings >= ka.config.MaxRetries {
			return ErrKeepAliveTimeout
		}
	}

	ka.lastPing = time.Now()

	if ka.config.PingHandler != nil {
		return ka.config.PingHandler(ka.conn)
	}

	return nil
}

func (ka *KeepAlive) OnPong() {
	ka.mu.Lock()
	defer ka.mu.Unlock()

	ka.lastPong = time.Now()
	ka.missedPings = 0
}

func (ka *KeepAlive) Stop() {
	ka.cancel()
	ka.wg.Wait()
}

func (ka *KeepAlive) LastPing() time.Time {
	ka.mu.RLock()
	defer ka.mu.RUnlock()
	return ka.lastPing
}

func (ka *KeepAlive) LastPong() time.Time {
	ka.mu.RLock()
	defer ka.mu.RUnlock()
	return ka.lastPong
}

func (ka *KeepAlive) MissedPings() int {
	ka.mu.RLock()
	defer ka.mu.RUnlock()
	return ka.missedPings
}

type KeepAliveManager struct {
	mu         sync.RWMutex
	keepAlives map[string]*KeepAlive
	config     *KeepAliveConfig
}

func NewKeepAliveManager(config *KeepAliveConfig) *KeepAliveManager {
	if config == nil {
		config = DefaultKeepAliveConfig()
	}

	return &KeepAliveManager{
		keepAlives: make(map[string]*KeepAlive),
		config:     config,
	}
}

func (kam *KeepAliveManager) Add(conn *Connection) *KeepAlive {
	ka := NewKeepAlive(conn, kam.config)

	kam.mu.Lock()
	kam.keepAlives[conn.ID()] = ka
	kam.mu.Unlock()

	ka.Start()
	return ka
}

func (kam *KeepAliveManager) Remove(connID string) {
	kam.mu.Lock()
	defer kam.mu.Unlock()

	if ka, ok := kam.keepAlives[connID]; ok {
		ka.Stop()
		delete(kam.keepAlives, connID)
	}
}

func (kam *KeepAliveManager) Get(connID string) (*KeepAlive, bool) {
	kam.mu.RLock()
	defer kam.mu.RUnlock()

	ka, ok := kam.keepAlives[connID]
	return ka, ok
}

func (kam *KeepAliveManager) Close() {
	kam.mu.Lock()
	defer kam.mu.Unlock()

	for _, ka := range kam.keepAlives {
		ka.Stop()
	}

	kam.keepAlives = make(map[string]*KeepAlive)
}
