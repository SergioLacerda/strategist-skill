package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

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

func addLine(run *telemetry.MissionRun) {
	if run != nil {
		run.AddLines(1)
	}
}

func printSyncReport(r syncReport, run *telemetry.MissionRun) {
	addLine(run)
	fmt.Printf("[Strategist] sync-governance fingerprint=%s\n", r.GovernanceFingerprint)
	addLine(run)
	fmt.Printf("[Strategist] mandates active=%d compliant=%d partial=%d missing=%d\n",
		len(r.MandatesActive), len(r.MandatesCompliant), len(r.MandatesPartial), len(r.MandatesMissing))

	if len(r.MandatesMissing) > 0 {
		addLine(run)
		fmt.Printf("[Strategist] mandates not covered:")
		for _, m := range r.MandatesMissing {
			addLine(run)
			fmt.Printf(" %s", m)
		}
		fmt.Println()
	}

	addLine(run)
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
