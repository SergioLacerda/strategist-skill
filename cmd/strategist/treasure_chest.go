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
	"github.com/SergioLacerda/strategist-skill/internal/domain"
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

// govGrade holds SQ-002 (Track T-G) source grading fields. All fields are
// human-reviewed only in this pass — no derived or learned values.
type govGrade struct {
	SourceGrade          string `yaml:"source_grade"`
	ReuseValue           string `yaml:"reuse_value"`
	ImplementationStatus string `yaml:"implementation_status"`
	Provenance           string `yaml:"provenance"`
}

type govChest struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Path     string   `yaml:"path"`
	Trust    govTrust `yaml:"trust"`
	Grade    govGrade `yaml:"grade"`
	OpenGaps []string `yaml:"open_gaps"`
	// Status is the SQ-006 tombstone marker: "" or "active" means active,
	// "inactive" means removed via `treasure-chest remove` but retained for audit.
	Status string `yaml:"status"`
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
	sourceGrade  string   // SQ-002: source_grade (A|B|C), human-reviewed
	reuseValue   string   // SQ-002: reuse_value (high|medium|low), human-reviewed
	openGaps     []string // SQ-002: known gaps in this source, human-reviewed
	jewelCount   int      // SQ-009: count of non-deprecated jewels for this chest
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
	jewels, jewelErr := loadJewels(root, governed)
	if jewelErr != nil {
		return fmt.Errorf("treasure-chest: %w", jewelErr)
	}

	rows := mergeChestRows(activeChests, governed, indexed, compiledIDs, jewels)

	if treasureChestDoIndex {
		return runTreasureChestIndex(root, rows)
	}

	rows = filterRowsByScope(rows, treasureChestScope)

	switch treasureChestFormat {
	case "", "table":
		// fall through to table rendering below
	case "json":
		return renderTreasureChestJSON(os.Stdout, root, rows, compErr, govErr, idxErr, compiledAt)
	default:
		return fmt.Errorf("treasure-chest: unknown --format %q (want table or json)", treasureChestFormat)
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
		if err := domain.ValidateChestGrade(c.ID, domain.ChestGrade{
			SourceGrade:          c.Grade.SourceGrade,
			ReuseValue:           c.Grade.ReuseValue,
			ImplementationStatus: c.Grade.ImplementationStatus,
		}); err != nil {
			return nil, fmt.Errorf("treasure-chests.yaml: %w", err)
		}
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
	jewels map[string][]jewelEntry,
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
			jewelCount: nonDeprecatedJewelCount(jewels[ac.ID]),
		}
		if gc, ok := governed[ac.ID]; ok {
			row.governed = true
			row.trustTier = gc.Trust.Tier
			row.reviewedBy = gc.Trust.ReviewedBy
			row.lastReviewed = gc.Trust.LastReviewed
			row.sourceGrade = gc.Grade.SourceGrade
			row.reuseValue = gc.Grade.ReuseValue
			row.openGaps = gc.OpenGaps
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
			sourceGrade:  gc.Grade.SourceGrade,
			reuseValue:   gc.Grade.ReuseValue,
			openGaps:     gc.OpenGaps,
			indexed:      indexed[id],
			compiled:     compiledIDs[id],
			jewelCount:   nonDeprecatedJewelCount(jewels[id]),
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

// --- scope filtering ---

// filterRowsByScope keeps rows whose scope contains value or "all". Rows with
// no declared scope (not configured in active.yaml) are excluded from a
// scoped view since they state no applicability. Empty value is a no-op.
func filterRowsByScope(rows []chestRow, value string) []chestRow {
	if value == "" {
		return rows
	}
	out := make([]chestRow, 0, len(rows))
	for _, r := range rows {
		for _, s := range r.scope {
			if s == value || s == "all" {
				out = append(out, r)
				break
			}
		}
	}
	return out
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

func renderTreasureChestJSON(w *os.File, root string, rows []chestRow, compErr, govErr, idxErr error, compiledAt int64) error {
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
			ID:          r.id,
			Path:        r.path,
			Scope:       r.scope,
			Trust:       r.trustTier,
			Freshness:   r.freshness,
			Drift:       r.drift,
			SourceGrade: r.sourceGrade,
			ReuseValue:  r.reuseValue,
			OpenGaps:    r.openGaps,
			JewelCount:  r.jewelCount,
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

func renderChestsSection(w *tabwriter.Writer, rows []chestRow) error {
	if _, err := fmt.Fprintln(w, "  CHESTS\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"ID", "PATH", "SCOPE", "TRUST", "FRESHNESS", "DRIFT", "GRADE", "REUSE", "GAPS", "JEWELS"); err != nil {
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
		grade := dashIfEmpty(r.sourceGrade)
		reuse := dashIfEmpty(r.reuseValue)
		gaps := "—"
		if len(r.openGaps) > 0 {
			gaps = fmt.Sprintf("%d", len(r.openGaps))
		}
		jewels := "—"
		if r.jewelCount > 0 {
			jewels = fmt.Sprintf("%d", r.jewelCount)
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.id, r.path, scope, trust, r.freshness, drift, grade, reuse, gaps, jewels); err != nil {
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
	treasureChestCmd.PersistentFlags().StringVar(&treasureChestRoot, "root", "", "path to .strategist/ root (default: auto-discovered from CWD)")
	treasureChestCmd.Flags().BoolVar(&treasureChestDoIndex, "index", false, "rebuild compiled knowledge index from declared sources")
	treasureChestCmd.Flags().BoolVar(&treasureChestIncludeHistorical, "include-historical", false, "include T2/T3 historical sources in index rebuild (requires --index)")
	treasureChestCmd.Flags().StringVar(&treasureChestFormat, "format", "table", "output format: table or json")
	treasureChestCmd.Flags().StringVar(&treasureChestScope, "scope", "", "filter output by slot scope (e.g. discovery, refinement, execution)")
	rootCmd.AddCommand(treasureChestCmd)
}
