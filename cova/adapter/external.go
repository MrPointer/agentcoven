package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// externalAdapter invokes an external cova-adapter-{name} executable using JSON
// over stdin/stdout.
type externalAdapter struct {
	commander utils.Commander
	execPath  string
}

var _ adapter = (*externalAdapter)(nil)

// newExternalAdapter creates an external adapter that invokes the executable at execPath.
func newExternalAdapter(execPath string, commander utils.Commander) *externalAdapter {
	return &externalAdapter{
		execPath:  execPath,
		commander: commander,
	}
}

// apply marshals req to JSON, pipes it to the adapter executable's stdin, and
// unmarshals the response from stdout. If the process exits non-zero, a detailed
// error including stderr is returned.
func (a *externalAdapter) apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
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

		return nil, fmt.Errorf("external adapter %q failed (exit %d): %w\nstderr: %s",
			a.execPath, result.ExitCode, err, stderr)
	}

	var resp ApplyResponse
	if err := json.Unmarshal(result.Stdout, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling apply response from %q: %w", a.execPath, err)
	}

	return &resp, nil
}
