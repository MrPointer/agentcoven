package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

const lockRetryDelay = 100 * time.Millisecond

// Locker provides kernel-level file locking for concurrent access control.
type Locker interface {
	// WithLock acquires an exclusive lock on the file at path, executes fn, and releases the lock.
	// The lock is released even if fn returns an error or panics.
	// Context cancellation is respected while waiting for the lock.
	WithLock(ctx context.Context, path string, fn func() error) error
}

// DefaultLocker is the production implementation of Locker using gofrs/flock.
type DefaultLocker struct{}

var _ Locker = (*DefaultLocker)(nil)

// NewDefaultLocker creates a new DefaultLocker.
func NewDefaultLocker() *DefaultLocker {
	return &DefaultLocker{}
}

// WithLock acquires an exclusive lock, executes fn, and releases the lock.
func (l *DefaultLocker) WithLock(ctx context.Context, path string, fn func() error) (retErr error) {
	fileLock := flock.New(path)

	locked, err := fileLock.TryLockContext(ctx, lockRetryDelay)
	if err != nil {
		return fmt.Errorf("acquiring lock on %s: %w", path, err)
	}

	if !locked {
		return fmt.Errorf("failed to acquire lock on %s", path)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = fileLock.Unlock() //nolint:errcheck // best-effort unlock before re-panicking

			panic(r)
		}

		if unlockErr := fileLock.Unlock(); unlockErr != nil && retErr == nil {
			retErr = fmt.Errorf("releasing lock on %s: %w", path, unlockErr)
		}
	}()

	return fn()
}
