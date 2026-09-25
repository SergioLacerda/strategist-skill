// Package mechanisms loads and validates the Mechanisms registry: the single
// machine-readable catalog of the tools Strategist gives an agent in a mission
// (Mechanisms, and the Abilities they sit beside), each with when to use it and
// how to invoke it. Its purpose is awareness: a role gets a compact, role-scoped
// brief at phase start instead of having to remember what exists.
package mechanisms

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// RegistryRelPath is the registry contract, relative to the .strategist root.
const RegistryRelPath = "contracts/machine/mechanisms.yaml"

// Families a registry row may belong to.
const (
	FamilyMechanism = "mechanism"
	FamilyAbility   = "ability"
)

// AllRoles marks a row every role should see.
const AllRoles = "all"

// OrchestratorRole marks a row owned by the orchestrator rather than a Role; it
// appears in no role brief.
const OrchestratorRole = "orchestrator"

var (
	validFamilies = map[string]bool{FamilyMechanism: true, FamilyAbility: true}
	validKinds    = map[string]bool{"code": true, "contract": true, "prose": true}
	validTiers    = map[string]bool{"": true, "machine_enforced": true, "machine_observed": true, "agent_only": true}
)

// Row is one registry entry.
type Row struct {
	ID              string   `yaml:"id"`
	DisplayName     string   `yaml:"display_name"`
	Family          string   `yaml:"family"`
	OwnerPackage    string   `yaml:"owner_package"`
	ContractFile    string   `yaml:"contract_file"`
	EnforcementKind string   `yaml:"enforcement_kind"`
	EnforcementTier string   `yaml:"enforcement_tier"`
	Summary         string   `yaml:"summary"`
	WhenToUse       string   `yaml:"when_to_use"`
	InvokedBy       []string `yaml:"invoked_by"`
	HowToInvoke     string   `yaml:"how_to_invoke"`
	PhaseScope      []string `yaml:"phase_scope"`
	// JudgmentFacet names the agent-judgment side of a hybrid item; the row's
	// Family is then its deterministic side.
	JudgmentFacet string `yaml:"judgment_facet"`
}

// Registry is the parsed catalog.
type Registry struct {
	SchemaVersion string `yaml:"schema_version"`
	Rows          []Row  `yaml:"mechanisms"`
}

// Load reads and validates the registry installed under a .strategist root.
func Load(strategistRoot string) (Registry, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, filepath.FromSlash(RegistryRelPath))) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return Registry{}, fmt.Errorf("mechanisms registry: %w", err)
	}
	return Parse(raw)
}

// Parse decodes and validates a registry document.
func Parse(raw []byte) (Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(raw, &reg); err != nil {
		return Registry{}, fmt.Errorf("mechanisms registry: decode: %w", err)
	}
	if len(reg.Rows) == 0 {
		return Registry{}, errors.New("mechanisms registry: no rows")
	}
	seen := map[string]bool{}
	for _, row := range reg.Rows {
		if err := validateRow(row); err != nil {
			return Registry{}, fmt.Errorf("mechanisms registry: row %q: %w", row.ID, err)
		}
		if seen[row.ID] {
			return Registry{}, fmt.Errorf("mechanisms registry: duplicate id %q", row.ID)
		}
		seen[row.ID] = true
	}
	return reg, nil
}

func validateRow(row Row) error {
	switch {
	case row.ID == "":
		return errors.New("id is required")
	case !validFamilies[row.Family]:
		return fmt.Errorf("unknown family %q", row.Family)
	case !validKinds[row.EnforcementKind]:
		return fmt.Errorf("unknown enforcement_kind %q", row.EnforcementKind)
	case !validTiers[row.EnforcementTier]:
		return fmt.Errorf("unknown enforcement_tier %q", row.EnforcementTier)
	case row.Summary == "":
		return errors.New("summary is required")
	case row.HowToInvoke == "":
		return errors.New("how_to_invoke is required")
	}
	return validateInvokedBy(row.InvokedBy)
}

func validateInvokedBy(invokedBy []string) error {
	if len(invokedBy) == 0 {
		return errors.New("invoked_by is required")
	}
	roles := domain.DefaultRoleRegistry()
	for _, who := range invokedBy {
		if who != AllRoles && who != OrchestratorRole && !roles.Has(who) {
			return fmt.Errorf("invoked_by %q is not a role", who)
		}
	}
	return nil
}
