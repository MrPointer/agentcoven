package exporter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

func TestList_ListingExportersShouldGroupByTypeWithConfiguredStatus(t *testing.T) {
	mockDispatcher := &MoqDispatcher{
		ListAvailableFunc: func(ctx context.Context) ([]InfoResponse, error) {
			return []InfoResponse{
				{Name: "claude-code", Description: "Places skills and agents for Claude Code", BuiltIn: true},
				{Name: "custom-agent", Description: "Exports blocks for custom agent", BuiltIn: false},
			}, nil
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
		"agents": []string{"claude-code"},
	})
	require.NoError(t, err)

	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return configYAML, nil
		},
	}

	deps := Deps{
		Dispatcher:  mockDispatcher,
		FileSystem:  mockFS,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	result, err := List(t.Context(), deps)

	require.NoError(t, err)
	require.Len(t, result.BuiltIn, 1)
	require.Equal(t, "claude-code", result.BuiltIn[0].Name)
	require.True(t, result.BuiltIn[0].Configured)
	require.True(t, result.BuiltIn[0].BuiltIn)
	require.Len(t, result.External, 1)
	require.Equal(t, "custom-agent", result.External[0].Name)
	require.False(t, result.External[0].Configured)
	require.False(t, result.External[0].BuiltIn)
}

func TestList_ListingExportersShouldReturnEmptyResultWhenNoneAvailable(t *testing.T) {
	mockDispatcher := &MoqDispatcher{
		ListAvailableFunc: func(ctx context.Context) ([]InfoResponse, error) {
			return []InfoResponse{}, nil
		},
	}

	deps := Deps{
		Dispatcher: mockDispatcher,
	}

	result, err := List(t.Context(), deps)

	require.NoError(t, err)
	require.Empty(t, result.BuiltIn)
	require.Empty(t, result.External)
}

func TestList_ListingExportersShouldReturnErrorWhenListAvailableFails(t *testing.T) {
	mockDispatcher := &MoqDispatcher{
		ListAvailableFunc: func(ctx context.Context) ([]InfoResponse, error) {
			return nil, errors.New("dispatcher failure")
		},
	}

	deps := Deps{
		Dispatcher: mockDispatcher,
	}

	_, err := List(t.Context(), deps)

	require.Error(t, err)
	require.Contains(t, err.Error(), "listing available exporters")
}

func TestList_ListingExportersShouldReturnErrorWhenConfigPathResolutionFails(t *testing.T) {
	mockDispatcher := &MoqDispatcher{
		ListAvailableFunc: func(ctx context.Context) ([]InfoResponse, error) {
			return []InfoResponse{
				{Name: "claude-code", Description: "Some description"},
			}, nil
		},
	}

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
		Dispatcher:  mockDispatcher,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	_, err := List(t.Context(), deps)

	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving config path")
}

func TestList_ListingExportersShouldReturnErrorWhenConfigLoadFails(t *testing.T) {
	mockDispatcher := &MoqDispatcher{
		ListAvailableFunc: func(ctx context.Context) ([]InfoResponse, error) {
			return []InfoResponse{
				{Name: "claude-code", Description: "Some description"},
			}, nil
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
			return false, errors.New("filesystem error")
		},
	}

	deps := Deps{
		Dispatcher:  mockDispatcher,
		FileSystem:  mockFS,
		EnvManager:  mockEnv,
		UserManager: mockUser,
	}

	_, err := List(t.Context(), deps)

	require.Error(t, err)
	require.Contains(t, err.Error(), "loading config")
}
