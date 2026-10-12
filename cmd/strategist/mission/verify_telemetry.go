package mission

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	telemetryverify "github.com/SergioLacerda/strategist-skill/internal/telemetry/verify"
	"github.com/spf13/cobra"
)

// NewVerifyTelemetry builds `mission verify-telemetry`: a read-only integrity
// check of one mission's lifecycle ledger, usage ledger and, optionally, an
// exported event stream. It reports gaps, duplicates, conflicting counts,
// invalid timestamps and unreadable lines; it never repairs anything, and it
// exits non-zero only when the evidence contradicts itself.
func NewVerifyTelemetry(deps ViewDependencies) *cobra.Command {
	var root, missionID, events string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "verify-telemetry",
		Short: "Check the integrity of a mission's lifecycle and usage evidence (read-only)",
		Long: "Cross-check the mission lifecycle ledger, the usage ledger and, with --events, an exported " +
			"JSONL event stream (STRATEGIST_TELEMETRY_JSONL). Findings are errors (the evidence contradicts " +
			"itself), warnings (a gap or a degradation) or info. The command is advisory and independent of " +
			"any governance provider.",
	}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&missionID, "mission-id", "", "mission identifier (required)")
	f.StringVar(&events, "events", "", "exported event stream (JSONL) to cross-check against the ledgers")
	f.BoolVar(&asJSON, "json", false, "emit strategist-telemetry-verify/v1 JSON")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunVerifyTelemetry(cmd, deps, root, missionID, events, asJSON)
	}
	return cmd
}

// verifySchemaVersion identifies the machine-readable verify output.
const verifySchemaVersion = "strategist-telemetry-verify/v1"

// RunVerifyTelemetry loads the evidence, verifies it and renders the report.
func RunVerifyTelemetry(cmd *cobra.Command, deps ViewDependencies, rootInput, missionID, eventsPath string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission verify-telemetry: %w", err)
	}
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return fmt.Errorf("mission verify-telemetry: %w", err)
	}
	if _, err := application.QueryMission(status); err != nil {
		return fmt.Errorf("mission verify-telemetry: %w", err)
	}
	in, err := loadVerifyInput(root, status.MissionID, eventsPath)
	if err != nil {
		return fmt.Errorf("mission verify-telemetry: %w", err)
	}
	in.Status = status
	report := telemetryverify.Verify(in)
	if err := renderVerify(cmd, report, asJSON); err != nil {
		return err
	}
	if !report.OK {
		return fmt.Errorf("mission verify-telemetry: %d integrity error(s) in the evidence of %s", report.Counts[string(telemetryverify.SeverityError)], report.MissionID)
	}
	return nil
}

func loadVerifyInput(root, missionID, eventsPath string) (telemetryverify.Input, error) {
	sources := loadUsageSources(root, missionID)
	if sources.lifecycleErr != nil {
		return telemetryverify.Input{}, fmt.Errorf("read lifecycle ledger: %w", sources.lifecycleErr)
	}
	if sources.usageErr != nil {
		return telemetryverify.Input{}, fmt.Errorf("read usage ledger: %w", sources.usageErr)
	}
	in := telemetryverify.Input{Lifecycle: sources.lifecycle, Usage: sources.usage, KnownInvocations: sources.invocationIDs}
	var err error
	if in.LifecycleMalformed, err = telemetry.CountUnreadableJSONLines(telemetry.MissionLifecyclePath(root)); err != nil {
		return in, fmt.Errorf("scan lifecycle ledger: %w", err)
	}
	if in.UsageMalformed, err = telemetry.CountUnreadableJSONLines(telemetry.MissionUsagePath(root)); err != nil {
		return in, fmt.Errorf("scan usage ledger: %w", err)
	}
	if eventsPath == "" {
		return in, nil
	}
	if in.Events, in.EventsMalformed, err = telemetry.ReadEventLines(eventsPath); err != nil {
		return in, fmt.Errorf("read event stream: %w", err)
	}
	in.EventsProvided = true
	return in, nil
}

func renderVerify(cmd *cobra.Command, r telemetryverify.Report, asJSON bool) error {
	out := cmd.OutOrStdout()
	if asJSON {
		payload := struct {
			Schema string `json:"schema"`
			telemetryverify.Report
		}{verifySchemaVersion, r}
		if err := json.NewEncoder(out).Encode(payload); err != nil {
			return fmt.Errorf("mission verify-telemetry: write output: %w", err)
		}
		return nil
	}
	return renderVerifyText(cmd, r)
}

func renderVerifyText(cmd *cobra.Command, r telemetryverify.Report) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "mission %s: ok=%t errors=%d warnings=%d info=%d (lifecycle=%d usage=%d events=%d)\n",
		r.MissionID, r.OK, r.Counts["error"], r.Counts["warning"], r.Counts["info"],
		r.Checked.LifecycleRecords, r.Checked.UsageRecords, r.Checked.Events); err != nil {
		return fmt.Errorf("mission verify-telemetry: write output: %w", err)
	}
	for _, f := range r.Findings {
		if _, err := fmt.Fprintf(out, "  %-7s %-34s %s %s\n", f.Severity, f.Code, f.Ref, f.Message); err != nil {
			return fmt.Errorf("mission verify-telemetry: write output: %w", err)
		}
	}
	return nil
}
