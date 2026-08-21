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
	// FallbackOutcomeAlwaysNative means the slot's resolution never depends on
	// this policy at all — discovery always resolves to the native Ranger role,
	// unconditionally, regardless of provider_resolution_policy or of what
	// active.slots.discovery is configured to (see 00-routing.md § Discovery
	// Weapon Resolution by Subtype). DecideSlotFallbackOutcome is the only
	// producer of this value; DecideFallbackOutcome alone never returns it.
	FallbackOutcomeAlwaysNative FallbackOutcome = "always_native_no_policy"
)

// discoverySlot names the one slot exempt from ADR-0028's policy table — see
// FallbackOutcomeAlwaysNative.
const discoverySlot = "discovery"

// DecideFallbackOutcome combines fallbackAvailable (whether a compatible native
// role exists for a slot) with policy's effective value into one deterministic
// FallbackOutcome (ADR-0028). Callers resolving the discovery slot must use
// DecideSlotFallbackOutcome instead — discovery is exempt from this table
// entirely (see FallbackOutcomeAlwaysNative).
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

// DecideSlotFallbackOutcome is DecideFallbackOutcome, adjusted for the one slot
// (discovery) whose resolution is not governed by provider_resolution_policy at
// all. Every other slot defers entirely to DecideFallbackOutcome.
func DecideSlotFallbackOutcome(slot string, policy ResolutionPolicy, fallbackAvailable bool) FallbackOutcome {
	if slot == discoverySlot {
		return FallbackOutcomeAlwaysNative
	}
	return DecideFallbackOutcome(policy, fallbackAvailable)
}
