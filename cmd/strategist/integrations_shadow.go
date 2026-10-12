package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/spf13/cobra"
)

var integrationsShadowCmd = &cobra.Command{
	Use:   "shadow",
	Short: "Keep the provider reading without letting it decide (on, off, report)",
	Long: `In shadow mode the pre-check is still called and every reading is recorded in the call ledger together
with what it would have decided, but no handoff uses it: no delegation, no hint, no confidence claim, and the main
path decides alone. Use it to collect readings on real missions before a calibration is trusted.
'shadow report' summarizes the readings (counts and numbers only); to compare one mission's reading with its gate
outcome run 'strategist dojo candidate --mission <id>'.`,
}

func init() {
	opts := &integrationsOptions{}
	on := &cobra.Command{Use: "on", Short: "Turn shadow mode on", Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return runShadowSet(cmd, *opts, true) }}
	off := &cobra.Command{Use: "off", Short: "Turn shadow mode off (the provider decides again)", Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return runShadowSet(cmd, *opts, false) }}
	report := &cobra.Command{Use: "report", Short: "Summarize the recorded shadow readings", Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return runShadowReport(cmd, *opts) }}
	for _, c := range []*cobra.Command{on, off, report} {
		c.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
		integrationsShadowCmd.AddCommand(c)
	}
	integrationsCmd.AddCommand(integrationsShadowCmd)
}

func runShadowSet(cmd *cobra.Command, opts integrationsOptions, on bool) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	file, found, err := loadIntegrations(root)
	if err != nil {
		return err
	}
	provider, declared := file.Providers[integrationProvider]
	if !found || !declared {
		return fmt.Errorf("integrations shadow: no provider is declared; run `strategist integrations enable` first")
	}
	provider.Shadow = on
	file.Providers[integrationProvider] = provider
	if err := config.Save(filepath.Join(root, config.FileName), file); err != nil {
		return fmt.Errorf("integrations shadow: %w", err)
	}
	if on {
		return printLines(cmd, "shadow: on (readings are recorded and nothing uses them; main decides alone)")
	}
	return printLines(cmd, "shadow: off (the provider's reading decides again, under its threshold)")
}

func runShadowReport(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	readings, err := handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}.Shadow()
	if err != nil {
		return fmt.Errorf("integrations shadow: %w", err)
	}
	rows := handoffconsumer.SummarizeShadow(readings)
	if len(rows) == 0 {
		return printLines(cmd, "no shadow readings recorded")
	}
	return printLines(cmd, shadowLines(rows)...)
}

func shadowLines(rows []handoffconsumer.ShadowRow) []string {
	var lines []string
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s: %d readings over %d missions, %d answered, would approve %d, confidence mean %.2f (min %.2f, max %.2f), under a calibration %d",
			row.Transition, row.Readings, row.Missions, row.Answered, row.WouldApprove, row.MeanConfident, row.MinConfident, row.MaxConfident, row.Calibrated))
		if hints := hintCounts(row.Hints); hints != "" {
			lines = append(lines, "  criteria found unmet: "+hints)
		}
	}
	return append(lines, "compare a mission with its gate outcome: strategist dojo candidate --mission <id>")
}

// hintCounts lists the unmet criteria, most frequent first.
func hintCounts(hints map[string]int) string {
	names := make([]string, 0, len(hints))
	for name := range hints {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if hints[names[i]] != hints[names[j]] {
			return hints[names[i]] > hints[names[j]]
		}
		return names[i] < names[j]
	})
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%s (%d)", name, hints[name])
	}
	return strings.Join(parts, ", ")
}
