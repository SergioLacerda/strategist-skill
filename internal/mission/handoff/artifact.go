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
	return validateRangerArtifact(path, missionID, []string{"ranger_pending", "ranger_done"})
}

// ValidateRangerArtifactForRefinement validates the same normalized discovery
// envelope after Archivist has claimed it. archivist_pending is a legitimate
// transient state at this boundary; accepting it here does not widen the
// earlier Ranger completion boundary.
func ValidateRangerArtifactForRefinement(path, missionID string) error {
	return validateRangerArtifact(path, missionID, []string{"ranger_pending", "ranger_done", "archivist_pending"})
}

func validateRangerArtifact(path, missionID string, statuses []string) error {
	content, err := readArtifact(path)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: read Ranger artifact: %w", err)
	}
	frontmatter, body, err := parseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	if err := validateIdentity(frontmatter, missionID, statuses); err != nil {
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

// RefinedPackage is the single typed projection authorized to cross the gate.
// Its target list is consumed by execution preparation instead of reparsing
// mutable tasks after lifecycle state has changed.
type RefinedPackage struct {
	MissionID            string
	DocumentationTargets []string
}

// LoadRefinedPackageForGate loads and validates the package authorized to cross
// the gate.
func LoadRefinedPackageForGate(refined, missionID string) (RefinedPackage, error) {
	analysisPath := filepath.Join(refined, "analysis.md")
	if err := validateArchivistAnalysis(analysisPath, missionID); err != nil {
		return RefinedPackage{}, err
	}
	files := make(map[string][]byte, len([]string{"analysis.md", "proposal.md", "design.md", "tasks.md"}))
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		raw, err := readArtifact(filepath.Join(refined, name))
		if err != nil {
			return RefinedPackage{}, fmt.Errorf("handoff_artifact_invalid: read Archivist %s: %w", name, err)
		}
		files[name] = raw
	}
	return ValidateRefinedPackageContent(files, missionID)
}

// ValidateRefinedPackageContent validates the staged package used by both
// provider publication and amendment planning before it reaches disk.
func ValidateRefinedPackageContent(files map[string][]byte, missionID string) (RefinedPackage, error) {
	frontmatter, err := validateRefinedAnalysis(files, missionID)
	if err != nil {
		return RefinedPackage{}, err
	}
	if err := validateRefinedSupportFiles(files); err != nil {
		return RefinedPackage{}, err
	}
	targets, err := validateRefinedTasks(files["tasks.md"])
	if err != nil {
		return RefinedPackage{}, err
	}
	if err := requireRefinedTargetFacts(frontmatter, targets); err != nil {
		return RefinedPackage{}, err
	}
	return RefinedPackage{MissionID: missionID, DocumentationTargets: targets}, nil
}

// ValidateRefinedPackageForGate validates the package at the gate boundary.
func ValidateRefinedPackageForGate(refined, missionID string) error {
	_, err := LoadRefinedPackageForGate(refined, missionID)
	return err
}

// ValidateArchivistPackage is retained as the boundary's historical name for
// callers outside the gate path. New package-to-gate callers use
// ValidateRefinedPackageForGate explicitly.
func ValidateArchivistPackage(refined, missionID string) error {
	return ValidateRefinedPackageForGate(refined, missionID)
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
	if _, declared := frontmatter[PolicyFactsKey]; declared {
		if _, err := ParsePolicyFacts(frontmatter); err != nil {
			return fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
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
