// Package apply implements the orchestration logic for applying coven blocks to the local filesystem.
package apply

import (
	"context"
	"errors"

	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
)

// conflictKind classifies the outcome of a conflict check for a single target path.
type conflictKind int

const (
	// conflictKindNew means the path does not exist on disk — safe to write.
	conflictKindNew conflictKind = iota
	// conflictKindUpdate means the path exists and is owned by this subscription+framework — safe to overwrite.
	conflictKindUpdate
	// conflictKindUserFile means the path exists but is not tracked — must not touch.
	conflictKindUserFile
	// conflictKindCrossSubscription means the path exists and is owned by a different subscription — must not touch.
	conflictKindCrossSubscription
)

// checkConflict determines whether it is safe to write to targetPath.
//
// It returns conflictKindNew when the path does not exist.
// It returns conflictKindUpdate when the path is owned by the same subscription+framework.
// It returns conflictKindUserFile when the path exists but has no state record.
// It returns conflictKindCrossSubscription when the path is owned by a different subscription.
func checkConflict(
	ctx context.Context,
	fs utils.FileSystem,
	store state.BlockStore,
	targetPath string,
	subscription string,
	framework string,
) (conflictKind, error) {
	exists, err := fs.PathExists(targetPath)
	if err != nil {
		return 0, err
	}

	if !exists {
		return conflictKindNew, nil
	}

	rec, err := store.QueryByPath(ctx, targetPath)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return conflictKindUserFile, nil
		}

		return 0, err
	}

	if rec.Subscription == subscription && rec.Framework == framework {
		return conflictKindUpdate, nil
	}

	return conflictKindCrossSubscription, nil
}
