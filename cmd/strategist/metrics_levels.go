package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

// defaultLedgerMaxRecords bounds .strategist/memory/role-levels.jsonl; the latest
// tuple of every mission, role and run is always kept regardless of the bound.
const defaultLedgerMaxRecords = 2000

type metricsLevelsOptions struct {
	Root       string
	Mission    string
	JSON       bool
	Rotate     bool
	MaxRecords int
}

var metricsLevelsCmd = &cobra.Command{
	Use:   "levels",
	Short: "Report the model x effort levels recorded per role",
	Long: `Report the levels recorded in .strategist/memory/role-levels.jsonl by
"strategist leveling label": records, missions, unknown levels, escalations, and
per role the levels and level sources (manual, host, policy). An empty history
reports zeros. With --rotate the ledger is compacted to --max-records records,
always keeping the latest tuple of every mission, role and run.`,
}

func runMetricsLevels(cmd *cobra.Command, opts metricsLevelsOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "levels", opts.Root)
	if err != nil {
		return err
	}
	return writeLevelsReport(cmd.OutOrStdout(), root, opts)
}

func writeLevelsReport(w io.Writer, root string, opts metricsLevelsOptions) error {
	path := filepath.Join(root, "memory", roleLevelLedger)
	if opts.Rotate {
		dropped, err := leveling.RotateLedger(path, opts.MaxRecords)
		if err != nil {
			return fmt.Errorf("metrics levels: %w", err)
		}
		if _, err := fmt.Fprintf(w, "rotated: dropped %d record(s), max_records=%d\n", dropped, opts.MaxRecords); err != nil {
			return fmt.Errorf("metrics levels: write output: %w", err)
		}
	}
	records, err := leveling.ReadRecords(path)
	if err != nil {
		return fmt.Errorf("metrics levels: %w", err)
	}
	if opts.Mission != "" {
		records = filterLevelRecords(records, opts.Mission)
	}
	report := leveling.Summarize(records)
	if opts.JSON {
		if err := json.NewEncoder(w).Encode(report); err != nil {
			return fmt.Errorf("metrics levels: write output: %w", err)
		}
		return nil
	}
	return printLevelsReport(w, report)
}

func filterLevelRecords(records []leveling.Record, mission string) []leveling.Record {
	out := records[:0:0]
	for _, record := range records {
		if record.MissionID == mission {
			out = append(out, record)
		}
	}
	return out
}

func printLevelsReport(w io.Writer, report leveling.Report) error {
	if _, err := fmt.Fprintf(w, "records: %d\nmissions: %d\nunknown: %d\nescalations: %d\n", report.Records, report.Missions, report.Unknown, report.Escalations); err != nil {
		return fmt.Errorf("metrics levels: write output: %w", err)
	}
	for _, role := range report.Roles {
		if _, err := fmt.Fprintf(w, "role.%s.records: %d\nrole.%s.unknown: %d\nrole.%s.escalations: %d\n", role.Role, role.Records, role.Role, role.Unknown, role.Role, role.Escalations); err != nil {
			return fmt.Errorf("metrics levels: write output: %w", err)
		}
		if err := printCounts(w, "role."+role.Role+".level.", role.Levels); err != nil {
			return err
		}
		if err := printCounts(w, "role."+role.Role+".source.", role.Sources); err != nil {
			return err
		}
	}
	return nil
}

func printCounts(w io.Writer, prefix string, counts map[string]int) error {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "%s%s: %d\n", prefix, key, counts[key]); err != nil {
			return fmt.Errorf("metrics levels: write output: %w", err)
		}
	}
	return nil
}

func init() {
	opts := metricsLevelsOptions{}
	f := metricsLevelsCmd.Flags()
	f.StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "scope the report to one mission")
	f.BoolVar(&opts.JSON, "json", false, "emit JSON")
	f.BoolVar(&opts.Rotate, "rotate", false, "compact the ledger first, keeping the latest tuple of every mission, role and run")
	f.IntVar(&opts.MaxRecords, "max-records", defaultLedgerMaxRecords, "maximum records kept by --rotate")
	metricsLevelsCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsLevels(cmd, opts)
	}
	metricsCmd.AddCommand(metricsLevelsCmd)
}
