package utils

import (
	"io"
	"os"

	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

// FileSystem defines the file system operations available in the application.
type FileSystem interface {
	// CreateFile creates a file at the specified path.
	// If the file already exists, it will be truncated.
	//
	// It returns the created file or an error if it fails.
	CreateFile(path string) (string, error)

	// CreateDirectory creates a directory at the specified path.
	CreateDirectory(path string) error

	// CreateDirectoryWithPermissions creates a directory at the specified path with the specified permissions.
	CreateDirectoryWithPermissions(path string, mode os.FileMode) error

	// CreateTemporaryFile creates a temporary file in the optional specified directory.
	// dir is the directory where the temporary file will be created.
	// If dir is nil, the system's default temporary directory will be used.
	//
	// It returns the created file or an error if it fails.
	CreateTemporaryFile(dir, pattern string) (string, error)

	// CreateTemporaryDirectory creates a temporary directory in the optional specified directory.
	// dir is the directory where the temporary directory will be created.
	// If dir is nil, the system's default temporary directory will be used.
	//
	// It returns the path of the created temporary directory or an error if it fails.
	CreateTemporaryDirectory(dir string) (string, error)

	// RemovePath removes a file or directory at the specified path.
	// If the path is a directory, it will be removed recursively.
	RemovePath(path string) error

	// PathExists checks if a file or directory exists at the specified path.
	// It returns true if the path exists, false if it does not, and an error if there was an issue checking the path.
	// This function does not distinguish between files and directories.
	PathExists(path string) (bool, error)

	// IsExecutable checks if the file at the specified path is executable.
	// It returns true if the file exists and has any execute permission bit set.
	IsExecutable(path string) (bool, error)

	// WriteFile writes data to a file at the specified path.
	// If the file does not exist, it will be created.
	// If the file exists, it will be truncated before writing.
	//
	// It accepts an io.Reader to read data from, which allows for streaming data into the file.
	//
	// It returns the number of bytes written and an error if the write operation fails.
	WriteFile(path string, reader io.Reader) (int64, error)

	// ReadFile reads data from a file at the specified path.
	// It writes the data to the provided receiver, which is an io.Writer.
	//
	// It returns the number of bytes read and an error if the read operation fails.
	ReadFile(path string, receiver io.Writer) (int64, error)

	// ReadFileContents reads the entire contents of a file and returns it as a byte slice.
	// This is a convenience method for small files where streaming is not needed.
	ReadFileContents(path string) ([]byte, error)

	// Rename renames (moves) a file or directory from oldPath to newPath.
	Rename(oldPath, newPath string) error

	// ReadDirectory reads the contents of a directory at the specified path.
	//
	// It returns a slice of os.DirEntry, which provides information about the files and directories within the specified directory.
	ReadDirectory(path string) ([]os.DirEntry, error)
}

// DefaultFileSystem is the default implementation of the FileSystem interface using os package.
type DefaultFileSystem struct {
	logger logger.Logger
}

var _ FileSystem = (*DefaultFileSystem)(nil)

// NewDefaultFileSystem creates a new DefaultFileSystem.
func NewDefaultFileSystem(logger logger.Logger) *DefaultFileSystem {
	return &DefaultFileSystem{
		logger: logger,
	}
}

// CreateFile creates or truncates the file at path and returns its name.
func (fs *DefaultFileSystem) CreateFile(path string) (string, error) {
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return file.Name(), nil
}

// CreateDirectory creates path and all parent directories with default permissions.
func (fs *DefaultFileSystem) CreateDirectory(path string) error {
	const defaultPermissions = 0o755 // Default permissions for directories.
	return os.MkdirAll(path, defaultPermissions)
}

// CreateDirectoryWithPermissions creates path and all parent directories with the given permissions.
func (fs *DefaultFileSystem) CreateDirectoryWithPermissions(path string, permissions os.FileMode) error {
	return os.MkdirAll(path, permissions)
}

// CreateTemporaryFile creates a temporary file in dir using the given pattern and returns its path.
func (fs *DefaultFileSystem) CreateTemporaryFile(dir, pattern string) (string, error) {
	var tempDir string
	if dir != "" {
		tempDir = dir
	}

	computedPattern := "tempfile-*.tmp"
	if pattern != "" {
		computedPattern = pattern
	}

	tempFile, err := os.CreateTemp(tempDir, computedPattern)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	return tempFile.Name(), nil
}

// CreateTemporaryDirectory creates a temporary directory inside dir and returns its path.
func (fs *DefaultFileSystem) CreateTemporaryDirectory(dir string) (string, error) {
	var tempDir string
	if dir != "" {
		tempDir = dir
	}

	return os.MkdirTemp(tempDir, "tempdir-*")
}

// WriteFile creates or truncates the file at path and writes all data from reader into it.
func (fs *DefaultFileSystem) WriteFile(path string, reader io.Reader) (int64, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	bytesWritten, err := io.Copy(file, reader)
	if err != nil {
		return 0, err
	}

	return bytesWritten, nil
}

// RemovePath removes the file or directory at path, recursively if necessary.
func (fs *DefaultFileSystem) RemovePath(path string) error {
	return os.RemoveAll(path)
}

// PathExists reports whether a file or directory exists at path.
func (fs *DefaultFileSystem) PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// IsExecutable reports whether the file at path has any execute permission bit set.
func (fs *DefaultFileSystem) IsExecutable(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	// Check if any execute bit is set
	return info.Mode()&0o111 != 0, nil
}

// ReadFile reads the file at path and writes its contents to receiver.
func (fs *DefaultFileSystem) ReadFile(path string, receiver io.Writer) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	bytesRead, err := io.Copy(receiver, file)
	if err != nil {
		return 0, err
	}

	return bytesRead, nil
}

// Rename renames (moves) oldPath to newPath atomically.
func (fs *DefaultFileSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// ReadFileContents reads and returns the entire contents of the file at path.
func (fs *DefaultFileSystem) ReadFileContents(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ReadDirectory returns the directory entries for the directory at path.
func (fs *DefaultFileSystem) ReadDirectory(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
