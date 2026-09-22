// Package refinement publishes provider planning output into the Strategist
// workspace lifecycle.
package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OpenSpecInput identifies one completed provider change and its Strategist
// mission. BasePath and RuntimeRoot must be resolved absolute paths.
type OpenSpecInput struct {
	MissionID           string
	BasePath            string
	RuntimeRoot         string
	ChangeID            string
	PendingAnalysisPath string
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

	refined, err := validatedRefinedPath(input)
	if err != nil {
		return OpenSpecResult{}, err
	}
	if err := publishOrPromote(refined, input.PendingAnalysisPath, contents); err != nil {
		return OpenSpecResult{}, err
	}
	// Published: the scratch change leaves the active list so the provider
	// runtime does not accumulate finished changes.
	if err := archiveChange(input.RuntimeRoot, changeDir, input.ChangeID); err != nil {
		return OpenSpecResult{}, err
	}
	return result(refined, input), nil
}

func publishOrPromote(refined, pending string, contents map[string][]byte) error {
	existing, err := existingPackage(refined, contents)
	if err != nil {
		return err
	}
	if existing {
		return removePending(pending)
	}
	return publish(refined, pending, contents)
}

func readContents(changeDir string, input OpenSpecInput) (map[string][]byte, error) {
	if err := requireFiles(changeDir, canonicalFiles[1:]); err != nil {
		return nil, fmt.Errorf("openspec bridge: incomplete change: %w", err)
	}
	analysis, err := os.ReadFile(input.PendingAnalysisPath)
	if err != nil {
		return nil, fmt.Errorf("openspec bridge: read pending analysis: %w", err)
	}
	if !hasMissionIdentity(analysis, input.MissionID) {
		return nil, fmt.Errorf("openspec bridge: pending analysis mission_id does not match %q", input.MissionID)
	}
	contents := map[string][]byte{"analysis.md": addMetadata(analysis, input)}
	for _, name := range canonicalFiles[1:] {
		contents[name], err = os.ReadFile(filepath.Join(changeDir, name)) //nolint:gosec // changeDir is contained under the validated runtime root
		if err != nil {
			return nil, fmt.Errorf("openspec bridge: read %s: %w", name, err)
		}
	}
	contents["design.md"], err = withAcceptanceScenarios(contents["design.md"], changeDir)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func result(refined string, input OpenSpecInput) OpenSpecResult {
	return OpenSpecResult{RefinedPath: refined, ProviderChangeID: input.ChangeID, ProviderRuntime: input.RuntimeRoot, PublishedAt: time.Now().UTC()}
}
