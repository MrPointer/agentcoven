// Package exporter translates cova blocks into agent-specific file placements.
package exporter

import "context"

// RequestManifest holds the coven manifest metadata sent in an apply request.
type RequestManifest struct {
	// Org is the organization name from the coven manifest.
	Org string `json:"org"`

	// Coven is the coven name from the manifest.
	Coven string `json:"coven"`
}

// RequestBlock represents a single block entry in an apply request.
type RequestBlock struct {
	// Name is the namespaced block name.
	Name string `json:"name"`

	// Source is the path of the block directory relative to the workspace root.
	Source string `json:"source"`
}

// ApplyRequest is the payload sent to an exporter for an apply operation.
type ApplyRequest struct {
	Blocks       map[string][]RequestBlock `json:"blocks"`
	Manifest     RequestManifest           `json:"manifest"`
	Operation    string                    `json:"operation"`
	Subscription string                    `json:"subscription"`
	Workspace    string                    `json:"workspace"`
}

// Placement describes where a single file from a block should be written.
type Placement struct {
	// Path is the absolute target path where the file should be written.
	Path string `json:"path"`

	// Source is the source file path relative to the workspace root.
	Source string `json:"source"`
}

// BlockResult holds the placement outcome for one input block.
type BlockResult struct {
	Error      *string     `json:"error"`
	Name       string      `json:"name"`
	Placements []Placement `json:"placements"`
}

// ApplyResponse is the payload returned by an exporter after an apply operation.
type ApplyResponse struct {
	// Results contains one entry per input block.
	Results []BlockResult `json:"results"`
}

// RemoveRequestBlock represents a single block entry in a remove request.
type RemoveRequestBlock struct {
	// Name is the namespaced block name.
	Name string `json:"name"`

	// Paths contains the absolute file paths that were placed for this block.
	Paths []string `json:"paths"`
}

// RemoveRequest is the payload sent to an exporter for a remove operation.
type RemoveRequest struct {
	Blocks       map[string][]RemoveRequestBlock `json:"blocks"`
	Manifest     RequestManifest                 `json:"manifest"`
	Operation    string                          `json:"operation"`
	Subscription string                          `json:"subscription"`
}

// RemoveBlockResult holds the outcome for one input block in a remove operation.
type RemoveBlockResult struct {
	Error *string `json:"error"`
	Name  string  `json:"name"`
}

// RemoveResponse is the payload returned by an exporter after a remove operation.
type RemoveResponse struct {
	// Results contains one entry per input block.
	Results []RemoveBlockResult `json:"results"`
}

// exporter is the internal interface implemented by all built-in exporters.
// It does not carry the agent parameter because the dispatcher already resolved
// which exporter to call.
type exporter interface {
	apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error)
	remove(ctx context.Context, req *RemoveRequest) (*RemoveResponse, error)
}
