package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// FailureReason is a stable classification for why a dojo check item failed,
// used to mine repeated failures across runs without depending on exact label text.
type FailureReason string

const (
	// FailureMissingArtifact indicates an expected artifact file was not found.
	FailureMissingArtifact FailureReason = "missing_artifact"
	// FailureMissingCanary indicates expected file content was absent or forbidden content was present.
	FailureMissingCanary FailureReason = "missing_canary"
	// FailureForbiddenEmit indicates a forbidden event appeared in emit logs.
	FailureForbiddenEmit FailureReason = "forbidden_emit"
	// FailureMissingEmit indicates a required event did not appear in emit logs.
	FailureMissingEmit FailureReason = "missing_emit"
	// FailureManifestDrift indicates a provider manifest differs from expectations.
	FailureManifestDrift FailureReason = "manifest_drift"
	// FailureTiming indicates a timing regression.
	FailureTiming FailureReason = "timing_regression"
	// FailurePipeline indicates the scenario pipeline violated expectations.
	FailurePipeline FailureReason = "pipeline_violation"
	// FailureCriteria indicates criteria.yaml failed validation.
	FailureCriteria FailureReason = "criteria_invalid"
)

// nextActionHints maps each failure reason to a short remediation pointer for lesson.md.
var nextActionHints = map[FailureReason]string{
	FailureMissingArtifact: "Confirm the run produced the expected file at the expected path.",
	FailureMissingCanary:   "Check that generated file content includes the required section or text.",
	FailureForbiddenEmit:   "A role emitted an event it should not have — review pipeline routing.",
	FailureMissingEmit:     "Re-run the scenario; a required emit event was not observed in emit.log.",
	FailureManifestDrift:   "Check the provider's skill.yaml for the expected fields.",
	FailureTiming:          "Investigate a wall-time regression in the run.",
	FailurePipeline:        "Review which slots were invoked and where the pipeline stopped.",
	FailureCriteria:        "Fix criteria.yaml — it failed schema validation before any check ran.",
}

// ClassifyFailures maps each failed item's label to a stable failure reason, deduplicated
// and in first-seen order. Items whose label does not match a known check family are
// skipped rather than guessed at.
func ClassifyFailures(items []domain.DojoCheckItem) []FailureReason {
	seen := make(map[FailureReason]bool)
	var reasons []FailureReason
	for _, it := range items {
		if it.Passed {
			continue
		}
		reason := classifyFailureLabel(it.Label)
		if reason == "" || seen[reason] {
			continue
		}
		seen[reason] = true
		reasons = append(reasons, reason)
	}
	return reasons
}

func classifyFailureLabel(label string) FailureReason {
	switch {
	case strings.HasPrefix(label, "files_created"):
		return FailureMissingArtifact
	case strings.HasPrefix(label, "must_contain"), strings.HasPrefix(label, "must_not_contain"), strings.HasPrefix(label, "section"):
		return FailureMissingCanary
	case strings.HasPrefix(label, "emit") && strings.Contains(label, "must NOT appear"):
		return FailureForbiddenEmit
	case strings.HasPrefix(label, "emit"):
		return FailureMissingEmit
	case strings.HasPrefix(label, "manifest"):
		return FailureManifestDrift
	case strings.HasPrefix(label, "timing"):
		return FailureTiming
	case strings.HasPrefix(label, "pipeline"):
		return FailurePipeline
	default:
		return ""
	}
}

// GenerateLesson builds a short, human-readable remediation note for a failed run:
// which checks failed, the likely cause(s), and a next action per cause.
func GenerateLesson(result domain.DojoCheckResult, reasons []FailureReason) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Dojo Lesson — %s\n\n", result.Scenario)
	fmt.Fprintf(&b, "Result: FAIL (%d of %d checks failed)\n\n", result.FailCount(), len(result.Items))

	b.WriteString("## Failed Checks\n\n")
	for _, it := range result.Items {
		if it.Passed {
			continue
		}
		fmt.Fprintf(&b, "- %s — %s\n", it.Label, it.Detail)
	}

	b.WriteString("\n## Likely Cause\n\n")
	if len(reasons) == 0 {
		b.WriteString("- unclassified\n")
	}
	for _, r := range reasons {
		fmt.Fprintf(&b, "- %s\n", r)
	}

	b.WriteString("\n## Next Action\n\n")
	b.WriteString(nextActionSection(reasons))
	b.WriteString("\n")
	return b.String()
}

func nextActionSection(reasons []FailureReason) string {
	if len(reasons) == 0 {
		return "- Review the failed checks above; no automatic classification matched."
	}
	var lines []string
	for _, r := range reasons {
		if hint, ok := nextActionHints[r]; ok {
			lines = append(lines, "- "+hint)
		}
	}
	return strings.Join(lines, "\n")
}

// WriteLesson writes lesson.md under <basePath>/dojo/.last-run/<scenario>/ summarizing a
// failed run for human review. It never writes production source or jewel files, and is a
// no-op for a passing result — a passing run has no lesson to teach.
func WriteLesson(basePath string, result domain.DojoCheckResult) error {
	if result.Passed() {
		return nil
	}
	content := GenerateLesson(result, ClassifyFailures(result.Items))
	dir := filepath.Join(basePath, "dojo", ".last-run", result.Scenario)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain, not source
		return fmt.Errorf("dojo: create %s: %w", dir, err)
	}
	path := filepath.Join(dir, "lesson.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo: write %s: %w", path, err)
	}
	return nil
}
