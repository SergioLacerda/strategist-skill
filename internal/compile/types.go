package compile

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// compiledConfig is the in-memory representation of a compiled config artifact.
// Active and Personas are raw maps so all YAML fields (including unknown/extended ones
// like content_by_lang or treasure_chests) are preserved verbatim in the artifact.
type compiledConfig struct {
	Schema     string           `json:"schema"`
	CompiledAt int64            `json:"compiled_at"`
	Sources    map[string]int64 `json:"sources"`
	Active     map[string]any   `json:"active"`
	Personas   map[string]any   `json:"personas"`
	Roles      map[string]any   `json:"roles"`
}

// compiledDomain is the in-memory representation of a compiled domain artifact.
type compiledDomain struct {
	Schema         string                    `json:"schema"`
	CompiledAt     int64                     `json:"compiled_at"`
	Sources        map[string]int64          `json:"sources"`
	LoadAlways     map[string]any            `json:"load_always"`
	LoadByTaskType map[string]map[string]any `json:"load_by_task_type"`
}

// compiledIndex is the in-memory representation of a compiled knowledge index artifact.
type compiledIndex struct {
	Schema     string              `json:"schema"`
	CompiledAt int64               `json:"compiled_at"`
	Sources    map[string]int64    `json:"sources"`
	Tags       map[string][]string `json:"tags"`
	SourceMeta map[string]any      `json:"source_meta"`
}

// compiledManifest records artifact paths and their SHA256 checksums.
type compiledManifest struct {
	Schema      string            `json:"schema"`
	GeneratedAt int64             `json:"generated_at"`
	Artifacts   map[string]string `json:"artifacts"` // name → "sha256:<hex>"
}

// IndexSource represents one entry in knowledge.index.yaml.
type IndexSource struct {
	ID   string   `yaml:"id"`
	Tags []string `yaml:"tags"`
}

// KnowledgeIndex is the structure of knowledge.index.yaml.
type KnowledgeIndex struct {
	Sources []IndexSource `yaml:"sources"`
}

// DomainIndex is the structure of a strategist domain index.yaml.
type DomainIndex struct {
	LoadAlways     []string            `yaml:"load_always"`
	LoadByTaskType map[string][]string `yaml:"load_by_task_type"`
}

// PhaseLabels is a re-export of domain.PhaseLabels for compile-package consumers.
type PhaseLabels = domain.PhaseLabels

// ActiveConfig is a re-export of domain.ActiveConfig for compile-package consumers.
type ActiveConfig = domain.ActiveConfig

// PersonaConfig is a re-export of domain.PersonaConfig for compile-package consumers.
type PersonaConfig = domain.PersonaConfig

// ApprovalGateContract is a re-export of domain.ApprovalGateContract for compile-package consumers.
type ApprovalGateContract = domain.ApprovalGateContract
