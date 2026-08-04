package eval

import (
	"regexp"
	"strings"
)

// sourceCitationPattern matches file-path-looking tokens ending in a
// documentation/source extension, used by AssertSourceCitations.
var sourceCitationPattern = regexp.MustCompile(`[\w./-]+\.(?:md|go|yaml|yml|txt)\b`)

// hasHeading reports whether content contains a markdown heading line
// (starting with one or more '#') whose text contains section.
func hasHeading(content, section string) bool {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			continue
		}
		t = strings.TrimSpace(strings.TrimLeft(t, "#"))
		if strings.Contains(t, section) {
			return true
		}
	}
	return false
}

// countSourceCitations counts distinct source-path-looking tokens in content.
func countSourceCitations(content string) int {
	matches := sourceCitationPattern.FindAllString(content, -1)
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		seen[m] = true
	}
	return len(seen)
}

// splitCSV splits a comma-separated Value into trimmed, non-empty parts.
func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// truncate shortens s for readable violation messages.
func truncate(s string) string {
	const maxLen = 120
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
