package domain

import "fmt"

// RoleInvocationPlan is the mission-scoped Role→Weapon composition consumed
// at mission invocation time: a pinned weapon binding plus the context and
// output-schema references a mission needs to invoke it. It is a distinct
// concept from connectors.InvocationEnvelope
// (internal/plugins/connectors/runtime_connector.go), the host/plugin-runtime
// dispatch payload for RuntimeConnector.Invoke — the two were never meant to
// be the same type; see
// docs/adr/0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md
// D1 for the naming rationale that keeps them separate.
type RoleInvocationPlan struct {
	Role string
	Slot string

	// Mode is the source binding's EffectiveMode() (SlotBindingModeCustom or
	// SlotBindingModeRanked). NewRoleInvocationPlanFromLock only resolves
	// Custom bindings — see its own doc comment.
	Mode string

	// For a Custom binding, WeaponID, WeaponDigest, BindingDigest,
	// BindingGeneration, and BindingStatus are sourced from today's flat
	// .strategist/plugins.lock (lock.nodes[]/bindings[]) via
	// NewRoleInvocationPlanFromLock — see D7
	// (docs/architecture/strategist-concepts.md §"Ranked Class" § Pipeline
	// scope). For a Ranked binding, the same fields are sourced from the
	// catalog's certification stamp instead via
	// NewRankedRoleInvocationPlanFromCatalog — a deliberately separate
	// resolution path (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md),
	// never a re-point of Custom's digest lookup.
	WeaponID          string
	WeaponDigest      string
	BindingDigest     string
	BindingGeneration int64
	BindingStatus     string
	Runtime           RankedRuntimeContract

	// RequiredContextRefs and OutputSchemaRef are populated by whatever
	// composes a mission invocation (ContextComposer, doc 05 — not
	// implemented here). This type only defines their shape.
	RequiredContextRefs []string
	OutputSchemaRef     string
}

// NewRoleInvocationPlanFromLock builds a RoleInvocationPlan for role/slot by
// resolving the single active SlotBinding for slot against lock.Bindings and
// the matching PluginLockNode digests in lock.Lock.Nodes. Every weapon-binding
// field is sourced by reference to the lock record, not recomputed — see D2's
// sibling rationale (PreflightResult) for why this project avoids a second,
// parallel implementation of logic another package already owns.
//
// This function only resolves Custom-pipeline bindings (see
// docs/architecture/strategist-concepts.md §"Ranked Class" § Pipeline
// scope). A binding with an invalid Mode is rejected; a Ranked-mode binding
// is explicitly rejected too, fail-closed — the digest lookup here (an
// adapter_contract/role_provider_binding node pair) is meaningless for a
// build-time-certified Ranked binding, and Ranked resolution is not
// implemented, so this must error rather than silently produce a wrong or
// empty plan.
func NewRoleInvocationPlanFromLock(role, slot string, lock PluginLockFile) (RoleInvocationPlan, error) {
	binding, err := SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return RoleInvocationPlan{}, err
	}
	if !binding.ValidMode() {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: slot %q has invalid binding mode %q", slot, binding.Mode)
	}
	if mode := binding.EffectiveMode(); mode != SlotBindingModeCustom {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: slot %q has mode %q — use NewRankedRoleInvocationPlanFromCatalog for a Ranked binding", slot, mode)
	}
	return RoleInvocationPlan{
		Role:              role,
		Slot:              slot,
		Mode:              binding.EffectiveMode(),
		WeaponID:          binding.InstalledInstanceID,
		WeaponDigest:      lock.NodeDigest(binding.InstalledInstanceID, "adapter_contract"),
		BindingDigest:     lock.NodeDigest(role+":"+binding.InstalledInstanceID, "role_provider_binding"),
		BindingGeneration: binding.Generation,
		BindingStatus:     binding.Status,
	}, nil
}

// NewRankedRoleInvocationPlanFromCatalog builds a RoleInvocationPlan for a
// mode: ranked binding, resolving weapon identity from the catalog's
// certification stamp (stamp.CertificationDigest) instead of plugins.lock's
// digest-lookup path — Custom's mechanism (see NewRoleInvocationPlanFromLock's
// own doc comment on why the two never share a lookup). binding must already
// be the slot's single persisted SlotBinding with EffectiveMode() ==
// SlotBindingModeRanked; this function additionally verifies stamp itself is
// certified, matches binding's provider, and declares affinity for role —
// fail-closed on any mismatch rather than trusting an inconsistent pairing.
func NewRankedRoleInvocationPlanFromCatalog(role, slot string, binding SlotBinding, stamp CatalogRankedStamp) (RoleInvocationPlan, error) {
	if !stamp.Certified() {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: provider %q is not a certified ranked candidate", stamp.ID)
	}
	if stamp.ID != binding.InstalledInstanceID {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: ranked binding provider %q does not match catalog stamp %q", binding.InstalledInstanceID, stamp.ID)
	}
	if !stamp.HasRole(role) {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: certified provider %q does not declare role affinity for %q", stamp.ID, role)
	}
	runtime := NormalizeRankedRuntime(stamp.Runtime)
	if err := runtime.Validate(); err != nil {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: ranked provider %q runtime: %w", stamp.ID, err)
	}
	return RoleInvocationPlan{
		Role:              role,
		Slot:              slot,
		Mode:              SlotBindingModeRanked,
		WeaponID:          stamp.ID,
		WeaponDigest:      stamp.CertificationDigest,
		BindingDigest:     stamp.CertificationDigest,
		BindingGeneration: binding.Generation,
		BindingStatus:     binding.Status,
		Runtime:           runtime,
	}, nil
}

// AttachInvocationContext copies the composer-owned context and output schema
// references into a resolved plan after checking that the envelope addresses
// the same role, slot, and weapon. It keeps binding resolution and context
// selection as separate authorities while making the resulting plan complete.
func AttachInvocationContext(plan RoleInvocationPlan, envelope InvocationEnvelope) (RoleInvocationPlan, error) {
	if envelope.Role != plan.Role || envelope.Slot != plan.Slot || envelope.Provider != plan.WeaponID {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: envelope binding does not match role/slot/provider")
	}
	if envelope.OutputSchemaRef == "" {
		return RoleInvocationPlan{}, fmt.Errorf("role invocation plan: output schema reference is required")
	}
	plan.RequiredContextRefs = append([]string(nil), envelope.RequiredContextRefs...)
	plan.OutputSchemaRef = envelope.OutputSchemaRef
	return plan, nil
}

// SingleLockBindingForSlot returns the single SlotBinding persisted for slot
// in lock.Bindings, erroring on zero or more than one match. Exported so
// rolevalidation.BuildRoleInvocationPlan can inspect a binding's mode before
// choosing which resolution path (Custom vs Ranked) to call.
func SingleLockBindingForSlot(lock PluginLockFile, slot string) (SlotBinding, error) {
	var matches []SlotBinding
	for _, b := range lock.Bindings {
		if b.Slot == slot {
			matches = append(matches, b)
		}
	}
	switch len(matches) {
	case 0:
		return SlotBinding{}, fmt.Errorf("role invocation plan: no persisted weapon binding for slot %q", slot)
	case 1:
		return matches[0], nil
	default:
		return SlotBinding{}, fmt.Errorf("role invocation plan: plugins.lock has %d bindings for slot %q", len(matches), slot)
	}
}
