package install

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// sidecarGeneratorVersion is stamped in the generated header. Bump it when the
// generator's derivation rules change (ADR-0061).
const sidecarGeneratorVersion = "v1"

// Sidecar scaffold outcomes.
const (
	SidecarCreated     = "created"
	SidecarUnchanged   = "unchanged"
	SidecarOverwritten = "overwritten"
)

// SidecarScaffoldOptions declares what the operator confirms for one package.
// Roles, Slots and (for openspec_root) the provenance runtime block are never
// inferred (ADR-0060 Decision 4, ADR-0061 Decisions 1-7).
type SidecarScaffoldOptions struct {
	PackageDir     string
	Roles          []string
	Slots          []string
	Version        string // used only when SKILL.md carries no metadata.version
	Runtime        string // "embedded" (default) or "openspec_root"
	ProvenancePath string
	Default        bool
	Force          bool
	Check          bool
}

// SidecarScaffoldResult reports what the command did to strategist.yaml.
type SidecarScaffoldResult struct {
	Path   string
	Status string
}

// SidecarScaffoldError carries one of the ADR-0061 failure codes.
type SidecarScaffoldError struct {
	Code   string
	Detail string
}

func (e *SidecarScaffoldError) Error() string {
	return e.Code + ": " + e.Detail
}

func scaffoldError(code, format string, args ...any) error {
	return &SidecarScaffoldError{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// ScaffoldSidecar generates <PackageDir>/strategist.yaml deterministically from
// the operator's declaration and the package. It writes only that file, never
// overwrites a differing one without Force, and with Check writes nothing.
func ScaffoldSidecar(opts SidecarScaffoldOptions) (SidecarScaffoldResult, error) {
	content, err := renderSidecar(opts)
	if err != nil {
		return SidecarScaffoldResult{}, err
	}
	path := filepath.Join(opts.PackageDir, externalSkillAdapterFileName)
	existing, err := os.ReadFile(path) //nolint:gosec // G304: operator-declared package directory
	switch {
	case errors.Is(err, os.ErrNotExist):
		return createSidecar(path, content, opts.Check)
	case err != nil:
		return SidecarScaffoldResult{}, fmt.Errorf("read %s: %w", path, err)
	case bytes.Equal(existing, content):
		return SidecarScaffoldResult{Path: path, Status: SidecarUnchanged}, nil
	}
	return replaceSidecar(path, content, opts)
}

func createSidecar(path string, content []byte, check bool) (SidecarScaffoldResult, error) {
	if check {
		return SidecarScaffoldResult{}, scaffoldError("sidecar_drift", "%s does not exist", path)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil { //nolint:gosec // G306: generated manifest is not sensitive
		return SidecarScaffoldResult{}, fmt.Errorf("write %s: %w", path, err)
	}
	return SidecarScaffoldResult{Path: path, Status: SidecarCreated}, nil
}

func replaceSidecar(path string, content []byte, opts SidecarScaffoldOptions) (SidecarScaffoldResult, error) {
	if opts.Check {
		return SidecarScaffoldResult{}, scaffoldError("sidecar_drift", "%s differs from the regenerated sidecar", path)
	}
	if !opts.Force {
		return SidecarScaffoldResult{}, scaffoldError("sidecar_exists_differs", "%s already exists and differs; pass --force to overwrite", path)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil { //nolint:gosec // G306: generated manifest is not sensitive
		return SidecarScaffoldResult{}, fmt.Errorf("write %s: %w", path, err)
	}
	return SidecarScaffoldResult{Path: path, Status: SidecarOverwritten}, nil
}
