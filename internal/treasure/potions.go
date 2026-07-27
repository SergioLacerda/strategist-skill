package treasure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// Potion is a compact runbook-index entry, a child of the "runbooks" treasure chest.
// Sibling of Jewel: a jewel is a fact extracted from a mission, a potion is an index
// entry for one whole runbook file under docs/runbooks/. Schema owned by
// internal/embed/defaults/potions.yaml (see the header comment there) — this type
// consumes that schema, it does not redecide it.
type Potion struct {
	ID           string   `yaml:"id"`
	ChestID      string   `yaml:"chest_id"`
	RunbookRef   string   `yaml:"runbook_ref"`
	WhenToUse    string   `yaml:"when_to_use"`
	WhenToAvoid  string   `yaml:"when_to_avoid,omitempty"`
	Trust        string   `yaml:"trust"`
	Status       string   `yaml:"status"`
	SourceRefs   []string `yaml:"source_refs"`
	ReviewedBy   string   `yaml:"reviewed_by"`
	LastReviewed string   `yaml:"last_reviewed,omitempty"`
}

// PotionManifest is the top-level potions.yaml document.
type PotionManifest struct {
	SchemaVersion string   `yaml:"schema_version"`
	Potions       []Potion `yaml:"potions"`
}

// ErrPotionNotFound is returned when no potion with the given id exists — lets callers
// that also check Jewel (the `items` CLI dispatch) distinguish "not a potion" from a
// genuine I/O or validation error.
var ErrPotionNotFound = errors.New("potion not found")

// --- loading ---

// LoadPotions reads monolithic potions.yaml plus optional potions/*.yaml partitions,
// validates each potion, and groups entries by chest id. Mirrors LoadJewels.
func LoadPotions(root string, governed map[string]GovernedChest) (map[string][]Potion, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	out := make(map[string][]Potion)
	seen := make(map[string]bool)
	for _, path := range paths {
		if err := loadPotionsFromManifest(path, root, governed, out, seen); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func loadPotionsFromManifest(path, root string, governed map[string]GovernedChest, out map[string][]Potion, seen map[string]bool) error {
	m, err := readPotionManifest(path, root)
	if err != nil {
		return err
	}
	for _, p := range m.Potions {
		if err := addPotionFromManifest(p, governed, out, seen); err != nil {
			return fmt.Errorf("%s: %w", potionManifestLabel(root, path), err)
		}
	}
	return nil
}

func readPotionManifest(path, root string) (PotionManifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return PotionManifest{}, fmt.Errorf("read %s: %w", potionManifestLabel(root, path), err)
	}
	var m PotionManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return PotionManifest{}, fmt.Errorf("parse %s: %w", potionManifestLabel(root, path), err)
	}
	if m.SchemaVersion != "1" {
		return PotionManifest{}, fmt.Errorf("%s: unsupported schema_version %q (expected \"1\")", potionManifestLabel(root, path), m.SchemaVersion)
	}
	return m, nil
}

func addPotionFromManifest(p Potion, governed map[string]GovernedChest, out map[string][]Potion, seen map[string]bool) error {
	if seen[p.ID] {
		return nil
	}
	if err := ValidatePotionEntry(p, governed); err != nil {
		return err
	}
	out[p.ChestID] = append(out[p.ChestID], p)
	seen[p.ID] = true
	return nil
}

// PotionManifestPaths returns monolithic and partitioned potion manifests in stable read order.
func PotionManifestPaths(root string) ([]string, error) {
	paths, err := monolithicPotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	partitionPaths, err := partitionedPotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	return append(paths, partitionPaths...), nil
}

func monolithicPotionManifestPaths(root string) ([]string, error) {
	path := filepath.Join(root, "potions.yaml")
	if _, err := os.Stat(path); err == nil {
		return []string{path}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat potions.yaml: %w", err)
	}
	return nil, nil
}

func partitionedPotionManifestPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "potions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read potions/: %w", err)
	}
	var partitionPaths []string
	for _, entry := range entries {
		if isJewelPartitionFile(entry) { // suffix check only, not jewel-specific despite the name
			partitionPaths = append(partitionPaths, filepath.Join(root, "potions", entry.Name()))
		}
	}
	sort.Strings(partitionPaths)
	return partitionPaths, nil
}

func potionManifestLabel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// ValidatePotionEntry validates a single potion against schema and parent trust rules.
func ValidatePotionEntry(p Potion, governed map[string]GovernedChest) error {
	for _, validate := range potionValidators(p, governed) {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func potionValidators(p Potion, governed map[string]GovernedChest) []func() error {
	return []func() error{
		func() error { return validatePotionChest(p) },
		func() error { return validatePotionRunbookRef(p) },
		func() error { return validatePotionSourceRefs(p) },
		func() error { return wrapJewelValidation("status", domain.ValidatePotionStatus(p.ID, p.Status)) },
		func() error { return validatePotionTrust(p, governed) },
	}
}

func validatePotionChest(p Potion) error {
	if p.ChestID == "" {
		return fmt.Errorf("potion %q missing chest_id", p.ID)
	}
	return nil
}

func validatePotionRunbookRef(p Potion) error {
	if p.RunbookRef == "" {
		return fmt.Errorf("potion %q missing runbook_ref", p.ID)
	}
	return nil
}

func validatePotionSourceRefs(p Potion) error {
	if len(p.SourceRefs) == 0 {
		return fmt.Errorf("potion %q missing source_refs", p.ID)
	}
	return nil
}

func validatePotionTrust(p Potion, governed map[string]GovernedChest) error {
	gc, ok := governed[p.ChestID]
	if !ok {
		return nil
	}
	return wrapJewelValidation("trust", domain.ValidatePotionTrust(p.ID, p.Trust, gc.Trust.Tier))
}

// --- query ---

// PotionFilter selects potions for read-only listing.
type PotionFilter struct {
	ChestID string
	Status  string
}

// FilterPotions flattens grouped potions, applies the requested filter, and returns a
// deterministic chest/id ordering for CLI and JSON output. Mirrors FilterJewels.
func FilterPotions(potionsByChest map[string][]Potion, filter PotionFilter) []Potion {
	filtered := make([]Potion, 0)
	for _, list := range potionsByChest {
		for _, p := range list {
			if potionMatchesFilter(p, filter) {
				filtered = append(filtered, p)
			}
		}
	}
	SortPotions(filtered)
	return filtered
}

// ProposedPotions returns the deterministic status:proposed curation queue.
func ProposedPotions(potionsByChest map[string][]Potion) []Potion {
	return FilterPotions(potionsByChest, PotionFilter{Status: domain.PotionStatusProposed})
}

// FindPotion returns the first potion with the given id from grouped potions.
func FindPotion(potionsByChest map[string][]Potion, id string) (Potion, bool) {
	for _, list := range potionsByChest {
		for _, p := range list {
			if p.ID == id {
				return p, true
			}
		}
	}
	return Potion{}, false
}

// SortPotions orders potions by chest id, then potion id.
func SortPotions(potions []Potion) {
	sort.Slice(potions, func(i, k int) bool {
		if potions[i].ChestID != potions[k].ChestID {
			return potions[i].ChestID < potions[k].ChestID
		}
		return potions[i].ID < potions[k].ID
	})
}

func potionMatchesFilter(p Potion, filter PotionFilter) bool {
	if filter.ChestID != "" && p.ChestID != filter.ChestID {
		return false
	}
	switch filter.Status {
	case "":
		return p.Status != domain.PotionStatusDeprecated
	case "all":
		return true
	default:
		return p.Status == filter.Status
	}
}

// --- lifecycle ---

// PromotePotion updates a potion lifecycle status while preserving YAML structure.
// Mirrors PromoteJewel. evidenceRef is accepted for CLI-surface parity with
// PromoteJewel/`items verify --evidence` but not persisted: the Potion schema (see
// internal/embed/defaults/potions.yaml) has no verification/evidence_refs field, and
// this function must not redecide that schema.
func PromotePotion(root, id, newStatus, _ string, reviewedAt time.Time) error {
	path, doc, entry, err := FindPotionDocument(root, id)
	if err != nil {
		return err
	}

	current := ""
	if v := MappingValue(entry, "status"); v != nil {
		current = v.Value
	}
	if current == domain.PotionStatusDeprecated && newStatus != domain.PotionStatusDeprecated {
		return fmt.Errorf("potion %q is deprecated, cannot promote to %s", id, newStatus)
	}

	SetMappingField(entry, "status", newStatus)
	SetMappingField(entry, "reviewed_by", "human")
	SetMappingField(entry, "last_reviewed", reviewedAt.UTC().Format("2006-01-02"))

	if _, err := WriteYAMLNodes(YAMLWrite{Path: path, Doc: doc}); err != nil {
		return err
	}
	return nil
}

// FindPotionDocument returns the manifest document and potion mapping for id across
// monolithic and partitioned potion manifests.
func FindPotionDocument(root, id string) (string, *yaml.Node, *yaml.Node, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return "", nil, nil, err
	}
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return "", nil, nil, err
		}
		entry, err := FindPotionEntry(doc, id)
		if err == nil {
			return path, doc, entry, nil
		}
	}
	return "", nil, nil, fmt.Errorf("potion %q: %w (checked potions.yaml and potions/)", id, ErrPotionNotFound)
}

// FindPotionEntry returns the potions.yaml potion entry mapping node for id, or an
// error if no potions are declared or id is not found.
func FindPotionEntry(doc *yaml.Node, id string) (*yaml.Node, error) {
	root, err := RootMapping(doc)
	if err != nil {
		return nil, err
	}
	seq := MappingValue(root, "potions")
	if seq == nil {
		return nil, fmt.Errorf("no potions declared in potions.yaml")
	}
	entry, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return nil, fmt.Errorf("potion %q not found in potions.yaml", id)
	}
	return entry, nil
}

// NewPotionsDocument creates an empty schema-versioned potions manifest.
func NewPotionsDocument() *yaml.Node {
	root := mapNode(
		strNode("schema_version"), strNode("1"),
		strNode("potions"), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
	)
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
}

// ReadOrCreatePotionsDocument loads a potions manifest or creates an empty one.
func ReadOrCreatePotionsDocument(path string) (*yaml.Node, error) {
	doc, err := ReadYAMLNode(path)
	if err == nil {
		return doc, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return NewPotionsDocument(), nil
	}
	return nil, err
}

// PotionPartitionPath returns the partition manifest path for a chest id.
func PotionPartitionPath(root, chestID string) string {
	name := strings.NewReplacer("/", "_", "\\", "_").Replace(chestID)
	return filepath.Join(root, "potions", name+".yaml")
}

// --- index scan extension (ask #1 / SQ-001) ---

var runbookHeadingRe = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// ScanRunbookDirectory scans a "runbooks"-kind chest directory (docs/runbooks/*.md) and
// builds status:proposed Potion candidates, one per runbook file. when_to_use is
// extracted from the first paragraph of the file's first "## " section (Symptom for
// diagnostic runbooks, Trigger for procedural ones) — header-extracted, not
// LLM-generated, per .analysis/refined/runbook-jewel-relevance-mechanism/design.md.
// A missing directory is not an error — it just yields no candidates.
func ScanRunbookDirectory(chestID, trustTier, dirPath string) ([]Potion, error) {
	entries, err := readRunbookDirEntries(dirPath)
	if err != nil {
		return nil, err
	}

	var candidates []Potion
	for _, entry := range entries {
		if !isRunbookCandidateFile(entry) {
			continue
		}
		candidate, err := runbookFileToPotion(chestID, trustTier, dirPath, entry.Name())
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates, nil
}

// readRunbookDirEntries reads dirPath's entries. A missing directory is not an
// error — it returns a nil slice so callers yield no candidates.
func readRunbookDirEntries(dirPath string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan runbook directory %s: %w", dirPath, err)
	}
	return entries, nil
}

func isRunbookCandidateFile(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	return strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "README.md")
}

func runbookFileToPotion(chestID, trustTier, dirPath, fileName string) (Potion, error) {
	path := filepath.Join(dirPath, fileName)
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return Potion{}, fmt.Errorf("read %s: %w", path, err)
	}
	slug := strings.TrimSuffix(fileName, ".md")
	runbookRef := "docs/runbooks/" + fileName
	return Potion{
		ID:         "potion-" + slug,
		ChestID:    chestID,
		RunbookRef: runbookRef,
		WhenToUse:  extractRunbookWhenToUse(string(raw)),
		Trust:      trustTier,
		Status:     domain.PotionStatusProposed,
		SourceRefs: []string{runbookRef},
		ReviewedBy: "agent",
	}, nil
}

// extractRunbookWhenToUse extracts a short summary from a runbook's first "## " section
// (e.g. "Symptom" or "Trigger") — the first non-empty paragraph after that heading,
// truncated. Falls back to the H1 title when no "## " section is found.
func extractRunbookWhenToUse(content string) string {
	if loc := runbookHeadingRe.FindStringIndex(content); loc != nil {
		if para := firstParagraphAfter(content[loc[1]:]); para != "" {
			return truncateRunbookSummary(para)
		}
	}
	return truncateRunbookSummary(runbookTitleFallback(content))
}

// isParagraphBoundary reports whether trimmed (a line already stripped of
// surrounding whitespace) ends paragraph collection: either a blank line
// (which only stops collection once some content has been gathered, via
// hasContent) or a heading line, which always stops it outright.
func isParagraphBoundary(trimmed string, hasContent bool) (boundary, stopEntirely bool) {
	if trimmed == "" {
		return true, hasContent
	}
	if strings.HasPrefix(trimmed, "#") {
		return true, true
	}
	return false, false
}

func firstParagraphAfter(rest string) string {
	lines := strings.Split(rest, "\n")
	var para []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if boundary, stop := isParagraphBoundary(trimmed, len(para) > 0); boundary {
			if stop {
				break
			}
			continue
		}
		para = append(para, trimmed)
	}
	return strings.Join(para, " ")
}

func runbookTitleFallback(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return ""
}

func truncateRunbookSummary(s string) string {
	const maxLen = 220
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen]) + "…"
}

// --- index write (ask #1 / SQ-001) ---

// WriteProposedPotions appends new proposed potions while skipping existing ids.
// Mirrors WriteProposedJewels.
func WriteProposedPotions(root string, candidates []Potion) (written, skipped int, err error) {
	if len(candidates) == 0 {
		return 0, 0, nil
	}

	docs, written, skipped, err := buildProposedPotionDocs(root, candidates)
	if err != nil {
		return written, skipped, err
	}

	if written == 0 {
		return written, skipped, nil
	}
	if err := writeProposedPotionDocs(root, docs); err != nil {
		return written, skipped, err
	}
	return written, skipped, nil
}

func buildProposedPotionDocs(root string, candidates []Potion) (map[string]*yaml.Node, int, int, error) {
	existing, err := ExistingPotionIDs(root)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("index proposed potions: %w", err)
	}
	docs := make(map[string]*yaml.Node)
	written, skipped := 0, 0
	for _, cand := range candidates {
		if existing[cand.ID] {
			skipped++
			continue
		}
		if err := appendCandidateToPotionPartition(root, docs, cand); err != nil {
			return docs, written, skipped, err
		}
		existing[cand.ID] = true
		written++
	}
	return docs, written, skipped, nil
}

func writeProposedPotionDocs(root string, docs map[string]*yaml.Node) error {
	return writeProposedItemDocs(root, "potions", docs, "index proposed potions")
}

func appendCandidateToPotionPartition(root string, docs map[string]*yaml.Node, cand Potion) error {
	path := PotionPartitionPath(root, cand.ChestID)
	doc, ok := docs[path]
	if !ok {
		var err error
		doc, err = ReadOrCreatePotionsDocument(path)
		if err != nil {
			return fmt.Errorf("index proposed potions: %w", err)
		}
		docs[path] = doc
	}
	mapping, err := RootMapping(doc)
	if err != nil {
		return fmt.Errorf("index proposed potions: %w", err)
	}
	seq := FindOrCreateSequence(mapping, "potions")
	seq.Style = 0
	var entry yaml.Node
	if err := entry.Encode(cand); err != nil {
		return fmt.Errorf("index proposed potions: encode %s: %w", cand.ID, err)
	}
	seq.Content = append(seq.Content, &entry)
	return nil
}

// ExistingPotionIDs returns all ids present across monolithic and partitioned potion manifests.
func ExistingPotionIDs(root string) (map[string]bool, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool)
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return nil, err
		}
		if err := collectPotionIDs(doc, ids); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func collectPotionIDs(doc *yaml.Node, ids map[string]bool) error {
	mapping, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(mapping, "potions")
	if seq == nil {
		return nil
	}
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		if id := MappingValue(entry, "id"); id != nil {
			ids[id.Value] = true
		}
	}
	return nil
}
