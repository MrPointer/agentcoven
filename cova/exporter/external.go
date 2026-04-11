package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// externalExporter invokes an external cova-exporter-{name} executable using JSON
// over stdin/stdout.
type externalExporter struct {
	commander utils.Commander
	execPath  string
}

var _ exporter = (*externalExporter)(nil)

// newExternalExporter creates an external exporter that invokes the executable at execPath.
func newExternalExporter(execPath string, commander utils.Commander) *externalExporter {
	return &externalExporter{
		execPath:  execPath,
		commander: commander,
	}
}

// info sends {"operation":"info"} to the exporter and returns the parsed response.
// On any failure (non-zero exit, bad JSON, timeout), a fallback InfoResponse is returned
// with the name derived from the binary and an empty description.
func (a *externalExporter) info(ctx context.Context) (*InfoResponse, error) {
	const infoTimeout = 5 * time.Second

	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()

	fallback := &InfoResponse{
		Name:        strings.TrimPrefix(filepath.Base(a.execPath), externalExporterPrefix),
		Description: "",
	}

	input, err := json.Marshal(map[string]string{"operation": "info"})
	if err != nil {
		return fallback, nil
	}

	result, err := a.commander.RunCommand(
		ctx,
		a.execPath,
		nil,
		utils.WithInput(input),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return fallback, nil
	}

	var resp InfoResponse
	if err := json.Unmarshal(result.Stdout, &resp); err != nil {
		return fallback, nil
	}

	return &resp, nil
}

// apply marshals req to JSON, pipes it to the exporter executable's stdin, and
// unmarshals the response from stdout. If the process exits non-zero, a detailed
// error including stderr is returned.
func (a *externalExporter) apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling apply request: %w", err)
	}

	result, err := a.commander.RunCommand(
		ctx,
		a.execPath,
		nil,
		utils.WithInput(input),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = result.StderrString()
		}

		return nil, fmt.Errorf("external exporter %q failed (exit %d): %w\nstderr: %s",
			a.execPath, result.ExitCode, err, stderr)
	}

	var resp ApplyResponse
	if err := json.Unmarshal(result.Stdout, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling apply response from %q: %w", a.execPath, err)
	}

	return &resp, nil
}

// remove marshals req to JSON, pipes it to the exporter executable's stdin, and
// unmarshals the response from stdout. If the process exits non-zero, a detailed
// error including stderr is returned.
func (a *externalExporter) remove(ctx context.Context, req *RemoveRequest) (*RemoveResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling remove request: %w", err)
	}

	result, err := a.commander.RunCommand(
		ctx,
		a.execPath,
		nil,
		utils.WithInput(input),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = result.StderrString()
		}

		return nil, fmt.Errorf("external exporter %q failed (exit %d): %w\nstderr: %s",
			a.execPath, result.ExitCode, err, stderr)
	}

	var resp RemoveResponse
	if err := json.Unmarshal(result.Stdout, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling remove response from %q: %w", a.execPath, err)
	}

	return &resp, nil
}
