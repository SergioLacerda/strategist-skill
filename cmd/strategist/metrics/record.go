package metrics

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// RecordOptions holds the flags of `strategist metrics record`.
type RecordOptions struct {
	Root, Mission, Run, Agent, ClaimFile, CorrelationKey, Reason string
	Missing                                                      bool
}

// NewRecord builds the `metrics record` command.
func NewRecord(deps Dependencies) *cobra.Command {
	opts := RecordOptions{}
	cmd := &cobra.Command{Use: "record", Short: "Record a confidence claim or an explicit missing-record for a boundary"}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id (required)")
	f.StringVar(&opts.Run, "run", "", "optional explicit run id for a repeated role execution")
	f.StringVar(&opts.Agent, "agent", "", "producing agent (required)")
	f.StringVar(&opts.ClaimFile, "claim-file", "", `YAML file with the claim and its evidence; "-" reads it from standard input (preferred: no file is created), and a file inside the workspace base_path is rejected`)
	f.BoolVar(&opts.Missing, "missing", false, "record an explicit missing-record instead of a claim")
	f.StringVar(&opts.CorrelationKey, "correlation-key", "", "boundary correlation key (with --missing)")
	f.StringVar(&opts.Reason, "reason", "", "why no summary was produced (with --missing)")
	for _, name := range []string{"mission", "agent"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunRecord(cmd, deps, opts) }
	return cmd
}

// RunRecord records one confidence claim, or an explicit missing-record, for
// a mission boundary.
func RunRecord(cmd *cobra.Command, deps Dependencies, opts RecordOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "record", opts.Root)
	if err != nil {
		return err
	}
	producer, err := telemetry.NewConfidenceProducerAdapter(telemetry.ConfidenceHistoryPath(root), opts.Agent, opts.Mission)
	if err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	producer = producer.WithRun(opts.Run)
	if opts.Missing {
		return recordMissing(cmd, producer, opts)
	}
	return recordClaim(cmd, deps, root, producer, opts.ClaimFile)
}
func recordMissing(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, opts RecordOptions) error {
	if opts.CorrelationKey == "" || opts.Reason == "" {
		return fmt.Errorf("metrics record: --missing requires --correlation-key and --reason")
	}
	if err := producer.RecordMissing(opts.CorrelationKey, opts.Reason); err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "missing-record recorded: agent=%s mission=%s correlation_key=%s\n", producer.Agent, producer.MissionID, opts.CorrelationKey)
	if err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	return nil
}
func recordClaim(cmd *cobra.Command, deps Dependencies, root string, producer telemetry.ConfidenceProducerAdapter, path string) error {
	if path == "" {
		return fmt.Errorf("metrics record: --claim-file is required unless --missing is set (use `--claim-file -` to read from standard input)")
	}
	input, err := readClaimInput(cmd, deps, root, path)
	if err != nil {
		return err
	}
	if input.batch {
		return recordClaimBatch(cmd, producer, input)
	}
	return recordSingleClaim(cmd, producer, input)
}
