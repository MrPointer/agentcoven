package utils_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/stretchr/testify/require"
)

func TestWithLock_LockingFileShouldExecuteCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	called := false
	err := locker.WithLock(t.Context(), lockPath, func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	require.True(t, called)
}

func TestWithLock_LockingFileShouldPropagateCallbackError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	expectedErr := errors.New("callback failed")
	err := locker.WithLock(t.Context(), lockPath, func() error {
		return expectedErr
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestWithLock_LockingFileShouldReleaseLockAfterCallbackError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	//nolint:errcheck // intentionally ignoring error from first call
	locker.WithLock(t.Context(), lockPath, func() error {
		return errors.New("first call fails")
	})

	err := locker.WithLock(t.Context(), lockPath, func() error {
		return nil
	})

	require.NoError(t, err)
}

func TestWithLock_LockingFileShouldReleaseLockAfterPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	require.Panics(t, func() {
		//nolint:errcheck // intentionally ignoring error; function panics
		locker.WithLock(t.Context(), lockPath, func() error {
			panic("test panic")
		})
	})

	err := locker.WithLock(t.Context(), lockPath, func() error {
		return nil
	})

	require.NoError(t, err)
}

func TestWithLock_LockingFileShouldRespectContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	// Hold the lock in a goroutine.
	held := make(chan struct{})
	release := make(chan struct{})

	go func() {
		//nolint:errcheck // lock holder goroutine; error not relevant to test
		locker.WithLock(t.Context(), lockPath, func() error {
			close(held)
			<-release

			return nil
		})
	}()

	<-held

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	err := locker.WithLock(ctx, lockPath, func() error {
		return nil
	})

	close(release)

	require.Error(t, err)
	require.Contains(t, err.Error(), "acquiring lock")
}

func TestWithLock_LockingFileShouldSerializeConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	var mu sync.Mutex

	active := 0
	maxActive := 0
	errs := make([]error, 5)

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Go(func() {
			errs[i] = locker.WithLock(t.Context(), lockPath, func() error {
				mu.Lock()

				active++
				if active > maxActive {
					maxActive = active
				}
				mu.Unlock()

				time.Sleep(10 * time.Millisecond)

				mu.Lock()
				active--
				mu.Unlock()

				return nil
			})
		})
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, 1, maxActive)
}

func TestWithLock_LockingFileShouldCreateLockFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := utils.NewDefaultLocker()

	err := locker.WithLock(t.Context(), lockPath, func() error {
		_, statErr := os.Stat(lockPath)
		require.NoError(t, statErr)

		return nil
	})

	require.NoError(t, err)
}
