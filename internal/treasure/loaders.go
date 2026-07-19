package treasure

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// Scope accepts both scalar ("all") and sequence (["discovery", "refinement"]) in YAML.
type Scope []string

// UnmarshalYAML decodes a scalar or sequence scope into a normalized string list.
func (s *Scope) UnmarshalYAML(value *yaml.Node) error {
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

// ActiveChest is a chest declaration from active.yaml.
type ActiveChest struct {
	ID    string `yaml:"id"`
	Path  string `yaml:"path"`
	Scope Scope  `yaml:"scope"`
}

type activeYAMLWithChests struct {
	TreasureChests []ActiveChest `yaml:"treasure_chests"`
}

// GovernedTrust is the trust metadata attached to a governed treasure chest.
type GovernedTrust struct {
	Tier         string `yaml:"tier"`
	ReviewedBy   string `yaml:"reviewed_by"`
	LastReviewed string `yaml:"last_reviewed"`
}

// GovernedGrade captures source quality and reuse metadata for a governed chest.
type GovernedGrade struct {
	SourceGrade          string `yaml:"source_grade"`
	ReuseValue           string `yaml:"reuse_value"`
	ImplementationStatus string `yaml:"implementation_status"`
	Provenance           string `yaml:"provenance"`
}

// GovernedChest is a chest entry from treasure-chests.yaml.
type GovernedChest struct {
	ID       string        `yaml:"id"`
	Title    string        `yaml:"title"`
	Path     string        `yaml:"path"`
	Trust    GovernedTrust `yaml:"trust"`
	Grade    GovernedGrade `yaml:"grade"`
	OpenGaps []string      `yaml:"open_gaps"`
	// Status is the tombstone marker: "" or "active" means active, "inactive" means removed.
	Status string `yaml:"status"`
}

type governedManifest struct {
	ScoringPolicy rawScoringPolicy `yaml:"scoring_policy"`
	Chests        []GovernedChest  `yaml:"chests"`
}

type rawScoringPolicy struct {
	ClusterBase          *int `yaml:"cluster_base"`
	ClusterMissionWeight *int `yaml:"cluster_mission_weight"`
	ClusterTagWeight     *int `yaml:"cluster_tag_weight"`
	GapBase              *int `yaml:"gap_base"`
	GapMissionWeight     *int `yaml:"gap_mission_weight"`
	MaxScore             *int `yaml:"max_score"`
}

// ScoringPolicy controls proposed-jewel score generation during treasure-chest index.
type ScoringPolicy struct {
	ClusterBase          int `yaml:"cluster_base" json:"cluster_base"`
	ClusterMissionWeight int `yaml:"cluster_mission_weight" json:"cluster_mission_weight"`
	ClusterTagWeight     int `yaml:"cluster_tag_weight" json:"cluster_tag_weight"`
	GapBase              int `yaml:"gap_base" json:"gap_base"`
	GapMissionWeight     int `yaml:"gap_mission_weight" json:"gap_mission_weight"`
	MaxScore             int `yaml:"max_score" json:"max_score"`
}

type indexedEntry struct {
	ID string `yaml:"id"`
}

type knowledgeIndexYAML struct {
	Sources []indexedEntry `yaml:"sources"`
}

// JewelScore stores a jewel's ranking value and supporting reasons.
type JewelScore struct {
	Value   int      `yaml:"value" json:"value"`
	Reasons []string `yaml:"reasons" json:"reasons"`
}

// JewelApplicability describes where and when a jewel should be applied.
type JewelApplicability struct {
	Scope       []string `yaml:"scope" json:"scope"`
	AppliesWhen []string `yaml:"applies_when" json:"applies_when"`
	AvoidWhen   []string `yaml:"avoid_when" json:"avoid_when"`
}

// JewelVerification stores evidence references for verified jewels.
type JewelVerification struct {
	EvidenceRefs []string `yaml:"evidence_refs" json:"evidence_refs"`
}

// JewelHistoryEntry records a lifecycle transition for a jewel.
type JewelHistoryEntry struct {
	Status      string `yaml:"status" json:"status"`
	At          string `yaml:"at" json:"at"`
	By          string `yaml:"by" json:"by"`
	EvidenceRef string `yaml:"evidence_ref,omitempty" json:"evidence_ref,omitempty"`
}

// Jewel is a compact knowledge unit anchored to a treasure chest.
type Jewel struct {
	ID            string              `yaml:"id"`
	ChestID       string              `yaml:"chest_id"`
	Kind          string              `yaml:"kind"`
	Statement     string              `yaml:"statement"`
	SourceRefs    []string            `yaml:"source_refs"`
	Trust         string              `yaml:"trust"`
	Status        string              `yaml:"status"`
	ReviewedBy    string              `yaml:"reviewed_by"`
	LastReviewed  string              `yaml:"last_reviewed"`
	Score         JewelScore          `yaml:"score"`
	Applicability JewelApplicability  `yaml:"applicability"`
	Verification  JewelVerification   `yaml:"verification"`
	History       []JewelHistoryEntry `yaml:"history,omitempty" json:"history,omitempty"`
}

// Manifest is the top-level jewels.yaml document.
type Manifest struct {
	SchemaVersion string  `yaml:"schema_version"`
	Jewels        []Jewel `yaml:"jewels"`
}

// LoadActiveChests reads active.yaml treasure_chests entries.
func LoadActiveChests(root string) ([]ActiveChest, error) {
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

// LoadGoverned reads treasure-chests.yaml and returns governed chests by id.
func LoadGoverned(root string) (map[string]GovernedChest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "treasure-chests.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	var m governedManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse treasure-chests.yaml: %w", err)
	}
	out := make(map[string]GovernedChest, len(m.Chests))
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

// DefaultScoringPolicy returns the legacy hardcoded score formula as configuration.
func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		ClusterBase:          40,
		ClusterMissionWeight: 10,
		ClusterTagWeight:     5,
		GapBase:              30,
		GapMissionWeight:     15,
		MaxScore:             100,
	}
}

// LoadScoringPolicy reads optional scoring_policy from treasure-chests.yaml.
func LoadScoringPolicy(root string) (ScoringPolicy, error) {
	raw, err := os.ReadFile(filepath.Join(root, "treasure-chests.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultScoringPolicy(), nil
		}
		return ScoringPolicy{}, fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	var m governedManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return ScoringPolicy{}, fmt.Errorf("parse treasure-chests.yaml: %w", err)
	}
	policy := DefaultScoringPolicy()
	applyScoringPolicyOverrides(&policy, m.ScoringPolicy)
	if err := ValidateScoringPolicy(policy); err != nil {
		return ScoringPolicy{}, fmt.Errorf("treasure-chests.yaml: scoring_policy: %w", err)
	}
	return policy, nil
}

func applyScoringPolicyOverrides(policy *ScoringPolicy, raw rawScoringPolicy) {
	if raw.ClusterBase != nil {
		policy.ClusterBase = *raw.ClusterBase
	}
	if raw.ClusterMissionWeight != nil {
		policy.ClusterMissionWeight = *raw.ClusterMissionWeight
	}
	if raw.ClusterTagWeight != nil {
		policy.ClusterTagWeight = *raw.ClusterTagWeight
	}
	if raw.GapBase != nil {
		policy.GapBase = *raw.GapBase
	}
	if raw.GapMissionWeight != nil {
		policy.GapMissionWeight = *raw.GapMissionWeight
	}
	if raw.MaxScore != nil {
		policy.MaxScore = *raw.MaxScore
	}
}

// ValidateScoringPolicy rejects negative weights and invalid score caps.
func ValidateScoringPolicy(policy ScoringPolicy) error {
	if policy.MaxScore < 1 || policy.MaxScore > 100 {
		return fmt.Errorf("max_score must be between 1 and 100")
	}
	for name, value := range map[string]int{
		"cluster_base":           policy.ClusterBase,
		"cluster_mission_weight": policy.ClusterMissionWeight,
		"cluster_tag_weight":     policy.ClusterTagWeight,
		"gap_base":               policy.GapBase,
		"gap_mission_weight":     policy.GapMissionWeight,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
	}
	return nil
}

// LoadIndexed reads knowledge.index.yaml and returns indexed source ids.
func LoadIndexed(root string) (map[string]bool, error) {
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

// LoadCompiledIndex reads the compiled fast-path index and returns source ids and timestamp.
func LoadCompiledIndex(root string) (present map[string]bool, compiledAt int64, err error) {
	path := filepath.Join(root, ".compiled", ".index.gz")
	idx, ok, err := readCompiledIndex(path)
	if err != nil || !ok {
		return nil, 0, err
	}
	return compiledSourcePresence(idx.SourceMeta), idx.CompiledAt, nil
}

type compiledIndexYAML struct {
	CompiledAt int64          `json:"compiled_at"`
	SourceMeta map[string]any `json:"source_meta"`
}

func readCompiledIndex(path string) (compiledIndexYAML, bool, error) {
	f, err := os.Open(path) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return compiledIndexYAML{}, false, nil
		}
		return compiledIndexYAML{}, false, fmt.Errorf("open .compiled/.index.gz: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close .compiled/.index.gz: %w", closeErr)
		}
	}()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return compiledIndexYAML{}, false, fmt.Errorf("decompress .compiled/.index.gz: %w", err)
	}
	defer func() {
		if closeErr := gz.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close decompressed .compiled/.index.gz: %w", closeErr)
		}
	}()

	var idx compiledIndexYAML
	if err := json.NewDecoder(gz).Decode(&idx); err != nil {
		return compiledIndexYAML{}, false, fmt.Errorf("decode .compiled/.index.gz: %w", err)
	}
	return idx, true, nil
}

func compiledSourcePresence(sourceMeta map[string]any) map[string]bool {
	present := make(map[string]bool, len(sourceMeta))
	for id := range sourceMeta {
		present[id] = true
	}
	return present
}

// LoadJewels reads monolithic jewels.yaml plus optional jewels/*.yaml partitions,
// validates each jewel, and groups entries by chest id. Duplicate jewel ids are loaded
// from the first file encountered so mixed-layout workspaces do not double-count jewels.
func LoadJewels(root string, governed map[string]GovernedChest) (map[string][]Jewel, error) {
	paths, err := JewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	out := make(map[string][]Jewel)
	seen := make(map[string]bool)
	for _, path := range paths {
		if err := loadJewelsFromManifest(path, root, governed, out, seen); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func loadJewelsFromManifest(path, root string, governed map[string]GovernedChest, out map[string][]Jewel, seen map[string]bool) error {
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return fmt.Errorf("read %s: %w", jewelManifestLabel(root, path), err)
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("parse %s: %w", jewelManifestLabel(root, path), err)
	}

	if m.SchemaVersion != "1" {
		return fmt.Errorf("%s: unsupported schema_version %q (expected \"1\")", jewelManifestLabel(root, path), m.SchemaVersion)
	}

	for _, j := range m.Jewels {
		if seen[j.ID] {
			continue
		}
		if err := ValidateJewelEntry(j, governed); err != nil {
			return fmt.Errorf("%s: %w", jewelManifestLabel(root, path), err)
		}
		out[j.ChestID] = append(out[j.ChestID], j)
		seen[j.ID] = true
	}
	return nil
}

// JewelManifestPaths returns monolithic and partitioned jewel manifests in stable read order.
func JewelManifestPaths(root string) ([]string, error) {
	paths, err := monolithicJewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	partitionPaths, err := partitionedJewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	return append(paths, partitionPaths...), nil
}

func monolithicJewelManifestPaths(root string) ([]string, error) {
	path := filepath.Join(root, "jewels.yaml")
	if _, err := os.Stat(path); err == nil {
		return []string{path}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat jewels.yaml: %w", err)
	}
	return nil, nil
}

func partitionedJewelManifestPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "jewels"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read jewels/: %w", err)
	}
	var partitionPaths []string
	for _, entry := range entries {
		if isJewelPartitionFile(entry) {
			partitionPaths = append(partitionPaths, filepath.Join(root, "jewels", entry.Name()))
		}
	}
	sort.Strings(partitionPaths)
	return partitionPaths, nil
}

func isJewelPartitionFile(entry os.DirEntry) bool {
	name := entry.Name()
	return !entry.IsDir() && (strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml"))
}

func jewelManifestLabel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// ValidateJewelEntry validates a single jewel against schema and parent trust rules.
func ValidateJewelEntry(j Jewel, governed map[string]GovernedChest) error {
	if j.ChestID == "" {
		return fmt.Errorf("jewel %q missing chest_id", j.ID)
	}
	if len(j.SourceRefs) == 0 {
		return fmt.Errorf("jewel %q missing source_refs", j.ID)
	}
	if err := domain.ValidateJewelStatus(j.ID, j.Status); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	if err := domain.ValidateJewelKind(j.ID, j.Kind); err != nil {
		return fmt.Errorf("kind: %w", err)
	}
	if err := domain.ValidateJewelScore(j.ID, j.Score.Value); err != nil {
		return fmt.Errorf("score: %w", err)
	}
	if gc, ok := governed[j.ChestID]; ok {
		if err := domain.ValidateJewelTrust(j.ID, j.Trust, gc.Trust.Tier); err != nil {
			return fmt.Errorf("trust: %w", err)
		}
	}
	return nil
}

// NonDeprecatedJewelCount counts jewels that are still active in the curation lifecycle.
func NonDeprecatedJewelCount(jewels []Jewel) int {
	n := 0
	for _, j := range jewels {
		if j.Status != domain.JewelStatusDeprecated {
			n++
		}
	}
	return n
}
