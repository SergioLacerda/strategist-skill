package treasure

import (
	"fmt"
	"os"
	"path/filepath"

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

// LoadActiveChests reads active.yaml treasure_chests entries.
func LoadActiveChests(root string) ([]ActiveChest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // G304: active.yaml path is derived from the selected runtime root
	if err != nil {
		return nil, fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg activeYAMLWithChests
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse active.yaml: %w", err)
	}
	return cfg.TreasureChests, nil
}

// LoadIndexed reads knowledge.index.yaml and returns indexed source ids.
func LoadIndexed(root string) (map[string]bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "knowledge.index.yaml")) //nolint:gosec // G304: knowledge index path is derived from the selected runtime root
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
