package handoff

// coverage_gaps_test.go – white-box tests targeting uncovered branches to push
// internal/handoff from 82.7% to ≥95%. Focuses on:
//   - metadata.go (0% → covered)
//   - outcome.go SignalsRecordOf (0% → covered)
//   - outcome_store.go uncovered branches
//   - outcome_authorize.go partial branches
//   - artifact.go / artifact_validation.go partial branches
//   - signals.go partial branches
//   - package_digest.go partial branches
//   - policy_risk_gate.go partial branches
//   - verify_policy_invalid.go partial branch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers shared across tests
// ---------------------------------------------------------------------------

// minimalArchivistPackage writes a complete, valid Archivist package to dir.
func minimalArchivistPackage(t *testing.T, dir, missionID string) {
	t.Helper()
	analysis := "---\nmission_id: " + missionID + "\nmission_status: archivist_done\n" +
		"handoff_policy_facts:\n" +
		"  schema_version: strategist-handoff-policy-facts/v1\n" +
		"  mandatory_constraints: []\n" +
		"  unresolved_questions: []\n" +
		"  forbidden_scope: []\n" +
		"  destructive_operation_possible: false\n" +
		"  security_sensitive_task: false\n" +
		"  informational_only: false\n" +
		"---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("# Proposal\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design.md"), []byte("# Design\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"),
		[]byte("- [ ] 1.1 [implementation_handoff] change source\n"), 0o600))
}

// ---------------------------------------------------------------------------
// metadata.go — ReadVerificationMetadata (0% → covered)
// ---------------------------------------------------------------------------

func TestReadVerificationMetadata_AbsentBlockReturnsNil(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	meta, err := ReadVerificationMetadata(dir)
	require.NoError(t, err)
	assert.Nil(t, meta)
}

func TestReadVerificationMetadata_FrontmatterBlockIsRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	enabled := true
	required := false
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n" +
		"handoff_policy_facts:\n" +
		"  schema_version: strategist-handoff-policy-facts/v1\n" +
		"  mandatory_constraints: []\n" +
		"  unresolved_questions: []\n" +
		"  forbidden_scope: []\n" +
		"  destructive_operation_possible: false\n" +
		"  security_sensitive_task: false\n" +
		"  informational_only: false\n" +
		"handoff_verification:\n" +
		"  required: false\n" +
		"  enabled: true\n" +
		"---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("# Proposal\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design.md"), []byte("# Design\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"),
		[]byte("- [ ] 1.1 [implementation_handoff] change source\n"), 0o600))

	meta, err := ReadVerificationMetadata(dir)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, &enabled, meta.Enabled)
	assert.Equal(t, &required, meta.Required)
}

func TestReadVerificationMetadata_FencedYAMLBlockIsRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	// Overwrite tasks.md with a fenced YAML block containing handoff_verification.
	tasks := "- [ ] 1.1 [implementation_handoff] change source\n\n" +
		"```yaml\nhandoff_verification:\n  required: true\n  enabled: true\n```\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasks), 0o600))

	meta, err := ReadVerificationMetadata(dir)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, meta.Required)
	assert.True(t, *meta.Required)
}

func TestReadVerificationMetadata_ConflictingDeclarationsAreRejected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Declare in frontmatter with required=true and in fenced block with required=false.
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n" +
		"handoff_policy_facts:\n" +
		"  schema_version: strategist-handoff-policy-facts/v1\n" +
		"  mandatory_constraints: []\n" +
		"  unresolved_questions: []\n" +
		"  forbidden_scope: []\n" +
		"  destructive_operation_possible: false\n" +
		"  security_sensitive_task: false\n" +
		"  informational_only: false\n" +
		"handoff_verification:\n  required: true\n  enabled: true\n" +
		"---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("# Proposal\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design.md"), []byte("# Design\n"), 0o600))
	tasks := "- [ ] 1.1 [implementation_handoff] change source\n\n" +
		"```yaml\nhandoff_verification:\n  required: false\n  enabled: true\n```\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasks), 0o600))

	_, err := ReadVerificationMetadata(dir)
	require.ErrorContains(t, err, "handoff_metadata_mismatch")
}

func TestReadVerificationMetadata_MissingAnalysisReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ReadVerificationMetadata(t.TempDir())
	require.ErrorContains(t, err, "handoff_artifact_invalid")
}

func TestReadVerificationMetadata_MissingTasksReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Write analysis.md but no tasks.md.
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n" +
		"handoff_policy_facts:\n" +
		"  schema_version: strategist-handoff-policy-facts/v1\n" +
		"  mandatory_constraints: []\n" +
		"  unresolved_questions: []\n" +
		"  forbidden_scope: []\n" +
		"  destructive_operation_possible: false\n" +
		"  security_sensitive_task: false\n" +
		"  informational_only: false\n" +
		"---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	_, err := ReadVerificationMetadata(dir)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// VerificationMetadata.ConsistentWith
// ---------------------------------------------------------------------------

func TestVerificationMetadataConsistentWith_NilIsAlwaysConsistent(t *testing.T) {
	t.Parallel()
	var m *VerificationMetadata
	require.NoError(t, m.ConsistentWith(true))
	require.NoError(t, m.ConsistentWith(false))
}

func TestVerificationMetadataConsistentWith_RequiredMismatch(t *testing.T) {
	t.Parallel()
	f := false
	m := &VerificationMetadata{Required: &f}
	require.ErrorContains(t, m.ConsistentWith(true), "handoff_metadata_mismatch")
}

func TestVerificationMetadataConsistentWith_EnabledMismatch(t *testing.T) {
	t.Parallel()
	f := false
	m := &VerificationMetadata{Enabled: &f}
	require.ErrorContains(t, m.ConsistentWith(true), "handoff_metadata_mismatch")
}

func TestVerificationMetadataConsistentWith_MatchingValuesAreConsistent(t *testing.T) {
	t.Parallel()
	tr := true
	m := &VerificationMetadata{Required: &tr, Enabled: &tr}
	require.NoError(t, m.ConsistentWith(true))
}

// ---------------------------------------------------------------------------
// firstSet and differs (internal helpers)
// ---------------------------------------------------------------------------

func TestFirstSet_ReturnsFirstWhenSet(t *testing.T) {
	t.Parallel()
	a := true
	b := false
	got := firstSet(&a, &b)
	assert.Equal(t, &a, got)
}

func TestFirstSet_ReturnsSecondWhenFirstNil(t *testing.T) {
	t.Parallel()
	b := false
	got := firstSet(nil, &b)
	assert.Equal(t, &b, got)
}

func TestDiffers_BothNilIsFalse(t *testing.T) {
	t.Parallel()
	assert.False(t, differs(nil, nil))
}

func TestDiffers_OneNilIsFalse(t *testing.T) {
	t.Parallel()
	tr := true
	assert.False(t, differs(&tr, nil))
	assert.False(t, differs(nil, &tr))
}

func TestDiffers_SameValueIsFalse(t *testing.T) {
	t.Parallel()
	tr := true
	tr2 := true
	assert.False(t, differs(&tr, &tr2))
}

func TestDiffers_DifferentValuesIsTrue(t *testing.T) {
	t.Parallel()
	tr := true
	fa := false
	assert.True(t, differs(&tr, &fa))
}

// ---------------------------------------------------------------------------
// fencedVerificationMetadata — non-handoff-verification fenced block is skipped
// ---------------------------------------------------------------------------

func TestFencedVerificationMetadata_NonHandoffBlockIsIgnored(t *testing.T) {
	t.Parallel()
	tasks := []byte("```yaml\nsome_other_key: value\n```\n")
	found, err := fencedVerificationMetadata(tasks)
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestFencedVerificationMetadata_NonYAMLBlockIsSkipped(t *testing.T) {
	t.Parallel()
	// A code block that doesn't parse as YAML mapping is gracefully skipped.
	tasks := []byte("```yaml\n- item one\n- item two\n```\n")
	found, err := fencedVerificationMetadata(tasks)
	require.NoError(t, err)
	assert.Empty(t, found)
}

// ---------------------------------------------------------------------------
// outcome.go — SignalsRecordOf (0%)
// ---------------------------------------------------------------------------

func TestSignalsRecordOf_ProjectsAllFields(t *testing.T) {
	t.Parallel()
	signals := RiskSignals{
		MandatoryConstraintsPresent:  true,
		UnresolvedQuestionsPresent:   true,
		ForbiddenScopePresent:        true,
		ImplementationHandoffPresent: true,
		DestructiveOperationPossible: true,
		SecuritySensitiveTask:        true,
		InformationalOnly:            false,
	}
	record := SignalsRecordOf(signals)
	assert.True(t, record.MandatoryConstraintsPresent)
	assert.True(t, record.UnresolvedQuestionsPresent)
	assert.True(t, record.ForbiddenScopePresent)
	assert.True(t, record.ImplementationHandoffPresent)
	assert.True(t, record.DestructiveOperationPossible)
	assert.True(t, record.SecuritySensitiveTask)
	assert.False(t, record.InformationalOnly)
}

// ---------------------------------------------------------------------------
// outcome_store.go — NewOutcomeStore (0%)
// ---------------------------------------------------------------------------

func TestNewOutcomeStore_HasUTCClock(t *testing.T) {
	t.Parallel()
	store := NewOutcomeStore(t.TempDir())
	assert.NotNil(t, store.Clock)
	now := store.now()
	assert.WithinDuration(t, time.Now().UTC(), now, 5*time.Second)
}

// ---------------------------------------------------------------------------
// outcome_store.go — now() with nil clock
// ---------------------------------------------------------------------------

func TestOutcomeStoreNow_NilClockFallsBackToRealTime(t *testing.T) {
	t.Parallel()
	store := OutcomeStore{Root: t.TempDir(), Clock: nil}
	now := store.now()
	assert.WithinDuration(t, time.Now().UTC(), now, 5*time.Second)
}

// ---------------------------------------------------------------------------
// outcome_store.go — dir() error branches
// ---------------------------------------------------------------------------

func TestOutcomeStoreDir_EmptyRootReturnsError(t *testing.T) {
	t.Parallel()
	store := OutcomeStore{Root: ""}
	_, err := store.dir("m1")
	require.ErrorContains(t, err, "runtime root is required")
}

func TestOutcomeStoreDir_DotDotMissionIDReturnsError(t *testing.T) {
	t.Parallel()
	store := OutcomeStore{Root: t.TempDir()}
	_, err := store.dir("..")
	require.ErrorContains(t, err, "malformed mission id")
}

func TestOutcomeStoreDir_SlashInMissionIDReturnsError(t *testing.T) {
	t.Parallel()
	store := OutcomeStore{Root: t.TempDir()}
	_, err := store.dir("a/b")
	require.ErrorContains(t, err, "malformed mission id")
}

// ---------------------------------------------------------------------------
// outcome_store.go — attempts() error branch (unreadable dir that exists)
// ---------------------------------------------------------------------------

func TestOutcomeStoreAttempts_UnreadableDirReturnsError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	store := fixedStore(t)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := store.NextAttempt("m1")
	require.ErrorContains(t, err, "handoff_outcome_unreadable")
}

// ---------------------------------------------------------------------------
// outcome_store.go — writeTemporary error branches
// ---------------------------------------------------------------------------

func TestOutcomeStoreAppend_UnwritableDirReturnsError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	store := fixedStore(t)
	// First append to create the directory, then lock it.
	_, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)

	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.Chmod(dir, 0o555)) // read+execute but not write
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err = store.Append(sampleOutcome(2, OutcomePassed))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// outcome_store.go — Latest error branches
// ---------------------------------------------------------------------------

func TestOutcomeStoreLatest_InvalidJSONIsDetectedAsTampering(t *testing.T) {
	t.Parallel()
	store := fixedStore(t)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "attempt-001.json"), []byte("not-json"), 0o644))

	_, err := store.Latest("m1")
	require.ErrorContains(t, err, "handoff_outcome_tampered")
}

// ---------------------------------------------------------------------------
// outcome_authorize.go — Consumed() branches
// ---------------------------------------------------------------------------

func TestConsumed_UnreadableMarkerFileReturnsError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	store := fixedStore(t)
	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)
	// Write a consumed marker that is unreadable.
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	markerPath := filepath.Join(dir, consumedFile)
	require.NoError(t, os.WriteFile(markerPath, []byte("{}"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(markerPath, 0o644) })

	_, err = store.Consumed(sealed)
	require.ErrorContains(t, err, "handoff_outcome_unreadable")
}

func TestConsumed_CorruptMarkerFileIsDetectedAsTampering(t *testing.T) {
	t.Parallel()
	store := fixedStore(t)
	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.WriteFile(filepath.Join(dir, consumedFile), []byte("not-json\n"), 0o644))

	_, err = store.Consumed(sealed)
	require.ErrorContains(t, err, "handoff_outcome_tampered")
}

func TestConsumed_MarkerWithDifferentIntegrityReturnsFalse(t *testing.T) {
	t.Parallel()
	store := fixedStore(t)
	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)
	require.NoError(t, store.Consume(sealed))

	// Create an outcome that differs from the consumed one.
	other := sealed
	other.Integrity = "sha256:different"
	consumed, err := store.Consumed(other)
	require.NoError(t, err)
	assert.False(t, consumed)
}

// ---------------------------------------------------------------------------
// outcome_authorize.go — Consume() with malformed mission id
// ---------------------------------------------------------------------------

func TestConsume_MalformedMissionIDReturnsError(t *testing.T) {
	t.Parallel()
	store := fixedStore(t)
	outcome := Outcome{MissionID: "a/b", Integrity: "sha256:abc"}
	err := store.Consume(outcome)
	require.ErrorContains(t, err, "malformed mission id")
}

// ---------------------------------------------------------------------------
// outcome_authorize.go — correlateOutcome — transition mismatch
// ---------------------------------------------------------------------------

func TestCorrelateOutcome_TransitionMismatchIsRejected(t *testing.T) {
	t.Parallel()
	store := fixedStore(t)
	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)

	// Manually overwrite with a different transition but valid integrity.
	sealed.Transition = "sniper_to_validation"
	digest, err := sealed.digest()
	require.NoError(t, err)
	sealed.Integrity = digest

	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	raw, err := jsonMarshalOutcome(sealed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, attemptFile(1)), raw, 0o644))

	_, err = store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_outcome_transition_mismatch")
}

// ---------------------------------------------------------------------------
// artifact_validation.go — validateIdentity missing mission_id key
// ---------------------------------------------------------------------------

func TestValidateIdentity_MissingMissionIDKeyIsRejected(t *testing.T) {
	t.Parallel()
	// frontmatter without mission_id key at all.
	err := validateIdentity(map[string]any{"mission_status": "ranger_done"}, "m-1", []string{"ranger_done"})
	require.ErrorContains(t, err, "mission_id")
}

// ---------------------------------------------------------------------------
// artifact_validation.go — hasSection — section in body
// ---------------------------------------------------------------------------

func TestHasSection_PresentSection(t *testing.T) {
	t.Parallel()
	body := []byte("## mission_objective\nsome content\n## handoff\nother\n")
	assert.True(t, hasSection(body, "mission_objective"))
	assert.True(t, hasSection(body, "handoff"))
	assert.False(t, hasSection(body, "missing_section"))
}

// ---------------------------------------------------------------------------
// signals.go — readPolicyFacts error paths
// ---------------------------------------------------------------------------

func TestReadPolicyFacts_MissingAnalysisReturnsError(t *testing.T) {
	t.Parallel()
	_, err := readPolicyFacts(t.TempDir())
	require.ErrorContains(t, err, "handoff_artifact_invalid")
}

// ---------------------------------------------------------------------------
// signals.go — taskID — unnumbered fallback
// ---------------------------------------------------------------------------

func TestTaskID_NoLeadingNumberReturnsUnnumbered(t *testing.T) {
	t.Parallel()
	// line has no field starting with a digit after the checkbox.
	line := "- [ ] [implementation_handoff] change source without number"
	got := taskID(line)
	assert.Equal(t, "unnumbered", got)
}

func TestTaskID_NumberedTaskReturnsNumber(t *testing.T) {
	t.Parallel()
	line := "- [ ] 3.2 [implementation_handoff] do something"
	got := taskID(line)
	assert.Equal(t, "3.2", got)
}

// ---------------------------------------------------------------------------
// package_digest.go — PackageDigest error branches
// ---------------------------------------------------------------------------

func TestPackageDigest_MissingAnalysisReturnsError(t *testing.T) {
	t.Parallel()
	_, err := PackageDigest(t.TempDir())
	require.ErrorContains(t, err, "handoff_artifact_invalid")
}

func TestPackageDigest_MissingSupportingFileReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n" +
		"handoff_policy_facts:\n" +
		"  schema_version: strategist-handoff-policy-facts/v1\n" +
		"  mandatory_constraints: []\n" +
		"  unresolved_questions: []\n" +
		"  forbidden_scope: []\n" +
		"  destructive_operation_possible: false\n" +
		"  security_sensitive_task: false\n" +
		"  informational_only: false\n" +
		"---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	// proposal.md is missing.
	_, err := PackageDigest(dir)
	require.ErrorContains(t, err, "handoff_artifact_invalid")
}

func TestPackageDigest_ValidPackageReturnsDeterministicDigest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	d1, err := PackageDigest(dir)
	require.NoError(t, err)
	d2, err := PackageDigest(dir)
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
	assert.Contains(t, d1, "sha256:")
}

// ---------------------------------------------------------------------------
// policy_risk_gate.go — riskGatedPolicy with RequireWhen/SkipWhen
// ---------------------------------------------------------------------------

func TestRiskGatedPolicy_PolicyWithRequireWhenIsEvaluatedDynamically(t *testing.T) {
	t.Parallel()
	base := RangerToArchivistPolicy()
	// With an empty riskLevel (no signals), the SkipWhen conditions govern.
	result := riskGatedPolicy(base, "low")
	// low risk → no require signals → Enabled should be false (advisory-first).
	assert.False(t, result.Enabled)
	assert.Nil(t, result.RequiredTypes)
}

func TestRiskGatedPolicy_PolicyWithNoRequireWhenUsesStatusForRisk(t *testing.T) {
	t.Parallel()
	// A base policy with no RequireWhen/SkipWhen falls back to StatusForRisk.
	base := Policy{Transition: "test-transition"}
	result := riskGatedPolicy(base, "high")
	// high risk → require signals → Enabled = true.
	assert.True(t, result.Enabled)
}

// ---------------------------------------------------------------------------
// policy_risk_gate.go — predicateMatches — default branch
// ---------------------------------------------------------------------------

func TestPredicateMatches_UnknownPredicateReturnsFalse(t *testing.T) {
	t.Parallel()
	result := predicateMatches("completely_unknown_predicate", RiskSignals{})
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// verify_policy_invalid.go — policyErrorMessages — single error (no unwrap)
// ---------------------------------------------------------------------------

func TestPolicyErrorMessages_SingleErrorReturnsSingleMessage(t *testing.T) {
	t.Parallel()
	err := errors.New("a single error")
	messages := policyErrorMessages(err)
	require.Len(t, messages, 1)
	assert.Equal(t, "a single error", messages[0])
}

func TestPolicyErrorMessages_JoinedErrorReturnsMultipleMessages(t *testing.T) {
	t.Parallel()
	err := errors.Join(errors.New("first"), errors.New("second"))
	messages := policyErrorMessages(err)
	require.Len(t, messages, 2)
	assert.Equal(t, "first", messages[0])
	assert.Equal(t, "second", messages[1])
}

// ---------------------------------------------------------------------------
// artifact.go — validateArchivistAnalysis bad frontmatter
// ---------------------------------------------------------------------------

func TestValidateArchivistAnalysis_BadFrontmatterReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// analysis.md with no closing frontmatter delimiter.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"),
		[]byte("---\nmission_id: m-1\n"), 0o600))
	err := validateArchivistAnalysis(filepath.Join(dir, "analysis.md"), "m-1")
	require.ErrorContains(t, err, "handoff_artifact_invalid")
}

// ---------------------------------------------------------------------------
// Additional coverage gap tests for metadata, signals, policy_risk_gate & outcome_store
// ---------------------------------------------------------------------------

func TestFencedVerificationMetadata_InvalidYAMLBlockSkipped(t *testing.T) {
	t.Parallel()
	tasks := []byte("```yaml\n[unclosed array\n```\n")
	found, err := fencedVerificationMetadata(tasks)
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestFencedVerificationMetadata_InvalidVerificationTypeErrors(t *testing.T) {
	t.Parallel()
	tasks := []byte("```yaml\nhandoff_verification:\n  required: [not, a, bool]\n```\n")
	_, err := fencedVerificationMetadata(tasks)
	require.ErrorContains(t, err, "handoff_metadata_invalid")
}

func TestReconcileMetadata_FirstSetFallback(t *testing.T) {
	t.Parallel()
	req := true
	ena := true
	m1 := VerificationMetadata{Required: nil, Enabled: &ena}
	m2 := VerificationMetadata{Required: &req, Enabled: nil}
	reconciled, err := reconcileMetadata([]VerificationMetadata{m1, m2})
	require.NoError(t, err)
	require.NotNil(t, reconciled.Required)
	assert.True(t, *reconciled.Required)
	require.NotNil(t, reconciled.Enabled)
	assert.True(t, *reconciled.Enabled)
}

func TestApplyCoarseRisk_UnknownRiskLevelErrors(t *testing.T) {
	t.Parallel()
	var e ExtractedSignals
	err := e.applyCoarseRisk("super-high")
	require.ErrorContains(t, err, "handoff_risk_level_unknown")
}

func TestTaskID_UnnumberedTask(t *testing.T) {
	t.Parallel()
	id := taskID("- [ ] [implementation_handoff] task_without_digits")
	assert.Equal(t, "unnumbered", id)
}

func TestSettleInformational_ContradictoryFactsErrors(t *testing.T) {
	t.Parallel()
	tru := true
	facts := PolicyFacts{
		InformationalOnly: &tru,
	}
	var e ExtractedSignals
	e.add(&e.Signals.DestructiveOperationPossible, PredicateDestructiveOperationPossible, "test", "declared")
	err := e.settleInformational(facts)
	require.ErrorContains(t, err, "handoff_policy_facts_contradictory")
}

func TestPredicateMatches_AllBranches(t *testing.T) {
	t.Parallel()
	signals := RiskSignals{
		MandatoryConstraintsPresent:  true,
		UnresolvedQuestionsPresent:   true,
		ForbiddenScopePresent:        true,
		ImplementationHandoffPresent: true,
		DestructiveOperationPossible: true,
		SecuritySensitiveTask:        true,
		InformationalOnly:            true,
	}

	assert.True(t, predicateMatches(PredicateMandatoryConstraintsPresent, signals))
	assert.True(t, predicateMatches(PredicateUnresolvedQuestionsPresent, signals))
	assert.True(t, predicateMatches(PredicateForbiddenScopePresent, signals))
	assert.True(t, predicateMatches(PredicateImplementationHandoffPresent, signals))
	assert.True(t, predicateMatches(PredicateDestructiveOperationPossible, signals))
	assert.True(t, predicateMatches(PredicateSecuritySensitiveTask, signals))
	assert.True(t, predicateMatches(PredicateInformationalOnly, signals))
	assert.False(t, predicateMatches(PredicateNoCriticalConstraints, signals))
	assert.False(t, predicateMatches(PredicateNoUnresolvedQuestions, signals))
	assert.False(t, predicateMatches(PredicateNoForbiddenScopeBeyondSniperDefaults, signals))
	assert.False(t, predicateMatches(PolicyPredicate("unknown"), signals))
}

func TestOutcomeStore_LatestCorruptedJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := OutcomeStore{Root: dir}
	missionDir := filepath.Join(dir, "missions", "handoff", "m-corrupt")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, attemptFile(1)), []byte("invalid json"), 0o600))

	_, err := store.Latest("m-corrupt")
	require.ErrorContains(t, err, "handoff_outcome_tampered")
}

func TestOutcome_ValidateShape_Gaps(t *testing.T) {
	t.Parallel()

	valid := Outcome{
		MissionID:     "m-1",
		Transition:    TransitionArchivistToSniper,
		PackageDigest: "sha256:abc",
		PolicyID:      "policy-1",
		Attempt:       1,
		GateObserved:  StatusRequired,
		Result:        OutcomePassed,
	}

	// Attempt < 1
	o := valid
	o.Attempt = 0
	require.ErrorContains(t, o.validateShape(), "attempt must be positive")

	// GateObserved empty
	o = valid
	o.GateObserved = ""
	require.ErrorContains(t, o.validateShape(), "gate observation is required")

	// Unknown result
	o = valid
	o.Result = "invalid_result"
	require.ErrorContains(t, o.validateShape(), "handoff_outcome_unknown_result")

	// Skipped but required
	o = valid
	o.Result = OutcomeSkipped
	o.Required = true
	require.ErrorContains(t, o.validateShape(), "skipped outcome cannot be a required challenge")
}

func TestOutcome_VerifyIntegrity_Tampered(t *testing.T) {
	t.Parallel()

	o := Outcome{
		MissionID:     "m-1",
		Transition:    TransitionArchivistToSniper,
		PackageDigest: "sha256:abc",
		PolicyID:      "policy-1",
		Attempt:       1,
		GateObserved:  StatusRequired,
		Result:        OutcomePassed,
		Integrity:     "",
	}
	require.ErrorContains(t, o.VerifyIntegrity(), "handoff_outcome_tampered")

	o.Integrity = "sha256:invalid"
	require.ErrorContains(t, o.VerifyIntegrity(), "handoff_outcome_tampered")
}

func TestOutcomeStore_Consumed_CorruptedJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := OutcomeStore{Root: dir}
	missionDir := filepath.Join(dir, "missions", "handoff", "m-consumed-corrupt")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "consumed.json"), []byte("invalid json"), 0o600))

	outcome := Outcome{MissionID: "m-consumed-corrupt"}
	_, err := store.Consumed(outcome)
	require.ErrorContains(t, err, "handoff_outcome_tampered")
}

func TestCorrelateOutcome_FailedAndUnknownResult(t *testing.T) {
	t.Parallel()
	check := ExecutionCheck{
		MissionID:     "m-1",
		PackageDigest: "sha256:abc",
		PolicyID:      "p-1",
	}
	base := Outcome{
		MissionID:     "m-1",
		Transition:    TransitionArchivistToSniper,
		PackageDigest: "sha256:abc",
		PolicyID:      "p-1",
		Attempt:       1,
	}

	failed := base
	failed.Result = OutcomeFailed
	require.ErrorContains(t, correlateOutcome(failed, check), "handoff_outcome_failed")

	unknown := base
	unknown.Result = "mystery"
	require.ErrorContains(t, correlateOutcome(unknown, check), "handoff_outcome_unknown_result")
}

func TestValidateIdentity_NonStringFields(t *testing.T) {
	t.Parallel()

	// mission_id is not a string
	fm1 := map[string]any{"mission_id": 123, "mission_status": "ready"}
	err := validateIdentity(fm1, "m-1", []string{"ready"})
	require.ErrorContains(t, err, "mission_id \"\" does not match \"m-1\"")

	// mission_status is not a string
	fm2 := map[string]any{"mission_id": "m-1", "mission_status": 123}
	err = validateIdentity(fm2, "m-1", []string{"ready"})
	require.ErrorContains(t, err, "mission_status \"\" is not valid")
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	t.Parallel()
	content := []byte("---\n: : :\n---\nbody")
	_, _, err := parseFrontmatter(content)
	require.ErrorContains(t, err, "parse frontmatter")
}

func TestParsePolicyFacts_MissingFlagField(t *testing.T) {
	t.Parallel()
	tru := true
	raw := map[string]any{
		"schema_version":          PolicyFactsSchemaVersion,
		"mandatory_constraints":   []any{},
		"unresolved_questions":    []any{},
		"forbidden_scope":         []any{},
		"security_sensitive_task": &tru,
		"informational_only":      &tru,
		// destructive_operation_possible is missing
	}
	fm := map[string]any{PolicyFactsKey: raw}
	_, err := ParsePolicyFacts(fm)
	require.ErrorContains(t, err, "handoff_policy_facts_missing: handoff_policy_facts.destructive_operation_possible is required")
}

func TestParseAttemptName_Gaps(t *testing.T) {
	t.Parallel()

	num, isAttempt, err := parseAttemptName("not-an-attempt.txt")
	require.NoError(t, err)
	assert.False(t, isAttempt)
	assert.Equal(t, 0, num)

	_, _, err = parseAttemptName("attempt-0.json")
	require.ErrorContains(t, err, "handoff_outcome_tampered: unrecognized outcome file \"attempt-0.json\"")

	_, _, err = parseAttemptName("attempt-abc.json")
	require.ErrorContains(t, err, "handoff_outcome_tampered: unrecognized outcome file \"attempt-abc.json\"")
}

func TestValidateRangerArtifact_MissingSourcesConsulted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "ranger.md")
	// frontmatter with sources_consulted missing or non-slice
	content := []byte("---\nmission_id: m-1\nmission_status: ranger_done\nsources_consulted: \"not a list\"\n---\n## mission_objective\n## known_facts\n## confidence_summary\n## handoff\n")
	require.NoError(t, os.WriteFile(path, content, 0o600))

	err := ValidateRangerArtifact(path, "m-1")
	require.ErrorContains(t, err, "missing list field \"sources_consulted\"")
}

func TestValidateRangerArtifact_MissingSection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "ranger.md")
	// missing handoff section
	content := []byte("---\nmission_id: m-1\nmission_status: ranger_done\nsources_consulted: []\n---\n## mission_objective\n## known_facts\n## confidence_summary\n")
	require.NoError(t, os.WriteFile(path, content, 0o600))

	err := ValidateRangerArtifact(path, "m-1")
	require.ErrorContains(t, err, "missing section \"handoff\"")
}

// ---------------------------------------------------------------------------
// Helper: marshal Outcome to JSON (needed for transition-mismatch test above)
// ---------------------------------------------------------------------------

func jsonMarshalOutcome(o Outcome) ([]byte, error) {
	raw, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
