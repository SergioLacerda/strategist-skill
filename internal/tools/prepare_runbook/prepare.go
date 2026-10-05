package preparerunbook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"gopkg.in/yaml.v3"
)

// Prepare builds a reviewable Runbook candidate without persisting it.
func Prepare(ctx context.Context, in Input) (Result, error) {
	if err := validateInput(in); err != nil {
		return Result{}, err
	}
	actual, err := packageDigest(in.RefinedPath, in.PackageDigest)
	if err != nil {
		return Result{}, err
	}
	artifacts, err := readArtifacts(in.RefinedPath)
	if err != nil {
		return Result{}, err
	}
	combined := joinArtifacts(artifacts)
	in.SelectedRunbooksHint = selectedRunbookHints(in, artifacts)
	evidence := inspectEvidence(combined)
	proposed := buildCandidate(in, artifacts, combined, evidence)
	if err := runbook.ValidateRunbook(proposed); err != nil {
		return Result{}, fmt.Errorf("prepare runbook: proposed sidecar is invalid: %w", err)
	}
	result := Result{
		SchemaVersion: SchemaVersion, Status: StatusPrepared, MissionID: in.MissionID,
		CandidateID: in.CandidateID, PackageDigest: in.PackageDigest, SourceDigest: actual,
		ProposedRunbook: proposed, Evidence: evidence,
		Provider:   ProviderReading{Path: ProviderPathMain, Reason: "jev_disabled"},
		Provenance: provenance(artifacts),
	}
	result.Comparison, err = compare(in, proposed)
	if err != nil {
		return Result{}, err
	}
	if applyPreparationStatus(&result, evidence) {
		return result, nil
	}
	result.Provider = evaluateProvider(ctx, in, actual, proposed, result.Comparison)
	return result, nil
}

func packageDigest(refinedPath, expected string) (string, error) {
	actual, err := handoff.PackageDigest(refinedPath)
	if err != nil {
		return "", fmt.Errorf("prepare runbook: calculate source digest: %w", err)
	}
	if actual != expected {
		return "", fmt.Errorf("prepare runbook: stale package digest: expected %s, got %s", expected, actual)
	}
	return actual, nil
}

func selectedRunbookHints(in Input, artifacts []artifact) []string {
	if len(in.SelectedRunbooksHint) > 0 {
		return in.SelectedRunbooksHint
	}
	return firstFrontStrings(artifacts, "selected_runbooks_hint")
}

func evaluateProvider(ctx context.Context, in Input, actual string, proposed runbook.Runbook, comparison Comparison) ProviderReading {
	if in.Evaluate == nil {
		return ProviderReading{Path: ProviderPathMain, Reason: "jev_disabled"}
	}
	reading, err := in.Evaluate(ctx, ProviderRequest{
		MissionID: in.MissionID, CandidateID: in.CandidateID, Subject: actual,
		Fields:   providerFields(proposed, comparison),
		Criteria: []string{"reusable_procedure", "novel_or_extension", "evidence_sufficiency", "scope_bounded"},
	})
	if err != nil {
		return ProviderReading{Path: ProviderPathMain, Reason: "jev_error: " + err.Error()}
	}
	return reading
}

func validateInput(in Input) error {
	if strings.TrimSpace(in.MissionID) == "" || strings.TrimSpace(in.CandidateID) == "" {
		return errors.New("prepare runbook: mission and candidate ids are required")
	}
	if strings.TrimSpace(in.PackageDigest) == "" {
		return errors.New("prepare runbook: package digest is required")
	}
	if strings.TrimSpace(in.RefinedPath) == "" {
		return errors.New("prepare runbook: refined path is required")
	}
	return nil
}

func applyPreparationStatus(result *Result, evidence Evidence) bool {
	if missing := evidence.Missing(); len(missing) > 0 {
		result.Status = StatusInsufficientEvidence
		result.UnresolvedQuestions = append(result.UnresolvedQuestions, missingQuestions(missing)...)
		return true
	}
	switch result.Comparison.Classification {
	case ComparisonInsufficient:
		result.Status = StatusNeedsReview
		result.UnresolvedQuestions = append(result.UnresolvedQuestions, "configured Runbook inventory could not be compared")
	case ComparisonDuplicate:
		result.Status = StatusDuplicate
		result.UnresolvedQuestions = append(result.UnresolvedQuestions, "an equivalent canonical Runbook already exists")
	case ComparisonNovel, ComparisonExtension:
		// Both classifications remain eligible for the normal preparation path.
	}
	return false
}

func readArtifacts(root string) ([]artifact, error) {
	names := []string{"analysis.md", "proposal.md", "design.md", "tasks.md"}
	artifacts := make([]artifact, 0, len(names))
	for _, name := range names {
		path := filepath.Join(root, name)
		raw, err := os.ReadFile(path) //nolint:gosec // path is resolved by the mission boundary
		if err != nil {
			return nil, fmt.Errorf("prepare runbook: read %s: %w", name, err)
		}
		if strings.TrimSpace(string(raw)) == "" {
			return nil, fmt.Errorf("prepare runbook: %s is empty", name)
		}
		front, body := splitFrontmatter(string(raw))
		artifacts = append(artifacts, artifact{path: path, text: body, front: front})
	}
	return artifacts, nil
}

func splitFrontmatter(text string) (map[string]any, string) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "---\n") {
		return map[string]any{}, trimmed
	}
	rest := strings.TrimPrefix(trimmed, "---\n")
	header, body, ok := strings.Cut(rest, "\n---")
	if !ok {
		return map[string]any{}, trimmed
	}
	front := map[string]any{}
	if yaml.Unmarshal([]byte(header), &front) != nil {
		return map[string]any{}, strings.TrimSpace(body)
	}
	return front, strings.TrimSpace(strings.TrimPrefix(body, "\n"))
}

func joinArtifacts(artifacts []artifact) string {
	var b strings.Builder
	for _, a := range artifacts {
		b.WriteString(a.text)
		b.WriteString("\n")
	}
	return b.String()
}
