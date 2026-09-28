package refinement

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
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

// documentationTargetPathToken matches a backtick-quoted, path-shaped token —
// the convention every observed tasks.md in this workspace uses to name a
// documentation_target's file (e.g. "Write `docs/adr/0057-....md`"). It
// requires at least one path separator and a dotted extension so a bare
// identifier or command name in backticks is not mistaken for a target path.
var documentationTargetPathToken = regexp.MustCompile("`([\\w./-]+/[\\w.-]+\\.[a-zA-Z0-9]+)`")

// DocumentationTargetPaths extracts the file paths named on each
// [documentation_target] line of tasksPath, in file order, deduplicated. A
// missing file, or a documentation_target line with no backtick-quoted
// path-shaped token, yields no paths for that line rather than an error —
// this is a best-effort extraction for the Sniper claim tripwire (ADR-0057
// §2.3), not a schema-validated field, and a line this cannot parse is a gap
// in coverage, not a fatal one.
func DocumentationTargetPaths(tasksPath string) ([]string, error) {
	raw, err := os.ReadFile(tasksPath) //nolint:gosec // path is <base_path>/refined/<mission_id>/tasks.md
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("refinement: read %s: %w", tasksPath, err)
	}
	seen := make(map[string]bool)
	var paths []string
	for _, line := range strings.Split(string(raw), "\n") {
		path, ok := documentationTargetPath(line)
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths, nil
}

func documentationTargetPath(line string) (string, bool) {
	if !documentationTargetMarker.MatchString(line) {
		return "", false
	}
	match := documentationTargetPathToken.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	return match[1], true
}
