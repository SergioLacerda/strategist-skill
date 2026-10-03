package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// InstallPlanSchemaVersion identifies the read-only plan envelope shared by
// Wizard and headless installation.
const InstallPlanSchemaVersion = "strategist-install-plan/v1"

// InstallPlan is the deterministic, inspectable result of installation
// planning. Applying it is a separate operation owned by the installer.
type InstallPlan struct {
	SchemaVersion   string            `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string            `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           Stage             `json:"stage" yaml:"stage"`
	Mode            string            `json:"mode,omitempty" yaml:"mode,omitempty"`
	BasePath        string            `json:"base_path,omitempty" yaml:"base_path,omitempty"`
	Slots           map[string]string `json:"slots" yaml:"slots"`
	SlotModes       map[string]string `json:"slot_modes,omitempty" yaml:"slot_modes,omitempty"`
	Lock            PluginLock        `json:"lock" yaml:"lock"`
	Bindings        []SlotBinding     `json:"bindings" yaml:"bindings"`
	PlanDigest      string            `json:"plan_digest" yaml:"plan_digest"`
}

// NewInstallPlan builds a plan and seals it with a digest. Maps and slices are
// copied so callers cannot change the plan after it has been created.
func NewInstallPlan(stage Stage, mode, basePath string, slots, slotModes map[string]string, lock PluginLock, bindings []SlotBinding) (InstallPlan, error) {
	plan := InstallPlan{
		SchemaVersion:   InstallPlanSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		Stage:           stage,
		Mode:            strings.TrimSpace(mode),
		BasePath:        strings.TrimSpace(basePath),
		Slots:           cloneStringMap(slots),
		SlotModes:       cloneStringMap(slotModes),
		Lock:            clonePluginLock(lock),
		Bindings:        cloneSlotBindings(bindings),
	}
	if err := plan.validateShape(); err != nil {
		return InstallPlan{}, err
	}
	digest, err := plan.computeDigest()
	if err != nil {
		return InstallPlan{}, fmt.Errorf("install plan: compute digest: %w", err)
	}
	plan.PlanDigest = digest
	return plan, nil
}

// Validate checks the envelope and verifies that PlanDigest covers the plan.
func (p InstallPlan) Validate() error {
	if err := p.validateShape(); err != nil {
		return err
	}
	if strings.TrimSpace(p.PlanDigest) == "" {
		return fmt.Errorf("install plan: plan_digest is required")
	}
	digest, err := p.computeDigest()
	if err != nil {
		return fmt.Errorf("install plan: compute digest: %w", err)
	}
	if digest != p.PlanDigest {
		return fmt.Errorf("install plan: plan_digest mismatch")
	}
	return nil
}

func (p InstallPlan) validateShape() error {
	if p.SchemaVersion != InstallPlanSchemaVersion {
		return fmt.Errorf("install plan: unsupported schema_version %q", p.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(p.TaxonomyVersion); err != nil {
		return fmt.Errorf("install plan: %w", err)
	}
	if err := p.Stage.Validate(); err != nil {
		return fmt.Errorf("install plan: %w", err)
	}
	if p.Stage != StageRoster {
		return fmt.Errorf("install plan: stage must be ROSTER")
	}
	if len(p.Slots) == 0 {
		return fmt.Errorf("install plan: slots are required")
	}
	if len(p.Bindings) == 0 {
		return fmt.Errorf("install plan: bindings are required")
	}
	if err := validatePlanSelections(p.Slots, p.SlotModes, p.Bindings); err != nil {
		return err
	}
	return nil
}

func validatePlanSelections(slots, slotModes map[string]string, bindings []SlotBinding) error {
	if err := validatePlanSlots(slots); err != nil {
		return err
	}
	if err := validatePlanModes(slots, slotModes); err != nil {
		return err
	}
	return validatePlanBindings(slots, bindings)
}

func validatePlanSlots(slots map[string]string) error {
	for slot, provider := range slots {
		if strings.TrimSpace(slot) == "" || strings.TrimSpace(provider) == "" {
			return fmt.Errorf("install plan: slot and provider are required")
		}
	}
	return nil
}

func validatePlanModes(slots, slotModes map[string]string) error {
	for slot, mode := range slotModes {
		if _, ok := slots[slot]; !ok {
			return fmt.Errorf("install plan: slot mode %q has no matching slot", slot)
		}
		if mode != "" && mode != SlotBindingModeCustom && mode != SlotBindingModeRanked {
			return fmt.Errorf("install plan: slot %q has unsupported mode %q", slot, mode)
		}
	}
	return nil
}

func validatePlanBindings(slots map[string]string, bindings []SlotBinding) error {
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if _, ok := slots[binding.Slot]; !ok {
			return fmt.Errorf("install plan: binding slot %q has no matching slot", binding.Slot)
		}
		if strings.TrimSpace(binding.InstalledInstanceID) == "" {
			return fmt.Errorf("install plan: binding for slot %q has no installed Weapon", binding.Slot)
		}
		if _, duplicate := seen[binding.Slot]; duplicate {
			return fmt.Errorf("install plan: duplicate binding for slot %q", binding.Slot)
		}
		seen[binding.Slot] = struct{}{}
	}
	return nil
}

func (p InstallPlan) computeDigest() (string, error) {
	unsigned := p
	unsigned.PlanDigest = ""
	raw, err := json.Marshal(unsigned)
	if err != nil {
		return "", fmt.Errorf("install plan: encode: %w", err)
	}
	digest := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func clonePluginLock(lock PluginLock) PluginLock {
	clone := lock
	clone.Nodes = append([]PluginLockNode(nil), lock.Nodes...)
	return clone
}

func cloneSlotBindings(bindings []SlotBinding) []SlotBinding {
	return append([]SlotBinding(nil), bindings...)
}
