package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
)

func TestExternalExporter_ApplyingRequestShouldReturnUnmarshalledResponse(t *testing.T) {
	expectedResp := &ApplyResponse{
		Results: []BlockResult{
			{
				Name: "acme-skill",
				Placements: []Placement{
					{Path: "/home/user/.claude/skills/acme-skill/SKILL.md", Source: "skills/acme-skill/SKILL.md"},
				},
			},
		},
	}

	respBytes, err := json.Marshal(expectedResp)
	require.NoError(t, err)

	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{Stdout: respBytes, ExitCode: 0}, nil
		},
	}

	a := newExternalExporter("/usr/local/bin/cova-exporter-myfw", mockCommander)
	req := &ApplyRequest{
		Operation:    "apply",
		Subscription: "platform",
		Workspace:    "/workspace",
		Blocks: map[string][]RequestBlock{
			"skills": {{Name: "acme-skill", Source: "skills/acme-skill"}},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "acme-skill", resp.Results[0].Name)
	require.Len(t, resp.Results[0].Placements, 1)
}

func TestExternalExporter_ApplyingRequestShouldReturnErrorWhenProcessFails(t *testing.T) {
	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{
				Stderr:   []byte("unexpected error in exporter"),
				ExitCode: 1,
			}, errors.New("exit status 1")
		},
	}

	a := newExternalExporter("/usr/local/bin/cova-exporter-myfw", mockCommander)
	req := &ApplyRequest{
		Operation: "apply",
		Blocks:    map[string][]RequestBlock{},
	}

	_, err := a.apply(t.Context(), req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed")
}

func TestExternalExporter_ApplyingRequestShouldReturnErrorWhenResponseIsInvalidJSON(t *testing.T) {
	mockCommander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{Stdout: []byte("not json"), ExitCode: 0}, nil
		},
	}

	a := newExternalExporter("/usr/local/bin/cova-exporter-myfw", mockCommander)
	req := &ApplyRequest{
		Operation: "apply",
		Blocks:    map[string][]RequestBlock{},
	}

	_, err := a.apply(t.Context(), req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshalling")
}
