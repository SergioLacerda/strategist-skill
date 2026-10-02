// Package refinement publishes provider planning output into the Strategist
// workspace lifecycle.
package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// OpenSpecInput identifies one completed provider change and its Strategist
// mission. BasePath and RuntimeRoot must be resolved absolute paths.
type OpenSpecInput struct {
	MissionID           string
	BasePath            string
	RuntimeRoot         string
	ChangeID            string
	PendingAnalysisPath string
	// RecordConfidence persists Archivist's own boundary claim before any
	// package is published. The callback is required so refinement cannot
	// finish silently when telemetry storage is unavailable.
	RecordConfidence func(domain.ConfidenceClaim, []domain.Evidence) error
	// HandoffFacts is the optional typed handoff_policy_facts mapping Archivist
	// declares at publication. It is validated with handoff.ParsePolicyFacts and
	// written into the published analysis.md frontmatter; nil publishes none and
	// leaves the package without an evaluable handoff policy.
	HandoffFacts map[string]any
	// RecordPublication persists a publication event separately from Sniper
	// materialization telemetry. Nil is permitted for library-only callers.
	RecordPublication func(PackagePublication) error
}

// PackagePublication identifies a newly published canonical package.
type PackagePublication struct {
	MissionID, ProviderChangeID, SourceDigest, PackageDigest string
	PublishedAt                                              time.Time
}

// OpenSpecResult describes the canonical package published by the bridge.
type OpenSpecResult struct {
	RefinedPath      string
	ProviderChangeID string
	ProviderRuntime  string
	PublishedAt      time.Time
}

var canonicalFiles = []string{"analysis.md", "proposal.md", "design.md", "tasks.md"}

// NormalizeOpenSpec validates a completed OpenSpec change and atomically
// publishes its four canonical files. Provider spec files and archive history
// are never copied as files; the specs' requirements and scenarios are carried
// into design.md under "Acceptance scenarios". After publishing, the change is
// moved to changes/archive/. Existing identical output is idempotent;
// conflicting output fails closed.
func NormalizeOpenSpec(input OpenSpecInput) (OpenSpecResult, error) {
	if err := validateInput(input); err != nil {
		return OpenSpecResult{}, err
	}
	changeDir, err := validatedChangeDir(input)
	if err != nil {
		return OpenSpecResult{}, err
	}
	contents, err := readContents(changeDir, input)
	if err != nil {
		return OpenSpecResult{}, err
	}
	if _, err := ValidateDocumentationTargetContent(contents["tasks.md"]); err != nil {
		return OpenSpecResult{}, fmt.Errorf("openspec bridge: validate documentation targets: %w", err)
	}

	return publishOpenSpec(input, changeDir, contents)
}

func publishOpenSpec(input OpenSpecInput, changeDir string, contents map[string][]byte) (OpenSpecResult, error) {
	refined, err := validatedRefinedPath(input)
	if err != nil {
		return OpenSpecResult{}, err
	}
	if err := recordArchivistConfidence(input); err != nil {
		return OpenSpecResult{}, fmt.Errorf("openspec bridge: record Archivist confidence: %w", err)
	}
	published, err := publishOrPromote(refined, input.PendingAnalysisPath, contents)
	if err != nil {
		return OpenSpecResult{}, err
	}
	if err := recordPackagePublication(input, contents, published); err != nil {
		return OpenSpecResult{}, err
	}
	if err := archiveChange(input.RuntimeRoot, changeDir, input.ChangeID); err != nil {
		return OpenSpecResult{}, err
	}
	return result(refined, input), nil
}

func recordPackagePublication(input OpenSpecInput, contents map[string][]byte, published bool) error {
	if !published || input.RecordPublication == nil {
		return nil
	}
	publication := PackagePublication{
		MissionID: input.MissionID, ProviderChangeID: input.ChangeID,
		SourceDigest: digest(contents["analysis.md"]), PackageDigest: canonicalPackageDigest(contents), PublishedAt: time.Now().UTC(),
	}
	if err := input.RecordPublication(publication); err != nil {
		return fmt.Errorf("openspec bridge: record package publication: %w", err)
	}
	return nil
}

func publishOrPromote(refined, pending string, contents map[string][]byte) (bool, error) {
	existing, err := existingPackage(refined, contents)
	if err != nil {
		return false, err
	}
	if existing {
		return false, removePending(pending)
	}
	return true, publish(refined, pending, contents)
}

func readContents(changeDir string, input OpenSpecInput) (map[string][]byte, error) {
	if err := requireFiles(changeDir, canonicalFiles[1:]); err != nil {
		return nil, fmt.Errorf("openspec bridge: incomplete change: %w", err)
	}
	analysis, err := readPendingAnalysis(input)
	if err != nil {
		return nil, err
	}
	withMetadata, err := addMetadata(analysis, input)
	if err != nil {
		return nil, err
	}
	contents := map[string][]byte{"analysis.md": withMetadata}
	canonical, err := readCanonicalFiles(changeDir)
	if err != nil {
		return nil, err
	}
	for name, content := range canonical {
		contents[name] = content
	}
	contents["design.md"], err = withAcceptanceScenarios(contents["design.md"], changeDir)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func readPendingAnalysis(input OpenSpecInput) ([]byte, error) {
	analysis, err := os.ReadFile(input.PendingAnalysisPath)
	if err != nil {
		return nil, fmt.Errorf("openspec bridge: read pending analysis: %w", err)
	}
	if !hasMissionIdentity(analysis, input.MissionID) {
		return nil, fmt.Errorf("openspec bridge: pending analysis mission_id does not match %q", input.MissionID)
	}
	if handoff.HasNormalizedRangerMetadata(input.PendingAnalysisPath) {
		if err := handoff.ValidateRangerArtifactForRefinement(input.PendingAnalysisPath, input.MissionID); err != nil {
			return nil, fmt.Errorf("openspec bridge: %w", err)
		}
	}
	return analysis, nil
}

func readCanonicalFiles(changeDir string) (map[string][]byte, error) {
	contents := make(map[string][]byte, len(canonicalFiles)-1)
	for _, name := range canonicalFiles[1:] {
		content, err := os.ReadFile(filepath.Join(changeDir, name)) //nolint:gosec // changeDir is contained under the validated runtime root
		if err != nil {
			return nil, fmt.Errorf("openspec bridge: read %s: %w", name, err)
		}
		contents[name] = content
	}
	return contents, nil
}

func result(refined string, input OpenSpecInput) OpenSpecResult {
	return OpenSpecResult{RefinedPath: refined, ProviderChangeID: input.ChangeID, ProviderRuntime: input.RuntimeRoot, PublishedAt: time.Now().UTC()}
}
