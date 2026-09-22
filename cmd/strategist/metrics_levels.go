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

var metricsLevelsCmd = newMetricsLevelsCommand()

func newMetricsLevelsCommand() *cobra.Command {
	opts := metricsLevelsOptions{}
	cmd := &cobra.Command{
		Use:   "levels",
		Short: "Report the model x effort levels recorded per role",
		Long: `Report the levels recorded in .strategist/memory/role-levels.jsonl by
"strategist leveling label": records, missions, unknown levels, escalations, and
per role the levels and level sources (manual, host, policy). An empty history
reports zeros. With --rotate the ledger is compacted to --max-records records,
always keeping the latest tuple of every mission, role and run.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "scope the report to one mission")
	f.BoolVar(&opts.JSON, "json", false, "emit JSON")
	f.BoolVar(&opts.Rotate, "rotate", false, "compact the ledger first, keeping the latest tuple of every mission, role and run")
	f.IntVar(&opts.MaxRecords, "max-records", defaultLedgerMaxRecords, "maximum records kept by --rotate")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsLevels(cmd, opts)
	}
	return cmd
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
	out := &reportWriter{w: w}
	if opts.Rotate {
		if err := rotateLevelsLedger(out, path, opts.MaxRecords); err != nil {
			return err
		}
	}
	report, err := loadLevelsReport(path, opts.Mission)
	if err != nil {
		return err
	}
	if opts.JSON {
		if err := json.NewEncoder(w).Encode(report); err != nil {
			return fmt.Errorf("metrics levels: write output: %w", err)
		}
		return nil
	}
	printLevelsReport(out, report)
	return out.result()
}

func rotateLevelsLedger(out *reportWriter, path string, maxRecords int) error {
	dropped, err := leveling.RotateLedger(path, maxRecords)
	if err != nil {
		return fmt.Errorf("metrics levels: %w", err)
	}
	out.printf("rotated: dropped %d record(s), max_records=%d\n", dropped, maxRecords)
	return out.result()
}

// loadLevelsReport reads the ledger, optionally scopes it to one mission, and
// summarizes it.
func loadLevelsReport(path, mission string) (leveling.Report, error) {
	records, err := leveling.ReadRecords(path)
	if err != nil {
		return leveling.Report{}, fmt.Errorf("metrics levels: %w", err)
	}
	if mission != "" {
		records = filterLevelRecords(records, mission)
	}
	return leveling.Summarize(records), nil
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

func printLevelsReport(out *reportWriter, report leveling.Report) {
	out.printf("records: %d\nmissions: %d\nunknown: %d\nescalations: %d\n", report.Records, report.Missions, report.Unknown, report.Escalations)
	for _, role := range report.Roles {
		out.printf("role.%s.records: %d\nrole.%s.unknown: %d\nrole.%s.escalations: %d\n", role.Role, role.Records, role.Role, role.Unknown, role.Role, role.Escalations)
		printCounts(out, "role."+role.Role+".level.", role.Levels)
		printCounts(out, "role."+role.Role+".source.", role.Sources)
	}
}

func printCounts(out *reportWriter, prefix string, counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.printf("%s%s: %d\n", prefix, key, counts[key])
	}
}

// reportWriter keeps the first write error so a report can be printed line by
// line without an error check after each one.
type reportWriter struct {
	w   io.Writer
	err error
}

func (r *reportWriter) printf(format string, args ...any) {
	if r.err == nil {
		_, r.err = fmt.Fprintf(r.w, format, args...)
	}
}

func (r *reportWriter) result() error {
	if r.err != nil {
		return fmt.Errorf("metrics levels: write output: %w", r.err)
	}
	return nil
}
