package opportunity_attack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var adrSignals = []string{"new pattern", "new interface", "new contract", "new schema", "new abstraction", "breaking change", "trade-off", "tradeoff", "alternative", "rejected", "external dependency", "new library", "new service", "new protocol"}
var runbookSignals = []string{"repeated failure", "recovery", "diagnos", "unblock", "provider", "governance", "decision procedure", "decision gate", "stop conditions", "resolution steps", "lookup", "canonical", "stale", "source of truth", "verified command", "expected output", "symptom", "recovery sequence", "operational"}

// Evaluate detects advisory ADR, Runbook, and Treasure Chest opportunities.
func Evaluate(in Input) (Evaluation, error) {
	if strings.TrimSpace(in.MissionID) == "" {
		return Evaluation{}, fmt.Errorf("opportunity attack: mission id is required")
	}
	contents, err := readRefinedContents(in.RefinedPath)
	if err != nil {
		return Evaluation{}, err
	}
	text := strings.ToLower(strings.Join(contents, "\n"))
	evaluation := Evaluation{ADR: containsAny(text, adrSignals), Runbook: containsAny(text, runbookSignals)}
	if in.ChestIndex == nil {
		return evaluation, nil
	}
	chest, err := evaluateChestOpportunity(contents[0], in.StrategistRoot, in.ChestIndex)
	if err != nil {
		return Evaluation{}, err
	}
	evaluation.Chest = chest.Warranted
	evaluation.ChestSource = chest.SourceRef
	return evaluation, nil
}

type chestOpportunity struct {
	Warranted bool
	SourceRef string
}

func evaluateChestOpportunity(analysis, root string, loadIndex ChestIndexLoader) (chestOpportunity, error) {
	frontmatter, err := analysisFrontmatter([]byte(analysis))
	if err != nil {
		return chestOpportunity{}, err
	}
	registered, err := loadIndex(root)
	if err != nil {
		return chestOpportunity{}, fmt.Errorf("opportunity attack: load Treasure Chest index: %w", err)
	}
	return firstUnregisteredSource(sourceEvidence(frontmatter["sources_consulted"]), registered), nil
}

func firstUnregisteredSource(sources []SourceEvidence, registered map[string]bool) chestOpportunity {
	for _, source := range sources {
		if source.CoverageStatus != "full" {
			continue
		}
		ref := sourceReference(source)
		if ref == "" || registered[ref] || source.Path != "" && registered[source.Path] {
			continue
		}
		return chestOpportunity{Warranted: true, SourceRef: ref}
	}
	return chestOpportunity{}
}

func sourceReference(source SourceEvidence) string {
	if source.ID != "" {
		return source.ID
	}
	return source.Path
}

func sourceEvidence(value any) []SourceEvidence {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]SourceEvidence, 0, len(items))
	for _, item := range items {
		mapping, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, SourceEvidence{
			ID:             stringValue(mapping["id"]),
			Path:           stringValue(mapping["source_path"]),
			CoverageStatus: strings.ToLower(stringValue(mapping["coverage_status"])),
		})
	}
	return result
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func readRefinedContents(refined string) ([]string, error) {
	contents := make([]string, 0, 4)
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		raw, err := os.ReadFile(filepath.Join(refined, name)) //nolint:gosec // path is resolved by the mission boundary
		if err != nil {
			return nil, fmt.Errorf("opportunity attack: read %s: %w", name, err)
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			return nil, fmt.Errorf("opportunity attack: %s is empty", name)
		}
		contents = append(contents, string(raw))
	}
	return contents, nil
}

func containsAny(text string, signals []string) bool {
	for _, signal := range signals {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}
