package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Styles for different types of messages using lipgloss.
var (
	DebugStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7f8c8d")).Bold(true) // Gray
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db")).Bold(true) // Blue
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71")).Bold(true) // Green
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f39c12")).Bold(true) // Yellow/Orange
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e74c3c")).Bold(true) // Red
)

// Backward compatibility aliases for internal use.
var (
	debugStyle   = DebugStyle
	infoStyle    = InfoStyle
	successStyle = SuccessStyle
	warningStyle = WarningStyle
	errorStyle   = ErrorStyle
)

type VerbosityLevel int

const (
	Minimal VerbosityLevel = iota
	Normal
	Verbose
	ExtraVerbose
)

// CliLogger implements the Logger interface using lipgloss styling.
type CliLogger struct {
	output    io.Writer
	verbosity VerbosityLevel
}

var _ Logger = (*CliLogger)(nil)

// NewCliLogger creates a new CLI logger that uses lipgloss styling.
func NewCliLogger(verbosity VerbosityLevel) *CliLogger {
	return &CliLogger{
		verbosity: verbosity,
		output:    os.Stdout,
	}
}

// NewProgressCliLogger creates a new CLI logger with progress indicator support.
func NewProgressCliLogger(verbosity VerbosityLevel) *CliLogger {
	return &CliLogger{
		verbosity: verbosity,
		output:    os.Stdout,
	}
}

// NewCliLoggerWithProgress creates a new CLI logger with a custom progress display.
func NewCliLoggerWithProgress(verbosity VerbosityLevel) *CliLogger {
	return &CliLogger{
		verbosity: verbosity,
		output:    os.Stdout,
	}
}

// NewCliLoggerWithOutput creates a new CLI logger with a custom output writer.
func NewCliLoggerWithOutput(verbosity VerbosityLevel, output io.Writer, withProgress bool) *CliLogger {
	return &CliLogger{
		verbosity: verbosity,
		output:    output,
	}
}

// Trace logs a trace message with gray styling.
func (l *CliLogger) Trace(format string, args ...any) {
	if l.verbosity >= ExtraVerbose {
		PrintStyled(l.output, debugStyle, format, args...)
	}
}

// Debug logs a debug message with gray styling.
func (l *CliLogger) Debug(format string, args ...any) {
	if l.verbosity >= Verbose {
		PrintStyled(l.output, debugStyle, format, args...)
	}
}

// Info logs an informational message with blue styling.
func (l *CliLogger) Info(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, infoStyle, format, args...)
	}
}

// Success logs a success message with green styling.
func (l *CliLogger) Success(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, successStyle, format, args...)
	}
}

// Warning logs a warning message with yellow styling.
func (l *CliLogger) Warning(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, warningStyle, format, args...)
	}
}

// Error logs an error message with red styling.
func (l *CliLogger) Error(format string, args ...any) {
	if l.verbosity >= Normal {
		if l.output == os.Stdout {
			PrintStyled(os.Stderr, errorStyle, format, args...)
		} else {
			PrintStyled(l.output, errorStyle, format, args...)
		}
	}
}

// Cleanup ensures proper cleanup of terminal state, including cursor restoration.
func (l *CliLogger) Close() error {
	return nil
}

// PrintStyled is a helper function to print styled text to the specified writer.
func PrintStyled(writer io.Writer, style lipgloss.Style, format string, args ...any) {
	if file, ok := writer.(*os.File); ok {
		fmt.Fprintln(file, style.Render(fmt.Sprintf(format, args...)))
	} else {
		fmt.Fprintln(writer, style.Render(fmt.Sprintf(format, args...)))
	}
}
