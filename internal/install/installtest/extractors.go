package installtest

import (
	"errors"
	"fmt"
)

// PartialExtractor serves MinimalExtractor files but fails for the given
// path(s). FailPaths supersedes the legacy single FailPath field when set.
type PartialExtractor struct {
	FailPath  string
	FailPaths map[string]bool
}

// Extract delegates to MinimalExtractor.
func (p PartialExtractor) Extract(targetDir string, overwrite bool) error {
	return MinimalExtractor{}.Extract(targetDir, overwrite)
}

// ReadFile serves MinimalExtractor files except for the configured failing paths.
func (p PartialExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == p.FailPath || p.FailPaths[relPath] {
		return nil, fmt.Errorf("PartialExtractor: injected failure for %s", relPath)
	}
	return MinimalExtractor{}.ReadFile(relPath)
}

// FixedFileExtractor serves fixed content (or a fixed error) per relative
// path, for exercising loaders' error branches directly without needing a
// full install fixture.
type FixedFileExtractor map[string]FixedFileEntry

// FixedFileEntry is one served file or error.
type FixedFileEntry struct {
	Data []byte
	Err  error
}

// Extract is a no-op.
func (f FixedFileExtractor) Extract(string, bool) error { return nil }

// ReadFile serves the configured entry for relPath.
func (f FixedFileExtractor) ReadFile(relPath string) ([]byte, error) {
	entry, ok := f[relPath]
	if !ok {
		return nil, errors.New("FixedFileExtractor: no entry for " + relPath)
	}
	return entry.Data, entry.Err
}

// AlwaysErrExtractor fails every ReadFile call, used to exercise loadSkillConfig's
// extractor-error fallback distinct from its YAML-parse-error fallback.
type AlwaysErrExtractor struct{}

// Extract is a no-op.
func (AlwaysErrExtractor) Extract(_ string, _ bool) error { return nil }

// ReadFile always fails.
func (AlwaysErrExtractor) ReadFile(relPath string) ([]byte, error) {
	return nil, fmt.Errorf("AlwaysErrExtractor: read error for %s", relPath)
}
