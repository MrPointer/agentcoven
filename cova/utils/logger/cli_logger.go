package logger

import (
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
)

// Styles for different types of messages using lipgloss.
var (
	DebugStyle   = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Bold(true)
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.BrightBlue).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightYellow).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.BrightRed).Bold(true)
)

// VerbosityLevel controls how much output the logger produces.
type VerbosityLevel int

// Verbosity levels from least to most verbose.
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
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Trace(format string, args ...any) {
	if l.verbosity >= ExtraVerbose {
		PrintStyled(l.output, DebugStyle, format, args...)
	}
}

// Debug logs a debug message with gray styling.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Debug(format string, args ...any) {
	if l.verbosity >= Verbose {
		PrintStyled(l.output, DebugStyle, format, args...)
	}
}

// Info logs an informational message with blue styling.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Info(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, InfoStyle, format, args...)
	}
}

// Success logs a success message with green styling.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Success(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, SuccessStyle, format, args...)
	}
}

// Warning logs a warning message with yellow styling.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Warning(format string, args ...any) {
	if l.verbosity >= Normal {
		PrintStyled(l.output, WarningStyle, format, args...)
	}
}

// Error logs an error message with red styling.
//
//nolint:goprintffuncname // Logger method names intentionally omit the 'f' suffix; they are semantic level names, not generic printf wrappers.
func (l *CliLogger) Error(format string, args ...any) {
	if l.verbosity >= Normal {
		if l.output == os.Stdout {
			PrintStyled(os.Stderr, ErrorStyle, format, args...)
		} else {
			PrintStyled(l.output, ErrorStyle, format, args...)
		}
	}
}

// Close cleans up terminal state, including cursor restoration.
func (l *CliLogger) Close() error {
	return nil
}

// PrintStyled is a helper function to print styled text to the specified writer.
//
//nolint:goprintffuncname // PrintStyled is a styled output helper, not a generic printf wrapper; the name is intentional.
func PrintStyled(writer io.Writer, style lipgloss.Style, format string, args ...any) {
	rendered := style.Render(fmt.Sprintf(format, args...))
	if file, ok := writer.(*os.File); ok {
		lipgloss.Fprintln(file, rendered)
	} else {
		// Non-file writers (e.g. test buffers) get raw ANSI output.
		fmt.Fprintln(writer, rendered)
	}
}
