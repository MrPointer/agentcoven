// Package manifest provides parsing and validation for coven repository manifests.
package manifest

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// ManifestFilename is the expected name of the root manifest file.
const ManifestFilename = "manifest.yaml"

// covensDirName is the directory under which multi-coven subdirectories live.
const covensDirName = "covens"

// segmentPattern matches a valid naming segment: lowercase alphanumeric with optional
// non-leading, non-trailing, non-consecutive hyphens.
var segmentPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// RootManifest represents a parsed and validated manifest.yaml.
type RootManifest struct {
	Org    string
	Covens []string
	single bool
}

// NewRootManifest creates a RootManifest with the given values.
func NewRootManifest(org string, covens []string, single bool) *RootManifest {
	return &RootManifest{
		Org:    org,
		Covens: covens,
		single: single,
	}
}

// IsSingleCoven reports whether the manifest declares a single-coven repository.
func (m *RootManifest) IsSingleCoven() bool {
	return m.single
}

// rawManifest is the intermediate YAML structure before validation.
type rawManifest struct {
	Org    string    `yaml:"org"`
	Covens rawCovens `yaml:"covens"`
}

// rawCovens handles the polymorphic covens field (string or list).
type rawCovens struct {
	names  []string
	single bool
}

// UnmarshalYAML implements custom unmarshalling for the covens field.
func (rc *rawCovens) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		rc.names = []string{value.Value}
		rc.single = true

		return nil
	case yaml.SequenceNode:
		var names []string
		if err := value.Decode(&names); err != nil {
			return fmt.Errorf("decoding covens list: %w", err)
		}

		rc.names = names
		rc.single = false

		return nil
	default:
		return fmt.Errorf("covens must be a string or a list, got YAML kind %d", value.Kind)
	}
}

// Parse reads and validates a manifest.yaml from the given repository root.
func Parse(fs utils.FileSystem, repoRoot string) (*RootManifest, error) {
	path := filepath.Join(repoRoot, ManifestFilename)

	data, err := fs.ReadFileContents(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing manifest YAML: %w", err)
	}

	if err := validateSegment(raw.Org, "org"); err != nil {
		return nil, err
	}

	if len(raw.Covens.names) == 0 {
		return nil, errors.New("covens must not be empty")
	}

	for i, name := range raw.Covens.names {
		if err := validateSegment(name, fmt.Sprintf("covens[%d]", i)); err != nil {
			return nil, err
		}
	}

	return &RootManifest{
		Org:    raw.Org,
		Covens: raw.Covens.names,
		single: raw.Covens.single,
	}, nil
}

// ValidateCovenDirectories checks that every coven listed in a multi-coven manifest
// has a corresponding directory under covens/ in the repository root.
func ValidateCovenDirectories(fs utils.FileSystem, repoRoot string, manifest *RootManifest) error {
	if manifest.IsSingleCoven() {
		return nil
	}

	entries, err := fs.ReadDirectory(filepath.Join(repoRoot, covensDirName))
	if err != nil {
		return fmt.Errorf("reading covens directory: %w", err)
	}

	dirs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs[entry.Name()] = struct{}{}
		}
	}

	for _, name := range manifest.Covens {
		if _, ok := dirs[name]; !ok {
			return fmt.Errorf("coven %q listed in manifest but no matching directory found at %s",
				name, filepath.Join(covensDirName, name))
		}
	}

	return nil
}

// validateSegment checks that a naming segment conforms to the spec:
// lowercase alphanumeric, optional hyphens, no leading/trailing/consecutive hyphens, non-empty.
func validateSegment(value, field string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", field)
	}

	if !segmentPattern.MatchString(value) {
		return fmt.Errorf("%s %q is invalid: must be lowercase alphanumeric with optional hyphens "+
			"(no leading/trailing/consecutive hyphens)", field, value)
	}

	return nil
}
