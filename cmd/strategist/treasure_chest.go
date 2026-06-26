package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- YAML parsing types ---

// scopeVal accepts both scalar ("all") and sequence (["discovery", "refinement"]) in YAML.
type scopeVal []string

func (s *scopeVal) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*s = []string{value.Value}
		return nil
	}
	var vs []string
	if err := value.Decode(&vs); err != nil {
		return fmt.Errorf("decode scope: %w", err)
	}
	*s = vs
	return nil
}

type activeChestEntry struct {
	ID    string   `yaml:"id"`
	Path  string   `yaml:"path"`
	Scope scopeVal `yaml:"scope"`
}

type activeYAMLWithChests struct {
	TreasureChests []activeChestEntry `yaml:"treasure_chests"`
}

type govTrust struct {
	Tier         string `yaml:"tier"`
	ReviewedBy   string `yaml:"reviewed_by"`
	LastReviewed string `yaml:"last_reviewed"`
}

type govChest struct {
	ID    string   `yaml:"id"`
	Title string   `yaml:"title"`
	Path  string   `yaml:"path"`
	Trust govTrust `yaml:"trust"`
}

type govManifest struct {
	Chests []govChest `yaml:"chests"`
}

type indexedEntry struct {
	ID string `yaml:"id"`
}

type knowledgeIndexYAML struct {
	Sources []indexedEntry `yaml:"sources"`
}

// --- row model ---

type chestRow struct {
	id           string
	path         string
	scope        []string
	trustTier    string
	reviewedBy   string
	lastReviewed string
	configured   bool // declared in active.yaml treasure_chests
	governed     bool // declared in treasure-chests.yaml
	indexed      bool // registered in knowledge.index.yaml
	compiled     bool // present in .compiled/.index.gz source_meta
	freshness    string
	drift        []string
}

// --- command flags ---

var (
	treasureChestRoot              string
	treasureChestDoIndex           bool
	treasureChestIncludeHistorical bool
	treasureChestFormat            string
	treasureChestScope             string
)

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
	RunE: runTreasureChest,
}

func runTreasureChest(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest: get cwd: %w", err)
	}

	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest: %w", err)
	}

	// Load the four truth layers (each is best-effort; only active.yaml is mandatory).
	activeChests, err := loadActiveChests(root)
	if err != nil {
		return fmt.Errorf("treasure-chest: %w", err)
	}

	governed, govErr := loadGoverned(root)
	indexed, idxErr := loadIndexed(root)
	compiledIDs, compiledAt, compErr := loadCompiledIndex(root)

	rows := mergeChestRows(activeChests, governed, indexed, compiledIDs)

	if treasureChestDoIndex {
		return runTreasureChestIndex(root, rows)
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

func runTreasureChestIndex(root string, rows []chestRow) error {
	indexPath := filepath.Join(root, "knowledge.index.yaml")

	if !treasureChestIncludeHistorical {
		historical := historicalCount(rows)
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

func historicalCount(rows []chestRow) int {
	n := 0
	for _, r := range rows {
		if r.trustTier == "T2" || r.trustTier == "T3" {
			n++
		}
	}
	return n
}

// --- loading helpers ---

func loadActiveChests(root string) ([]activeChestEntry, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // G304
	if err != nil {
		return nil, fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg activeYAMLWithChests
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse active.yaml: %w", err)
	}
	return cfg.TreasureChests, nil
}

func loadGoverned(root string) (map[string]govChest, error) { //nolint:dupl
	raw, err := os.ReadFile(filepath.Join(root, "treasure-chests.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	var m govManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse treasure-chests.yaml: %w", err)
	}
	out := make(map[string]govChest, len(m.Chests))
	for _, c := range m.Chests {
		out[c.ID] = c
	}
	return out, nil
}

func loadIndexed(root string) (map[string]bool, error) { //nolint:dupl
	raw, err := os.ReadFile(filepath.Join(root, "knowledge.index.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read knowledge.index.yaml: %w", err)
	}
	var ki knowledgeIndexYAML
	if err := yaml.Unmarshal(raw, &ki); err != nil {
		return nil, fmt.Errorf("parse knowledge.index.yaml: %w", err)
	}
	out := make(map[string]bool, len(ki.Sources))
	for _, s := range ki.Sources {
		out[s.ID] = true
	}
	return out, nil
}

func loadCompiledIndex(root string) (map[string]bool, int64, error) {
	path := filepath.Join(root, ".compiled", ".index.gz")
	f, err := os.Open(path) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("open .compiled/.index.gz: %w", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, 0, fmt.Errorf("decompress .compiled/.index.gz: %w", err)
	}
	defer func() { _ = gz.Close() }() //nolint:errcheck

	var idx struct {
		CompiledAt int64          `json:"compiled_at"`
		SourceMeta map[string]any `json:"source_meta"`
	}
	if err := json.NewDecoder(gz).Decode(&idx); err != nil {
		return nil, 0, fmt.Errorf("decode .compiled/.index.gz: %w", err)
	}

	present := make(map[string]bool, len(idx.SourceMeta))
	for id := range idx.SourceMeta {
		present[id] = true
	}
	return present, idx.CompiledAt, nil
}

// --- merge ---

func mergeChestRows(
	active []activeChestEntry,
	governed map[string]govChest,
	indexed map[string]bool,
	compiledIDs map[string]bool,
) []chestRow {
	seen := make(map[string]bool)
	var rows []chestRow

	for _, ac := range active {
		row := chestRow{
			id:         ac.ID,
			path:       ac.Path,
			scope:      ac.Scope,
			configured: true,
			indexed:    indexed[ac.ID],
			compiled:   compiledIDs[ac.ID],
		}
		if gc, ok := governed[ac.ID]; ok {
			row.governed = true
			row.trustTier = gc.Trust.Tier
			row.reviewedBy = gc.Trust.ReviewedBy
			row.lastReviewed = gc.Trust.LastReviewed
		}
		row.freshness = deriveFreshness(row)
		row.drift = deriveDrift(row)
		rows = append(rows, row)
		seen[ac.ID] = true
	}

	// Governed chests not declared in active.yaml
	for id, gc := range governed {
		if seen[id] {
			continue
		}
		row := chestRow{
			id:           id,
			path:         gc.Path,
			governed:     true,
			trustTier:    gc.Trust.Tier,
			reviewedBy:   gc.Trust.ReviewedBy,
			lastReviewed: gc.Trust.LastReviewed,
			indexed:      indexed[id],
			compiled:     compiledIDs[id],
		}
		row.freshness = deriveFreshness(row)
		row.drift = deriveDrift(row)
		rows = append(rows, row)
		seen[id] = true
	}

	// Indexed sources not declared anywhere else
	for id := range indexed {
		if seen[id] {
			continue
		}
		row := chestRow{
			id:        id,
			indexed:   true,
			compiled:  compiledIDs[id],
			freshness: "unknown",
		}
		row.drift = deriveDrift(row)
		rows = append(rows, row)
		seen[id] = true
	}

	return rows
}

func deriveFreshness(r chestRow) string {
	if r.lastReviewed != "" {
		return "fresh"
	}
	return "unknown"
}

func deriveDrift(r chestRow) []string {
	var d []string
	if r.configured && !r.governed {
		d = append(d, "missing_governance")
	}
	if (r.configured || r.governed) && !r.indexed {
		d = append(d, "missing_index")
	}
	if !r.configured && (r.governed || r.indexed) {
		d = append(d, "unscoped")
	}
	return d
}

// --- rendering ---

func renderChestsSection(w *tabwriter.Writer, rows []chestRow) error {
	if _, err := fmt.Fprintln(w, "  CHESTS\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
		"ID", "PATH", "SCOPE", "TRUST", "FRESHNESS", "DRIFT"); err != nil {
		return fmt.Errorf("treasure-chest: write column header: %w", err)
	}
	for _, r := range rows {
		scope := strings.Join(r.scope, ",")
		if scope == "" {
			scope = "—"
		}
		trust := r.trustTier
		if trust == "" {
			trust = "—"
		}
		drift := "none"
		if len(r.drift) > 0 {
			drift = strings.Join(r.drift, " ")
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
			r.id, r.path, scope, trust, r.freshness, drift); err != nil {
			return fmt.Errorf("treasure-chest: write row: %w", err)
		}
	}
	return nil
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

func collectWarnings(rows []chestRow, govErr, idxErr, compErr error, compiledAt int64) []string {
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
		if len(r.drift) > 0 {
			driftIDs = append(driftIDs, r.id+"("+strings.Join(r.drift, ",")+")")
		}
		if (r.trustTier == "T2" || r.trustTier == "T3") && r.lastReviewed == "" {
			historicalMissing = append(historicalMissing, r.id)
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

// printTreasureChestBanner prints the ASCII header for the treasure-chest command.
func printTreasureChestBanner() {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  treasure-chest                                     │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func init() {
	treasureChestCmd.Flags().StringVar(&treasureChestRoot, "root", "", "path to .strategist/ root (default: auto-discovered from CWD)")
	treasureChestCmd.Flags().BoolVar(&treasureChestDoIndex, "index", false, "rebuild compiled knowledge index from declared sources")
	treasureChestCmd.Flags().BoolVar(&treasureChestIncludeHistorical, "include-historical", false, "include T2/T3 historical sources in index rebuild (requires --index)")
	treasureChestCmd.Flags().StringVar(&treasureChestFormat, "format", "table", "output format: table or json")
	treasureChestCmd.Flags().StringVar(&treasureChestScope, "scope", "", "filter output by slot scope (e.g. discovery, refinement, execution)")
	rootCmd.AddCommand(treasureChestCmd)
}
