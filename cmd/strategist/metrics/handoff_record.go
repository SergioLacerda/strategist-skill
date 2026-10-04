package metrics

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// HandoffRecordOptions holds the flags of `strategist metrics handoff-record`.
type HandoffRecordOptions struct {
	Root, Mission, Model, Effort, LevelSource string
	DiscoveryTokens, BriefTokens              int64
	Reopens, Revision                         int
	CompressionRatio, EvidenceCoverage        float64
}

// NewHandoffRecord builds the `metrics handoff-record` command, the sanctioned
// writer of the Archivist's line in handoff-metrics.jsonl.
func NewHandoffRecord(deps Dependencies) *cobra.Command {
	opts := HandoffRecordOptions{}
	cmd := &cobra.Command{
		Use:   "handoff-record",
		Short: "Record the Archivist's handoff metrics line for a mission",
		Long:  "Append the Archivist's per-refinement line to .strategist/memory/handoff-metrics.jsonl. Only the values passed are recorded; every other field is null (an unset --reopens is null, not 0), and the two ratios are never derived because the contracts do not define them. A mission that already has a line for the same revision is left unchanged; --revision <n> records one further line per gate revision.",
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id (required)")
	f.Int64Var(&opts.DiscoveryTokens, "discovery-tokens", 0, "tokens spent by discovery (omit when not measured)")
	f.Int64Var(&opts.BriefTokens, "brief-tokens", 0, "tokens in the discovery brief (omit when not measured)")
	f.Float64Var(&opts.CompressionRatio, "brief-compression-ratio", 0, "brief compression ratio, as measured by the caller (omit when not measured)")
	f.IntVar(&opts.Reopens, "reopens", 0, "sources the Archivist reopened, each with a declared reason (omit when not measured; 0 means measured, none)")
	f.IntVar(&opts.Revision, "revision", 0, "gate revision number this line records (omit for the base line; one line is kept per revision)")
	f.Float64Var(&opts.EvidenceCoverage, "evidence-coverage-ratio", 0, "evidence coverage ratio, as measured by the caller (omit when not measured)")
	f.StringVar(&opts.Model, "model", "", "Archivist model (omit when unknown)")
	f.StringVar(&opts.Effort, "effort", "", "Archivist effort (omit when unknown)")
	f.StringVar(&opts.LevelSource, "level-source", "", "how the level was resolved, for example host (omit when unknown)")
	if err := cmd.MarkFlagRequired("mission"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunHandoffRecord(cmd, deps, opts) }
	return cmd
}

// RunHandoffRecord appends the handoff metrics line for a mission.
func RunHandoffRecord(cmd *cobra.Command, deps Dependencies, opts HandoffRecordOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "handoff-record", opts.Root)
	if err != nil {
		return err
	}
	appended, err := telemetry.AppendRefinementHandoffLine(telemetry.HandoffMetricsPath(root), handoffLineFromFlags(cmd, opts))
	if err != nil {
		return fmt.Errorf("metrics handoff-record: %w", err)
	}
	status := "handoff metrics recorded"
	if !appended {
		status = "handoff metrics already recorded for this mission; nothing written"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: mission=%s\n", status, opts.Mission); err != nil {
		return fmt.Errorf("metrics handoff-record: write output: %w", err)
	}
	return nil
}

// handoffLineFromFlags keeps a value null unless its flag was passed, so an
// unmeasured field is never recorded as a zero.
func handoffLineFromFlags(cmd *cobra.Command, opts HandoffRecordOptions) telemetry.RefinementHandoffLine {
	flags := cmd.Flags()
	line := application.HandoffMetricsLine{MissionID: opts.Mission}
	if flags.Changed("reopens") {
		line.RefinementReopens = &opts.Reopens
	}
	if flags.Changed("revision") {
		line.Revision = &opts.Revision
	}
	if flags.Changed("discovery-tokens") {
		line.DiscoveryTokens = &opts.DiscoveryTokens
	}
	if flags.Changed("brief-tokens") {
		line.BriefTokens = &opts.BriefTokens
	}
	if flags.Changed("brief-compression-ratio") {
		line.BriefCompressionRatio = &opts.CompressionRatio
	}
	if flags.Changed("evidence-coverage-ratio") {
		line.EvidenceCoverageRatio = &opts.EvidenceCoverage
	}
	line.Model, line.Effort, line.LevelSource = opts.Model, opts.Effort, opts.LevelSource
	return telemetry.RefinementHandoffLine{
		MissionID: line.MissionID, DiscoveryTokens: line.DiscoveryTokens, BriefTokens: line.BriefTokens,
		BriefCompressionRatio: line.BriefCompressionRatio, RefinementReopens: line.RefinementReopens,
		Revision: line.Revision, EvidenceCoverageRatio: line.EvidenceCoverageRatio,
		Model: optionalString(line.Model), Effort: optionalString(line.Effort), LevelSource: optionalString(line.LevelSource),
	}
}

func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
