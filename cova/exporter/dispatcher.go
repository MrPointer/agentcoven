package exporter

import (
	"context"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

const (
	// externalExporterPrefix is the executable name prefix for external exporters.
	externalExporterPrefix = "cova-exporter-"

	// agentClaudeCode is the agent name for the Claude Code built-in exporter.
	agentClaudeCode = "claude-code"
)

// Dispatcher resolves the appropriate exporter for a given agent and applies or removes blocks.
type Dispatcher interface {
	// Apply resolves the exporter for the given agent and applies the request.
	Apply(ctx context.Context, agent string, req *ApplyRequest) (*ApplyResponse, error)

	// Remove resolves the exporter for the given agent and removes the blocks in the request.
	Remove(ctx context.Context, agent string, req *RemoveRequest) (*RemoveResponse, error)
}

// DefaultDispatcher implements Dispatcher by first checking built-in exporters,
// then falling back to external executables on $PATH.
type DefaultDispatcher struct {
	programQuery osmanager.ProgramQuery
	commander    utils.Commander
	builtins     map[string]exporter
}

var _ Dispatcher = (*DefaultDispatcher)(nil)

// NewDefaultDispatcher creates a DefaultDispatcher with built-in exporters registered.
// homeDir is used to construct absolute target paths for the Claude Code exporter.
func NewDefaultDispatcher(
	programQuery osmanager.ProgramQuery,
	commander utils.Commander,
	fs utils.FileSystem,
	homeDir string,
) *DefaultDispatcher {
	d := &DefaultDispatcher{
		programQuery: programQuery,
		commander:    commander,
		builtins:     make(map[string]exporter),
	}

	d.builtins[agentClaudeCode] = newClaudeCodeExporter(fs, homeDir)

	return d
}

// Apply resolves the exporter for agent and invokes it with req.
// Built-in exporters are checked first; if none match, the dispatcher looks for
// a cova-exporter-{agent} executable on $PATH. An error is returned if
// neither is found.
func (d *DefaultDispatcher) Apply(ctx context.Context, agent string, req *ApplyRequest) (*ApplyResponse, error) {
	if a, ok := d.builtins[agent]; ok {
		return a.apply(ctx, req)
	}

	execName := externalExporterPrefix + agent

	path, err := d.programQuery.GetProgramPath(execName)
	if err != nil {
		return nil, fmt.Errorf("no exporter found for agent %q: %w", agent, err)
	}

	ext := newExternalExporter(path, d.commander)

	return ext.apply(ctx, req)
}

// Remove resolves the exporter for agent and invokes it with req.
// Built-in exporters are checked first; if none match, the dispatcher looks for
// a cova-exporter-{agent} executable on $PATH. An error is returned if
// neither is found.
func (d *DefaultDispatcher) Remove(ctx context.Context, agent string, req *RemoveRequest) (*RemoveResponse, error) {
	if a, ok := d.builtins[agent]; ok {
		return a.remove(ctx, req)
	}

	execName := externalExporterPrefix + agent

	path, err := d.programQuery.GetProgramPath(execName)
	if err != nil {
		return nil, fmt.Errorf("no exporter found for agent %q: %w", agent, err)
	}

	ext := newExternalExporter(path, d.commander)

	return ext.remove(ctx, req)
}
