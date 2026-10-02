// Package governance reconciles .strategist/skill.yaml with an explicitly
// provisioned governance source.
package governance

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SyncReport summarizes the outcome of a governance sync run.
type SyncReport struct {
	GovernanceFingerprint string
	MandatesActive        []string
	MandatesCompliant     []string
	MandatesPartial       []string
	MandatesMissing       []string
	FieldsApplied         []string
	DryRun                bool
}

// RunSync reads an explicitly selected governance source and reconciles
// skillRoot/skill.yaml against it, applying missing governance fields unless
// dryRun is set.
func RunSync(skillRoot string, source Source, governanceDir string, dryRun bool) (SyncReport, error) {
	report := SyncReport{DryRun: dryRun}
	snapshot, err := snapshotFor(source, governanceDir)
	if err != nil {
		return report, err
	}
	report.GovernanceFingerprint = snapshot.Fingerprint
	report.MandatesActive = snapshot.ActiveMandates

	skillPath := filepath.Join(skillRoot, "skill.yaml")
	skill, err := readSkill(skillPath)
	if err != nil {
		return report, err
	}

	computeComplianceGaps(&report, skill)
	changed := applyMissingFields(skill, &report)

	if err := persistIfNeeded(skillPath, skill, changed, dryRun); err != nil {
		return report, err
	}

	return report, nil
}

func snapshotFor(source Source, governanceDir string) (Snapshot, error) {
	if source == nil {
		return Snapshot{}, fmt.Errorf("governance source is required")
	}
	snapshot, err := source.Snapshot(governanceDir)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read governance source snapshot: %w", err)
	}
	return validateSnapshot(source, governanceDir, snapshot)
}

func persistIfNeeded(skillPath string, skill map[string]any, changed, dryRun bool) error {
	if !changed || dryRun {
		return nil
	}
	return writeSyncedSkill(skillPath, skill)
}

func writeSyncedSkill(skillPath string, skill map[string]any) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("marshal skill.yaml: %v", recovered)
		}
	}()
	out, err := yaml.Marshal(skill)
	if err != nil {
		return fmt.Errorf("marshal skill.yaml: %w", err)
	}
	if err := os.WriteFile(skillPath, out, 0o644); err != nil {
		return fmt.Errorf("write skill.yaml: %w", err)
	}
	return nil
}

func readSkill(skillPath string) (map[string]any, error) {
	skillRaw, err := os.ReadFile(skillPath) //nolint:gosec // G304: skill manifest path is derived from the configured skill root
	if err != nil {
		return nil, fmt.Errorf("read skill.yaml: %w", err)
	}
	var skill map[string]any
	if err := yaml.Unmarshal(skillRaw, &skill); err != nil {
		return nil, fmt.Errorf("parse skill.yaml: %w", err)
	}
	return skill, nil
}

func computeComplianceGaps(report *SyncReport, skill map[string]any) {
	covered := make(map[string]bool)
	for _, m := range stringSlice(skill, "compliance", "mandates") {
		covered[m] = true
		report.MandatesCompliant = append(report.MandatesCompliant, m)
	}
	for _, m := range stringSlice(skill, "compliance", "partial") {
		covered[m] = true
		report.MandatesPartial = append(report.MandatesPartial, m)
	}
	for _, m := range report.MandatesActive {
		if !covered[m] {
			report.MandatesMissing = append(report.MandatesMissing, m)
		}
	}
}

func applyMissingFields(skill map[string]any, report *SyncReport) (changed bool) {
	defaults := []struct {
		key string
		val map[string]any
	}{
		{"validation_policy", map[string]any{"require_preflight": true, "require_postcheck": false}},
		{"budget_policy", map[string]any{"token_budget": "high", "timeout_seconds": 600, "max_retries": 1}},
		{"telemetry_policy", map[string]any{"emit_runtime_event": true, "otel_required_if_enabled": false}},
	}
	for _, d := range defaults {
		if _, ok := skill[d.key]; !ok {
			skill[d.key] = d.val
			report.FieldsApplied = append(report.FieldsApplied, d.key)
			changed = true
		}
	}
	return changed
}

// stringSlice extracts a []string from a nested map[string]any path.
func stringSlice(m map[string]any, keys ...string) []string {
	cur := any(m)
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	raw, ok := cur.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
