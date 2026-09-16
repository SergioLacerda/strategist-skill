package domain

import "fmt"

// RoleContractSchemaVersion is the accepted RoleContract schema version. It
// follows the same naming convention already used by plugins/catalog.yaml's
// top-level schema_version and per-provider provider_schema_version, rather
// than inventing a new versioning scheme (see .analysis/refined/
// strategist-papeis-personagens-skills-nativas/proposal.md — Resolved
// decisions, item 2).
const RoleContractSchemaVersion = "strategist-role-contract/v1"

// RoleContract is the canonical, Provider-independent pipeline obligation for
// one native role (ranger/archivist/sniper). It formalizes RoleConfig with an
// explicit schema version so ProviderContract can declare compatibility
// against it without a second role registry (Decision 4: lifecycle reuse).
type RoleContract struct {
	SchemaVersion string   `yaml:"schema_version"`
	Role          string   `yaml:"role"`
	Slot          string   `yaml:"slot"`
	Must          []string `yaml:"must"`
	MustNot       []string `yaml:"must_not"`
	HandoffSchema string   `yaml:"handoff_schema,omitempty"`
}

// RoleContractFromConfig derives a RoleContract from an existing native role
// definition (roles/<name>.yaml), reusing RoleConfig as the source of truth
// instead of creating a parallel role registry.
func RoleContractFromConfig(cfg RoleConfig, handoffSchema string) RoleContract {
	return RoleContract{
		SchemaVersion: RoleContractSchemaVersion,
		Role:          cfg.Role,
		Slot:          cfg.Slot,
		Must:          cfg.Must,
		MustNot:       cfg.MustNot,
		HandoffSchema: handoffSchema,
	}
}

// Validate returns an error if the role contract is missing required fields
// or targets an unknown slot.
func (r RoleContract) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "schema_version", r.SchemaVersion)
	requireNonEmpty(&errs, "role", r.Role)
	if r.Slot == "" {
		errs = append(errs, "slot is required")
	} else if !IsValidSlot(r.Slot) {
		errs = append(errs, fmt.Sprintf("slot %q is not one of %s", r.Slot, requiredSlotList))
	}
	return joinPluginValidation("role contract", errs)
}

// ProviderSource is the provenance dimension for a Provider — independent
// from default selection, installation, and readiness (design.md Target
// architecture; EC-03).
type ProviderSource string

const (
	// ProviderSourceEmbedded is a Strategist-shipped provider
	// (catalog.yaml compatibility_source: embedded).
	ProviderSourceEmbedded ProviderSource = "embedded"
	// ProviderSourceExternal is an externally installed skill plugin.
	ProviderSourceExternal ProviderSource = "external"
	// ProviderSourceNativeRole is a Strategist native role acting as its
	// own provider (e.g. sniper, archivist).
	ProviderSourceNativeRole ProviderSource = "native_role"
)

var validProviderSources = stringSet(
	string(ProviderSourceEmbedded),
	string(ProviderSourceExternal),
	string(ProviderSourceNativeRole),
)

// MaterializationState is the installation/runtime-availability dimension
// for a Provider — independent from ProviderSource and from Default
// selection (design.md Target architecture; EC-03).
type MaterializationState string

const (
	// MaterializationUnavailable means the provider is cataloged but not
	// installed or invocable.
	MaterializationUnavailable MaterializationState = "unavailable"
	// MaterializationInstalled means the provider's package/legacy
	// manifest is present in the workspace.
	MaterializationInstalled MaterializationState = "installed"
	// MaterializationActive means the provider currently holds an active
	// SlotBinding.
	MaterializationActive MaterializationState = "active"
)

var validMaterializationStates = stringSet(
	string(MaterializationUnavailable),
	string(MaterializationInstalled),
	string(MaterializationActive),
)

// ProviderContract is the canonical, Role-independent capability
// declaration for one Provider (embedded or external), reusing
// plugins/catalog.yaml fields instead of introducing a parallel provider
// registry (Decision 4: lifecycle reuse).
type ProviderContract struct {
	SchemaVersion         string `yaml:"schema_version"`
	ID                    string `yaml:"id"`
	Version               string `yaml:"version"`
	ProviderSchemaVersion string `yaml:"provider_schema_version"`
	CanonicalRole         string `yaml:"canonical_role"`
	// Roles is the canonical multi-role affinity declaration. CanonicalRole is
	// retained as a compatibility alias for older single-role catalog entries.
	Roles           []string             `yaml:"roles,omitempty"`
	RiskScore       string               `yaml:"risk_score"`
	Source          ProviderSource       `yaml:"source"`
	Materialization MaterializationState `yaml:"materialization"`
	Default         bool                 `yaml:"default,omitempty"`
	// Ranked and CertificationDigest mark a build-time-certified Ranked
	// binding candidate (docs/adr/0041-...md §"Ranked Class Resolution";
	// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
	// DEC-001). Independent of Default: a candidate can be Default without
	// being Ranked, or Ranked without being Default — "preferred?" and
	// "certified?" are separate questions, never conflated.
	Ranked              bool   `yaml:"ranked,omitempty"`
	CertificationDigest string `yaml:"certification_digest,omitempty"`
	// RankedBindingGeneration and RankedBindingStatus are the pre-generated
	// runtime SlotBinding fragment's Generation/Status (ADR-0043 DEC-005),
	// computed once at certification time (`strategist plugin
	// prepare-embedded`) and copied — never recomputed — by the Wizard's
	// Ranked activation path.
	RankedBindingGeneration       int64    `yaml:"ranked_binding_generation,omitempty"`
	RankedBindingStatus           string   `yaml:"ranked_binding_status,omitempty"`
	Capabilities                  []string `yaml:"capabilities,omitempty"`
	Guarantees                    []string `yaml:"guarantees,omitempty"`
	SupportedRoleContractVersions []string `yaml:"supported_role_contract_versions"`
	// SupportedHandoffSchemas declares which handoff_schema value(s)
	// (RoleContract.HandoffSchema) this Provider's real output actually
	// conforms to. Populated through the catalog/ingestion pipeline and
	// carried on this type, but not currently read by any compatibility
	// check — CheckRoleAffinity (role_provider_compatibility.go), the
	// method actually used by the Wizard and `strategist check`, only
	// evaluates role affinity and role-contract-version, not this
	// dimension. The stricter handoff_schema-aware check that used to read
	// this field (CheckRoleCompatibility) was removed as dead code
	// (2026-09-15, zero production callers) — see
	// .analysis/pending/skills_plugaveis/04-wizard-cli-surface/20260914-wizard-weapon-options-not-listed/
	// for why the Wizard was loosened to CheckRoleAffinity instead of
	// `strategist check` being tightened to match the old stricter check.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
}

// Validate returns an error if required ProviderContract fields are missing
// or hold an unknown enum value.
func (p ProviderContract) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "schema_version", p.SchemaVersion)
	requireNonEmpty(&errs, "id", p.ID)
	requireNonEmpty(&errs, "version", p.Version)
	requireNonEmpty(&errs, "provider_schema_version", p.ProviderSchemaVersion)
	if p.CanonicalRole == "" && len(p.Roles) == 0 {
		errs = append(errs, "roles or canonical_role is required")
	}
	requireNonEmpty(&errs, "risk_score", p.RiskScore)
	if p.Source == "" {
		errs = append(errs, "source is required")
	} else if !hasString(validProviderSources, string(p.Source)) {
		errs = append(errs, fmt.Sprintf("source %q is not a known provider source", p.Source))
	}
	if p.Materialization != "" && !hasString(validMaterializationStates, string(p.Materialization)) {
		errs = append(errs, fmt.Sprintf("materialization %q is not a known materialization state", p.Materialization))
	}
	if len(p.SupportedRoleContractVersions) == 0 {
		errs = append(errs, "supported_role_contract_versions must have at least one entry")
	}
	return joinPluginValidation("provider contract", errs)
}

// CheckRoleAffinity and ProviderBinding live in
// role_provider_compatibility.go, split out to keep this file under the
// repo's file-size budget.
