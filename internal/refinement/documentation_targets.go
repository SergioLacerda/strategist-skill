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

// DocumentationTargetPaths validates and extracts the file paths named on each
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
	return ValidateDocumentationTargetContent(raw)
}

// ValidateDocumentationTargetContent validates and extracts every declared
// documentation target from tasks content. It is intentionally content-based
// so normalization can validate provider output before writing the canonical
// refined package.
func ValidateDocumentationTargetContent(raw []byte) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	for lineNumber, line := range strings.Split(string(raw), "\n") {
		if !documentationTargetMarker.MatchString(line) {
			continue
		}
		path, ok := documentationTargetPath(line)
		if !ok {
			return nil, fmt.Errorf("refinement: documentation_target on line %d requires an explicit backtick-quoted repository-relative path", lineNumber+1)
		}
		if seen[path] {
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
	path := match[1]
	if unsafeDocumentationPath(path) {
		return "", false
	}
	return path, true
}

func unsafeDocumentationPath(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "./") || strings.Contains(path, "\\") {
		return true
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
