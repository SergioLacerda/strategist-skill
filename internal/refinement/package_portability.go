package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PortabilityFinding is an advisory package-durability observation. Findings
// do not block unrelated missions and never rewrite an existing package.
type PortabilityFinding struct {
	Path   string
	Reason string
}

// CheckPackagePortability reports absolute runtime-only pointers and evidence
// rows that contain no captured excerpt. It is intentionally advisory so old
// packages can be reviewed or migrated explicitly rather than rewritten during
// an unrelated install or mission.
func CheckPackagePortability(refined string) ([]PortabilityFinding, error) {
	analysisPath := filepath.Join(refined, "analysis.md")
	content, err := os.ReadFile(analysisPath) //nolint:gosec // refined is resolved by the caller
	if err != nil {
		return nil, fmt.Errorf("read analysis artifact: %w", err)
	}
	var findings []PortabilityFinding
	hasExcerpt := strings.Contains(string(content), "excerpt:")
	for _, line := range strings.Split(string(content), "\n") {
		finding, ok := packagePortabilityFinding(analysisPath, line, hasExcerpt)
		if ok {
			findings = append(findings, finding)
		}
		if finding.Reason == "evidence has no captured excerpt" {
			break
		}
	}
	return findings, nil
}

func packagePortabilityFinding(path, line string, hasExcerpt bool) (PortabilityFinding, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "provider_runtime:") && strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "provider_runtime:")), "/") {
		return PortabilityFinding{Path: path, Reason: "absolute provider_runtime"}, true
	}
	if strings.Contains(trimmed, "source_ref:") && !hasExcerpt {
		return PortabilityFinding{Path: path, Reason: "evidence has no captured excerpt"}, true
	}
	return PortabilityFinding{}, false
}
