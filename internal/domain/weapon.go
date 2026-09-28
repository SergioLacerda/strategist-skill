package domain

import (
	"fmt"
	"strings"
)

// WeaponKind identifies whether a skill package executes as one unit or
// composes other Weapons.
type WeaponKind string

// Weapon kinds.
const (
	WeaponKindAtomic    WeaponKind = "atomic"
	WeaponKindComposite WeaponKind = "composite"
)

// WeaponRuntimeKind names a runtime contract kind.
type WeaponRuntimeKind = string

// Weapon runtime kinds.
const (
	WeaponRuntimeNone       = RankedRuntimeNone
	WeaponRuntimeHost       = RankedRuntimeHost
	WeaponRuntimeEmbedded   = RankedRuntimeEmbedded
	WeaponRuntimeExecutable = RankedRuntimeExecutable
	WeaponRuntimeOpenSpec   = RankedRuntimeOpenSpecRoot
)

// WeaponManifest is the canonical identity and compatibility contract for one
// atomic or composite Weapon.
type WeaponManifest struct {
	ID             string             `yaml:"id"`
	Version        string             `yaml:"version"`
	Kind           WeaponKind         `yaml:"kind"`
	Origin         WeaponOrigin       `yaml:"origin"`
	Roles          []string           `yaml:"roles"`
	SupportedSlots []string           `yaml:"supported_slots"`
	RiskScore      string             `yaml:"risk_score"`
	Runtime        WeaponRuntime      `yaml:"runtime"`
	Contract       WeaponContract     `yaml:"weapon_contract,omitempty"`
	Composition    *WeaponComposition `yaml:"composition,omitempty"`
}

// Validate checks the manifest's identity, Role/slot compatibility, runtime,
// contract, and composition shape, reporting every problem found.
func (w WeaponManifest) Validate() error {
	var errs []string
	errs = append(errs, w.validateIdentity()...)
	errs = append(errs, w.validateOrigin()...)
	errs = append(errs, w.validateRoles()...)
	errs = append(errs, w.validateSlots()...)
	if err := w.Runtime.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := w.Contract.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	errs = append(errs, w.validateComposition()...)
	if len(errs) > 0 {
		return fmt.Errorf("weapon manifest invalid: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (w WeaponManifest) validateIdentity() []string {
	var errs []string
	if strings.TrimSpace(w.ID) == "" {
		errs = append(errs, "id is required")
	}
	if strings.TrimSpace(w.Version) == "" {
		errs = append(errs, "version is required")
	}
	switch w.Kind {
	case WeaponKindAtomic, WeaponKindComposite:
	default:
		errs = append(errs, fmt.Sprintf("kind %q must be atomic or composite", w.Kind))
	}
	return errs
}

func (w WeaponManifest) validateRoles() []string {
	var errs []string
	if len(w.Roles) == 0 {
		errs = append(errs, "roles must not be empty")
	}
	return append(errs, validateUniqueEach(w.Roles, "role", ValidateRoleReference)...)
}

func (w WeaponManifest) validateSlots() []string {
	var errs []string
	if len(w.SupportedSlots) == 0 {
		errs = append(errs, "supported_slots must not be empty")
	}
	return append(errs, validateUniqueEach(w.SupportedSlots, "slot", func(slot string) error {
		if !IsValidSlot(slot) {
			return fmt.Errorf("slot %q is not one of %s", slot, requiredSlotList)
		}
		return nil
	})...)
}

func (w WeaponManifest) validateComposition() []string {
	if w.Kind != WeaponKindComposite {
		if w.Composition != nil {
			return []string{"atomic Weapon must not declare composition"}
		}
		return nil
	}
	if w.Composition == nil {
		return []string{"composite Weapon requires composition"}
	}
	if err := w.Composition.Validate(); err != nil {
		return []string{err.Error()}
	}
	return nil
}

// validateUniqueEach reports duplicate values and any error from check for
// each first occurrence.
func validateUniqueEach(values []string, label string, check func(string) error) []string {
	var errs []string
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, dup := seen[value]; dup {
			errs = append(errs, fmt.Sprintf("duplicate %s %q", label, value))
			continue
		}
		seen[value] = struct{}{}
		if err := check(value); err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

// ValidateActive additionally requires the runtime contract to be invocable.
func (w WeaponManifest) ValidateActive() error {
	if err := w.Validate(); err != nil {
		return err
	}
	if err := w.Runtime.ValidateActive(); err != nil {
		return fmt.Errorf("weapon %q: %w", w.ID, err)
	}
	return nil
}
