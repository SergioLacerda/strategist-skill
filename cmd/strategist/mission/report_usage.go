package mission

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// ReportUsageOptions carries the `mission report-usage` flag values.
type ReportUsageOptions struct {
	Root, MissionID     string
	TokensIn, TokensOut int64
}

// ReportUsageDependencies injects validation, root resolution, the known-mission
// check and telemetry silencing.
type ReportUsageDependencies struct {
	RootFlag        string
	SilenceRun      func(*cobra.Command)
	ResolveBasePath func(string) (string, string, error)
	MissionKnown    func(string, string) bool
	Validate        func(*cobra.Command, ReportUsageOptions) error
}

// NewReportUsage builds `mission report-usage`.
func NewReportUsage(deps ReportUsageDependencies) *cobra.Command {
	opts := ReportUsageOptions{}
	cmd := &cobra.Command{
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
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.MissionID, "mission-id", "", "mission_id to record usage against (required) — must match an existing pending/refined/archived artifact")
	f.Int64Var(&opts.TokensIn, "tokens-in", 0, "real input token count from the invoking agent's own provider response (required, >= 0)")
	f.Int64Var(&opts.TokensOut, "tokens-out", 0, "real output token count from the invoking agent's own provider response (required, >= 0)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunReportUsage(cmd, deps, opts) }
	return cmd
}

// RunReportUsage appends one agent-reported token usage record for a known
// mission.
func RunReportUsage(cmd *cobra.Command, deps ReportUsageDependencies, opts ReportUsageOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	if err := deps.Validate(cmd, opts); err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	strategistRoot, basePath, err := deps.ResolveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	if !deps.MissionKnown(basePath, opts.MissionID) {
		return fmt.Errorf("mission report-usage: unknown mission_id %q (no pending/refined/archived artifact found under %s)", opts.MissionID, basePath)
	}
	run := telemetry.NewMissionRun(opts.MissionID)
	run.SetTokens(opts.TokensIn, opts.TokensOut)
	snapshot := run.Snapshot()
	rec := telemetry.MissionTokenUsageRecord{MissionID: snapshot.MissionID, TokensIn: snapshot.TokensIn, TokensOut: snapshot.TokensOut, Source: telemetry.MissionUsageSourceAgentReport, ReportedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := telemetry.AppendMissionTokenUsage(telemetry.MissionTokenUsageHistoryPath(strategistRoot), rec); err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s tokens_in=%d tokens_out=%d source=%s recorded\n", rec.MissionID, rec.TokensIn, rec.TokensOut, rec.Source); err != nil {
		return fmt.Errorf("mission report-usage: write output: %w", err)
	}
	return nil
}
