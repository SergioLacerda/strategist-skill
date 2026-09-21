package domain

import (
	"encoding/json"
	"fmt"
)

// RankedRuntimeStatePath is the Strategist-root-relative file recording the
// verified runtime of every Ranked binding.
const RankedRuntimeStatePath = "ranked-runtimes.yaml"

// RankedRuntimeStateSchemaVersion identifies the host-Node runtime record: an
// absolute host Node plus a digest-verified OpenSpec bundle under
// weapon-runtime/. Earlier versions (private-Node payloads) are rejected, never
// reinterpreted.
const RankedRuntimeStateSchemaVersion = "strategist-ranked-runtime/v2"

// RankedRuntimeComponentOpenSpec names the embedded OpenSpec bundle component;
// its SHA256 is the bundle tree digest.
const RankedRuntimeComponentOpenSpec = "openspec"

// Ranked runtime readiness reason codes, cataloged in machine/errors.yaml.
const (
	ReasonRankedRuntimeStateLegacy   = "ranked_runtime_state_legacy"
	ReasonRankedRuntimeStateInvalid  = "ranked_runtime_state_invalid"
	ReasonRankedRuntimeBundleMissing = "ranked_runtime_bundle_missing"
	ReasonRankedRuntimeBundleAltered = "ranked_runtime_bundle_tampered"
)

// RankedRuntimeState is the ranked-runtimes.yaml document, written by the
// installer and read by check and role invocation.
type RankedRuntimeState struct {
	SchemaVersion string                    `yaml:"schema_version" json:"schema_version"`
	Entries       []RankedRuntimeStateEntry `yaml:"entries" json:"entries"`
}

// RankedRuntimeStateEntry records one Ranked binding's runtime.
type RankedRuntimeStateEntry struct {
	Role           string `yaml:"role" json:"role"`
	Slot           string `yaml:"slot" json:"slot"`
	Provider       string `yaml:"provider" json:"provider"`
	ContractDigest string `yaml:"contract_digest" json:"contract_digest"`
	Root           string `yaml:"root,omitempty" json:"root,omitempty"`
	Kind           string `yaml:"kind" json:"kind"`

	Runtime *RankedRuntimeStateRuntime `yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

// RankedRuntimeStateRuntime is one absolute host Node executable plus the
// private OpenSpec launcher, a slash path relative to the Strategist root.
type RankedRuntimeStateRuntime struct {
	Node       string                        `yaml:"node" json:"node"`
	Script     string                        `yaml:"script" json:"script"`
	Components []RankedRuntimeStateComponent `yaml:"components" json:"components"`
}

// RankedRuntimeStateComponent is one verified runtime component.
type RankedRuntimeStateComponent struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	SHA256  string `yaml:"sha256" json:"sha256"`
}

// RankedRuntimeStateLegacyError reports a record written under an earlier
// schema; the remedy is `strategist upgrade` or a Ranked reinstall.
type RankedRuntimeStateLegacyError struct {
	SchemaVersion string
}

func (e *RankedRuntimeStateLegacyError) Error() string {
	return fmt.Sprintf("ranked runtime state schema %q is not %q", e.SchemaVersion, RankedRuntimeStateSchemaVersion)
}

// ParseRankedRuntimeState decodes ranked-runtimes.yaml (written as JSON) and
// rejects any schema other than RankedRuntimeStateSchemaVersion.
func ParseRankedRuntimeState(raw []byte) (RankedRuntimeState, error) {
	var state RankedRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		return RankedRuntimeState{}, fmt.Errorf("decode ranked runtime state: %w", err)
	}
	if state.SchemaVersion != RankedRuntimeStateSchemaVersion {
		return RankedRuntimeState{}, &RankedRuntimeStateLegacyError{SchemaVersion: state.SchemaVersion}
	}
	return state, nil
}

// Entry returns the entry recorded for slot/provider.
func (s RankedRuntimeState) Entry(slot, provider string) (RankedRuntimeStateEntry, bool) {
	for _, entry := range s.Entries {
		if entry.Slot == slot && entry.Provider == provider {
			return entry, true
		}
	}
	return RankedRuntimeStateEntry{}, false
}

// Component returns the named component, if recorded.
func (r *RankedRuntimeStateRuntime) Component(name string) (RankedRuntimeStateComponent, bool) {
	if r == nil {
		return RankedRuntimeStateComponent{}, false
	}
	for _, component := range r.Components {
		if component.Name == name {
			return component, true
		}
	}
	return RankedRuntimeStateComponent{}, false
}
