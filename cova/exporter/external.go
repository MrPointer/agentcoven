package exporter

import (
	"context"
	"encoding/json"
	"fmt"

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
