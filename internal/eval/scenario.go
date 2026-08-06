// Package eval provides a deterministic scenario harness for exercising
// Strategist's orchestration policy (internal/domain, internal/treasure)
// directly, in-process — no LLM or skill-provider invocation involved. See
// .analysis/refined/20260804-test-framework-v2/design.md for the full
// design rationale, including why this Phase 1 scope excludes a
// provider-scripting layer (FakeProvider).
package eval

// Scenario is a single deterministic test case exercising a domain/treasure
// exported function directly.
type Scenario struct {
	ID          string      `yaml:"id"`
	Description string      `yaml:"description"`
	Input       Input       `yaml:"input"`
	Expected    Expected    `yaml:"expected"`
	Assertions  []Assertion `yaml:"assertions"`
}

// Target names which internal/domain or internal/treasure surface a
// Scenario dispatches to. This vocabulary is Phase 1's own — it extends
// design.md's original three-value sketch (state_machine, route_decision,
// scope_filter) with slot_write_scope and artifact_check, each backed by
// an existing exported function or a plain file/YAML check, to cover the
// full set of scenarios the refined tasks.md assigns to Phase 1 (see
// harness.go for the dispatch and the rationale for each addition).
type Target string

// Target values, one per dispatchable internal/domain or internal/treasure surface.
const (
	TargetStateMachine   Target = "state_machine"
	TargetRouteDecision  Target = "route_decision"
	TargetScopeFilter    Target = "scope_filter"
	TargetSlotWriteScope Target = "slot_write_scope"
	TargetArtifactCheck  Target = "artifact_check"
	// TargetChestGrade dispatches to domain.ValidateChestGrade — treasure-chest
	// grading field validation (source_grade/reuse_value/implementation_status).
	TargetChestGrade Target = "chest_grade"
	// TargetJewelTrust dispatches to domain.ValidateJewelTrust — the safeguard
	// that a jewel's trust tier may never exceed its parent chest's trust tier.
	TargetJewelTrust Target = "jewel_trust"
	// TargetCriticalHitTrigger dispatches to domain.EvaluateCriticalHit — the
	// plain-move/closure-move trigger conditions from
	// contracts/machine/critical-hit.yaml#trigger_conditions.
	TargetCriticalHitTrigger Target = "critical_hit_trigger"
)

// Input carries whatever the target function needs. Shape depends on
// Target — kept as a loosely-typed map rather than one shared struct
// across five unrelated call signatures.
type Input struct {
	Target Target         `yaml:"target"`
	Params map[string]any `yaml:"params"`
}

// Expected declares the pass condition. Not every field applies to every
// Target — see harness.go's per-target evaluation for which fields it
// reads.
type Expected struct {
	// State is the expected final domain.MissionState (target: state_machine).
	State string `yaml:"state,omitempty"`
	// Status is "allowed" or "blocked" (target: route_decision, slot_write_scope).
	Status string `yaml:"status,omitempty"`
	// Reason, when non-empty, must be a substring of the actual reason/error text
	// (target: route_decision, slot_write_scope).
	Reason string `yaml:"reason,omitempty"`
	// IDs is the expected set of StatusRow.ID values, order-independent
	// (target: scope_filter).
	IDs []string `yaml:"ids,omitempty"`
}
