package add

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

func TestBuildSubscriptions_BuildingSingleCovenShouldReturnOneSubscription(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "", nil)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "acme-platform", subs[0].Name)
	require.Equal(t, "https://github.com/acme/blocks.git", subs[0].Repo)
	require.Empty(t, subs[0].Path)
	require.Empty(t, subs[0].Ref)
}

func TestBuildSubscriptions_BuildingSingleCovenWithRefShouldSetRef(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "v1.0.0", nil)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "v1.0.0", subs[0].Ref)
}

func TestBuildSubscriptions_BuildingSingleCovenShouldIgnoreExtraCovenArgs(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"extra", "args"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "acme-platform", subs[0].Name)
}

func TestBuildSubscriptions_BuildingMultiCovenWithValidArgsShouldReturnSubscriptions(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend", "backend"}, false)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"platform", "frontend"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 2)
	require.Equal(t, "acme-platform", subs[0].Name)
	require.Equal(t, "covens/platform", subs[0].Path)
	require.Equal(t, "acme-frontend", subs[1].Name)
	require.Equal(t, "covens/frontend", subs[1].Path)
}

func TestBuildSubscriptions_BuildingMultiCovenWithNoArgsShouldReturnError(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	_, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple covens")
	require.Contains(t, err.Error(), "platform")
	require.Contains(t, err.Error(), "frontend")
}

func TestBuildSubscriptions_BuildingMultiCovenWithUnknownNameShouldReturnError(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	_, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"nonexistent"},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent")
	require.Contains(t, err.Error(), "not found")
}

func TestBuildSubscriptions_BuildingMultiCovenWithRefShouldSetRefOnAll(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "main", []string{"platform", "frontend"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 2)
	require.Equal(t, "main", subs[0].Ref)
	require.Equal(t, "main", subs[1].Ref)
}

func TestLogUpsertResult_LoggingShouldUseCorrectLevel(t *testing.T) {
	tests := []struct {
		name           string
		expectedSubstr string
		result         config.UpsertResult
		expectSuccess  bool
		expectInfo     bool
	}{
		{"WhenAdded", "added", config.UpsertAdded, true, false},
		{"WhenUpdated", "updated", config.UpsertUpdated, false, true},
		{"WhenNoOp", "already up to date", config.UpsertNoOp, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var successMsg, infoMsg string

			mock := &logger.MoqLogger{
				SuccessFunc: func(format string, _ ...any) {
					successMsg = format
				},
				InfoFunc: func(format string, _ ...any) {
					infoMsg = format
				},
				TraceFunc:   func(string, ...any) {},
				DebugFunc:   func(string, ...any) {},
				WarningFunc: func(string, ...any) {},
				ErrorFunc:   func(string, ...any) {},
				CloseFunc:   func() error { return nil },
			}

			LogUpsertResult(mock, "test-sub", tt.result)

			if tt.expectSuccess {
				require.Contains(t, successMsg, tt.expectedSubstr)
			}

			if tt.expectInfo {
				require.Contains(t, infoMsg, tt.expectedSubstr)
			}
		})
	}
}
