// Package config provides YAML-based configuration for cova subscriptions.
package config

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

const (
	// configDirName is the directory name under the XDG config root.
	configDirName = "cova"
	// configFileName is the config file name.
	configFileName = "config.yaml"
	// lockSuffix is appended to the config path to form the lock file path.
	lockSuffix = ".lock"
	// tempFilePattern is the pattern for temporary files during atomic writes.
	tempFilePattern = "config-*.yaml.tmp"
)

// Config represents the top-level cova configuration.
type Config struct {
	Subscriptions []Subscription `yaml:"subscriptions,omitempty"`
	Frameworks    []string       `yaml:"frameworks,omitempty"`
}

// Subscription represents a single coven subscription entry.
type Subscription struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
	Path string `yaml:"path,omitempty"`
	Ref  string `yaml:"ref,omitempty"`
}

// UpsertResult indicates the outcome of an upsert operation.
type UpsertResult int

const (
	// UpsertAdded indicates a new subscription was added.
	UpsertAdded UpsertResult = iota
	// UpsertUpdated indicates an existing subscription was updated.
	UpsertUpdated
	// UpsertNoOp indicates the subscription already existed with identical settings.
	UpsertNoOp
)

// DefaultPath resolves the config file path using XDG conventions.
// It checks $XDG_CONFIG_HOME first; if unset or empty, falls back to ~/.config.
func DefaultPath(
	envMgr osmanager.EnvironmentManager,
	userMgr osmanager.UserManager,
) (string, error) {
	configRoot := envMgr.Getenv("XDG_CONFIG_HOME")
	if configRoot == "" {
		homeDir, err := userMgr.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}

		configRoot = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configRoot, configDirName, configFileName), nil
}

// Load reads and parses the config file at the given path.
// If the file does not exist, it returns an empty Config without writing anything.
func Load(fs utils.FileSystem, path string) (Config, error) {
	exists, err := fs.PathExists(path)
	if err != nil {
		return Config{}, fmt.Errorf("checking config file: %w", err)
	}

	if !exists {
		return Config{}, nil
	}

	data, err := fs.ReadFileContents(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config YAML: %w", err)
	}

	return cfg, nil
}

// Save writes the config atomically to the given path.
// It creates the parent directory if needed, writes to a temp file in the same directory,
// then renames over the target. The temp file is cleaned up on failure.
func Save(fs utils.FileSystem, path string, cfg Config) error {
	dir := filepath.Dir(path)

	if err := fs.CreateDirectory(dir); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	tmpPath, err := fs.CreateTemporaryFile(dir, tempFilePattern)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := fs.WriteFile(tmpPath, bytes.NewReader(data)); err != nil {
		_ = fs.RemovePath(tmpPath) //nolint:errcheck // best-effort cleanup of temp file
		return fmt.Errorf("writing temp config file: %w", err)
	}

	if err := fs.Rename(tmpPath, path); err != nil {
		_ = fs.RemovePath(tmpPath) //nolint:errcheck // best-effort cleanup of temp file
		return fmt.Errorf("renaming temp config file: %w", err)
	}

	return nil
}

// UpsertSubscription performs a locked read-modify-write to add or update a subscription.
// It returns the upsert result indicating whether the subscription was added, updated, or unchanged.
func UpsertSubscription(
	ctx context.Context,
	fs utils.FileSystem,
	locker utils.Locker,
	path string,
	sub Subscription,
) (UpsertResult, error) {
	var result UpsertResult

	lockPath := path + lockSuffix

	err := locker.WithLock(ctx, lockPath, func() error {
		cfg, err := Load(fs, path)
		if err != nil {
			return err
		}

		result = upsert(&cfg, sub)

		if result == UpsertNoOp {
			return nil
		}

		return Save(fs, path, cfg)
	})
	if err != nil {
		return 0, fmt.Errorf("upserting subscription %q: %w", sub.Name, err)
	}

	return result, nil
}

// upsert modifies cfg in place, returning the result.
func upsert(cfg *Config, sub Subscription) UpsertResult {
	for i, existing := range cfg.Subscriptions {
		if existing.Name != sub.Name {
			continue
		}

		if existing == sub {
			return UpsertNoOp
		}

		cfg.Subscriptions[i] = sub

		return UpsertUpdated
	}

	cfg.Subscriptions = append(cfg.Subscriptions, sub)

	return UpsertAdded
}
