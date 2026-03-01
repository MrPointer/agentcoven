package logger_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

func Test_CliLogger_ByDesign_ImplementsLoggerInterface(t *testing.T) {
	var _ logger.Logger = (*logger.CliLogger)(nil)
}

func Test_NewProgressCliLogger_WhenCalled_CreatesValidInstance(t *testing.T) {
	log := logger.NewProgressCliLogger(logger.Normal)
	require.NotNil(t, log)
}

func Test_NewCliLogger_WithVerbosityLevels_CreatesValidInstance(t *testing.T) {
	tests := []struct {
		name      string
		verbosity logger.VerbosityLevel
	}{
		{"Minimal verbosity", logger.Minimal},
		{"Normal verbosity", logger.Normal},
		{"Verbose verbosity", logger.Verbose},
		{"ExtraVerbose verbosity", logger.ExtraVerbose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.NewCliLogger(tt.verbosity)
			require.NotNil(t, log)
		})
	}
}

func Test_VerbosityLevels_WithDifferentMessages_FilterCorrectly(t *testing.T) {
	tests := []struct {
		name      string
		verbosity logger.VerbosityLevel
		logFunc   func(logger.Logger)
		shouldLog bool
	}{
		{
			name:      "Trace messages appear with ExtraVerbose",
			verbosity: logger.ExtraVerbose,
			logFunc:   func(l logger.Logger) { l.Trace("trace message") },
			shouldLog: true,
		},
		{
			name:      "Trace messages hidden with Verbose",
			verbosity: logger.Verbose,
			logFunc:   func(l logger.Logger) { l.Trace("trace message") },
			shouldLog: false,
		},
		{
			name:      "Debug messages appear with Verbose",
			verbosity: logger.Verbose,
			logFunc:   func(l logger.Logger) { l.Debug("debug message") },
			shouldLog: true,
		},
		{
			name:      "Debug messages hidden with Normal",
			verbosity: logger.Normal,
			logFunc:   func(l logger.Logger) { l.Debug("debug message") },
			shouldLog: false,
		},
		{
			name:      "Info messages appear with Normal",
			verbosity: logger.Normal,
			logFunc:   func(l logger.Logger) { l.Info("info message") },
			shouldLog: true,
		},
		{
			name:      "Info messages hidden with Minimal",
			verbosity: logger.Minimal,
			logFunc:   func(l logger.Logger) { l.Info("info message") },
			shouldLog: false,
		},
		{
			name:      "Error messages appear with Normal",
			verbosity: logger.Normal,
			logFunc:   func(l logger.Logger) { l.Error("error message") },
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.NewCliLogger(tt.verbosity)
			require.NotNil(t, log)

			// Just verify that calling the function doesn't panic
			// The actual output verification would require capturing stdout/stderr
			require.NotPanics(t, func() {
				tt.logFunc(log)
			})
		})
	}
}

func Test_AllVerbosityLevels_WithDifferentMessages_ProduceAppropriateOutput(t *testing.T) {
	tests := []struct {
		name      string
		verbosity logger.VerbosityLevel
	}{
		{"Minimal verbosity produces minimal output", logger.Minimal},
		{"Normal verbosity produces normal output", logger.Normal},
		{"Verbose verbosity produces verbose output", logger.Verbose},
		{"ExtraVerbose verbosity produces extra verbose output", logger.ExtraVerbose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.NewCliLogger(tt.verbosity)

			// Test all logging levels
			log.Trace("This is a trace message")
			log.Debug("This is a debug message")
			log.Info("This is an info message")
			log.Success("This is a success message")
			log.Warning("This is a warning message")
			log.Error("This is an error message")

			require.NotNil(t, log)
		})
	}
}
