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

	// Info resolves the exporter for the given agent and returns its info.
	Info(ctx context.Context, agent string) (*InfoResponse, error)

	// ListAvailable returns info for all discoverable exporters (built-in + external on PATH).
	ListAvailable(ctx context.Context) ([]InfoResponse, error)

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

// Info resolves the exporter for agent and returns its info.
// Built-in exporters are checked first; if none match, the dispatcher looks for
// a cova-exporter-{agent} executable on $PATH. An error is returned if
// neither is found.
func (d *DefaultDispatcher) Info(ctx context.Context, agent string) (*InfoResponse, error) {
	if a, ok := d.builtins[agent]; ok {
		return a.info(ctx)
	}

	execName := externalExporterPrefix + agent

	path, err := d.programQuery.GetProgramPath(execName)
	if err != nil {
		return nil, fmt.Errorf("no exporter found for agent %q: %w", agent, err)
	}

	ext := newExternalExporter(path, d.commander)

	return ext.info(ctx)
}

// ListAvailable returns info for all discoverable exporters.
// Built-ins are enumerated first. External exporters found on $PATH via
// FindProgramsByPrefix are then added; if a name already appears from a
// built-in, the external entry is skipped (built-in wins).
func (d *DefaultDispatcher) ListAvailable(ctx context.Context) ([]InfoResponse, error) {
	seen := make(map[string]struct{})
	results := make([]InfoResponse, 0, len(d.builtins))

	for _, exp := range d.builtins {
		resp, err := exp.info(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting info for built-in exporter: %w", err)
		}

		resp.BuiltIn = true
		seen[resp.Name] = struct{}{}
		results = append(results, *resp)
	}

	paths, err := d.programQuery.FindProgramsByPrefix(externalExporterPrefix)
	if err != nil {
		return nil, fmt.Errorf("scanning PATH for external exporters: %w", err)
	}

	for _, p := range paths {
		ext := newExternalExporter(p, d.commander)

		resp, err := ext.info(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting info for external exporter %q: %w", p, err)
		}

		if _, exists := seen[resp.Name]; exists {
			continue
		}

		seen[resp.Name] = struct{}{}
		results = append(results, *resp)
	}

	return results, nil
}
