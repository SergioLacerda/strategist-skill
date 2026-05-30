package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	syncGovernanceRoot   string
	syncGovernanceSddDir string
	syncGovernanceDryRun bool
)

var syncGovernanceCmd = &cobra.Command{
	Use:   "sync-governance",
	Short: "Sync .strategist/skill.yaml with active SDD governance mandates",
	Long: `Reads .sdd/ governance mandates and reconciles .strategist/skill.yaml.

Checks performed:
  - Reads .sdd/metadata.json to verify governance fingerprint
  - Reads .sdd/source/governance-core.json to extract active mandates
  - Compares active mandates against compliance.mandates in skill.yaml
  - Applies missing governance fields (validation_policy, budget_policy, telemetry_policy)
  - Reports drift before applying changes

Use --dry-run to preview changes without writing.`,
	RunE: runSyncGovernanceCmd,
}

func runSyncGovernanceCmd(cmd *cobra.Command, _ []string) (retErr error) {
	if syncGovernanceRoot == "" {
		syncGovernanceRoot = ".strategist"
	}
	if syncGovernanceSddDir == "" {
		syncGovernanceSddDir = ".sdd"
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.sync_governance")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	report, err := runSyncGovernance(syncGovernanceRoot, syncGovernanceSddDir, syncGovernanceDryRun)
	if err != nil {
		return fmt.Errorf("sync-governance: %w", err)
	}

	span.SetAttributes(
		attribute.Int(telemetry.AttrMandates, len(report.MandatesActive)),
		attribute.StringSlice("strategist.mandates.missing", report.MandatesMissing),
	)
	slog.InfoContext(ctx, "[Strategist] sync-governance complete",
		"fingerprint", report.GovernanceFingerprint,
		"active", len(report.MandatesActive),
		"missing", len(report.MandatesMissing),
	)

	printSyncReport(report)
	return nil
}

type syncReport struct {
	GovernanceFingerprint string
	MandatesActive        []string
	MandatesCompliant     []string
	MandatesPartial       []string
	MandatesMissing       []string
	FieldsApplied         []string
	DryRun                bool
}

type sddMetadata struct {
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

func runSyncGovernance(skillRoot, sddDir string, dryRun bool) (syncReport, error) {
	report := syncReport{DryRun: dryRun}

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
		out, err := yaml.Marshal(skill)
		if err != nil {
			return report, fmt.Errorf("marshal skill.yaml: %w", err)
		}
		if err := os.WriteFile(skillPath, out, 0o644); err != nil {
			return report, fmt.Errorf("write skill.yaml: %w", err)
		}
	}

	return report, nil
}

func readGovernance(sddDir string) (fingerprint string, activeMandates []string, err error) {
	metaPath := filepath.Join(sddDir, "metadata.json")
	metaRaw, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf(".sdd/metadata.json not found — is SDD active in this workspace? (path: %s)", metaPath)
		}
		return "", nil, fmt.Errorf("read metadata: %w", err)
	}
	var meta sddMetadata
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return "", nil, fmt.Errorf("parse metadata: %w", err)
	}
	fp := meta.Fingerprints.Combined
	if fp == "" {
		fp = meta.GovernanceFingerprint
	}

	corePath := filepath.Join(sddDir, "source", "governance-core.json")
	coreRaw, err := os.ReadFile(corePath)
	if err != nil {
		return "", nil, fmt.Errorf("read governance-core.json: %w", err)
	}
	var core governanceCore
	if err := json.Unmarshal(coreRaw, &core); err != nil {
		return "", nil, fmt.Errorf("parse governance-core.json: %w", err)
	}
	for _, item := range core.Items {
		if item.Type == "MANDATE" && item.Status == "required" {
			activeMandates = append(activeMandates, item.ID)
		}
	}
	return fp, activeMandates, nil
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

func computeComplianceGaps(report *syncReport, skill map[string]any) {
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

func applyMissingFields(skill map[string]any, report *syncReport) (changed bool) {
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

func printSyncReport(r syncReport) {
	fmt.Printf("[Strategist] sync-governance fingerprint=%s\n", r.GovernanceFingerprint)
	fmt.Printf("[Strategist] mandates active=%d compliant=%d partial=%d missing=%d\n",
		len(r.MandatesActive), len(r.MandatesCompliant), len(r.MandatesPartial), len(r.MandatesMissing))

	if len(r.MandatesMissing) > 0 {
		fmt.Printf("[Strategist] mandates not covered:")
		for _, m := range r.MandatesMissing {
			fmt.Printf(" %s", m)
		}
		fmt.Println()
	}

	if len(r.FieldsApplied) == 0 {
		fmt.Println("[Strategist] sync-governance status=ok — skill.yaml already compliant")
		return
	}

	if r.DryRun {
		fmt.Printf("[Strategist] sync-governance status=dry-run — would apply: %v\n", r.FieldsApplied)
	} else {
		fmt.Printf("[Strategist] sync-governance status=applied — fields written: %v\n", r.FieldsApplied)
	}
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

func init() {
	syncGovernanceCmd.Flags().StringVar(&syncGovernanceRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	syncGovernanceCmd.Flags().StringVar(&syncGovernanceSddDir, "sdd", "", "path to .sdd/ directory (default: .sdd)")
	syncGovernanceCmd.Flags().BoolVar(&syncGovernanceDryRun, "dry-run", false, "preview changes without writing")
}
