package refinement

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

// documentationTargetMarker matches the structural ways a refined package
// declares a Sniper-executable item: a `[documentation_target]` checkbox tag
// or a `task_type: documentation_target` handoff field. Prose that merely
// mentions the word does not count.
var documentationTargetMarker = regexp.MustCompile(`\[documentation_target\]|task_type:\s*documentation_target\b`)

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
