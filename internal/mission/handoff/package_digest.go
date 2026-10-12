package handoff

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PackageDigest is the content identity of a refined package. It covers
// proposal.md, design.md and tasks.md in full and analysis.md's body plus its
// typed facts, but not the rest of analysis.md's frontmatter: mission_status
// legitimately changes as the mission advances, and that must not make an
// unchanged package look stale.
func PackageDigest(refined string) (string, error) {
	hash := sha256.New()
	analysis, err := readArtifact(filepath.Join(refined, "analysis.md"))
	if err != nil {
		return "", fmt.Errorf("handoff_artifact_invalid: read Archivist analysis: %w", err)
	}
	frontmatter, body, err := parseFrontmatter(analysis)
	if err != nil {
		return "", fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	facts, err := yaml.Marshal(frontmatter[PolicyFactsKey])
	if err != nil {
		return "", fmt.Errorf("handoff_policy_facts_invalid: encode %s: %w", PolicyFactsKey, err)
	}
	writeDigestPart(hash, "analysis.md#body", body)
	writeDigestPart(hash, "analysis.md#"+PolicyFactsKey, facts)
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		content, err := os.ReadFile(filepath.Join(refined, name)) //nolint:gosec // refined is resolved by the mission boundary
		if err != nil {
			return "", fmt.Errorf("handoff_artifact_invalid: read Archivist %s: %w", name, err)
		}
		writeDigestPart(hash, name, content)
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func writeDigestPart(hash interface{ Write([]byte) (int, error) }, name string, content []byte) {
	_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", name, len(content)) //nolint:errcheck // hash writes never fail
	_, _ = hash.Write(content)                                   //nolint:errcheck // hash writes never fail
}
