package eval

// AssertionType enumerates Phase 1's deterministic, LLM-content-free
// assertion kinds, plus Phase 2's content-assertion kinds (contains,
// not-contains, regex, max-tokens, forbidden-tool-call, required-sections,
// source-citations). Phase 2's types inspect real fixture content — hand
// authored golden files, or artifacts harvested from completed missions —
// never a scripted provider response. There is no FakeProvider in this
// design; see .analysis/archived/20260804-eval-fake-provider-adr.md (DEC-2).
type AssertionType string

const (
	// AssertEqualState checks Expected.State/Status against the actual
	// result — most scenarios already get this for free via Expected, this
	// type exists for scenarios that want it as an explicit, named check.
	AssertEqualState AssertionType = "equal-state"
	// AssertRequiredEvent checks that a specific domain.TransitionEvent
	// (Value) was present in the state_machine target's input event
	// sequence — i.e., confirms the scenario actually exercised the event
	// under test, not just that the final state matched.
	AssertRequiredEvent AssertionType = "required-event"
	// AssertArtifactExists checks that a file exists at Value (repo-relative path).
	AssertArtifactExists AssertionType = "artifact-exists"
	// AssertScopeIncludes checks that Value is present in a scope_filter
	// target's resulting Expected.IDs set.
	AssertScopeIncludes AssertionType = "scope-includes"
	// AssertScopeExcludes checks that Value is absent from a scope_filter
	// target's resulting Expected.IDs set.
	AssertScopeExcludes AssertionType = "scope-excludes"

	// Phase 2 (20260804-eval-fake-provider): content assertions, evaluated by
	// evaluateContentAssertions (content_assert.go) against a TargetArtifactCheck
	// fixture's raw file content.

	// AssertContains checks that Value is a substring of the fixture content.
	AssertContains AssertionType = "contains"
	// AssertNotContains checks that Value is NOT a substring of the fixture content.
	AssertNotContains AssertionType = "not-contains"
	// AssertRegex checks that Value, compiled as a regexp, matches somewhere
	// in the fixture content.
	AssertRegex AssertionType = "regex"
	// AssertMaxTokens checks that the fixture content's whitespace-split word
	// count does not exceed the integer in Value. This is a deliberately
	// simple approximation, not a real tokenizer — sufficient for a fixture
	// size budget, not for measuring real LLM token usage.
	AssertMaxTokens AssertionType = "max-tokens"
	// AssertForbiddenToolCall checks that Value (a tool/command name or
	// fragment, e.g. "git commit") does NOT appear in the fixture content —
	// a regression guard for recorded-transcript-shaped fixtures.
	AssertForbiddenToolCall AssertionType = "forbidden-tool-call"
	// AssertRequiredSections checks that every comma-separated heading name
	// in Value appears as a markdown heading line (a line starting with one
	// or more '#') in the fixture content.
	AssertRequiredSections AssertionType = "required-sections"
	// AssertSourceCitations checks that the fixture content references at
	// least N distinct source-looking paths (files ending .md/.go/.yaml/
	// .yml/.txt), where N is the integer in Value.
	AssertSourceCitations AssertionType = "source-citations"
)

// Assertion is one explicit, named check evaluated in addition to Expected.
type Assertion struct {
	Type  AssertionType `yaml:"type"`
	Value string        `yaml:"value"`
}
