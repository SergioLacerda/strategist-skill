package handoff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// RangerToArchivistArtifact is the discovery artifact boundary.
	RangerToArchivistArtifact = "ranger_to_archivist"
	// ArchivistToSniperArtifact is the refined package boundary.
	ArchivistToSniperArtifact = "archivist_to_sniper"
)

var requiredRangerSections = []string{
	"mission_objective", "known_facts", "confidence_summary", "handoff",
}

// ValidateRangerArtifact validates the identity envelope and the required
// handoff sections before Archivist consumes a discovery artifact. Provider
// compatibility is deliberately not checked here; that belongs to the
// provider binding boundary.
func ValidateRangerArtifact(path, missionID string) error {
	content, err := readArtifact(path)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: read Ranger artifact: %w", err)
	}
	frontmatter, body, err := parseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	if err := validateIdentity(frontmatter, missionID, []string{"ranger_pending", "ranger_done"}); err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	if _, ok := frontmatter["sources_consulted"].([]any); !ok {
		return fmt.Errorf("handoff_artifact_invalid: Ranger artifact is missing list field %q (use an empty list when no source was opened)", "sources_consulted")
	}
	for _, section := range requiredRangerSections {
		if !hasSection(body, section) {
			return fmt.Errorf("handoff_artifact_invalid: Ranger artifact is missing section %q", section)
		}
	}
	return nil
}

// HasHandoffMetadata reports whether an artifact carries the normalized
// Strategist envelope. Legacy provider fixtures and pre-contract packages may
// still be plain Markdown; callers use this only to select the compatibility
// path while new normalized artifacts are always validated strictly.
func HasHandoffMetadata(path string) bool {
	content, err := os.ReadFile(path) //nolint:gosec // caller resolves the mission path
	if err != nil {
		return false
	}
	return bytes.HasPrefix(bytes.TrimSpace(content), []byte("---\n"))
}

// HasNormalizedRangerMetadata identifies the provider-normalized discovery
// envelope. It is narrower than HasHandoffMetadata because older bridge tests
// and legacy packages can have generic Markdown frontmatter without the
// provider-owned schema marker.
func HasNormalizedRangerMetadata(path string) bool {
	content, err := os.ReadFile(path) //nolint:gosec // caller resolves the mission path
	if err != nil {
		return false
	}
	return HasHandoffMetadata(path) && bytes.Contains(content, []byte("schema_version:"))
}

// ValidateArchivistPackage validates the package consumed at the Archivist to
// Sniper boundary. It accepts Markdown task plans because OpenSpec's task
// artifact is intentionally human-readable, but every task must carry an
// explicit handoff classification.
func ValidateArchivistPackage(refined, missionID string) error {
	analysisPath := filepath.Join(refined, "analysis.md")
	if err := validateArchivistAnalysis(analysisPath, missionID); err != nil {
		return err
	}
	if err := validateArchivistFiles(refined); err != nil {
		return err
	}
	tasks, err := readArchivistTasks(refined)
	if err != nil {
		return err
	}
	if err := validateTaskClassifications(tasks); err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Archivist tasks: %w", err)
	}
	return nil
}

func validateArchivistAnalysis(path, missionID string) error {
	content, err := readArtifact(path)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: read Archivist analysis: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	if err := validateIdentity(frontmatter, missionID, []string{
		"archivist_done", "gate_pending", "gate_analysis_accepted", "sniper_running", "documentation_applied",
	}); err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	return nil
}

func validateArchivistFiles(refined string) error {
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		content, err := readArtifact(filepath.Join(refined, name))
		if err != nil {
			return fmt.Errorf("handoff_artifact_invalid: read Archivist %s: %w", name, err)
		}
		if len(bytes.TrimSpace(content)) == 0 {
			return fmt.Errorf("handoff_artifact_invalid: Archivist %s is empty", name)
		}
	}
	return nil
}

func readArchivistTasks(refined string) ([]byte, error) {
	tasks, err := os.ReadFile(filepath.Join(refined, "tasks.md")) //nolint:gosec // refined is resolved by the mission boundary
	if err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: read Archivist tasks: %w", err)
	}
	return tasks, nil
}
