package manifest_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/utils"
)

// fakeDirEntry implements os.DirEntry for test purposes.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string      { return f.name }
func (f fakeDirEntry) IsDir() bool       { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) {
	return fakeFileInfo(f), nil
}

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

// parseManifest is a test helper that parses YAML content into a RootManifest.
func parseManifest(t *testing.T, yamlContent string) *manifest.RootManifest {
	t.Helper()

	mockFS := &utils.MoqFileSystem{
		ReadFileContentsFunc: func(_ string) ([]byte, error) {
			return []byte(yamlContent), nil
		},
	}

	m, err := manifest.Parse(mockFS, "/repo")
	require.NoError(t, err)

	return m
}

func TestParse_ParsingManifestShouldSucceed(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectedOrg    string
		expectedCovens []string
		expectedSingle bool
	}{
		{
			"WhenSingleCovenIsSpecified",
			"org: acme\ncovens: platform\n",
			"acme",
			[]string{"platform"},
			true,
		},
		{
			"WhenMultipleCovensAreSpecified",
			"org: acme\ncovens:\n  - platform\n  - frontend\n",
			"acme",
			[]string{"platform", "frontend"},
			false,
		},
		{
			"WhenOrgContainsHyphens",
			"org: my-org\ncovens: my-coven\n",
			"my-org",
			[]string{"my-coven"},
			true,
		},
		{
			"WhenCovenListHasSingleEntry",
			"org: acme\ncovens:\n  - platform\n",
			"acme",
			[]string{"platform"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &utils.MoqFileSystem{
				ReadFileContentsFunc: func(_ string) ([]byte, error) {
					return []byte(tt.yaml), nil
				},
			}

			m, err := manifest.Parse(mockFS, "/repo")

			require.NoError(t, err)
			require.Equal(t, tt.expectedOrg, m.Org)
			require.Equal(t, tt.expectedCovens, m.Covens)
			require.Equal(t, tt.expectedSingle, m.IsSingleCoven())
		})
	}
}

func TestParse_ParsingManifestShouldReturnError(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		readErr     error
		errContains string
	}{
		{
			"WhenManifestFileCannotBeRead",
			"",
			errors.New("no such file"),
			"reading manifest",
		},
		{
			"WhenYAMLIsMalformed",
			"org: acme\ncovens: [invalid yaml",
			nil,
			"parsing manifest YAML",
		},
		{
			"WhenOrgIsEmpty",
			"org: \"\"\ncovens: platform\n",
			nil,
			"org must not be empty",
		},
		{
			"WhenOrgIsMissing",
			"covens: platform\n",
			nil,
			"org must not be empty",
		},
		{
			"WhenOrgContainsUppercase",
			"org: Acme\ncovens: platform\n",
			nil,
			"org \"Acme\" is invalid",
		},
		{
			"WhenOrgHasLeadingHyphen",
			"org: -acme\ncovens: platform\n",
			nil,
			"org \"-acme\" is invalid",
		},
		{
			"WhenOrgHasTrailingHyphen",
			"org: acme-\ncovens: platform\n",
			nil,
			"org \"acme-\" is invalid",
		},
		{
			"WhenOrgHasConsecutiveHyphens",
			"org: ac--me\ncovens: platform\n",
			nil,
			"org \"ac--me\" is invalid",
		},
		{
			"WhenOrgContainsSpecialCharacters",
			"org: acme_org\ncovens: platform\n",
			nil,
			"org \"acme_org\" is invalid",
		},
		{
			"WhenCovensIsEmptyString",
			"org: acme\ncovens: \"\"\n",
			nil,
			"covens[0]",
		},
		{
			"WhenCovensIsEmptyList",
			"org: acme\ncovens: []\n",
			nil,
			"covens must not be empty",
		},
		{
			"WhenCovenNameContainsUppercase",
			"org: acme\ncovens: Platform\n",
			nil,
			"covens[0] \"Platform\" is invalid",
		},
		{
			"WhenCovenNameHasLeadingHyphen",
			"org: acme\ncovens:\n  - -platform\n",
			nil,
			"covens[0] \"-platform\" is invalid",
		},
		{
			"WhenCovenNameHasTrailingHyphen",
			"org: acme\ncovens:\n  - platform-\n",
			nil,
			"covens[0] \"platform-\" is invalid",
		},
		{
			"WhenCovenNameHasConsecutiveHyphens",
			"org: acme\ncovens:\n  - plat--form\n",
			nil,
			"covens[0] \"plat--form\" is invalid",
		},
		{
			"WhenSecondCovenNameIsInvalid",
			"org: acme\ncovens:\n  - platform\n  - FRONTEND\n",
			nil,
			"covens[1] \"FRONTEND\" is invalid",
		},
		{
			"WhenCovensFieldIsAMap",
			"org: acme\ncovens:\n  key: value\n",
			nil,
			"covens must be a string or a list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &utils.MoqFileSystem{
				ReadFileContentsFunc: func(_ string) ([]byte, error) {
					if tt.readErr != nil {
						return nil, tt.readErr
					}

					return []byte(tt.yaml), nil
				},
			}

			m, err := manifest.Parse(mockFS, "/repo")

			require.Error(t, err)
			require.Nil(t, m)
			require.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestValidateCovenDirectories_ValidatingShouldSucceed(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		entries []os.DirEntry
	}{
		{
			"WhenSingleCovenSkipsValidation",
			"org: acme\ncovens: platform\n",
			nil,
		},
		{
			"WhenAllCovenDirectoriesExist",
			"org: acme\ncovens:\n  - platform\n  - frontend\n",
			[]os.DirEntry{
				fakeDirEntry{name: "platform", isDir: true},
				fakeDirEntry{name: "frontend", isDir: true},
				fakeDirEntry{name: "shared", isDir: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := parseManifest(t, tt.yaml)

			mockFS := &utils.MoqFileSystem{
				ReadDirectoryFunc: func(_ string) ([]os.DirEntry, error) {
					return tt.entries, nil
				},
			}

			err := manifest.ValidateCovenDirectories(mockFS, "/repo", m)

			require.NoError(t, err)
		})
	}
}

func TestValidateCovenDirectories_ValidatingShouldReturnError(t *testing.T) {
	tests := []struct {
		readErr     error
		name        string
		yaml        string
		errContains string
		entries     []os.DirEntry
	}{
		{
			name:        "WhenCovensDirectoryCannotBeRead",
			yaml:        "org: acme\ncovens:\n  - platform\n",
			readErr:     errors.New("permission denied"),
			errContains: "reading covens directory",
		},
		{
			name: "WhenCovenDirectoryIsMissing",
			yaml: "org: acme\ncovens:\n  - platform\n  - frontend\n",
			entries: []os.DirEntry{
				fakeDirEntry{name: "platform", isDir: true},
			},
			errContains: "coven \"frontend\" listed in manifest but no matching directory",
		},
		{
			name: "WhenMatchingEntryIsAFileNotDirectory",
			yaml: "org: acme\ncovens:\n  - platform\n",
			entries: []os.DirEntry{
				fakeDirEntry{name: "platform", isDir: false},
			},
			errContains: "coven \"platform\" listed in manifest but no matching directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := parseManifest(t, tt.yaml)

			mockFS := &utils.MoqFileSystem{
				ReadDirectoryFunc: func(_ string) ([]os.DirEntry, error) {
					if tt.readErr != nil {
						return nil, tt.readErr
					}

					return tt.entries, nil
				},
			}

			err := manifest.ValidateCovenDirectories(mockFS, "/repo", m)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errContains)
		})
	}
}
