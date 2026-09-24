package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ValidateOpenSpecHealthcheck verifies the semantic root reported by OpenSpec
// against the physical provider runtime directory. OpenSpec keeps its config
// under .strategist/openspec while context resolves the containing
// .strategist directory as the semantic root.
func ValidateOpenSpecHealthcheck(output []byte, runtimeRoot string) error {
	var contextResult struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
	}
	payload := output
	if start := bytes.IndexByte(output, '{'); start >= 0 {
		if end := bytes.LastIndexByte(output, '}'); end > start {
			payload = output[start : end+1]
		}
	}
	if err := json.Unmarshal(payload, &contextResult); err != nil {
		return fmt.Errorf("parse OpenSpec context: %w", err)
	}
	if contextResult.Root.Path == "" {
		return fmt.Errorf("OpenSpec context did not report root.path")
	}
	expected := canonicalRuntimePath(filepath.Dir(filepath.Clean(runtimeRoot)))
	observed := canonicalRuntimePath(contextResult.Root.Path)
	if observed != expected && !sameDirectory(expected, observed) {
		return fmt.Errorf("OpenSpec semantic root mismatch: expected %s, got %s (resolved paths differ; verify --root points at the prepared .strategist directory)", expected, observed)
	}
	return nil
}

// canonicalRuntimePath makes a path comparable regardless of spelling:
// relative, dotted, trailing-slash, or symlinked forms resolve to the same
// absolute physical path. A path that cannot be resolved (for example one
// that does not exist) falls back to its cleaned absolute form.
func canonicalRuntimePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// sameDirectory covers filesystems where two canonical spellings still name
// one directory (case-insensitive volumes, bind mounts).
func sameDirectory(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}
