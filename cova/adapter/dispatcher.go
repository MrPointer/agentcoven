package adapter

import (
	"context"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

const (
	// externalAdapterPrefix is the executable name prefix for external adapters.
	externalAdapterPrefix = "cova-adapter-"

	// frameworkClaudeCode is the framework name for the Claude Code built-in adapter.
	frameworkClaudeCode = "claude-code"
)

// Dispatcher resolves the appropriate adapter for a given framework and applies blocks.
type Dispatcher interface {
	// Apply resolves the adapter for the given framework and applies the request.
	Apply(ctx context.Context, framework string, req *ApplyRequest) (*ApplyResponse, error)
}

// DefaultDispatcher implements Dispatcher by first checking built-in adapters,
// then falling back to external executables on $PATH.
type DefaultDispatcher struct {
	programQuery osmanager.ProgramQuery
	commander    utils.Commander
	builtins     map[string]adapter
}

var _ Dispatcher = (*DefaultDispatcher)(nil)

// NewDefaultDispatcher creates a DefaultDispatcher with built-in adapters registered.
// homeDir is used to construct absolute target paths for the Claude Code adapter.
func NewDefaultDispatcher(
	programQuery osmanager.ProgramQuery,
	commander utils.Commander,
	fs utils.FileSystem,
	homeDir string,
) *DefaultDispatcher {
	d := &DefaultDispatcher{
		programQuery: programQuery,
		commander:    commander,
		builtins:     make(map[string]adapter),
	}

	d.builtins[frameworkClaudeCode] = newClaudeCodeAdapter(fs, homeDir)

	return d
}

// Apply resolves the adapter for framework and invokes it with req.
// Built-in adapters are checked first; if none match, the dispatcher looks for
// a cova-adapter-{framework} executable on $PATH. An error is returned if
// neither is found.
func (d *DefaultDispatcher) Apply(ctx context.Context, framework string, req *ApplyRequest) (*ApplyResponse, error) {
	if a, ok := d.builtins[framework]; ok {
		return a.apply(ctx, req)
	}

	execName := externalAdapterPrefix + framework

	path, err := d.programQuery.GetProgramPath(execName)
	if err != nil {
		return nil, fmt.Errorf("no adapter found for framework %q: %w", framework, err)
	}

	ext := newExternalAdapter(path, d.commander)

	return ext.apply(ctx, req)
}
