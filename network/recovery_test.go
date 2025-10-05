package network

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBackoffConfig(t *testing.T) {
	config := DefaultBackoffConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 1*time.Second, config.InitialInterval)
	assert.Equal(t, 60*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 10, config.MaxRetries)
	assert.True(t, config.Jitter)
	assert.Equal(t, 0.2, config.JitterFactor)
}

func TestBackoffConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *BackoffConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: &BackoffConfig{
				InitialInterval: 1 * time.Second,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.2,
			},
			expectErr: false,
		},
		{
			name: "invalid initial interval",
			config: &BackoffConfig{
				InitialInterval: 0,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			expectErr: true,
		},
		{
			name: "invalid max interval",
			config: &BackoffConfig{
				InitialInterval: 10 * time.Second,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
			},
			expectErr: true,
		},
		{
			name: "invalid multiplier",
			config: &BackoffConfig{
				InitialInterval: 1 * time.Second,
				MaxInterval:     10 * time.Second,
				Multiplier:      0,
			},
			expectErr: true,
		},
		{
			name: "invalid jitter factor negative",
			config: &BackoffConfig{
				InitialInterval: 1 * time.Second,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    -0.1,
			},
			expectErr: true,
		},
		{
			name: "invalid jitter factor too large",
			config: &BackoffConfig{
				InitialInterval: 1 * time.Second,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    1.5,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewBackoff(t *testing.T) {
	backoff, err := NewBackoff(nil)
	assert.NoError(t, err)
	assert.NotNil(t, backoff)
}

func TestNewBackoffWithInvalidConfig(t *testing.T) {
	config := &BackoffConfig{
		InitialInterval: 0,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}
	backoff, err := NewBackoff(config)
	assert.Error(t, err)
	assert.Nil(t, backoff)
}

func TestBackoffNext(t *testing.T) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      3,
		Jitter:          false,
	}

	backoff, err := NewBackoff(config)
	require.NoError(t, err)

	interval1, ok := backoff.Next()
	assert.True(t, ok)
	assert.Equal(t, 100*time.Millisecond, interval1)

	interval2, ok := backoff.Next()
	assert.True(t, ok)
	assert.Equal(t, 200*time.Millisecond, interval2)

	interval3, ok := backoff.Next()
	assert.True(t, ok)
	assert.Equal(t, 400*time.Millisecond, interval3)

	_, ok = backoff.Next()
	assert.False(t, ok)
}

func TestBackoffNextWithJitter(t *testing.T) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      3,
		Jitter:          true,
		JitterFactor:    0.2,
	}

	backoff, err := NewBackoff(config)
	require.NoError(t, err)

	interval1, ok := backoff.Next()
	assert.True(t, ok)
	assert.True(t, interval1 >= 80*time.Millisecond && interval1 <= 120*time.Millisecond)
}

func TestBackoffNextMaxInterval(t *testing.T) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     300 * time.Millisecond,
		Multiplier:      2.0,
		MaxRetries:      10,
		Jitter:          false,
	}

	backoff, err := NewBackoff(config)
	require.NoError(t, err)

	backoff.Next()
	backoff.Next()
	interval3, ok := backoff.Next()
	assert.True(t, ok)
	assert.Equal(t, 300*time.Millisecond, interval3)
}

func TestBackoffReset(t *testing.T) {
	config := &BackoffConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      3,
		Jitter:          false,
	}

	backoff, err := NewBackoff(config)
	require.NoError(t, err)

	backoff.Next()
	backoff.Next()
	assert.Equal(t, 2, backoff.Attempt())

	backoff.Reset()
	assert.Equal(t, 0, backoff.Attempt())

	interval, ok := backoff.Next()
	assert.True(t, ok)
	assert.Equal(t, 100*time.Millisecond, interval)
}

func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()
	assert.NotNil(t, config)
	assert.NotNil(t, config.BackoffConfig)
	assert.True(t, config.EnableRecovery)
}

func TestNewRecovery(t *testing.T) {
	recovery, err := NewRecovery(nil)
	assert.NoError(t, err)
	assert.NotNil(t, recovery)
}

func TestNewRecoveryWithInvalidConfig(t *testing.T) {
	config := &RecoveryConfig{
		BackoffConfig: &BackoffConfig{
			InitialInterval: 0,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		},
		EnableRecovery: true,
	}
	recovery, err := NewRecovery(config)
	assert.Error(t, err)
	assert.Nil(t, recovery)
}

func TestRecoveryRetrySuccess(t *testing.T) {
	recovery, err := NewRecovery(nil)
	require.NoError(t, err)

	attempts := 0
	err = recovery.Retry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRecoveryRetryFailure(t *testing.T) {
	config := &RecoveryConfig{
		BackoffConfig: &BackoffConfig{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      2.0,
			MaxRetries:      3,
			Jitter:          false,
		},
		EnableRecovery: true,
	}

	recovery, err := NewRecovery(config)
	require.NoError(t, err)

	attempts := 0
	err = recovery.Retry(context.Background(), func() error {
		attempts++
		return errors.New("permanent error")
	})

	assert.Equal(t, ErrMaxRetriesExceeded, err)
	assert.Equal(t, 3, attempts)
}

func TestRecoveryRetryContextCanceled(t *testing.T) {
	recovery, err := NewRecovery(nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	err = recovery.Retry(ctx, func() error {
		attempts++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestRecoveryRetryDisabled(t *testing.T) {
	config := &RecoveryConfig{
		BackoffConfig:  DefaultBackoffConfig(),
		EnableRecovery: false,
	}

	recovery, err := NewRecovery(config)
	require.NoError(t, err)

	attempts := 0
	testErr := errors.New("error")
	err = recovery.Retry(context.Background(), func() error {
		attempts++
		return testErr
	})

	assert.Equal(t, testErr, err)
	assert.Equal(t, 1, attempts)
}

func TestRecoveryRetryWithHealthCheck(t *testing.T) {
	healthCheckCalled := false
	config := &RecoveryConfig{
		BackoffConfig: &BackoffConfig{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      2.0,
			MaxRetries:      5,
			Jitter:          false,
		},
		EnableRecovery: true,
		HealthCheckFunc: func(ctx context.Context) error {
			healthCheckCalled = true
			return nil
		},
	}

	recovery, err := NewRecovery(config)
	require.NoError(t, err)

	attempts := 0
	err = recovery.Retry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, healthCheckCalled)
}

func TestRecoveryReset(t *testing.T) {
	recovery, err := NewRecovery(nil)
	require.NoError(t, err)

	attempts := 0
	recovery.Retry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	})

	assert.True(t, recovery.Attempt() > 0)
	recovery.Reset()
	assert.Equal(t, 0, recovery.Attempt())
}

func TestNewReconnector(t *testing.T) {
	connectFn := func() (*Connection, error) {
		server, _ := net.Pipe()
		return NewConnection(server, "test", nil), nil
	}

	reconnector, err := NewReconnector(context.Background(), nil, connectFn)
	assert.NoError(t, err)
	assert.NotNil(t, reconnector)
	defer reconnector.Close()
}

func TestNewReconnectorNilConnectFn(t *testing.T) {
	reconnector, err := NewReconnector(context.Background(), nil, nil)
	assert.Error(t, err)
	assert.Nil(t, reconnector)
}

func TestReconnectorConnect(t *testing.T) {
	attempts := 0
	connectFn := func() (*Connection, error) {
		attempts++
		if attempts < 2 {
			return nil, errors.New("connection failed")
		}
		server, _ := net.Pipe()
		return NewConnection(server, "test", nil), nil
	}

	config := &RecoveryConfig{
		BackoffConfig: &BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			Multiplier:      2.0,
			MaxRetries:      5,
			Jitter:          false,
		},
		EnableRecovery: true,
	}

	reconnector, err := NewReconnector(context.Background(), config, connectFn)
	require.NoError(t, err)
	defer reconnector.Close()

	conn, err := reconnector.Connect()
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, 2, attempts)
}

func TestReconnectorConnectFailure(t *testing.T) {
	connectFn := func() (*Connection, error) {
		return nil, errors.New("connection failed")
	}

	config := &RecoveryConfig{
		BackoffConfig: &BackoffConfig{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      2.0,
			MaxRetries:      3,
			Jitter:          false,
		},
		EnableRecovery: true,
	}

	reconnector, err := NewReconnector(context.Background(), config, connectFn)
	require.NoError(t, err)
	defer reconnector.Close()

	conn, err := reconnector.Connect()
	assert.Equal(t, ErrMaxRetriesExceeded, err)
	assert.Nil(t, conn)
}

func TestReconnectorClose(t *testing.T) {
	connectFn := func() (*Connection, error) {
		server, _ := net.Pipe()
		return NewConnection(server, "test", nil), nil
	}

	reconnector, err := NewReconnector(context.Background(), nil, connectFn)
	require.NoError(t, err)

	reconnector.Close()
}
