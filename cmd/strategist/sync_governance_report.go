package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/governance"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func addLine(run *telemetry.MissionRun) {
	if run != nil {
		run.AddLines(1)
	}
}

func printSyncReport(r governance.SyncReport, run *telemetry.MissionRun) {
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
