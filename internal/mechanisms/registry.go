// Package mechanisms loads and validates the compatibility registry of mission
// primitives Strategist gives an agent: Tools, Mechanisms, and Feats, each with
// when to use it and how to invoke it. Its purpose is awareness: a role gets a
// compact, role-scoped brief at phase start instead of having to remember what exists.
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

// RegistrySchemaVersion identifies the compatibility catalog shape.
const RegistrySchemaVersion = "1"

// Families a registry row may belong to. The registry remains named
// mechanisms.yaml for compatibility, but its identity vocabulary is the
// canonical seven-family taxonomy. RoleLoadoutCapabilities deliberately
// projects only Feats and Tools into a Role's executable loadout.
const (
	FamilyMechanism = "mechanism"
	FamilyFeat      = "feat"
	FamilyTool      = "tool"
	FamilyRole      = "role"
	FamilyWeapon    = "weapon"
	FamilyStage     = "stage"
	FamilyArtifact  = "artifact"
)

// AllRoles marks a row every role should see.
const AllRoles = "all"

// OrchestratorRole marks a row owned by the orchestrator rather than a Role; it
// appears in no role brief.
const OrchestratorRole = "orchestrator"

var (
	validFamilies = map[string]bool{
		FamilyMechanism: true, FamilyFeat: true, FamilyTool: true,
		FamilyRole: true, FamilyWeapon: true, FamilyStage: true, FamilyArtifact: true,
	}
	validKinds = map[string]bool{"code": true, "contract": true, "prose": true}
	validTiers = map[string]bool{"": true, "machine_enforced": true, "machine_observed": true, "agent_only": true}
)

// Row is one registry entry.
type Row struct {
	ID              string   `yaml:"id"`
	Version         string   `yaml:"version,omitempty"`
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

// RoleLoadoutCapabilities projects the registry's permitted Feats and Tools
// for a Role. Mechanisms stay in their enforcement boundary and are not
// smuggled into the loadout as capabilities.
func (r Registry) RoleLoadoutCapabilities(role, slot string) (feats, tools []domain.LoadoutCapability) {
	for _, row := range r.ForRole(role) {
		if slot != "" && !rowAppliesToSlot(row, slot) {
			continue
		}
		capability := domain.LoadoutCapability{ID: row.ID, Availability: domain.CapabilityAvailable}
		switch row.Family {
		case FamilyFeat:
			capability.Family = domain.TaxonomyFeat
			feats = append(feats, capability)
		case FamilyTool:
			capability.Family = domain.TaxonomyTool
			tools = append(tools, capability)
		}
	}
	return feats, tools
}

// BuildRoleLoadout composes the validated registry projection with the pinned
// Weapon plan. The registry contributes only Feats and Tools; Stage and
// binding validation remain domain-owned.
func (r Registry) BuildRoleLoadout(resolution domain.StageResolution, weapon domain.RoleInvocationPlan) (domain.RoleLoadout, error) {
	feats, tools := r.RoleLoadoutCapabilities(weapon.Role, weapon.Slot)
	loadout, err := domain.NewRoleLoadout(resolution, weapon, feats, tools)
	if err != nil {
		return domain.RoleLoadout{}, fmt.Errorf("build role loadout: %w", err)
	}
	return loadout, nil
}

func rowAppliesToSlot(row Row, slot string) bool {
	if len(row.PhaseScope) == 0 {
		return true
	}
	for _, phase := range row.PhaseScope {
		if phase == "all" || phase == slot {
			return true
		}
	}
	return false
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
	if err := reg.Validate(); err != nil {
		return Registry{}, err
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
