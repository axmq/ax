package network

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type PoolConfig struct {
	MaxConnections     int
	MaxIdleConnections int
	MaxIdleTime        time.Duration
	MaxLifetime        time.Duration
	CleanupInterval    time.Duration
}

func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:     10000,
		MaxIdleConnections: 1000,
		MaxIdleTime:        5 * time.Minute,
		MaxLifetime:        30 * time.Minute,
		CleanupInterval:    1 * time.Minute,
	}
}

type Pool struct {
	config *PoolConfig

	mu    sync.RWMutex
	conns map[string]*Connection
	idle  []*Connection

	active atomic.Int32
	total  atomic.Int32

	closed    atomic.Bool
	closeCh   chan struct{}
	closeOnce sync.Once

	wg sync.WaitGroup
}

func NewPool(config *PoolConfig) (*Pool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	if config.MaxConnections <= 0 {
		return nil, ErrInvalidPoolConfig
	}

	if config.MaxIdleConnections > config.MaxConnections {
		config.MaxIdleConnections = config.MaxConnections
	}

	p := &Pool{
		config:  config,
		conns:   make(map[string]*Connection),
		idle:    make([]*Connection, 0, config.MaxIdleConnections),
		closeCh: make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		p.wg.Add(1)
		go p.cleanupLoop()
	}

	return p, nil
}

func (p *Pool) Add(conn *Connection) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if int(p.total.Load()) >= p.config.MaxConnections {
		return ErrConnectionPoolExhausted
	}

	p.conns[conn.ID()] = conn
	p.total.Add(1)
	p.active.Add(1)

	return nil
}

func (p *Pool) Get(id string) (*Connection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conn, ok := p.conns[id]
	return conn, ok
}

func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, ok := p.conns[id]
	if !ok {
		return ErrConnectionNotFound
	}

	delete(p.conns, id)
	p.total.Add(-1)

	if conn.State() == StateConnected {
		p.active.Add(-1)
	}

	return conn.Close()
}

func (p *Pool) Release(conn *Connection) error {
	if p.closed.Load() {
		return conn.Close()
	}

	if conn.State() != StateConnected {
		return p.Remove(conn.ID())
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.idle) >= p.config.MaxIdleConnections {
		return p.removeUnlocked(conn.ID())
	}

	p.idle = append(p.idle, conn)
	p.active.Add(-1)

	return nil
}

func (p *Pool) removeUnlocked(id string) error {
	conn, ok := p.conns[id]
	if !ok {
		return ErrConnectionNotFound
	}

	delete(p.conns, id)
	p.total.Add(-1)

	if conn.State() == StateConnected {
		p.active.Add(-1)
	}

	return conn.Close()
}

func (p *Pool) cleanupLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.closeCh:
			return
		}
	}
}

func (p *Pool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	newIdle := p.idle[:0]

	for _, conn := range p.idle {
		if conn.State() != StateConnected {
			p.removeUnlocked(conn.ID())
			continue
		}

		if p.config.MaxIdleTime > 0 && conn.IdleDuration() > p.config.MaxIdleTime {
			p.removeUnlocked(conn.ID())
			continue
		}

		if p.config.MaxLifetime > 0 && now.Sub(conn.LastActivity()) > p.config.MaxLifetime {
			p.removeUnlocked(conn.ID())
			continue
		}

		newIdle = append(newIdle, conn)
	}

	p.idle = newIdle
}

func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		Active: int(p.active.Load()),
		Idle:   len(p.idle),
		Total:  int(p.total.Load()),
	}
}

func (p *Pool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.closeOnce.Do(func() {
		close(p.closeCh)
	})

	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	for id := range p.conns {
		_ = p.removeUnlocked(id)
	}

	return nil
}

func (p *Pool) ForEach(fn func(*Connection) bool) {
	p.mu.RLock()
	conns := make([]*Connection, 0, len(p.conns))
	for _, conn := range p.conns {
		conns = append(conns, conn)
	}
	p.mu.RUnlock()

	for _, conn := range conns {
		if !fn(conn) {
			break
		}
	}
}

func (p *Pool) IsClosed() bool {
	return p.closed.Load()
}

type PoolStats struct {
	Active int
	Idle   int
	Total  int
}

type PoolManager struct {
	pool   *Pool
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewPoolManager(config *PoolConfig) (*PoolManager, error) {
	pool, err := NewPool(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PoolManager{
		pool:   pool,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (pm *PoolManager) Pool() *Pool {
	return pm.pool
}

func (pm *PoolManager) Close() error {
	pm.cancel()
	return pm.pool.Close()
}
