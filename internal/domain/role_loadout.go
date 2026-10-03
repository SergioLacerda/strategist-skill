package domain

import (
	"fmt"
	"sort"
	"strings"
)

// RoleLoadoutSchemaVersion identifies the serialized Role loadout contract.
const RoleLoadoutSchemaVersion = "strategist-role-loadout/v1"

// CapabilityAvailability is deliberately closed: a loadout may contain only
// capabilities that are proven available for the selected Role and Stage.
type CapabilityAvailability string

const (
	// CapabilityAvailable indicates that a capability can be invoked.
	CapabilityAvailable CapabilityAvailability = "available"
	// CapabilityUnavailable indicates that a capability is not currently available.
	CapabilityUnavailable CapabilityAvailability = "unavailable"
	// CapabilityIncompatible indicates that a capability does not fit the loadout.
	CapabilityIncompatible CapabilityAvailability = "incompatible"
	// CapabilityStaleDigest indicates that a capability digest is stale.
	CapabilityStaleDigest CapabilityAvailability = "stale_digest"
	// CapabilityUnsupportedRuntime indicates that the runtime cannot invoke a capability.
	CapabilityUnsupportedRuntime CapabilityAvailability = "unsupported_runtime"
)

// LoadoutCapability is a Feat or Tool admitted to one role loadout.
type LoadoutCapability struct {
	Family       TaxonomyFamily         `json:"family" yaml:"family"`
	ID           string                 `json:"id" yaml:"id"`
	Availability CapabilityAvailability `json:"availability" yaml:"availability"`
	Reason       string                 `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// RoleLoadout is the Stage-aware composition consumed by a role invocation.
// Binding identity remains sourced from RoleInvocationPlan/SlotBinding.
type RoleLoadout struct {
	SchemaVersion   string              `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string              `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Resolution      StageResolution     `json:"stage_resolution" yaml:"stage_resolution"`
	Role            string              `json:"role" yaml:"role"`
	Slot            string              `json:"slot" yaml:"slot"`
	Weapon          RoleInvocationPlan  `json:"weapon" yaml:"weapon"`
	Feats           []LoadoutCapability `json:"feats,omitempty" yaml:"feats,omitempty"`
	Tools           []LoadoutCapability `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// NewRoleLoadout admits only a pinned, invocable Weapon and explicitly
// available Feats/Tools. It fails closed on stale or unsupported inputs.
func NewRoleLoadout(resolution StageResolution, weapon RoleInvocationPlan, feats, tools []LoadoutCapability) (RoleLoadout, error) {
	loadout := RoleLoadout{
		SchemaVersion:   RoleLoadoutSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		Resolution:      resolution,
		Role:            weapon.Role,
		Slot:            weapon.Slot,
		Weapon:          weapon,
		Feats:           append([]LoadoutCapability(nil), feats...),
		Tools:           append([]LoadoutCapability(nil), tools...),
	}
	sort.Slice(loadout.Feats, func(i, j int) bool { return loadout.Feats[i].ID < loadout.Feats[j].ID })
	sort.Slice(loadout.Tools, func(i, j int) bool { return loadout.Tools[i].ID < loadout.Tools[j].ID })
	if err := loadout.Validate(); err != nil {
		return RoleLoadout{}, err
	}
	return loadout, nil
}

// Validate checks the schema, stage, identity, runtime, and capabilities.
func (l RoleLoadout) Validate() error {
	if l.SchemaVersion != RoleLoadoutSchemaVersion {
		return fmt.Errorf("role loadout: unsupported schema_version %q", l.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(l.TaxonomyVersion); err != nil {
		return fmt.Errorf("role loadout: %w", err)
	}
	if err := validateLoadoutResolution(l.Resolution); err != nil {
		return err
	}
	if err := l.validateIdentity(); err != nil {
		return err
	}
	if err := validateLoadoutRuntime(l.Weapon.Runtime); err != nil {
		return err
	}
	if err := validateCapabilities(TaxonomyFeat, l.Feats); err != nil {
		return fmt.Errorf("role loadout: feats: %w", err)
	}
	if err := validateCapabilities(TaxonomyTool, l.Tools); err != nil {
		return fmt.Errorf("role loadout: tools: %w", err)
	}
	return nil
}

func validateLoadoutResolution(resolution StageResolution) error {
	if err := resolution.Validate(); err != nil {
		return fmt.Errorf("role loadout: %w", err)
	}
	if resolution.Stage != StageFull && resolution.Stage != StageShort {
		return fmt.Errorf("role loadout: stage %s cannot invoke a Role", resolution.Stage)
	}
	return nil
}

func (l RoleLoadout) validateIdentity() error {
	if strings.TrimSpace(l.Role) == "" || strings.TrimSpace(l.Slot) == "" || strings.TrimSpace(l.Weapon.WeaponID) == "" {
		return fmt.Errorf("role loadout: role, slot, and Weapon identity are required")
	}
	if l.Resolution.Role != "" && l.Resolution.Role != l.Role {
		return fmt.Errorf("role loadout: resolution Role %q does not match %q", l.Resolution.Role, l.Role)
	}
	if strings.TrimSpace(l.Weapon.WeaponVersion) == "" || strings.TrimSpace(l.Weapon.WeaponDigest) == "" || strings.TrimSpace(l.Weapon.BindingDigest) == "" {
		return fmt.Errorf("role loadout: Weapon version and digests are required")
	}
	return nil
}

func validateLoadoutRuntime(runtime WeaponRuntime) error {
	switch NormalizeRankedRuntime(runtime).Kind {
	case RankedRuntimeOpenSpecRoot, RankedRuntimeHost, RankedRuntimeEmbedded, RankedRuntimeExecutable:
		return nil
	case RankedRuntimeNone:
		return fmt.Errorf("role loadout: Weapon runtime is not invocable")
	default:
		return fmt.Errorf("role loadout: Weapon runtime %q is unsupported", runtime.Kind)
	}
}

func validateCapabilities(family TaxonomyFamily, capabilities []LoadoutCapability) error {
	seen := map[string]struct{}{}
	for _, capability := range capabilities {
		if err := validateCapability(family, capability); err != nil {
			return err
		}
		if _, exists := seen[capability.ID]; exists {
			return fmt.Errorf("duplicate %s %q", family, capability.ID)
		}
		seen[capability.ID] = struct{}{}
	}
	return nil
}

func validateCapability(family TaxonomyFamily, capability LoadoutCapability) error {
	if capability.Family != family {
		return fmt.Errorf("%s %q has family %q", family, capability.ID, capability.Family)
	}
	if strings.TrimSpace(capability.ID) == "" {
		return fmt.Errorf("%s identity is required", family)
	}
	if capability.Availability != CapabilityAvailable {
		return fmt.Errorf("%s %q is %s", family, capability.ID, capability.Availability)
	}
	return nil
}
