package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ValidateEmbeddedInvocationBinding rejects runtime kinds that cannot be
// handled by the embedded prompt bridge.
func ValidateEmbeddedInvocationBinding(binding domain.RoleWeaponBinding) error {
	if binding.Mode != domain.SlotBindingModeRanked {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports Ranked Embedded prompt-bridge bindings")
	}
	if binding.RuntimeKind == domain.RankedRuntimeOpenSpecRoot {
		return fmt.Errorf("role_invocation_failed: Ranked openspec_root bindings execute through their declared private runtime and mission normalize-openspec, not mission invoke")
	}
	if binding.RuntimeKind != domain.RankedRuntimeEmbedded {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports runtime kind %q, got %q", domain.RankedRuntimeEmbedded, binding.RuntimeKind)
	}
	return nil
}

// ValidateInvocationCompletion validates host-controlled completion data
// against the immutable request record before any normalization or publish.
func ValidateInvocationCompletion(record domain.MissionInvocationRecord, completion domain.MissionInvocationCompletion) error {
	if err := completion.Validate(); err != nil {
		return fmt.Errorf("validate mission completion: %w", err)
	}
	if completion.RequestID != record.Request.RequestID {
		return fmt.Errorf("invocation_binding_mismatch: completion request_id does not match pending request")
	}
	if !completionRegistered(record.Request.Role, record.Request.Slot) {
		return fmt.Errorf("role_invocation_failed: completion normalization is not registered for %s/%s", record.Request.Role, record.Request.Slot)
	}
	return nil
}

// VerifyExecutionAdapter checks the completion path against the adapter
// committed when the request was dispatched.
func VerifyExecutionAdapter(record domain.MissionInvocationRecord, adapter domain.MissionExecutionAdapter) error {
	committed := record.EffectiveAdapter()
	switch {
	case adapter == "":
		return fmt.Errorf("invocation_adapter_missing: the completion path did not declare an execution adapter")
	case !adapter.Committable():
		return fmt.Errorf("invocation_adapter_unknown: %q is not an execution adapter", adapter)
	case !committed.Known():
		return fmt.Errorf("invocation_adapter_unknown: request %q carries unrecognized adapter %q", record.Request.RequestID, committed)
	case committed == domain.ExecutionAdapterCurrentHostUnverified && adapter == domain.ExecutionAdapterCurrentHost:
		return nil
	case committed != adapter:
		return fmt.Errorf("invocation_adapter_mismatch: request %q was committed as %s but completion arrived as %s", record.Request.RequestID, committed, adapter)
	case committed.IsChild() && record.ChildPolicyID == "":
		return fmt.Errorf("invocation_adapter_unknown: child request %q has no policy identity", record.Request.RequestID)
	}
	return nil
}

// completionRegistered lists the Role and slot pairs whose completion has a
// registered normalizer. The completion dispatcher in cmd/strategist selects
// its normalizer from the same pairs and fails closed for any other.
func completionRegistered(role, slot string) bool {
	switch {
	case role == "ranger" && slot == string(domain.SlotDiscovery):
		return true
	case role == "archivist" && slot == string(domain.SlotRefinement):
		return true
	case role == "sniper" && slot == string(domain.SlotExecution):
		return true
	}
	return false
}
