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
	opts.Root = stringFlag(cmd, "root", opts.Root)
	opts.DoIndex = boolFlag(cmd, "index", opts.DoIndex)
	opts.IncludeHistorical = boolFlag(cmd, "include-historical", opts.IncludeHistorical)
	opts.Format = stringFlag(cmd, "format", opts.Format)
	opts.Scope = stringFlag(cmd, "scope", opts.Scope)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest: get cwd: %w", err)
	}

	root, _, err := resolveStrategistRoot(opts.Root, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest: %w", err)
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

	printTreasureChestBanner()

	// Chests and Index sections each get their own tabwriter so column widths
	// are scoped per section (warnings would inflate column 0 if shared).
	wc := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := renderChestsSection(wc, rows); err != nil {
		return err
	}
	if err := wc.Flush(); err != nil {
		return fmt.Errorf("flush chests: %w", err)
	}
	fmt.Println()

	wi := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := renderIndexSection(wi, root, compiledAt, compErr); err != nil {
		return err
	}
	if err := wi.Flush(); err != nil {
		return fmt.Errorf("flush index: %w", err)
	}

	warnings := collectWarnings(rows, govErr, idxErr, compErr, compiledAt)
	renderWarningsSection(warnings)
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
		scope := strings.Join(r.Scope, ",")
		if scope == "" {
			scope = "—"
		}
		trust := r.TrustTier
		if trust == "" {
			trust = "—"
		}
		drift := "none"
		if len(r.Drift) > 0 {
			drift = strings.Join(r.Drift, " ")
		}
		grade := dashIfEmpty(r.SourceGrade)
		reuse := dashIfEmpty(r.ReuseValue)
		gaps := "—"
		if len(r.OpenGaps) > 0 {
			gaps = fmt.Sprintf("%d", len(r.OpenGaps))
		}
		jewels := "—"
		if r.JewelCount > 0 {
			jewels = fmt.Sprintf("%d", r.JewelCount)
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Path, scope, trust, r.Freshness, drift, grade, reuse, gaps, jewels); err != nil {
			return fmt.Errorf("treasure-chest: write row: %w", err)
		}
	}
	return nil
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

	if govErr != nil {
		w = append(w, "⚠ treasure-chests.yaml unavailable: "+govErr.Error())
	}
	if idxErr != nil {
		w = append(w, "⚠ knowledge.index.yaml unavailable: "+idxErr.Error())
	}
	if compErr != nil {
		w = append(w, "⚠ .compiled/.index.gz corrupt — run: strategist treasure-chest --index")
	} else if compiledAt == 0 {
		w = append(w, "⚠ compiled index absent — run: strategist treasure-chest --index")
	}

	var driftIDs []string
	var historicalMissing []string
	for _, r := range rows {
		if len(r.Drift) > 0 {
			driftIDs = append(driftIDs, r.ID+"("+strings.Join(r.Drift, ",")+")")
		}
		if (r.TrustTier == "T2" || r.TrustTier == "T3") && r.LastReviewed == "" {
			historicalMissing = append(historicalMissing, r.ID)
		}
	}
	if len(driftIDs) > 0 {
		w = append(w, "⚠ drift detected: "+strings.Join(driftIDs, " "))
		if compiledAt != 0 {
			w = append(w, "  → run: strategist treasure-chest --index to refresh")
		}
	}
	if len(historicalMissing) > 0 {
		w = append(w, "⚠ historical sources missing last_reviewed (freshness=unknown): "+strings.Join(historicalMissing, ", "))
	}
	return w
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
