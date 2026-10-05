package opportunity_attack

import preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"

// SchemaVersion identifies the Opportunity Attack manifest format.
const SchemaVersion = "strategist-opportunity-attack/v1"

const (
	// KindADR identifies an architecture-decision opportunity.
	KindADR = "adr"
	// KindRunbook identifies an operational Runbook opportunity.
	KindRunbook = "runbook"
	// KindChest identifies a Treasure Chest opportunity.
	KindChest = "treasure_chest"
	// KindTreasureChest is the compatibility alias for KindChest.
	KindTreasureChest = KindChest
	// StatusProposed identifies a detected but not materialized opportunity.
	StatusProposed = "proposed"
)

// ChestIndexLoader is the composition boundary for Treasure Chest state. The
// Opportunity Attack feat evaluates source evidence, but the Treasure Chest
// package remains the owner of index loading and mutation.
type ChestIndexLoader func(root string) (map[string]bool, error)

// SourceEvidence is the minimal source coverage record needed by the Chest
// opportunity. It deliberately does not expose Treasure Chest domain types.
type SourceEvidence struct {
	ID             string
	Path           string
	CoverageStatus string
}

// Candidate is an independent opportunity. Detection never writes the
// canonical ADR or Runbook repositories.
type Candidate struct {
	ID                    string                 `yaml:"id" json:"id"`
	Kind                  string                 `yaml:"kind" json:"kind"`
	Warranted             bool                   `yaml:"warranted" json:"warranted"`
	Status                string                 `yaml:"status" json:"status"`
	Description           string                 `yaml:"description" json:"description"`
	Strategy              string                 `yaml:"strategy" json:"strategy"`
	EstimatedImpact       string                 `yaml:"estimated_impact" json:"estimated_impact"`
	InvestigationRequired bool                   `yaml:"investigation_required" json:"investigation_required"`
	SourceRef             string                 `yaml:"source_ref,omitempty" json:"source_ref,omitempty"`
	RunbookPreparation    *preparerunbook.Result `yaml:"runbook_preparation,omitempty" json:"runbook_preparation,omitempty"`
}

// Manifest is the persisted Opportunity Attack projection for one package.
type Manifest struct {
	SchemaVersion string      `yaml:"schema_version" json:"schema_version"`
	MissionID     string      `yaml:"mission_id" json:"mission_id"`
	PackageDigest string      `yaml:"package_digest" json:"package_digest"`
	EvaluatedAt   string      `yaml:"evaluated_at" json:"evaluated_at"`
	Candidates    []Candidate `yaml:"candidates" json:"candidates"`
}

// Input is the immutable input to Opportunity Attack evaluation and persistence.
type Input struct {
	RefinedPath    string
	StrategistRoot string
	MissionID      string
	PackageDigest  string
	ChestIndex     ChestIndexLoader
}

// Evaluation contains advisory Opportunity Attack detections.
type Evaluation struct {
	ADR         bool
	Runbook     bool
	Chest       bool
	ChestSource string
}
