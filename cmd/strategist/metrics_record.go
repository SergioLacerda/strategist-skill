package main

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type metricsRecordOptions struct {
	Root, Mission, Run, Agent, ClaimFile, CorrelationKey, Reason string
	Missing                                                      bool
}

// confidenceClaimFile is the producer input: one claim plus the evidence it cites.
type confidenceClaimFile struct {
	Claim    domain.ConfidenceClaim `yaml:"claim"`
	Evidence []domain.Evidence      `yaml:"evidence"`
}

var metricsRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a confidence claim or an explicit missing-record for a boundary",
	Long: `Persist one confidence observation from a producing boundary (scout, ranger,
archivist, critic, mission_quality, handoff_challenge, sniper) to
.strategist/memory/confidence-records.jsonl.

  --claim-file FILE   YAML with "claim:" and optional "evidence:" entries
  --missing           record that the boundary produced no confidence summary
                      (requires --correlation-key and --reason)

An invalid claim is stored as a rejected record and reported, never dropped.
Replays are idempotent. The record is advisory input to the Approval Gate.`,
}

func runMetricsRecord(cmd *cobra.Command, opts metricsRecordOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "record", opts.Root)
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
	return recordClaim(cmd, producer, opts.ClaimFile)
}

func recordMissing(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, opts metricsRecordOptions) error {
	if opts.CorrelationKey == "" || opts.Reason == "" {
		return fmt.Errorf("metrics record: --missing requires --correlation-key and --reason")
	}
	if err := producer.RecordMissing(opts.CorrelationKey, opts.Reason); err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "missing-record recorded: agent=%s mission=%s correlation_key=%s\n", producer.Agent, producer.MissionID, opts.CorrelationKey); err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	return nil
}

func recordClaim(cmd *cobra.Command, producer telemetry.ConfidenceProducerAdapter, path string) error {
	if path == "" {
		return fmt.Errorf("metrics record: --claim-file is required unless --missing is set")
	}
	raw, err := os.ReadFile(path) //nolint:gosec // operator-supplied claim file
	if err != nil {
		return fmt.Errorf("metrics record: read claim file: %w", err)
	}
	var input confidenceClaimFile
	if err := yaml.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("metrics record: parse claim file: %w", err)
	}
	record, err := producer.RecordClaim(input.Claim, input.Evidence)
	if err != nil {
		return fmt.Errorf("metrics record: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "coverage_status: %s\nclaim: %s\nviolation: %s\n", record.CoverageStatus, record.ClaimID, record.Violation); err != nil {
		return fmt.Errorf("metrics record: write output: %w", err)
	}
	return nil
}

func init() {
	opts := metricsRecordOptions{}
	f := metricsRecordCmd.Flags()
	f.StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id (required)")
	f.StringVar(&opts.Run, "run", "", "optional explicit run id for a repeated role execution")
	f.StringVar(&opts.Agent, "agent", "", "producing agent (required)")
	f.StringVar(&opts.ClaimFile, "claim-file", "", "YAML file with the claim and its evidence")
	f.BoolVar(&opts.Missing, "missing", false, "record an explicit missing-record instead of a claim")
	f.StringVar(&opts.CorrelationKey, "correlation-key", "", "boundary correlation key (with --missing)")
	f.StringVar(&opts.Reason, "reason", "", "why no summary was produced (with --missing)")
	requireFlags(metricsRecordCmd, "mission", "agent")
	metricsRecordCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsRecord(cmd, opts)
	}
	metricsCmd.AddCommand(metricsRecordCmd)
}
