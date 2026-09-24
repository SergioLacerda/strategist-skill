package domain

// FallbackOutcome is the deterministic result of combining a slot's native-role
// fallback availability with the effective ResolutionPolicy (ADR-0028). It is a
// static classification only — it never observes live invocation state (whether
// the *configured* provider actually failed at mission time is decided by the
// agent, not by this function). Its purpose is to be the single source of truth
// both `strategist check`'s own display and the agent-facing contracts
// (contracts/narrative/00-routing.md § Provider Resolution Policy) point to, so
// the two can never silently drift on what a given policy+availability
// combination is supposed to mean.
type FallbackOutcome string

const (
	// FallbackOutcomeUnavailable means no compatible native role exists for
	// the slot. The policy is irrelevant — there is nothing to fall back to.
	FallbackOutcomeUnavailable FallbackOutcome = "no_fallback_available"
	// FallbackOutcomeBlocked means a compatible native role exists, but the
	// effective policy is "block" — strict failure behavior, no fallback offered.
	FallbackOutcomeBlocked FallbackOutcome = "blocked"
	// FallbackOutcomeAskRequired means a compatible native role exists and the
	// effective policy is "ask" — the agent must request explicit user
	// confirmation before using it. This is also what any unrecognized policy
	// value defaults to (fail toward the more conservative, confirmation-gated
	// outcome rather than silently auto-falling-back).
	FallbackOutcomeAskRequired FallbackOutcome = "ask_required"
	// FallbackOutcomeAutoNative means a compatible native role exists and the
	// effective policy is "native" — the agent uses it automatically, but MUST
	// still emit degradation evidence (configured provider, effective provider,
	// reason). Never implies Strategist Approval Gate acceptance.
	FallbackOutcomeAutoNative FallbackOutcome = "auto_native"
)

// discoverySlot names the route whose selected Weapon is required and cannot
// degrade to native behavior.
const discoverySlot = "discovery"

// DecideFallbackOutcome combines fallbackAvailable (whether a compatible native
// role exists for a slot) with policy's effective value into one deterministic
// FallbackOutcome (ADR-0028). Callers resolving the discovery slot must use
// DecideSlotFallbackOutcome instead — discovery is exempt from this table
// entirely; discovery is handled as FallbackOutcomeUnavailable.
func DecideFallbackOutcome(policy ResolutionPolicy, fallbackAvailable bool) FallbackOutcome {
	if !fallbackAvailable {
		return FallbackOutcomeUnavailable
	}
	switch policy.EffectivePolicy() {
	case ResolutionPolicyBlock:
		return FallbackOutcomeBlocked
	case ResolutionPolicyNative:
		return FallbackOutcomeAutoNative
	case ResolutionPolicyAsk:
		return FallbackOutcomeAskRequired
	default:
		// Unrecognized policy value: fail toward the more conservative,
		// confirmation-gated outcome rather than silently auto-falling-back.
		return FallbackOutcomeAskRequired
	}
}

// DecideSlotFallbackOutcome is DecideFallbackOutcome, adjusted for discovery:
// no fallback outcome is available because Ranger must invoke the selected
// Weapon and fail closed when it cannot. Every other slot defers entirely to
// DecideFallbackOutcome.
func DecideSlotFallbackOutcome(slot string, policy ResolutionPolicy, fallbackAvailable bool) FallbackOutcome {
	if slot == discoverySlot {
		return FallbackOutcomeUnavailable
	}
	return DecideFallbackOutcome(policy, fallbackAvailable)
}
