package handoff

import (
	"fmt"
	"regexp"
	"strings"
)

// DocumentationTargetMarker matches the structural ways a refined package
// declares a Sniper-executable item. Prose that merely mentions the term does
// not count as a target.
var DocumentationTargetMarker = regexp.MustCompile(`\[documentation_target\]|task_type:\s*documentation_target\b`)

var documentationTargetPathToken = regexp.MustCompile("`([\\w./-]+/[\\w.-]+\\.[a-zA-Z0-9]+)`")

// ValidateDocumentationTargetContent validates and extracts every declared
// documentation target from tasks content. Keeping this in handoff lets the
// same boundary protect publication, amendment, gate acceptance, and
// execution authorization without an import cycle.
func ValidateDocumentationTargetContent(raw []byte) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	for lineNumber, line := range strings.Split(string(raw), "\n") {
		if !DocumentationTargetMarker.MatchString(line) {
			continue
		}
		path, ok := documentationTargetPath(line)
		if !ok {
			return nil, fmt.Errorf("handoff_artifact_invalid: documentation_target on line %d requires an explicit backtick-quoted repository-relative path", lineNumber+1)
		}
		if seen[path] {
			return nil, fmt.Errorf("handoff_artifact_invalid: documentation_target on line %d duplicates executable target %q", lineNumber+1, path)
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths, nil
}

func documentationTargetPath(line string) (string, bool) {
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
