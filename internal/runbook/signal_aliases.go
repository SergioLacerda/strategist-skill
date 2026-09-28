package runbook

// signalAliases maps controlled signals to the free-text phrases that resolve
// to them. Most aliases are lifted from runbook applies_when entries; the
// remaining entries are operator synonyms for the same condition.
var signalAliases = map[CanonicalSignal][]string{
	SignalCITestFailure: {
		"go test -tags spec ./tests/spec/...", "ci test suite is red", "flaky test", "tests are failing",
	},
	SignalDependencyUpgrade: {
		"npm audit fix --force", "breaking change", "major-version jump", "dependency bump", "go.mod dependency bumped",
	},
	SignalReleaseToolVersionDrift: {
		"tag-triggered release fails", "bump, pin, or unpin a tool version", "deprecation warning", "adding a new tool to the pipeline",
	},
	SignalConcurrentSessionCollision: {
		"two claude sessions running against the same", "sniper materializing to the same file", "git conflict at commit time",
	},
	SignalProviderInvocationFailure: {
		"role_invocation_failed", "role_provider_invalid", "slot_provider_not_found", "slot_risk_mismatch",
	},
	SignalExecutionProviderMissing: {
		"local_execution_provider_missing", "execution_provider_unavailable", "local_execution_context_bypass",
	},
	SignalTreasureChestPartialWrite: {
		"left in an inconsistent state", "write <path>: create temp", "rename temp", "already committed",
	},
	SignalVerifyingImplementedDemands: {
		"already finished", "move it to done", "bootstrap stale scan", "refined/<mission_id>",
	},
	SignalComplexityRefactor: {
		"golangci-lint", "gocritic", "wrapcheck", "complexity tooling", "reduce complexity below a numeric limit",
	},
	SignalSkillCorpusHealthReview: {
		"periodic health review of the skill corpus", "structural refactor", "drift/consistency incidents",
		"onboarding review of an unfamiliar skill", "diagnostic", "hardening",
	},
}
