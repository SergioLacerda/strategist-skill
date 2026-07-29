// Package governance reconciles .strategist/skill.yaml with the active SDD
// governance mandates declared under .sdd/.
package governance

import (
	"encoding/json"
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

// SDDMetadata is the subset of .sdd/metadata.json read during a sync.
type SDDMetadata struct {
	Version      string `json:"version"`
	Fingerprints struct {
		Combined string `json:"combined"`
	} `json:"fingerprints"`
	GovernanceFingerprint string            `json:"governance_fingerprint"` // fallback field name
	Mandates              map[string]string `json:"mandates"`
}

type governanceCore struct {
	Items []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	} `json:"items"`
}

// RunSync reads .sdd/ governance state and reconciles skillRoot/skill.yaml
// against it, applying missing governance fields unless dryRun is set.
func RunSync(skillRoot, sddDir string, dryRun bool) (SyncReport, error) {
	report := SyncReport{DryRun: dryRun}

	fp, activeMandates, err := readGovernance(sddDir)
	if err != nil {
		return report, err
	}
	report.GovernanceFingerprint = fp
	report.MandatesActive = activeMandates

	skillPath := filepath.Join(skillRoot, "skill.yaml")
	skill, err := readSkill(skillPath)
	if err != nil {
		return report, err
	}

	computeComplianceGaps(&report, skill)
	changed := applyMissingFields(skill, &report)

	if changed && !dryRun {
		if err := writeSyncedSkill(skillPath, skill); err != nil {
			return report, err
		}
	}

	return report, nil
}

func writeSyncedSkill(skillPath string, skill map[string]any) error {
	out, err := yaml.Marshal(skill)
	if err != nil {
		return fmt.Errorf("marshal skill.yaml: %w", err)
	}
	if err := os.WriteFile(skillPath, out, 0o644); err != nil {
		return fmt.Errorf("write skill.yaml: %w", err)
	}
	return nil
}

func readGovernance(sddDir string) (fingerprint string, activeMandates []string, err error) {
	metaPath := filepath.Join(sddDir, "metadata.json")
	meta, err := readSDDMetadata(metaPath)
	if err != nil {
		return "", nil, err
	}
	fp := meta.Fingerprints.Combined
	if fp == "" {
		fp = meta.GovernanceFingerprint
	}

	corePath := filepath.Join(sddDir, "source", "governance-core.json")
	core, err := readGovernanceCore(corePath)
	if err != nil {
		return "", nil, err
	}
	return fp, activeMandateIDs(core), nil
}

func readSDDMetadata(metaPath string) (SDDMetadata, error) {
	metaRaw, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return SDDMetadata{}, fmt.Errorf(".sdd/metadata.json not found — is SDD active in this workspace? (path: %s)", metaPath)
	}
	if err != nil {
		return SDDMetadata{}, fmt.Errorf("read metadata: %w", err)
	}
	var meta SDDMetadata
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return SDDMetadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

func readGovernanceCore(corePath string) (governanceCore, error) {
	coreRaw, err := os.ReadFile(corePath)
	if err != nil {
		return governanceCore{}, fmt.Errorf("read governance-core.json: %w", err)
	}
	var core governanceCore
	if err := json.Unmarshal(coreRaw, &core); err != nil {
		return governanceCore{}, fmt.Errorf("parse governance-core.json: %w", err)
	}
	return core, nil
}

func activeMandateIDs(core governanceCore) []string {
	var activeMandates []string
	for _, item := range core.Items {
		if item.Type == "MANDATE" && item.Status == "required" {
			activeMandates = append(activeMandates, item.ID)
		}
	}
	return activeMandates
}

func readSkill(skillPath string) (map[string]any, error) {
	skillRaw, err := os.ReadFile(skillPath)
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
