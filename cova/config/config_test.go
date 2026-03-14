package config_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// --- DefaultPath tests ---

func TestDefaultPath_ResolvingPathShouldUseXDGConfigHomeWhenSet(t *testing.T) {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			if key == "XDG_CONFIG_HOME" {
				return "/custom/config"
			}

			return ""
		},
	}
	userMgr := &osmanager.MoqUserManager{}

	path, err := config.DefaultPath(envMgr, userMgr)

	require.NoError(t, err)
	require.Equal(t, "/custom/config/cova/config.yaml", path)
}

func TestDefaultPath_ResolvingPathShouldFallBackToHomeDirWhenXDGIsEmpty(t *testing.T) {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(_ string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/testuser", nil },
	}

	path, err := config.DefaultPath(envMgr, userMgr)

	require.NoError(t, err)
	require.Equal(t, "/home/testuser/.config/cova/config.yaml", path)
}

func TestDefaultPath_ResolvingPathShouldReturnErrorWhenHomeDirFails(t *testing.T) {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(_ string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "", errors.New("no home dir") },
	}

	_, err := config.DefaultPath(envMgr, userMgr)

	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving home directory")
}

// --- Load tests ---

func TestLoad_LoadingConfigShouldReturnEmptyConfigWhenFileDoesNotExist(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(_ string) (bool, error) { return false, nil },
	}

	cfg, err := config.Load(fs, "/some/config.yaml")

	require.NoError(t, err)
	require.Empty(t, cfg.Subscriptions)
}

func TestLoad_LoadingConfigShouldParseSubscriptions(t *testing.T) {
	yamlData := []byte(`subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
    path: covens/platform
    ref: main
`)
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return yamlData, nil },
	}

	cfg, err := config.Load(fs, "/some/config.yaml")

	require.NoError(t, err)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-platform", cfg.Subscriptions[0].Name)
	require.Equal(t, "github.com/acme/coven-blocks", cfg.Subscriptions[0].Repo)
	require.Equal(t, "covens/platform", cfg.Subscriptions[0].Path)
	require.Equal(t, "main", cfg.Subscriptions[0].Ref)
}

func TestLoad_LoadingConfigShouldReturnErrorWhenPathCheckFails(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(_ string) (bool, error) { return false, errors.New("permission denied") },
	}

	_, err := config.Load(fs, "/some/config.yaml")

	require.Error(t, err)
	require.Contains(t, err.Error(), "checking config file")
}

func TestLoad_LoadingConfigShouldReturnErrorWhenReadFails(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return nil, errors.New("read error") },
	}

	_, err := config.Load(fs, "/some/config.yaml")

	require.Error(t, err)
	require.Contains(t, err.Error(), "reading config file")
}

func TestLoad_LoadingConfigShouldReturnErrorWhenYAMLIsInvalid(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return []byte("subscriptions:\n  - [invalid"), nil },
	}

	_, err := config.Load(fs, "/some/config.yaml")

	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing config YAML")
}

func TestLoad_LoadingConfigShouldParseFrameworks(t *testing.T) {
	yamlData := []byte(`frameworks:
  - claude-code
  - openai-gpt
`)
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return yamlData, nil },
	}

	cfg, err := config.Load(fs, "/some/config.yaml")

	require.NoError(t, err)
	require.Len(t, cfg.Frameworks, 2)
	require.Equal(t, "claude-code", cfg.Frameworks[0])
	require.Equal(t, "openai-gpt", cfg.Frameworks[1])
}

func TestLoad_LoadingConfigShouldReturnEmptyFrameworksWhenOmitted(t *testing.T) {
	yamlData := []byte(`subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
`)
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return yamlData, nil },
	}

	cfg, err := config.Load(fs, "/some/config.yaml")

	require.NoError(t, err)
	require.Empty(t, cfg.Frameworks)
}

// --- Save tests ---

func TestSave_SavingConfigShouldWriteAtomically(t *testing.T) {
	var writtenData []byte

	var renamedFrom, renamedTo string

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc: func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) {
			return dir + "/config-123.yaml.tmp", nil
		},
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			writtenData = data

			return int64(len(data)), nil
		},
		RenameFunc: func(oldPath, newPath string) error {
			renamedFrom = oldPath
			renamedTo = newPath

			return nil
		},
	}

	cfg := config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-platform", Repo: "github.com/acme/blocks"},
		},
	}

	err := config.Save(fs, "/config/cova/config.yaml", cfg)

	require.NoError(t, err)
	require.Contains(t, string(writtenData), "acme-platform")
	require.Equal(t, "/config/cova/config-123.yaml.tmp", renamedFrom)
	require.Equal(t, "/config/cova/config.yaml", renamedTo)
}

func TestSave_SavingConfigShouldCleanUpTempFileWhenWriteFails(t *testing.T) {
	var removedPath string

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc: func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) {
			return dir + "/config-123.yaml.tmp", nil
		},
		WriteFileFunc: func(_ string, _ io.Reader) (int64, error) {
			return 0, errors.New("disk full")
		},
		RemovePathFunc: func(path string) error {
			removedPath = path

			return nil
		},
	}

	err := config.Save(fs, "/config/cova/config.yaml", config.Config{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "writing temp config file")
	require.Equal(t, "/config/cova/config-123.yaml.tmp", removedPath)
}

func TestSave_SavingConfigShouldCleanUpTempFileWhenRenameFails(t *testing.T) {
	var removedPath string

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc:           func(_ string, _ io.Reader) (int64, error) { return 10, nil },
		RenameFunc:              func(_, _ string) error { return errors.New("rename failed") },
		RemovePathFunc: func(path string) error {
			removedPath = path

			return nil
		},
	}

	err := config.Save(fs, "/config/cova/config.yaml", config.Config{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "renaming temp config file")
	require.Equal(t, "/config/cova/tmp.yaml", removedPath)
}

// --- Save round-trip test ---

func Test_SavingAndLoadingShouldPreserveSubscriptions(t *testing.T) {
	var stored []byte

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			stored = data

			return int64(len(data)), nil
		},
		RenameFunc:     func(_, _ string) error { return nil },
		PathExistsFunc: func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) {
			return stored, nil
		},
	}

	original := config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-platform", Repo: "github.com/acme/blocks", Path: "covens/platform", Ref: "v1.0"},
			{Name: "other-tools", Repo: "github.com/other/tools"},
		},
	}

	err := config.Save(fs, "/cfg/config.yaml", original)
	require.NoError(t, err)

	loaded, err := config.Load(fs, "/cfg/config.yaml")

	require.NoError(t, err)
	require.Equal(t, original, loaded)
}

func Test_SavingAndLoadingShouldPreserveFrameworks(t *testing.T) {
	var stored []byte

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			stored = data

			return int64(len(data)), nil
		},
		RenameFunc:     func(_, _ string) error { return nil },
		PathExistsFunc: func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) {
			return stored, nil
		},
	}

	original := config.Config{
		Frameworks: []string{"claude-code", "openai-gpt"},
	}

	err := config.Save(fs, "/cfg/config.yaml", original)
	require.NoError(t, err)

	loaded, err := config.Load(fs, "/cfg/config.yaml")

	require.NoError(t, err)
	require.Equal(t, original, loaded)
}

func Test_SavingAndLoadingShouldPreserveSubscriptionsAndFrameworks(t *testing.T) {
	var stored []byte

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			stored = data

			return int64(len(data)), nil
		},
		RenameFunc:     func(_, _ string) error { return nil },
		PathExistsFunc: func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) {
			return stored, nil
		},
	}

	original := config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-platform", Repo: "github.com/acme/blocks"},
		},
		Frameworks: []string{"claude-code"},
	}

	err := config.Save(fs, "/cfg/config.yaml", original)
	require.NoError(t, err)

	loaded, err := config.Load(fs, "/cfg/config.yaml")

	require.NoError(t, err)
	require.Equal(t, original, loaded)
}

// --- UpsertSubscription tests ---

func TestUpsertSubscription_UpsertingShouldAddNewSubscription(t *testing.T) {
	var savedCfg config.Config

	fs := &utils.MoqFileSystem{
		PathExistsFunc:          func(_ string) (bool, error) { return false, nil },
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			if unmarshalErr := yaml.Unmarshal(data, &savedCfg); unmarshalErr != nil {
				return 0, unmarshalErr
			}

			return int64(len(data)), nil
		},
		RenameFunc: func(_, _ string) error { return nil },
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(_ context.Context, _ string, fn func() error) error { return fn() },
	}

	sub := config.Subscription{Name: "acme-platform", Repo: "github.com/acme/blocks"}
	result, err := config.UpsertSubscription(t.Context(), fs, locker, "/cfg/config.yaml", sub)

	require.NoError(t, err)
	require.Equal(t, config.UpsertAdded, result)
	require.Len(t, savedCfg.Subscriptions, 1)
	require.Equal(t, "acme-platform", savedCfg.Subscriptions[0].Name)
}

func TestUpsertSubscription_UpsertingShouldUpdateExistingSubscription(t *testing.T) {
	existingYAML := []byte(`subscriptions:
  - name: acme-platform
    repo: github.com/acme/old-blocks
`)

	var savedCfg config.Config

	fs := &utils.MoqFileSystem{
		PathExistsFunc:          func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc:    func(_ string) ([]byte, error) { return existingYAML, nil },
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			if unmarshalErr := yaml.Unmarshal(data, &savedCfg); unmarshalErr != nil {
				return 0, unmarshalErr
			}

			return int64(len(data)), nil
		},
		RenameFunc: func(_, _ string) error { return nil },
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(_ context.Context, _ string, fn func() error) error { return fn() },
	}

	sub := config.Subscription{Name: "acme-platform", Repo: "github.com/acme/new-blocks"}
	result, err := config.UpsertSubscription(t.Context(), fs, locker, "/cfg/config.yaml", sub)

	require.NoError(t, err)
	require.Equal(t, config.UpsertUpdated, result)
	require.Len(t, savedCfg.Subscriptions, 1)
	require.Equal(t, "github.com/acme/new-blocks", savedCfg.Subscriptions[0].Repo)
}

func TestUpsertSubscription_UpsertingShouldNoOpWhenSubscriptionIsIdentical(t *testing.T) {
	existingYAML := []byte(`subscriptions:
  - name: acme-platform
    repo: github.com/acme/blocks
`)
	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(_ string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(_ string) ([]byte, error) { return existingYAML, nil },
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(_ context.Context, _ string, fn func() error) error { return fn() },
	}

	sub := config.Subscription{Name: "acme-platform", Repo: "github.com/acme/blocks"}
	result, err := config.UpsertSubscription(t.Context(), fs, locker, "/cfg/config.yaml", sub)

	require.NoError(t, err)
	require.Equal(t, config.UpsertNoOp, result)
}

func TestUpsertSubscription_UpsertingShouldUseLockFileSuffix(t *testing.T) {
	var lockedPath string

	fs := &utils.MoqFileSystem{
		PathExistsFunc:          func(_ string) (bool, error) { return false, nil },
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc:           func(_ string, _ io.Reader) (int64, error) { return 10, nil },
		RenameFunc:              func(_, _ string) error { return nil },
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(_ context.Context, path string, fn func() error) error {
			lockedPath = path

			return fn()
		},
	}

	sub := config.Subscription{Name: "test", Repo: "github.com/test/repo"}
	_, err := config.UpsertSubscription(t.Context(), fs, locker, "/cfg/config.yaml", sub)

	require.NoError(t, err)
	require.Equal(t, "/cfg/config.yaml.lock", lockedPath)
}

func TestUpsertSubscription_UpsertingShouldReturnErrorWhenLockFails(t *testing.T) {
	fs := &utils.MoqFileSystem{}
	locker := &utils.MoqLocker{
		WithLockFunc: func(_ context.Context, _ string, _ func() error) error {
			return errors.New("lock timeout")
		},
	}

	sub := config.Subscription{Name: "test", Repo: "github.com/test/repo"}
	_, err := config.UpsertSubscription(t.Context(), fs, locker, "/cfg/config.yaml", sub)

	require.Error(t, err)
	require.Contains(t, err.Error(), "upserting subscription")
}

// --- Save edge case: WriteFile receives bytes.Reader ---

func TestSave_SavingConfigShouldPassBytesReaderToWriteFile(t *testing.T) {
	var receivedReader io.Reader

	fs := &utils.MoqFileSystem{
		CreateDirectoryFunc:     func(_ string) error { return nil },
		CreateTemporaryFileFunc: func(dir, _ string) (string, error) { return dir + "/tmp.yaml", nil },
		WriteFileFunc: func(_ string, reader io.Reader) (int64, error) {
			receivedReader = reader

			data, err := io.ReadAll(reader)
			if err != nil {
				return 0, err
			}

			return int64(len(data)), nil
		},
		RenameFunc: func(_, _ string) error { return nil },
	}

	err := config.Save(fs, "/cfg/config.yaml", config.Config{})

	require.NoError(t, err)

	_, ok := receivedReader.(*bytes.Reader)
	require.True(t, ok, "WriteFile should receive a *bytes.Reader")
}
