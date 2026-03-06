package osmanager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

// UserManager defines operations for managing system users.
type UserManager interface {
	// GetHomeDirectory returns the home directory of the current user.
	GetHomeDir() (string, error)

	// GetConfigDir returns the configuration directory of the current user.
	GetConfigDir() (string, error)

	// GetCurrentUsername returns the current user's username.
	GetCurrentUsername() (string, error)
}

// VersionExtractor is a function type that defines how to extract version information from a program's output.
type VersionExtractor func(string) (string, error)

type ProgramQuery interface {
	// GetProgramPath retrieves the full path of a program if it's available in one of the system's PATH directories.
	// If the program is not found, it returns an error.
	GetProgramPath(program string) (string, error)

	// ProgramExists checks if a program exists in the system's PATH directories.
	// It returns true if the program is found, false if not, and an error if there was an issue checking.
	ProgramExists(program string) (bool, error)

	// GetProgramVersion retrieves the version of a program by executing it with the provided query arguments.
	GetProgramVersion(program string, versionExtractor VersionExtractor, queryArgs ...string) (string, error)
}

// EnvironmentManager defines operations for managing environment variables.
type EnvironmentManager interface {
	// Getenv retrieves the value of the environment variable named by the key.
	Getenv(key string) string
}

// OsManager combines all system operation interfaces.
type OsManager interface {
	UserManager
	ProgramQuery
	EnvironmentManager
}

// DefaultOsManager implements OsManager for Unix-like systems.
type DefaultOsManager struct {
	logger     logger.Logger
	fileSystem utils.FileSystem
	commander  utils.Commander
}

var _ OsManager = (*DefaultOsManager)(nil)

// NewDefaultOsManager creates a new DefaultOsManager with injected Escalator and FileSystem.
// Intended for deterministic unit tests.
func NewDefaultOsManager(
	logger logger.Logger,
	commander utils.Commander,
	fileSystem utils.FileSystem,
) *DefaultOsManager {
	return &DefaultOsManager{
		logger:     logger,
		fileSystem: fileSystem,
		commander:  commander,
	}
}

func (u *DefaultOsManager) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (u *DefaultOsManager) GetConfigDir() (string, error) {
	return os.UserConfigDir()
}

// GetCurrentUsername returns the current user's username.
func (u *DefaultOsManager) GetCurrentUsername() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return currentUser.Username, nil
}

func (u *DefaultOsManager) GetProgramPath(program string) (string, error) {
	return exec.LookPath(program)
}

func (u *DefaultOsManager) ProgramExists(program string) (bool, error) {
	_, err := u.GetProgramPath(program)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return false, nil // Program not found.
		}
		return false, fmt.Errorf("error checking program existence: %w", err)
	}
	return true, nil // Program found.
}

func (u *DefaultOsManager) GetProgramVersion(
	program string,
	versionExtractor VersionExtractor,
	queryArgs ...string,
) (string, error) {
	args := []string{"--version"} // Default argument for version query.
	if len(queryArgs) > 0 {
		args = queryArgs
	}

	cmd := exec.Command(program, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", program, err)
	}

	version, err := versionExtractor(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to extract version from output: %w", err)
	}

	return version, nil
}

func (u *DefaultOsManager) Getenv(key string) string {
	return os.Getenv(key)
}
