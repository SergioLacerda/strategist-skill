package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// RuntimePathResolver is the host filesystem port used to compare the
// semantic root reported by a runtime. Domain owns the comparison contract;
// adapters own symlink and filesystem inspection.
type RuntimePathResolver interface {
	Canonical(path string) string
	SameDirectory(left, right string) bool
}

// ValidateOpenSpecHealthcheck verifies the semantic root reported by OpenSpec
// against the physical provider runtime directory. OpenSpec keeps its config
// under .strategist/openspec while context resolves the containing
// .strategist directory as the semantic root.
func ValidateOpenSpecHealthcheck(output []byte, runtimeRoot string, resolver RuntimePathResolver) error {
	observedRoot, err := parseOpenSpecContextRoot(output)
	if err != nil {
		return err
	}
	if resolver == nil {
		return fmt.Errorf("OpenSpec path resolver is unavailable")
	}
	expected := resolver.Canonical(filepath.Dir(filepath.Clean(runtimeRoot)))
	observed := resolver.Canonical(observedRoot)
	if observed != expected && !resolver.SameDirectory(expected, observed) {
		return fmt.Errorf("OpenSpec semantic root mismatch: expected %s, got %s (resolved paths differ; verify --root points at the prepared .strategist directory)", expected, observed)
	}
	return nil
}

func parseOpenSpecContextRoot(output []byte) (string, error) {
	var contextResult struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
	}
	payload := openSpecJSONPayload(output)
	if err := json.Unmarshal(payload, &contextResult); err != nil {
		return "", fmt.Errorf("parse OpenSpec context: %w", err)
	}
	if contextResult.Root.Path == "" {
		return "", fmt.Errorf("OpenSpec context did not report root.path")
	}
	return contextResult.Root.Path, nil
}

func openSpecJSONPayload(output []byte) []byte {
	start := bytes.IndexByte(output, '{')
	if start < 0 {
		return output
	}
	end := bytes.LastIndexByte(output, '}')
	if end <= start {
		return output
	}
	return output[start : end+1]
}
