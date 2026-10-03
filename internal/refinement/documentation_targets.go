package refinement

import (
	"errors"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// documentationTargetMarker matches the structural ways a refined package
// declares a Sniper-executable item: a `[documentation_target]` checkbox tag
// or a `task_type: documentation_target` handoff field. Prose that merely
// mentions the word does not count.
var documentationTargetMarker = handoff.DocumentationTargetMarker

// HasDocumentationTargets reports whether a refined tasks.md declares any
// documentation_target. A missing file declares none.
func HasDocumentationTargets(tasksPath string) (bool, error) {
	raw, err := os.ReadFile(tasksPath) //nolint:gosec // path is <base_path>/refined/<mission_id>/tasks.md
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("refinement: read %s: %w", tasksPath, err)
	}
	return documentationTargetMarker.Match(raw), nil
}

// DocumentationTargetPaths matches a backtick-quoted, path-shaped token —
// the convention every observed tasks.md in this workspace uses to name a
// documentation_target's file (e.g. "Write `docs/adr/0057-....md`"). It
// requires at least one path separator and a dotted extension so a bare
// identifier or command name in backticks is not mistaken for a target path.
// It validates and extracts the file paths named on each
// documentation_target line of tasksPath, in file order, deduplicated. A
// documentation_target without an explicit repository-relative path is an
// invalid package and returns an error. The same validator is used by
// normalization and execution authorization so a package cannot pass the gate
// and fail only when Sniper starts.
func DocumentationTargetPaths(tasksPath string) ([]string, error) {
	raw, err := os.ReadFile(tasksPath) //nolint:gosec // path is <base_path>/refined/<mission_id>/tasks.md
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("refinement: read %s: %w", tasksPath, err)
	}
	paths, err := handoff.ValidateDocumentationTargetContent(raw)
	if err != nil {
		return nil, fmt.Errorf("refinement: validate documentation target paths: %w", err)
	}
	return paths, nil
}

// ValidateDocumentationTargetContent validates and extracts every declared
// documentation target from tasks content. It is intentionally content-based
// so normalization can validate provider output before writing the canonical
// refined package.
func ValidateDocumentationTargetContent(raw []byte) ([]string, error) {
	paths, err := handoff.ValidateDocumentationTargetContent(raw)
	if err != nil {
		return nil, fmt.Errorf("refinement: validate documentation targets: %w", err)
	}
	return paths, nil
}
