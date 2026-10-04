package domain

import (
	"fmt"
	"sort"
	"strings"
)

// WeaponBindingArtifactSchemaVersion identifies the serialized binding contract.
const WeaponBindingArtifactSchemaVersion = "strategist-weapon-binding/v1"

// WeaponBindingArtifact is the canonical projection of a fully pinned binding.
// SlotBinding remains the persisted authority; this artifact is evidence.
type WeaponBindingArtifact struct {
	SchemaVersion    string `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion  string `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage            Stage  `json:"stage" yaml:"stage"`
	Role             string `json:"role" yaml:"role"`
	Slot             string `json:"slot" yaml:"slot"`
	WeaponID         string `json:"weapon_id" yaml:"weapon_id"`
	WeaponVersion    string `json:"weapon_version" yaml:"weapon_version"`
	WeaponDigest     string `json:"weapon_digest" yaml:"weapon_digest"`
	SourceDigest     string `json:"source_digest,omitempty" yaml:"source_digest,omitempty"`
	BindingDigest    string `json:"binding_digest" yaml:"binding_digest"`
	RuntimeKind      string `json:"runtime_kind" yaml:"runtime_kind"`
	ContractIdentity string `json:"contract_identity,omitempty" yaml:"contract_identity,omitempty"`
	Origin           string `json:"origin" yaml:"origin"`
}

// NewWeaponBindingArtifact projects a complete persisted binding and refuses
// sparse identity, so evidence cannot claim a pinned Weapon prematurely.
func NewWeaponBindingArtifact(binding SlotBinding) (WeaponBindingArtifact, error) {
	if err := ValidateTaxonomyVersion(binding.TaxonomyVersion); err != nil {
		return WeaponBindingArtifact{}, fmt.Errorf("weapon binding: %w", err)
	}
	artifact := WeaponBindingArtifact{
		SchemaVersion: WeaponBindingArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion, Stage: StageRoster,
		Role: binding.Role, Slot: binding.Slot, WeaponID: binding.InstalledInstanceID,
		WeaponVersion: binding.WeaponVersion, WeaponDigest: binding.WeaponDigest,
		SourceDigest: binding.SourceDigest, BindingDigest: binding.BindingDigest,
		RuntimeKind: binding.RuntimeKind, ContractIdentity: binding.ConnectorID + ":" + binding.Entrypoint,
		Origin: binding.Origin,
	}
	if err := artifact.Validate(); err != nil {
		return WeaponBindingArtifact{}, err
	}
	return artifact, nil
}

// Validate checks the evidence projection without consulting or mutating the
// persisted binding. SlotBinding remains the authority for runtime decisions.
func (a WeaponBindingArtifact) Validate() error {
	if a.SchemaVersion != WeaponBindingArtifactSchemaVersion {
		return fmt.Errorf("weapon binding: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("weapon binding: %w", err)
	}
	if a.Stage != StageRoster {
		return fmt.Errorf("weapon binding: stage must be ROSTER")
	}
	values := []string{a.Role, a.Slot, a.WeaponID, a.WeaponVersion, a.WeaponDigest, a.BindingDigest, a.RuntimeKind, a.ContractIdentity, a.Origin}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("weapon binding: role, slot, version, weapon_digest, binding_digest, runtime_kind, contract_identity, and origin are required")
		}
	}
	return nil
}

// NewWeaponBindingArtifacts creates a deterministic projection for every
// binding with complete identity. Legacy sparse bindings are omitted until a
// later activation enriches them.
func NewWeaponBindingArtifacts(bindings []SlotBinding) ([]WeaponBindingArtifact, error) {
	if len(bindings) == 0 {
		return nil, fmt.Errorf("weapon bindings: bindings are required")
	}
	artifacts := make([]WeaponBindingArtifact, 0, len(bindings))
	seenSlots := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		var err error
		artifacts, err = appendWeaponBindingArtifact(artifacts, seenSlots, binding)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		left := strings.Join([]string{artifacts[i].Slot, artifacts[i].Role, artifacts[i].WeaponID}, "\x00")
		right := strings.Join([]string{artifacts[j].Slot, artifacts[j].Role, artifacts[j].WeaponID}, "\x00")
		return left < right
	})
	return artifacts, nil
}

func appendWeaponBindingArtifact(artifacts []WeaponBindingArtifact, seenSlots map[string]struct{}, binding SlotBinding) ([]WeaponBindingArtifact, error) {
	if !bindingHasArtifactIdentity(binding) {
		return artifacts, nil
	}
	if _, exists := seenSlots[binding.Slot]; exists {
		return nil, fmt.Errorf("weapon bindings: duplicate slot %q", binding.Slot)
	}
	artifact, err := NewWeaponBindingArtifact(binding)
	if err != nil {
		return nil, err
	}
	seenSlots[binding.Slot] = struct{}{}
	return append(artifacts, artifact), nil
}

// ValidateWeaponBindingArtifacts proves persisted evidence is exactly the
// projection of the current bindings, never an alternate runtime authority.
func ValidateWeaponBindingArtifacts(bindings []SlotBinding, artifacts []WeaponBindingArtifact) error {
	if len(artifacts) == 0 {
		return nil // legacy locks predate this optional projection
	}
	if len(bindings) == 0 {
		return fmt.Errorf("weapon bindings: artifacts exist without bindings")
	}
	want, err := NewWeaponBindingArtifacts(bindings)
	if err != nil {
		return err
	}
	if len(want) != len(artifacts) {
		return fmt.Errorf("weapon bindings: artifact count does not match bindings")
	}
	return compareWeaponBindingArtifacts(want, artifacts)
}

func compareWeaponBindingArtifacts(want, artifacts []WeaponBindingArtifact) error {
	for i, artifact := range artifacts {
		if err := artifact.Validate(); err != nil {
			return err
		}
		if artifact != want[i] {
			return fmt.Errorf("weapon bindings: artifact %d does not match its binding", i)
		}
	}
	return nil
}

func bindingHasArtifactIdentity(binding SlotBinding) bool {
	values := []string{binding.Role, binding.Slot, binding.InstalledInstanceID, binding.WeaponVersion, binding.WeaponDigest, binding.BindingDigest, binding.RuntimeKind, binding.ConnectorID, binding.Entrypoint, binding.Origin}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}
