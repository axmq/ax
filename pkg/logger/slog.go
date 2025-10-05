package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// SlogLogger wraps slog.Logger to implement the Logger interface
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with colored output and configurable minimum level
func NewSlogLogger(minLevel slog.Level, writer io.Writer) *SlogLogger {
	if writer == nil {
		writer = os.Stdout
	}

	handler := &ColoredHandler{
		writer:   writer,
		minLevel: minLevel,
	}

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// ColoredHandler implements slog.Handler with colored level output
type ColoredHandler struct {
	writer   io.Writer
	minLevel slog.Level
	attrs    []slog.Attr
	groups   []string
}

func (h *ColoredHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

func (h *ColoredHandler) Handle(_ context.Context, r slog.Record) error {
	// Format timestamp
	timestamp := r.Time.Format("2006-01-02 15:04:05")

	// Get colored level string
	levelStr := h.coloredLevel(r.Level)

	// Build the log message
	buf := fmt.Sprintf("%s %s %s", timestamp, levelStr, r.Message)

	// Add attributes from handler
	for _, attr := range h.attrs {
		buf += fmt.Sprintf(" %s=%v", attr.Key, attr.Value)
	}

	// Add attributes from record
	r.Attrs(func(a slog.Attr) bool {
		buf += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	buf += "\n"

	_, err := h.writer.Write([]byte(buf))
	return err
}

func (h *ColoredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &ColoredHandler{
		writer:   h.writer,
		minLevel: h.minLevel,
		attrs:    newAttrs,
		groups:   h.groups,
	}
}

func (h *ColoredHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &ColoredHandler{
		writer:   h.writer,
		minLevel: h.minLevel,
		attrs:    h.attrs,
		groups:   newGroups,
	}
}

func (h *ColoredHandler) coloredLevel(level slog.Level) string {
	var color string
	var levelStr string

	switch level {
	case slog.LevelDebug:
		color = colorGray
		levelStr = "DBG"
	case slog.LevelInfo:
		color = colorBlue
		levelStr = "INF"
	case slog.LevelWarn:
		color = colorYellow
		levelStr = "WRN"
	case slog.LevelError:
		color = colorRed
		levelStr = "ERR"
	default:
		color = colorReset
		levelStr = level.String()
	}

	return color + levelStr + colorReset
}

// Info logs an informational message
func (l *SlogLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, formatArgs(args...)...)
}

// Warn logs a warning message
func (l *SlogLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, formatArgs(args...)...)
}

// Error logs an error message
func (l *SlogLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, formatArgs(args...)...)
}

// Debug logs a debug message
func (l *SlogLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, formatArgs(args...)...)
}

// formatArgs converts variadic interface{} args to slog.Attr
func formatArgs(args ...interface{}) []any {
	if len(args) == 0 {
		return nil
	}

	// Convert pairs of key-value to slog attributes
	attrs := make([]any, 0, len(args))
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				attrs = append(attrs, slog.Any(key, args[i+1]))
			}
		}
	}
	return attrs
}
