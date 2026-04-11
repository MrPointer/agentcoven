// Package config provides YAML-based configuration for cova subscriptions.
package config

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"

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
	Agents        []string       `yaml:"agents,omitempty"`
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

// AgentAddResult indicates the outcome of adding a single agent.
type AgentAddResult struct {
	Name  string
	Added bool // true = newly added, false = already existed
}

// AgentRemoveResult indicates the outcome of removing a single agent.
type AgentRemoveResult struct {
	Name    string
	Removed bool // true = found and removed, false = not found
}

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

	if err := fs.CreateDirectory(filepath.Dir(lockPath)); err != nil {
		return 0, fmt.Errorf("creating config directory: %w", err)
	}

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

// RemoveSubscription performs a locked read-modify-write to remove a subscription by name.
// It returns true if the subscription was found and removed, false if it was not found.
func RemoveSubscription(
	ctx context.Context,
	fs utils.FileSystem,
	locker utils.Locker,
	path string,
	name string,
) (bool, error) {
	var found bool

	lockPath := path + lockSuffix

	if err := fs.CreateDirectory(filepath.Dir(lockPath)); err != nil {
		return false, fmt.Errorf("creating config directory: %w", err)
	}

	err := locker.WithLock(ctx, lockPath, func() error {
		cfg, err := Load(fs, path)
		if err != nil {
			return err
		}

		found = remove(&cfg, name)

		if !found {
			return nil
		}

		return Save(fs, path, cfg)
	})
	if err != nil {
		return false, fmt.Errorf("removing subscription %q: %w", name, err)
	}

	return found, nil
}

// remove modifies cfg in place, removing the subscription with the given name.
// It returns true if the subscription was found and removed, false otherwise.
func remove(cfg *Config, name string) bool {
	for i, sub := range cfg.Subscriptions {
		if sub.Name != name {
			continue
		}

		cfg.Subscriptions = append(cfg.Subscriptions[:i], cfg.Subscriptions[i+1:]...)

		return true
	}

	return false
}

// AddAgents performs a locked read-modify-write to add agent names to the config.
// For each name, if not already present, it is added. Returns per-name results.
func AddAgents(
	ctx context.Context,
	fs utils.FileSystem,
	locker utils.Locker,
	path string,
	names []string,
) ([]AgentAddResult, error) {
	var results []AgentAddResult

	lockPath := path + lockSuffix

	if err := fs.CreateDirectory(filepath.Dir(lockPath)); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	err := locker.WithLock(ctx, lockPath, func() error {
		cfg, err := Load(fs, path)
		if err != nil {
			return err
		}

		results = addAgentsToConfig(&cfg, names)

		// Only save if at least one agent was actually added
		anyAdded := false

		for _, result := range results {
			if result.Added {
				anyAdded = true
				break
			}
		}

		if !anyAdded {
			return nil
		}

		return Save(fs, path, cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("adding agents: %w", err)
	}

	return results, nil
}

// addAgentsToConfig modifies cfg in place, adding agent names that are not already present.
// Returns per-name results indicating which were added vs. already existed.
func addAgentsToConfig(cfg *Config, names []string) []AgentAddResult {
	results := make([]AgentAddResult, len(names))

	for i, name := range names {
		if slices.Contains(cfg.Agents, name) {
			results[i] = AgentAddResult{Name: name, Added: false}
		} else {
			cfg.Agents = append(cfg.Agents, name)
			results[i] = AgentAddResult{Name: name, Added: true}
		}
	}

	return results
}

// RemoveAgents performs a locked read-modify-write to remove agent names from the config.
// For each name, if present, it is removed. Returns per-name results.
func RemoveAgents(
	ctx context.Context,
	fs utils.FileSystem,
	locker utils.Locker,
	path string,
	names []string,
) ([]AgentRemoveResult, error) {
	var results []AgentRemoveResult

	lockPath := path + lockSuffix

	if err := fs.CreateDirectory(filepath.Dir(lockPath)); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	err := locker.WithLock(ctx, lockPath, func() error {
		cfg, err := Load(fs, path)
		if err != nil {
			return err
		}

		results = removeAgentsFromConfig(&cfg, names)

		// Only save if at least one agent was actually removed
		anyRemoved := false

		for _, result := range results {
			if result.Removed {
				anyRemoved = true
				break
			}
		}

		if !anyRemoved {
			return nil
		}

		return Save(fs, path, cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("removing agents: %w", err)
	}

	return results, nil
}

// removeAgentsFromConfig modifies cfg in place, removing agent names that are present.
// Returns per-name results indicating which were removed vs. not found.
func removeAgentsFromConfig(cfg *Config, names []string) []AgentRemoveResult {
	results := make([]AgentRemoveResult, len(names))

	for i, name := range names {
		found := false

		for j, existing := range cfg.Agents {
			if existing == name {
				cfg.Agents = append(cfg.Agents[:j], cfg.Agents[j+1:]...)
				found = true

				break
			}
		}

		results[i] = AgentRemoveResult{Name: name, Removed: found}
	}

	return results
}
