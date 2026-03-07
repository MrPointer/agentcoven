// Package logger provides a leveled logging interface and implementations for CLI output.
package logger

import "io"

// Logger defines a minimal logging interface that our installer utilities need.
type Logger interface {
	io.Closer

	// Trace logs a trace message
	Trace(format string, args ...any)
	// Debug logs a debug message
	Debug(format string, args ...any)
	// Info logs an informational message
	Info(format string, args ...any)
	// Success logs a success message
	Success(format string, args ...any)
	// Warning logs a warning message
	Warning(format string, args ...any)
	// Error logs an error message
	Error(format string, args ...any)
}

// NoopLogger implements Logger but does nothing.
type NoopLogger struct{}

var _ Logger = (*NoopLogger)(nil)

// Trace is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Trace(format string, args ...any) {}

// Debug is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Debug(format string, args ...any) {}

// Info is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Info(format string, args ...any) {}

// Success is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Success(format string, args ...any) {}

// Warning is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Warning(format string, args ...any) {}

// Error is a no-op implementation.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l NoopLogger) Error(format string, args ...any) {}

// Close is a no-op implementation.
func (l NoopLogger) Close() error { return nil }

// DefaultLogger is the default logger used if none is provided.
var DefaultLogger Logger = NoopLogger{}
