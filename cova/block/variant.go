package block

import (
	"fmt"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// variantsFilename is the name of the file that declares framework variants for a block.
const variantsFilename = "variants.yaml"

// rawVariants is the intermediate YAML structure for variants.yaml.
type rawVariants struct {
	Variants []string `yaml:"variants"`
}

// ResolveVariant resolves the effective source directory for a block given an adapter name.
//
// If the block has no variants.yaml, the original sourceDir is returned as-is with include=true.
// If variants.yaml exists and adapterName is listed, the variant subdirectory is returned with include=true.
// If variants.yaml exists but adapterName is not listed, empty string is returned with include=false.
//
// sourceDir must be relative to covenRoot (e.g., "skills/acme-platform-deploy-pipeline").
// The resolved path is also relative to covenRoot (e.g., "skills/acme-platform-deploy-pipeline/claude-code").
func ResolveVariant(fs utils.FileSystem, covenRoot, sourceDir, adapterName string) (string, bool, error) {
	variantsPath := filepath.Join(covenRoot, sourceDir, variantsFilename)

	exists, err := fs.PathExists(variantsPath)
	if err != nil {
		return "", false, fmt.Errorf("checking variants file at %q: %w", variantsPath, err)
	}

	if !exists {
		return sourceDir, true, nil
	}

	data, err := fs.ReadFileContents(variantsPath)
	if err != nil {
		return "", false, fmt.Errorf("reading variants file at %q: %w", variantsPath, err)
	}

	var raw rawVariants
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", false, fmt.Errorf("parsing variants file at %q: %w", variantsPath, err)
	}

	if slices.Contains(raw.Variants, adapterName) {
		return filepath.Join(sourceDir, adapterName), true, nil
	}

	return "", false, nil
}
