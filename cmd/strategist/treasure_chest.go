package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- command flags ---

type treasureChestOptions struct {
	Root              string
	DoIndex           bool
	IncludeHistorical bool
	Format            string
	Scope             string
}

var treasureChestCmd = &cobra.Command{
	Use:   "treasure-chest",
	Short: "Show treasure chest runtime status and index health",
	Long: `Show the current treasure chest configuration, governance, index, and compiled artifact status.

Default: read-only status view. Merges four runtime truth layers:
  - .strategist/active.yaml         (configured/scoped chests)
  - .strategist/treasure-chests.yaml (governed trust and routing policy)
  - .strategist/knowledge.index.yaml (indexed retrieval sources)
  - .strategist/.compiled/.index.gz  (compiled fast-path artifact)

Use --index to rebuild the compiled knowledge index from declared sources.
Use --include-historical with --index to opt in to indexing T2/T3 sources.`,
}

func runTreasureChest(cmd *cobra.Command, _ []string, opts treasureChestOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts = treasureChestOptionsFromFlags(cmd, opts)
	root, err := resolveTreasureChestRoot(opts.Root)
	if err != nil {
		return err
	}

	// Load the four truth layers (each is best-effort; only active.yaml is mandatory).
	activeChests, err := treasure.LoadActiveChests(root)
	if err != nil {
		return fmt.Errorf("treasure-chest: %w", err)
	}

	governed, govErr := treasure.LoadGoverned(root)
	indexed, idxErr := treasure.LoadIndexed(root)
	compiledIDs, compiledAt, compErr := treasure.LoadCompiledIndex(root)
	jewels, jewelErr := treasure.LoadJewels(root, governed)
	if jewelErr != nil {
		return fmt.Errorf("treasure-chest: %w", jewelErr)
	}

	rows := treasure.MergeChestRows(activeChests, governed, indexed, compiledIDs, jewels)

	if opts.DoIndex {
		return runTreasureChestIndex(root, rows, opts.IncludeHistorical)
	}

	rows = treasure.FilterRowsByScope(rows, opts.Scope)

	switch opts.Format {
	case "", "table":
		// fall through to table rendering below
	case "json":
		return renderTreasureChestJSON(os.Stdout, root, rows, compErr, govErr, idxErr, compiledAt)
	default:
		return fmt.Errorf("treasure-chest: unknown --format %q (want table or json)", opts.Format)
	}

	return renderTreasureChestTable(root, rows, govErr, idxErr, compErr, compiledAt)
}

func treasureChestOptionsFromFlags(cmd *cobra.Command, opts treasureChestOptions) treasureChestOptions {
	opts.Root = stringFlag(cmd, "root", opts.Root)
	opts.DoIndex = boolFlag(cmd, "index", opts.DoIndex)
	opts.IncludeHistorical = boolFlag(cmd, "include-historical", opts.IncludeHistorical)
	opts.Format = stringFlag(cmd, "format", opts.Format)
	opts.Scope = stringFlag(cmd, "scope", opts.Scope)
	return opts
}

func resolveTreasureChestRoot(rootFlag string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("treasure-chest: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(rootFlag, cwd)
	if err != nil {
		return "", fmt.Errorf("treasure-chest: %w", err)
	}
	return root, nil
}

func renderTreasureChestTable(root string, rows []treasure.StatusRow, govErr, idxErr, compErr error, compiledAt int64) error {
	printTreasureChestBanner()
	if err := renderTableSection("chests", func(w *tabwriter.Writer) error {
		return renderChestsSection(w, rows)
	}); err != nil {
		return err
	}
	if err := renderTableSection("index", func(w *tabwriter.Writer) error {
		return renderIndexSection(w, root, compiledAt, compErr)
	}); err != nil {
		return err
	}
	renderWarningsSection(collectWarnings(rows, govErr, idxErr, compErr, compiledAt))
	fmt.Println()
	return nil
}

func renderTableSection(label string, render func(*tabwriter.Writer) error) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := render(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush %s: %w", label, err)
	}
	fmt.Println()
	return nil
}

// --- index action ---

func runTreasureChestIndex(root string, rows []treasure.StatusRow, includeHistorical bool) error {
	indexPath := filepath.Join(root, "knowledge.index.yaml")

	if !includeHistorical {
		historical := treasure.HistoricalCount(rows)
		if historical > 0 {
			fmt.Printf("[Strategist] treasure-chest --index: %d historical/lower-trust source(s) excluded from default indexing.\n", historical)
			fmt.Println("             Use --include-historical to opt in.")
		}
	}

	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest --index: %w", err)
	}
	fmt.Printf("[Strategist] treasure-chest --index complete → %s/.compiled/\n", root)
	return nil
}

// --- json output ---

type jsonChestRow struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Scope       []string `json:"scope"`
	Trust       string   `json:"trust,omitempty"`
	Freshness   string   `json:"freshness"`
	Drift       []string `json:"drift,omitempty"`
	SourceGrade string   `json:"source_grade,omitempty"` // SQ-002/SQ-001
	ReuseValue  string   `json:"reuse_value,omitempty"`  // SQ-002/SQ-001
	OpenGaps    []string `json:"open_gaps,omitempty"`    // SQ-002/SQ-001
	JewelCount  int      `json:"jewel_count,omitempty"`  // SQ-009
}

type jsonIndex struct {
	Artifact   string `json:"artifact"`
	Health     string `json:"health"`
	CompiledAt string `json:"compiled_at,omitempty"`
}

type jsonTreasureChestOutput struct {
	Chests   []jsonChestRow `json:"chests"`
	Index    jsonIndex      `json:"index"`
	Warnings []string       `json:"warnings,omitempty"`
}

func renderTreasureChestJSON(w *os.File, root string, rows []treasure.StatusRow, compErr, govErr, idxErr error, compiledAt int64) error {
	indexPath := filepath.Join(root, ".compiled", ".index.gz")
	var health, ts string
	switch {
	case compErr != nil:
		health = "corrupt"
	case compiledAt == 0:
		health = "absent"
	default:
		health = "ok"
		ts = time.Unix(compiledAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	out := jsonTreasureChestOutput{
		Chests:   make([]jsonChestRow, 0, len(rows)),
		Index:    jsonIndex{Artifact: indexPath, Health: health, CompiledAt: ts},
		Warnings: collectWarnings(rows, govErr, idxErr, compErr, compiledAt),
	}
	for _, r := range rows {
		out.Chests = append(out.Chests, jsonChestRow{
			ID:          r.ID,
			Path:        r.Path,
			Scope:       r.Scope,
			Trust:       r.TrustTier,
			Freshness:   r.Freshness,
			Drift:       r.Drift,
			SourceGrade: r.SourceGrade,
			ReuseValue:  r.ReuseValue,
			OpenGaps:    r.OpenGaps,
			JewelCount:  r.JewelCount,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest: encode json: %w", err)
	}
	return nil
}

// --- rendering ---

func renderChestsSection(w *tabwriter.Writer, rows []treasure.StatusRow) error {
	if _, err := fmt.Fprintln(w, "  CHESTS\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"ID", "PATH", "SCOPE", "TRUST", "FRESHNESS", "DRIFT", "GRADE", "REUSE", "GAPS", "JEWELS"); err != nil {
		return fmt.Errorf("treasure-chest: write column header: %w", err)
	}
	for _, r := range rows {
		if err := renderChestRow(w, r); err != nil {
			return fmt.Errorf("treasure-chest: write row: %w", err)
		}
	}
	return nil
}

func renderChestRow(w *tabwriter.Writer, r treasure.StatusRow) error {
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		r.ID,
		r.Path,
		dashIfEmpty(strings.Join(r.Scope, ",")),
		dashIfEmpty(r.TrustTier),
		r.Freshness,
		driftText(r.Drift),
		dashIfEmpty(r.SourceGrade),
		dashIfEmpty(r.ReuseValue),
		countOrDash(len(r.OpenGaps)),
		countOrDash(r.JewelCount),
	); err != nil {
		return fmt.Errorf("treasure-chest status: write chest row: %w", err)
	}
	return nil
}

func driftText(drift []string) string {
	if len(drift) == 0 {
		return "none"
	}
	return strings.Join(drift, " ")
}

func countOrDash(count int) string {
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", count)
}

func dashIfEmpty(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

func renderIndexSection(w *tabwriter.Writer, root string, compiledAt int64, compErr error) error {
	if _, err := fmt.Fprintln(w, "  INDEX\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}

	indexPath := filepath.Join(root, ".compiled", ".index.gz")
	var health, ts string

	switch {
	case compErr != nil:
		health = "corrupt"
		ts = "—"
	case compiledAt == 0:
		health = "absent"
		ts = "—"
	default:
		health = "ok"
		ts = time.Unix(compiledAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	rows := []struct{ k, v string }{
		{"  artifact", indexPath},
		{"  health", health},
		{"  compiled_at", ts},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\t\t\t\t\n", row.k, row.v); err != nil {
			return fmt.Errorf("treasure-chest: write index row: %w", err)
		}
	}
	return nil
}

func renderWarningsSection(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("  WARNINGS")
	for _, warn := range warnings {
		fmt.Println("  " + warn)
	}
}

func collectWarnings(rows []treasure.StatusRow, govErr, idxErr, compErr error, compiledAt int64) []string {
	var w []string

	w = append(w, loadWarnings(govErr, idxErr)...)
	w = append(w, compiledIndexWarnings(compErr, compiledAt)...)

	var driftIDs []string
	var historicalMissing []string
	for _, r := range rows {
		driftIDs = appendDriftID(driftIDs, r)
		historicalMissing = appendHistoricalMissing(historicalMissing, r)
	}
	return appendTreasureWarnings(w, driftIDs, historicalMissing, compiledAt)
}

func loadWarnings(govErr, idxErr error) []string {
	var warnings []string
	if govErr != nil {
		warnings = append(warnings, "⚠ treasure-chests.yaml unavailable: "+govErr.Error())
	}
	if idxErr != nil {
		warnings = append(warnings, "⚠ knowledge.index.yaml unavailable: "+idxErr.Error())
	}
	return warnings
}

func compiledIndexWarnings(compErr error, compiledAt int64) []string {
	if compErr != nil {
		return []string{"⚠ .compiled/.index.gz corrupt — run: strategist treasure-chest --index"}
	}
	if compiledAt == 0 {
		return []string{"⚠ compiled index absent — run: strategist treasure-chest --index"}
	}
	return nil
}

func appendDriftID(ids []string, r treasure.StatusRow) []string {
	if len(r.Drift) == 0 {
		return ids
	}
	return append(ids, r.ID+"("+strings.Join(r.Drift, ",")+")")
}

func appendHistoricalMissing(ids []string, r treasure.StatusRow) []string {
	if (r.TrustTier == "T2" || r.TrustTier == "T3") && r.LastReviewed == "" {
		return append(ids, r.ID)
	}
	return ids
}

func appendTreasureWarnings(warnings, driftIDs, historicalMissing []string, compiledAt int64) []string {
	if len(driftIDs) > 0 {
		warnings = append(warnings, "⚠ drift detected: "+strings.Join(driftIDs, " "))
		if compiledAt != 0 {
			warnings = append(warnings, "  → run: strategist treasure-chest --index to refresh")
		}
	}
	if len(historicalMissing) > 0 {
		warnings = append(warnings, "⚠ historical sources missing last_reviewed (freshness=unknown): "+strings.Join(historicalMissing, ", "))
	}
	return warnings
}

// telemetryRunFromCmd extracts the MissionRun from the command context, if any.
func telemetryRunFromCmd(cmd *cobra.Command) *telemetry.MissionRun {
	if cmd == nil {
		return nil
	}
	ctx := cmd.Context()
	if ctx == nil {
		return nil
	}
	return telemetry.MissionRunFromContext(ctx)
}

func treasureChestRootFromCmd(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	root, err := cmd.Flags().GetString("root")
	if err == nil {
		return root
	}
	root, err = cmd.InheritedFlags().GetString("root")
	if err == nil {
		return root
	}
	return ""
}

func stringFlag(cmd *cobra.Command, name, fallback string) string {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetString(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetString(name)
	if err == nil {
		return value
	}
	return fallback
}

func boolFlag(cmd *cobra.Command, name string, fallback bool) bool {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetBool(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetBool(name)
	if err == nil {
		return value
	}
	return fallback
}

// printTreasureChestBanner prints the ASCII header for the treasure-chest command.
func printTreasureChestBanner() {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  treasure-chest                                     │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func init() {
	opts := treasureChestOptions{Format: "table"}
	treasureChestCmd.PersistentFlags().StringVar(&opts.Root, "root", "", "path to .strategist/ root (default: auto-discovered from CWD)")
	treasureChestCmd.Flags().BoolVar(&opts.DoIndex, "index", false, "rebuild compiled knowledge index from declared sources")
	treasureChestCmd.Flags().BoolVar(&opts.IncludeHistorical, "include-historical", false, "include T2/T3 historical sources in index rebuild (requires --index)")
	treasureChestCmd.Flags().StringVar(&opts.Format, "format", "table", "output format: table or json")
	treasureChestCmd.Flags().StringVar(&opts.Scope, "scope", "", "filter output by slot scope (e.g. discovery, refinement, execution)")
	treasureChestCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChest(cmd, args, opts)
	}
	rootCmd.AddCommand(treasureChestCmd)
}
