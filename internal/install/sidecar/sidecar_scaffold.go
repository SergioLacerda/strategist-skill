package sidecar

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
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

// ScaffoldOptions declares what the operator confirms for one package.
// Roles, Slots and (for openspec_root) the provenance runtime block are never
// inferred (ADR-0060 Decision 4, ADR-0061 Decisions 1-7).
type ScaffoldOptions struct {
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

// ScaffoldResult reports what the command did to strategist.yaml.
type ScaffoldResult struct {
	Path   string
	Status string
}

// ScaffoldError carries one of the ADR-0061 failure codes.
type ScaffoldError struct {
	Code   string
	Detail string
}

func (e *ScaffoldError) Error() string {
	return e.Code + ": " + e.Detail
}

func scaffoldError(code, format string, args ...any) error {
	return &ScaffoldError{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// ScaffoldSidecar generates <PackageDir>/strategist.yaml deterministically from
// the operator's declaration and the package. It writes only that file, never
// overwrites a differing one without Force, and with Check writes nothing.
func ScaffoldSidecar(opts ScaffoldOptions) (ScaffoldResult, error) {
	content, err := renderSidecar(opts)
	if err != nil {
		return ScaffoldResult{}, err
	}
	path := filepath.Join(opts.PackageDir, plugincatalog.ExternalSkillAdapterFileName)
	existing, err := os.ReadFile(path) //nolint:gosec // G304: operator-declared package directory
	switch {
	case errors.Is(err, os.ErrNotExist):
		return createSidecar(path, content, opts.Check)
	case err != nil:
		return ScaffoldResult{}, fmt.Errorf("read %s: %w", path, err)
	case bytes.Equal(existing, content):
		return ScaffoldResult{Path: path, Status: SidecarUnchanged}, nil
	}
	return ReplaceSidecar(path, content, opts)
}

func createSidecar(path string, content []byte, check bool) (ScaffoldResult, error) {
	if check {
		return ScaffoldResult{}, scaffoldError("sidecar_drift", "%s does not exist", path)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil { //nolint:gosec // G306: generated manifest is not sensitive
		return ScaffoldResult{}, fmt.Errorf("write %s: %w", path, err)
	}
	return ScaffoldResult{Path: path, Status: SidecarCreated}, nil
}

// ReplaceSidecar replaces the sidecar at path with content; it is refused in check mode and without Force.
func ReplaceSidecar(path string, content []byte, opts ScaffoldOptions) (ScaffoldResult, error) {
	if opts.Check {
		return ScaffoldResult{}, scaffoldError("sidecar_drift", "%s differs from the regenerated sidecar", path)
	}
	if !opts.Force {
		return ScaffoldResult{}, scaffoldError("sidecar_exists_differs", "%s already exists and differs; pass --force to overwrite", path)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil { //nolint:gosec // G306: generated manifest is not sensitive
		return ScaffoldResult{}, fmt.Errorf("write %s: %w", path, err)
	}
	return ScaffoldResult{Path: path, Status: SidecarOverwritten}, nil
}
