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

func TestDefaultDispatcher_GettingInfoForBuiltinAgentShouldDelegateToBuiltinExporter(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			return "", errors.New("should not be called")
		},
	}

	mockExporter := &Moqexporter{
		infoFunc: func(ctx context.Context) (*InfoResponse, error) {
			return &InfoResponse{Name: "test-agent", Description: "A test agent"}, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		builtins:     map[string]exporter{"test-agent": mockExporter},
	}

	resp, err := d.Info(t.Context(), "test-agent")

	require.NoError(t, err)
	require.Equal(t, "test-agent", resp.Name)
	require.Equal(t, "A test agent", resp.Description)
}

func TestDefaultDispatcher_GettingInfoForExternalAgentShouldDelegateToExternalExporter(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			if program == "cova-exporter-myfw" {
				return "/usr/local/bin/cova-exporter-myfw", nil
			}

			return "", errors.New("not found")
		},
	}

	respBytes := []byte(`{"name":"myfw","description":"My external framework"}`)

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

	resp, err := d.Info(t.Context(), "myfw")

	require.NoError(t, err)
	require.Equal(t, "myfw", resp.Name)
}

func TestDefaultDispatcher_GettingInfoForUnknownAgentShouldReturnError(t *testing.T) {
	mockPQ := &osmanager.MoqProgramQuery{
		GetProgramPathFunc: func(program string) (string, error) {
			return "", errors.New("not found: " + program)
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		builtins:     map[string]exporter{},
	}

	_, err := d.Info(t.Context(), "unknown-agent")

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown-agent")
}

func TestDefaultDispatcher_ListingAvailableShouldReturnBuiltinsAndExternals(t *testing.T) {
	mockExporter := &Moqexporter{
		infoFunc: func(ctx context.Context) (*InfoResponse, error) {
			return &InfoResponse{Name: "builtin-agent", Description: "A built-in agent"}, nil
		},
	}

	mockPQ := &osmanager.MoqProgramQuery{
		FindProgramsByPrefixFunc: func(prefix string) ([]string, error) {
			return []string{"/usr/local/bin/cova-exporter-ext-agent"}, nil
		},
	}

	extRespBytes := []byte(`{"name":"ext-agent","description":"An external agent"}`)

	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{Stdout: extRespBytes, ExitCode: 0}, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		commander:    mockCommander,
		builtins:     map[string]exporter{"builtin-agent": mockExporter},
	}

	results, err := d.ListAvailable(t.Context())

	require.NoError(t, err)
	require.Len(t, results, 2)

	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Name
	}

	require.Contains(t, names, "builtin-agent")
	require.Contains(t, names, "ext-agent")

	for _, r := range results {
		if r.Name == "builtin-agent" {
			require.True(t, r.BuiltIn)
		} else {
			require.False(t, r.BuiltIn)
		}
	}
}

func TestDefaultDispatcher_ListingAvailableShouldPreferBuiltinWhenExternalHasSameName(t *testing.T) {
	mockExporter := &Moqexporter{
		infoFunc: func(ctx context.Context) (*InfoResponse, error) {
			return &InfoResponse{Name: "claude-code", Description: "Built-in"}, nil
		},
	}

	mockPQ := &osmanager.MoqProgramQuery{
		FindProgramsByPrefixFunc: func(prefix string) ([]string, error) {
			return []string{"/usr/local/bin/cova-exporter-claude-code"}, nil
		},
	}

	extRespBytes := []byte(`{"name":"claude-code","description":"External override"}`)

	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{Stdout: extRespBytes, ExitCode: 0}, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		commander:    mockCommander,
		builtins:     map[string]exporter{"claude-code": mockExporter},
	}

	results, err := d.ListAvailable(t.Context())

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Built-in", results[0].Description)
}

func TestDefaultDispatcher_ListingAvailableShouldReturnOnlyBuiltinsWhenNoExternalsFound(t *testing.T) {
	mockExporter := &Moqexporter{
		infoFunc: func(ctx context.Context) (*InfoResponse, error) {
			return &InfoResponse{Name: "claude-code", Description: "Built-in"}, nil
		},
	}

	mockPQ := &osmanager.MoqProgramQuery{
		FindProgramsByPrefixFunc: func(prefix string) ([]string, error) {
			return nil, nil
		},
	}

	d := &DefaultDispatcher{
		programQuery: mockPQ,
		builtins:     map[string]exporter{"claude-code": mockExporter},
	}

	results, err := d.ListAvailable(t.Context())

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "claude-code", results[0].Name)
	require.True(t, results[0].BuiltIn)
}
