package mission

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// ReportUsageOptions carries the `mission report-usage` flag values. The
// *Set fields record which token flags the caller actually passed: an omitted
// count is "not observed", never zero.
type ReportUsageOptions struct {
	Root, MissionID, Phase, Role       string
	InvocationID, PhaseRunID, EventID  string
	Scope, SessionID, RecordKind       string
	Source, Quality, Provider, Model   string
	Attempt                            int
	TokensIn, TokensOut                int64
	TokensCache, TokensReasoning       int64
	InSet, OutSet, CacheSet, ReasonSet bool
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
		Short: "Record one token usage observation for a mission phase, declared by the invoking agent",
		Long: `strategist mission report-usage records ONE observation of token usage for a
mission, declared by the LLM agent (e.g. Claude Code) that invoked this CLI.

This binary never calls an LLM API and cannot verify the numbers: every record
is stored as reported_unverified (or as an explicit estimate with
--source estimator). It never claims provider_measured or host_measured; those
qualities need an adapter that observes a provider or host counter itself.

Report each observation once, with the counts of that call alone (an
incremental record). Never report a running total as incremental: either use
--record-kind cumulative_snapshot with --session-id, so only the delta between
two snapshots of the same session is counted, or report the call's own counts.
Pass --invocation-id (the request id printed by 'mission invoke') or --event-id
so a repeated delivery is recognised and counted once; without one the record is
accepted but flagged dedup_unavailable. Omit a --tokens-* flag you did not
observe: unobserved is never recorded as zero.

The command is refused until the mission has a pending, refined or archived
artifact; report earlier invocations afterwards with an explicit --phase.
The Approval Gate carries no tokens; use --phase unattributed for consumption
outside any phase. Records are appended to
.strategist/memory/mission-usage.jsonl and read by 'strategist mission usage'.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.MissionID, "mission-id", "", "mission_id to record usage against (required) — must have a pending/refined/archived artifact")
	f.StringVar(&opts.Phase, "phase", "", "phase the usage belongs to (required): bootstrap, intake, discovery, refinement, execution or unattributed")
	f.StringVar(&opts.Role, "role", "", "role that consumed the tokens (required): scout, ranger, archivist, sniper, shell, ...")
	f.StringVar(&opts.InvocationID, "invocation-id", "", "the 'mission invoke' request id (inv_...) this usage belongs to; makes the record idempotent")
	f.StringVar(&opts.EventID, "event-id", "", "caller-chosen idempotency key when there is no invocation id")
	f.StringVar(&opts.PhaseRunID, "phase-run-id", "", "phase run (r1, r2, ...) the usage belongs to, when known")
	f.IntVar(&opts.Attempt, "attempt", 1, "attempt of the invocation; a real retry that consumed tokens is a new attempt")
	f.StringVar(&opts.Scope, "scope", "invocation", "what the counts cover: invocation, session, phase, ...")
	f.StringVar(&opts.RecordKind, "record-kind", "incremental", "incremental (this call's own counts) or cumulative_snapshot (a running total, needs --session-id)")
	f.StringVar(&opts.SessionID, "session-id", "", "session of a cumulative counter; deltas are only taken inside one session")
	f.StringVar(&opts.Source, "source", "agent_report", "origin of the numbers: agent_report or estimator")
	f.StringVar(&opts.Quality, "quality", "", "claimed quality, never stronger than --source allows (default: reported_unverified, or heuristic_estimated for an estimator)")
	f.StringVar(&opts.Provider, "provider", "", "provider that served the call, when known")
	f.StringVar(&opts.Model, "model", "", "model that served the call, when known")
	f.Int64Var(&opts.TokensIn, "tokens-in", 0, "input tokens observed for this call (omit when not observed)")
	f.Int64Var(&opts.TokensOut, "tokens-out", 0, "output tokens observed for this call (omit when not observed)")
	f.Int64Var(&opts.TokensCache, "tokens-cache", 0, "cache tokens the provider reported; stored separately, never added to the total")
	f.Int64Var(&opts.TokensReasoning, "tokens-reasoning", 0, "reasoning tokens the provider reported; stored separately, never added to the total")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunReportUsage(cmd, deps, opts) }
	return cmd
}

// RunReportUsage appends one declared usage observation for a known mission.
func RunReportUsage(cmd *cobra.Command, deps ReportUsageDependencies, opts ReportUsageOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	opts = withSetFlags(cmd, opts)
	if err := deps.Validate(cmd, opts); err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	strategistRoot, basePath, err := deps.ResolveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	result, err := application.RecordMissionUsage(strategistRoot, basePath, usageInput(opts), time.Now(), application.MissionUsagePorts{
		MissionKnown: deps.MissionKnown,
		Append:       appendUsage,
	})
	if err != nil {
		return fmt.Errorf("mission report-usage: %w", err)
	}
	return writeUsageResult(cmd, result)
}

// withSetFlags records which token flags were actually passed.
func withSetFlags(cmd *cobra.Command, opts ReportUsageOptions) ReportUsageOptions {
	if cmd == nil {
		return opts
	}
	changed := cmd.Flags().Changed
	opts.InSet, opts.OutSet = changed("tokens-in"), changed("tokens-out")
	opts.CacheSet, opts.ReasonSet = changed("tokens-cache"), changed("tokens-reasoning")
	return opts
}

func usageInput(o ReportUsageOptions) application.MissionUsageInput {
	return application.MissionUsageInput{
		MissionID: o.MissionID, Phase: o.Phase, Role: o.Role,
		InvocationID: o.InvocationID, PhaseRunID: o.PhaseRunID, EventID: o.EventID,
		UsageScope: o.Scope, SessionID: o.SessionID, RecordKind: o.RecordKind,
		Source: o.Source, Quality: o.Quality, Provider: o.Provider, Model: o.Model, Attempt: o.Attempt,
		TokensInput: optionalCount(o.InSet, o.TokensIn), TokensOutput: optionalCount(o.OutSet, o.TokensOut),
		TokensCache: optionalCount(o.CacheSet, o.TokensCache), TokensReasoning: optionalCount(o.ReasonSet, o.TokensReasoning),
	}
}

func optionalCount(set bool, v int64) *int64 {
	if !set {
		return nil
	}
	return &v
}

func appendUsage(root string, r application.MissionUsageRecord) (bool, error) {
	rec := telemetry.MissionUsageRecord{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, EventID: r.EventID, MissionID: r.MissionID,
		Phase: r.Phase, PhaseRunID: r.PhaseRunID, Role: r.Role, InvocationID: r.InvocationID,
		Attempt: r.Attempt, UsageScope: r.UsageScope, SessionID: r.SessionID, RecordKind: r.RecordKind,
		TokensInput: r.TokensInput, TokensOutput: r.TokensOutput, TokensCache: r.TokensCache, TokensReasoning: r.TokensReasoning,
		Provider: r.Provider, Model: r.Model, MeasurementSource: r.MeasurementSource, MeasurementQuality: r.MeasurementQuality,
		DedupUnavailable: r.DedupUnavailable, ObservedAt: r.ObservedAt,
	}
	rec.RecordID = telemetry.MissionUsageRecordID(rec)
	appended, err := telemetry.AppendMissionUsageRecord(telemetry.MissionUsagePath(root), rec)
	if err != nil {
		return false, fmt.Errorf("append mission usage: %w", err)
	}
	return appended, nil
}

func writeUsageResult(cmd *cobra.Command, res application.MissionUsageResult) error {
	r := res.Record
	status := "recorded"
	if !res.Appended {
		status = "duplicate (already recorded, not counted again)"
	}
	if r.DedupUnavailable {
		status += "; dedup_unavailable (no --invocation-id or --event-id)"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s phase=%s role=%s tokens_in=%s tokens_out=%s kind=%s quality=%s %s\n",
		r.MissionID, r.Phase, r.Role, countText(r.TokensInput), countText(r.TokensOutput), r.RecordKind, r.MeasurementQuality, status); err != nil {
		return fmt.Errorf("mission report-usage: write output: %w", err)
	}
	return nil
}

func countText(p *int64) string {
	if p == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%d", *p)
}
