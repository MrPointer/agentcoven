package exporter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

func TestDefaultDispatcher_ApplyingBuiltinAgentShouldDelegateToBuiltinExporter(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			return "", errors.New("should not be called")
		},
	}

	expectedResp := &ApplyResponse{
		Results: []BlockResult{{Name: "my-skill"}},
	}

	mockExporter := &Moqexporter{
		applyFunc: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			return expectedResp, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		builtins:     map[string]exporter{"test-agent": mockExporter},
	}

	req := &ApplyRequest{Operation: "apply", Blocks: map[string][]RequestBlock{}}

	resp, err := d.Apply(t.Context(), "test-agent", req)

	require.NoError(t, err)
	require.Equal(t, expectedResp, resp)
}

func TestDefaultDispatcher_ApplyingUnknownAgentShouldLookUpExternalExporter(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			if program == "cova-exporter-myfw" {
				return "/usr/local/bin/cova-exporter-myfw", nil
			}

			return "", errors.New("not found")
		},
	}

	respBytes := []byte(`{"results":[{"name":"acme-block","placements":null,"error":null}]}`)

	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{Stdout: respBytes, ExitCode: 0}, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		commander:    mockCommander,
		builtins:     map[string]exporter{},
	}

	req := &ApplyRequest{Operation: "apply", Blocks: map[string][]RequestBlock{}}

	resp, err := d.Apply(t.Context(), "myfw", req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "acme-block", resp.Results[0].Name)
}

func TestDefaultDispatcher_ApplyingUnknownAgentShouldReturnErrorWhenNoExporterFound(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			return "", errors.New("not found: " + program)
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		builtins:     map[string]exporter{},
	}

	req := &ApplyRequest{Operation: "apply", Blocks: map[string][]RequestBlock{}}

	_, err := d.Apply(t.Context(), "unknown-agent", req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown-agent")
}
