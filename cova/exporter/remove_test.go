package exporter

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

func TestRemove_RemovingExportersShouldLogPerNameResults(t *testing.T) {
	var (
		infoCalls []string
		warnCalls []string
	)

	mockLogger := &logger.MoqLogger{
		InfoFunc: func(format string, args ...any) {
			infoCalls = append(infoCalls, format)
		},
		WarningFunc: func(format string, args ...any) {
			warnCalls = append(warnCalls, format)
		},
	}

	mockEnv := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			if key == "XDG_CONFIG_HOME" {
				return "/tmp/testconfig"
			}

			return ""
		},
	}

	mockUser := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "/home/test", nil
		},
	}

	configYAML, err := yaml.Marshal(map[string]any{
		"agents": []string{"configured-exporter"},
	})
	require.NoError(t, err)

	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return configYAML, nil
		},
		CreateDirectoryFunc: func(path string) error {
			return nil
		},
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) {
			return dir + "/config-tmp.yaml.tmp", nil
		},
		WriteFileFunc: func(path string, reader io.Reader) (int64, error) {
			return 0, nil
		},
		RenameFunc: func(oldPath, newPath string) error {
			return nil
		},
	}

	mockLocker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, path string, fn func() error) error {
			return fn()
		},
	}

	deps := Deps{
		Logger:      mockLogger,
		FileSystem:  mockFS,
		Locker:      mockLocker,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	err = Remove(t.Context(), deps, []string{"configured-exporter", "missing-exporter"})

	require.NoError(t, err)
	require.Len(t, infoCalls, 1)
	require.Len(t, warnCalls, 1)
}

func TestRemove_RemovingExportersShouldWarnWhenExporterIsNotConfigured(t *testing.T) {
	var warnCalls []string

	mockLogger := &logger.MoqLogger{
		InfoFunc: func(format string, args ...any) {},
		WarningFunc: func(format string, args ...any) {
			warnCalls = append(warnCalls, format)
		},
	}

	mockEnv := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			if key == "XDG_CONFIG_HOME" {
				return "/tmp/testconfig"
			}

			return ""
		},
	}

	mockUser := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "/home/test", nil
		},
	}

	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
		CreateDirectoryFunc: func(path string) error {
			return nil
		},
	}

	mockLocker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, path string, fn func() error) error {
			return fn()
		},
	}

	deps := Deps{
		Logger:      mockLogger,
		FileSystem:  mockFS,
		Locker:      mockLocker,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	err := Remove(t.Context(), deps, []string{"missing-exporter"})

	require.NoError(t, err)
	require.Len(t, warnCalls, 1)
	require.Contains(t, warnCalls[0], "not configured")
}

func TestRemove_RemovingExportersShouldReturnErrorWhenConfigPathResolutionFails(t *testing.T) {
	mockEnv := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			return ""
		},
	}

	mockUser := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "", errors.New("cannot determine home directory")
		},
	}

	deps := Deps{
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	err := Remove(t.Context(), deps, []string{"some-exporter"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving config path")
}

func TestRemove_RemovingExportersShouldReturnErrorWhenRemoveAgentsFails(t *testing.T) {
	mockEnv := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			if key == "XDG_CONFIG_HOME" {
				return "/tmp/testconfig"
			}

			return ""
		},
	}

	mockUser := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "/home/test", nil
		},
	}

	mockFS := &utils.MoqFileSystem{
		CreateDirectoryFunc: func(path string) error {
			return errors.New("permission denied")
		},
	}

	mockLocker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, path string, fn func() error) error {
			return fn()
		},
	}

	deps := Deps{
		FileSystem:  mockFS,
		Locker:      mockLocker,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	err := Remove(t.Context(), deps, []string{"some-exporter"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "removing exporters")
}
