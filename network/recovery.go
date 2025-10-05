package network

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxRetries      int
	Jitter          bool
	JitterFactor    float64
}

func DefaultBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		InitialInterval: 1 * time.Second,
		MaxInterval:     60 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      10,
		Jitter:          true,
		JitterFactor:    0.2,
	}
}

func (bc *BackoffConfig) Validate() error {
	if bc.InitialInterval <= 0 {
		return ErrInvalidBackoffConfig
	}
	if bc.MaxInterval < bc.InitialInterval {
		return ErrInvalidBackoffConfig
	}
	if bc.Multiplier <= 0 {
		return ErrInvalidBackoffConfig
	}
	if bc.JitterFactor < 0 || bc.JitterFactor > 1 {
		return ErrInvalidBackoffConfig
	}
	return nil
}

type Backoff struct {
	config  *BackoffConfig
	attempt int
}

func NewBackoff(config *BackoffConfig) (*Backoff, error) {
	if config == nil {
		config = DefaultBackoffConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Backoff{
		config: config,
	}, nil
}

func (b *Backoff) Next() (time.Duration, bool) {
	if b.config.MaxRetries > 0 && b.attempt >= b.config.MaxRetries {
		return 0, false
	}

	interval := b.calculate()
	b.attempt++

	return interval, true
}

func (b *Backoff) calculate() time.Duration {
	interval := float64(b.config.InitialInterval) * math.Pow(b.config.Multiplier, float64(b.attempt))

	if interval > float64(b.config.MaxInterval) {
		interval = float64(b.config.MaxInterval)
	}

	if b.config.Jitter {
		jitter := interval * b.config.JitterFactor
		interval = interval - jitter + (rand.Float64() * 2 * jitter)
	}

	return time.Duration(interval)
}

func (b *Backoff) Reset() {
	b.attempt = 0
}

func (b *Backoff) Attempt() int {
	return b.attempt
}

type RecoveryConfig struct {
	BackoffConfig   *BackoffConfig
	EnableRecovery  bool
	HealthCheckFunc func(context.Context) error
}

func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		BackoffConfig:  DefaultBackoffConfig(),
		EnableRecovery: true,
	}
}

type Recovery struct {
	config  *RecoveryConfig
	backoff *Backoff
}

func NewRecovery(config *RecoveryConfig) (*Recovery, error) {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	backoff, err := NewBackoff(config.BackoffConfig)
	if err != nil {
		return nil, err
	}

	return &Recovery{
		config:  config,
		backoff: backoff,
	}, nil
}

func (r *Recovery) Retry(ctx context.Context, fn func() error) error {
	if !r.config.EnableRecovery {
		return fn()
	}

	r.backoff.Reset()

	for {
		// Check if we've exceeded max retries before attempting
		if r.config.BackoffConfig.MaxRetries > 0 && r.backoff.Attempt() >= r.config.BackoffConfig.MaxRetries {
			return ErrMaxRetriesExceeded
		}

		err := fn()
		if err == nil {
			return nil
		}

		// Increment attempt counter
		r.backoff.attempt++

		// Check again after the attempt
		if r.config.BackoffConfig.MaxRetries > 0 && r.backoff.Attempt() >= r.config.BackoffConfig.MaxRetries {
			return ErrMaxRetriesExceeded
		}

		// Calculate backoff interval
		interval := r.backoff.calculate()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		if r.config.HealthCheckFunc != nil {
			if err := r.config.HealthCheckFunc(ctx); err != nil {
				return err
			}
		}
	}
}

func (r *Recovery) Reset() {
	r.backoff.Reset()
}

func (r *Recovery) Attempt() int {
	return r.backoff.Attempt()
}

type Reconnector struct {
	config    *RecoveryConfig
	recovery  *Recovery
	connectFn func() (*Connection, error)

	ctx    context.Context
	cancel context.CancelFunc
}

func NewReconnector(ctx context.Context, config *RecoveryConfig, connectFn func() (*Connection, error)) (*Reconnector, error) {
	if connectFn == nil {
		return nil, ErrInvalidBackoffConfig
	}

	recovery, err := NewRecovery(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	return &Reconnector{
		config:    config,
		recovery:  recovery,
		connectFn: connectFn,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

func (r *Reconnector) Connect() (*Connection, error) {
	var conn *Connection

	err := r.recovery.Retry(r.ctx, func() error {
		var err error
		conn, err = r.connectFn()
		return err
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (r *Reconnector) Close() {
	r.cancel()
}
