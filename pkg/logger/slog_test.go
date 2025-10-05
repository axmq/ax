package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlogLogger(t *testing.T) {
	t.Run("creates logger with custom writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewSlogLogger(slog.LevelInfo, buf)

		require.NotNil(t, logger)
		require.NotNil(t, logger.logger)
	})

	t.Run("creates logger with default writer when nil", func(t *testing.T) {
		logger := NewSlogLogger(slog.LevelInfo, nil)

		require.NotNil(t, logger)
		require.NotNil(t, logger.logger)
	})
}

func TestSlogLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelInfo, buf)

	logger.Info("test message")
	output := buf.String()

	assert.Contains(t, output, "INF")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "2025-")
}

func TestSlogLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelWarn, buf)

	logger.Warn("warning message")
	output := buf.String()

	assert.Contains(t, output, "WRN")
	assert.Contains(t, output, "warning message")
}

func TestSlogLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelError, buf)

	logger.Error("error message")
	output := buf.String()

	assert.Contains(t, output, "ERR")
	assert.Contains(t, output, "error message")
}

func TestSlogLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelDebug, buf)

	logger.Debug("debug message")
	output := buf.String()

	assert.Contains(t, output, "DBG")
	assert.Contains(t, output, "debug message")
}

func TestSlogLogger_WithArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelInfo, buf)

	logger.Info("test message", "key1", "value1", "key2", 123)
	output := buf.String()

	assert.Contains(t, output, "key1=value1")
	assert.Contains(t, output, "key2=123")
}

func TestSlogLogger_MultipleArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelInfo, buf)

	logger.Info("Sensor initialized",
		"sensor", "simulated-sensor",
		"id", "d364a655-f226-4682-9252-f914c230081e")
	output := buf.String()

	assert.Contains(t, output, "Sensor initialized")
	assert.Contains(t, output, "sensor=simulated-sensor")
	assert.Contains(t, output, "id=d364a655-f226-4682-9252-f914c230081e")
}

func TestSlogLogger_OddNumberOfArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelInfo, buf)

	logger.Info("test message", "key1", "value1", "key2")
	output := buf.String()

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key1=value1")
}

func TestSlogLogger_MinLevel(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  slog.Level
		logLevel  string
		logFunc   func(*SlogLogger)
		shouldLog bool
	}{
		{
			name:     "Debug not logged when min level is Info",
			minLevel: slog.LevelInfo,
			logLevel: "DBG",
			logFunc: func(l *SlogLogger) {
				l.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name:     "Info logged when min level is Info",
			minLevel: slog.LevelInfo,
			logLevel: "INF",
			logFunc: func(l *SlogLogger) {
				l.Info("info message")
			},
			shouldLog: true,
		},
		{
			name:     "Warn logged when min level is Info",
			minLevel: slog.LevelInfo,
			logLevel: "WRN",
			logFunc: func(l *SlogLogger) {
				l.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name:     "Error logged when min level is Info",
			minLevel: slog.LevelInfo,
			logLevel: "ERR",
			logFunc: func(l *SlogLogger) {
				l.Error("error message")
			},
			shouldLog: true,
		},
		{
			name:     "Info not logged when min level is Error",
			minLevel: slog.LevelError,
			logLevel: "INF",
			logFunc: func(l *SlogLogger) {
				l.Info("info message")
			},
			shouldLog: false,
		},
		{
			name:     "Debug logged when min level is Debug",
			minLevel: slog.LevelDebug,
			logLevel: "DBG",
			logFunc: func(l *SlogLogger) {
				l.Debug("debug message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewSlogLogger(tt.minLevel, buf)

			tt.logFunc(logger)
			output := buf.String()

			if tt.shouldLog {
				assert.NotEmpty(t, output)
				assert.Contains(t, output, tt.logLevel)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestColoredHandler_Enabled(t *testing.T) {
	handler := &ColoredHandler{
		minLevel: slog.LevelInfo,
	}

	tests := []struct {
		name    string
		level   slog.Level
		enabled bool
	}{
		{"Debug below Info", slog.LevelDebug, false},
		{"Info equals Info", slog.LevelInfo, true},
		{"Warn above Info", slog.LevelWarn, true},
		{"Error above Info", slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled := handler.Enabled(context.Background(), tt.level)
			assert.Equal(t, tt.enabled, enabled)
		})
	}
}

func TestColoredHandler_WithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := &ColoredHandler{
		writer:   buf,
		minLevel: slog.LevelInfo,
	}

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	newHandler := handler.WithAttrs(attrs)
	coloredHandler, ok := newHandler.(*ColoredHandler)
	require.True(t, ok)
	assert.Len(t, coloredHandler.attrs, 2)
}

func TestColoredHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := &ColoredHandler{
		writer:   buf,
		minLevel: slog.LevelInfo,
	}

	newHandler := handler.WithGroup("testgroup")
	coloredHandler, ok := newHandler.(*ColoredHandler)
	require.True(t, ok)
	require.Len(t, coloredHandler.groups, 1)
	assert.Equal(t, "testgroup", coloredHandler.groups[0])
}

func TestColoredHandler_coloredLevel(t *testing.T) {
	handler := &ColoredHandler{}

	tests := []struct {
		name     string
		level    slog.Level
		expected string
	}{
		{"Debug", slog.LevelDebug, colorGray + "DBG" + colorReset},
		{"Info", slog.LevelInfo, colorBlue + "INF" + colorReset},
		{"Warn", slog.LevelWarn, colorYellow + "WRN" + colorReset},
		{"Error", slog.LevelError, colorRed + "ERR" + colorReset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.coloredLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected int
	}{
		{
			name:     "empty args",
			args:     []interface{}{},
			expected: 0,
		},
		{
			name:     "single key-value pair",
			args:     []interface{}{"key", "value"},
			expected: 1,
		},
		{
			name:     "multiple key-value pairs",
			args:     []interface{}{"key1", "value1", "key2", "value2"},
			expected: 2,
		},
		{
			name:     "odd number of args",
			args:     []interface{}{"key1", "value1", "key2"},
			expected: 1,
		},
		{
			name:     "non-string key",
			args:     []interface{}{123, "value"},
			expected: 0,
		},
		{
			name:     "mixed types",
			args:     []interface{}{"key1", 42, "key2", true, "key3", 3.14},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatArgs(tt.args...)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestSlogLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = (*SlogLogger)(nil)
}

func TestLogFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewSlogLogger(slog.LevelInfo, buf)

	logger.Info("Sensor initialized", "sensor", "simulated-sensor", "id", "test-id-123")
	output := buf.String()

	parts := strings.Fields(output)
	require.GreaterOrEqual(t, len(parts), 4)

	datePart := parts[0]
	assert.Contains(t, datePart, "-")

	timePart := parts[1]
	assert.Contains(t, timePart, ":")

	assert.Contains(t, output, "INF")
	assert.Contains(t, output, "Sensor initialized")
}
