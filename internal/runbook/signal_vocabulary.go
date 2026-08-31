package runbook

import "strings"

// SignalVocabularyVersion identifies the current revision of the controlled
// signal vocabulary below (canonicalSignal consts + signalAliases). Bump it
// whenever a canonical term is added, removed, or has its alias set
// changed materially, so callers pinning behavior (e.g. the golden
// selection tests in select_runbook_golden_test.go) have an explicit
// signal that the vocabulary — not just the code around it — has drifted.
const SignalVocabularyVersion = 1

// CanonicalSignal is a controlled-vocabulary term naming one underlying
// trigger condition that both a runbook's applies_when prose and a
// mission's free-text MissionSignals can be normalized to. It exists
// because Select's original matching was pure case-insensitive substring
// containment: an operator's phrasing ("flaky test") and a runbook
// author's phrasing ("CI test suite is red") never share a literal
// substring even though they describe the same condition, so a purely
// substring-based matcher silently misses the connection. Normalizing both
// sides to a shared CanonicalSignal closes that gap without requiring
// runbook authors and mission callers to agree on exact wording.
type CanonicalSignal string

// Canonical signal terms, derived from what docs/runbooks/*.runbook.yaml
// files actually declare in applies_when as of SignalVocabularyVersion 1
// (see signalAliases for the source runbook of each real-text alias).
// Adding a new runbook whose applies_when describes a genuinely new
// trigger condition should add a new canonical term here (and bump
// SignalVocabularyVersion) rather than folding an unrelated concept into
// an existing alias list.
const (
	// SignalCITestFailure covers a red/flaky test suite.
	// Real source: docs/runbooks/verifying-test-failures.runbook.yaml.
	SignalCITestFailure CanonicalSignal = "ci_test_failure"

	// SignalDependencyUpgrade covers a breaking/major-version dependency
	// bump that needs confirming safe before landing.
	// Real source: docs/runbooks/verifying-dependency-upgrades.runbook.yaml.
	SignalDependencyUpgrade CanonicalSignal = "dependency_upgrade"

	// SignalReleaseToolVersionDrift covers a release/CI failure caused by a
	// tool version that changed without a corresponding commit.
	// Real source: docs/runbooks/release-tool-version-drift.runbook.yaml.
	SignalReleaseToolVersionDrift CanonicalSignal = "release_tool_version_drift"

	// SignalConcurrentSessionCollision covers two concurrent sessions'
	// Snipers colliding on the same file.
	// Real source: docs/runbooks/concurrent-session-sniper-collision.runbook.yaml.
	SignalConcurrentSessionCollision CanonicalSignal = "concurrent_session_collision"

	// SignalProviderInvocationFailure covers a configured slot provider
	// that cannot be invoked or resolved (role_invocation_failed and its
	// sibling slot-resolution error codes).
	// Real source: docs/runbooks/role-invocation-failed.runbook.yaml,
	// docs/runbooks/provider-fallback-policy.runbook.yaml.
	SignalProviderInvocationFailure CanonicalSignal = "provider_invocation_failure"

	// SignalExecutionProviderMissing covers a delegated invocation whose
	// execution_provider was never supplied, or was supplied but cannot be
	// invoked in the current environment — distinct from
	// SignalProviderInvocationFailure, which is about slot/role provider
	// *resolution*, not execution-provider *availability*.
	// Real source: docs/runbooks/delegated-execution-blocked.runbook.yaml.
	SignalExecutionProviderMissing CanonicalSignal = "execution_provider_missing"

	// SignalTreasureChestPartialWrite covers a treasure-chest add/remove
	// that exited non-zero partway through its multi-file batch write.
	// Real source: docs/runbooks/treasure-chest-partial-write.runbook.yaml.
	SignalTreasureChestPartialWrite CanonicalSignal = "treasure_chest_partial_write"

	// SignalVerifyingImplementedDemands covers a request to implement a
	// demand that may already have a refined package, or already be done.
	// Real source: docs/runbooks/verifying-implemented-demands.runbook.yaml.
	SignalVerifyingImplementedDemands CanonicalSignal = "verifying_implemented_demands"

	// SignalComplexityRefactor covers a lint/complexity-tooling signal
	// asking for a narrow, behavior-preserving refactor.
	// Real source: docs/runbooks/refactoring-for-agent-operations.runbook.yaml.
	SignalComplexityRefactor CanonicalSignal = "complexity_refactor"
)

// signalAliases maps each canonical signal to the free-text phrases that
// should resolve to it. Most entries are phrases lifted verbatim from a
// real docs/runbooks/*.runbook.yaml applies_when entry (see the const
// block above for which file); a few are explicitly-marked plausible
// operator synonyms for the same underlying condition, included so that a
// mission's own phrasing of a symptom — which will rarely match a
// runbook author's prose verbatim — still resolves to the right
// canonical signal. Matching is case-insensitive substring containment
// (see canonicalSignalsIn), so an alias should be specific enough that it
// does not also occur, by coincidence, in an unrelated runbook's prose.
var signalAliases = map[CanonicalSignal][]string{
	SignalCITestFailure: {
		"go test -tags spec ./tests/spec/...", // real: verifying-test-failures
		"ci test suite is red",                // operator synonym
		"flaky test",                          // operator synonym
		"tests are failing",                   // operator synonym
	},
	SignalDependencyUpgrade: {
		"npm audit fix --force",    // real: verifying-dependency-upgrades
		"breaking change",          // real: verifying-dependency-upgrades
		"major-version jump",       // real: verifying-dependency-upgrades
		"dependency bump",          // operator synonym
		"go.mod dependency bumped", // operator synonym
	},
	SignalReleaseToolVersionDrift: {
		"tag-triggered release fails",        // real: release-tool-version-drift
		"bump, pin, or unpin a tool version", // real: release-tool-version-drift
		"deprecation warning",                // real: release-tool-version-drift
		"adding a new tool to the pipeline",  // real: release-tool-version-drift
	},
	SignalConcurrentSessionCollision: {
		"two claude sessions running against the same", // real: concurrent-session-sniper-collision
		"sniper materializing to the same file",        // real: concurrent-session-sniper-collision
		"git conflict at commit time",                  // real: concurrent-session-sniper-collision
	},
	SignalProviderInvocationFailure: {
		"role_invocation_failed",  // real: role-invocation-failed, provider-fallback-policy
		"role_provider_invalid",   // real: provider-fallback-policy
		"slot_provider_not_found", // real: provider-fallback-policy
		"slot_risk_mismatch",      // real: provider-fallback-policy
	},
	SignalExecutionProviderMissing: {
		"local_execution_provider_missing", // real: delegated-execution-blocked
		"execution_provider_unavailable",   // real: delegated-execution-blocked
		"local_execution_context_bypass",   // real: delegated-execution-blocked
	},
	SignalTreasureChestPartialWrite: {
		"left in an inconsistent state", // real: treasure-chest-partial-write
		"write <path>: create temp",     // real: treasure-chest-partial-write
		"rename temp",                   // real: treasure-chest-partial-write
		"already committed",             // real: treasure-chest-partial-write
	},
	SignalVerifyingImplementedDemands: {
		"already finished",     // real: verifying-implemented-demands
		"move it to done",      // real: verifying-implemented-demands
		"bootstrap stale scan", // real: verifying-implemented-demands
		"refined/<mission_id>", // real: verifying-implemented-demands
	},
	SignalComplexityRefactor: {
		"golangci-lint",      // real: refactoring-for-agent-operations
		"gocritic",           // real: refactoring-for-agent-operations
		"wrapcheck",          // real: refactoring-for-agent-operations
		"complexity tooling", // real: refactoring-for-agent-operations
		"reduce complexity below a numeric limit", // real: refactoring-for-agent-operations
	},
}

// canonicalSignalsIn returns the set of canonical signals whose canonical
// term itself, or one of its aliases, appears (case-insensitively) as a
// substring of text. An empty text or one that matches no known term/alias
// returns an empty (non-nil) set.
func canonicalSignalsIn(text string) map[CanonicalSignal]bool {
	lower := strings.ToLower(text)
	found := make(map[CanonicalSignal]bool)
	for canonical, aliases := range signalAliases {
		if strings.Contains(lower, string(canonical)) {
			found[canonical] = true
			continue
		}
		for _, alias := range aliases {
			if strings.Contains(lower, strings.ToLower(alias)) {
				found[canonical] = true
				break
			}
		}
	}
	return found
}

// sharesCanonicalSignal reports whether a and b have at least one
// canonical signal in common.
func sharesCanonicalSignal(a, b map[CanonicalSignal]bool) bool {
	for signal := range a {
		if b[signal] {
			return true
		}
	}
	return false
}
