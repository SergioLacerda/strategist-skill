package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// missionCmd is the parent for Strategist mission-level commands that the
// invoking agent (not this binary) drives directly, as opposed to
// "metrics" (read-only, computed from runtime memory) or "handoff" (a
// scoped verification tool for one transition).
var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Report and inspect mission-level facts this binary cannot observe directly",
}

type missionReportUsageOptions struct {
	Root      string
	MissionID string
	TokensIn  int64
	TokensOut int64
}

// missionIDPattern restricts --mission-id to the same safe character set
// GenerateMissionID (internal/install/mission_id.go) produces:
// lowercase/digits/hyphens. This also protects the filepath.Glob existence
// check below from path-traversal or glob-metacharacter injection via an
// operator-supplied string.
var missionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

var missionReportUsageCmd = &cobra.Command{
	Use:   "report-usage",
	Short: "Record real token usage for a mission, reported by the invoking agent",
	Long: `strategist mission report-usage records tokens_in/tokens_out for a
mission_id, as reported by the LLM agent (e.g. Claude Code) that invoked
this CLI.

This binary is a CLI/contract layer: it never calls an LLM API itself and
has no direct visibility into how many tokens a model consumed. The
tokens_in/tokens_out values shown inline in persona chat templates
(mission_metrics diagnostics) are the agent's own self-reported estimate,
with no persisted record behind them until this command is run.

--tokens-in and --tokens-out must be the real counts the invoking agent
read from its own provider response for this mission (e.g. Claude's
usage.input_tokens / usage.output_tokens across the conversation turns
spent on this mission_id) — not another estimate. Only the agent has
access to those numbers; this binary cannot verify them and does not try
to.

The record is appended to .strategist/memory/mission-token-usage.jsonl,
one JSONL line per report. Comparing the reported total against
skill.yaml's declarative token_budget is a natural follow-up, not done by
this command.`,
}

func runMissionReportUsage(cmd *cobra.Command, opts missionReportUsageOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	if err := validateMissionReportUsageOptions(cmd, opts); err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}

	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(stringFlag(cmd, flagRoot, opts.Root))
	if err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	if !missionIDKnown(basePath, opts.MissionID) {
		return fmt.Errorf(
			"mission report-usage: unknown mission_id %q (no pending/refined/archived artifact found under %s)",
			opts.MissionID, basePath,
		)
	}

	// Feed the reported counts through SetTokens on a MissionRun scoped to
	// the target mission_id, then persist the snapshot's token fields as the
	// durable record — this is the one production call site SetTokens's doc
	// comment (internal/telemetry/mission_run.go) describes.
	run := telemetry.NewMissionRun(opts.MissionID)
	run.SetTokens(opts.TokensIn, opts.TokensOut)
	snapshot := run.Snapshot()

	rec := telemetry.MissionTokenUsageRecord{
		MissionID:  snapshot.MissionID,
		TokensIn:   snapshot.TokensIn,
		TokensOut:  snapshot.TokensOut,
		Source:     telemetry.MissionUsageSourceAgentReport,
		ReportedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := telemetry.AppendMissionTokenUsage(telemetry.MissionTokenUsageHistoryPath(strategistRoot), rec); err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s tokens_in=%d tokens_out=%d source=%s recorded\n",
		rec.MissionID, rec.TokensIn, rec.TokensOut, rec.Source); err != nil {
		return fmt.Errorf("mission report-usage: write output: %w", err)
	}
	return nil
}

// validateMissionReportUsageOptions checks the flags this command treats as
// required and range-validates the token counts. Cobra's Int64Var flag type
// already rejects a non-numeric --tokens-in/--tokens-out at parse time
// (returns "invalid argument ... strconv.ParseInt" before RunE is ever
// called), so this only needs to catch: the flag never passed at all, and a
// value that parsed fine but is out of range (negative).
func validateMissionReportUsageOptions(cmd *cobra.Command, opts missionReportUsageOptions) error {
	var problems []string
	if opts.MissionID == "" {
		problems = append(problems, "--mission-id is required")
	} else if !missionIDPattern.MatchString(opts.MissionID) {
		problems = append(problems, fmt.Sprintf("--mission-id %q is malformed (want lowercase letters, digits, and hyphens, e.g. 20260830-skill-gaps-triage)", opts.MissionID))
	}
	problems = append(problems, validateTokenFlag(cmd, "tokens-in", opts.TokensIn)...)
	problems = append(problems, validateTokenFlag(cmd, "tokens-out", opts.TokensOut)...)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

// validateTokenFlag reports a "required" problem when --name was never set,
// or a "must be >= 0" problem when an explicitly-set value is negative. cmd
// may be nil in unit tests that call runMissionReportUsage's helpers
// directly against a bare opts struct — treated as "flag not set".
func validateTokenFlag(cmd *cobra.Command, name string, value int64) []string {
	changed := cmd != nil && cmd.Flags().Changed(name)
	switch {
	case !changed:
		return []string{fmt.Sprintf("--%s is required", name)}
	case value < 0:
		return []string{fmt.Sprintf("--%s must be >= 0, got %d", name, value)}
	default:
		return nil
	}
}

// missionIDKnown reports whether id matches an existing mission artifact
// under any of the three canonical analysis directories, mirroring
// internal/install.GenerateMissionID's own collision check (that function
// cannot be reused directly: it is unexported and lives in a package this
// task is not scoped to touch).
func missionIDKnown(basePath, id string) bool {
	for _, sub := range []string{"pending", "refined", "archived"} {
		matches, err := filepath.Glob(filepath.Join(basePath, sub, id+"*"))
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}

func init() {
	opts := missionReportUsageOptions{}
	missionReportUsageCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	missionReportUsageCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission_id to record usage against (required) — must match an existing pending/refined/archived artifact")
	missionReportUsageCmd.Flags().Int64Var(&opts.TokensIn, "tokens-in", 0, "real input token count from the invoking agent's own provider response (required, >= 0)")
	missionReportUsageCmd.Flags().Int64Var(&opts.TokensOut, "tokens-out", 0, "real output token count from the invoking agent's own provider response (required, >= 0)")
	missionReportUsageCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMissionReportUsage(cmd, opts)
	}
	missionCmd.AddCommand(missionReportUsageCmd)
	rootCmd.AddCommand(missionCmd)
}
