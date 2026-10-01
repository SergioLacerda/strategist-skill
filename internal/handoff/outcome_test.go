package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleOutcome(attempt int, result string) Outcome {
	return Outcome{
		MissionID: "m1", Transition: TransitionArchivistToSniper, PackageDigest: "sha256:pkg", Attempt: attempt, Result: result,
		PolicyID: PolicyIdentity(DefaultPolicy()), Required: result != OutcomeSkipped,
		Signals: SignalsRecord{ImplementationHandoffPresent: true}, GateObserved: "handoff_challenge",
		Provenance: []SignalProvenance{{Signal: "implementation_handoff_present", Source: "tasks.md", Detail: "2.1"}},
	}
}

func fixedStore(t *testing.T) OutcomeStore {
	t.Helper()
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	return OutcomeStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
}

func currentCheck() ExecutionCheck {
	return ExecutionCheck{MissionID: "m1", PackageDigest: "sha256:pkg", PolicyID: PolicyIdentity(DefaultPolicy())}
}

func TestOutcomeRoundTripsAndVerifiesItsIntegrity(t *testing.T) {
	store := fixedStore(t)

	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)
	got, err := store.Latest("m1")

	require.NoError(t, err)
	assert.Equal(t, sealed, got)
	assert.Equal(t, OutcomeSchemaVersion, got.SchemaVersion)
	assert.NotEmpty(t, got.Integrity)
	assert.Equal(t, "2026-10-01T12:00:00Z", got.CreatedAt)
}

func TestOutcomeTamperingIsDetected(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"result flipped to passed": func(m map[string]any) { m["result"] = OutcomePassed },
		"package digest swapped":   func(m map[string]any) { m["package_digest"] = "sha256:other" },
		"signals cleared":          func(m map[string]any) { m["signals"] = map[string]any{} },
		"integrity removed":        func(m map[string]any) { delete(m, "integrity") },
		"attempt rewritten":        func(m map[string]any) { m["attempt"] = 7 },
	} {
		store := fixedStore(t)
		_, err := store.Append(sampleOutcome(1, OutcomeFailed))
		require.NoError(t, err)
		path := filepath.Join(store.Root, "missions", "handoff", "m1", attemptFile(1))
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var record map[string]any
		require.NoError(t, json.Unmarshal(raw, &record))
		edit(record)
		edited, err := json.Marshal(record)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, edited, 0o644))

		_, err = store.Latest("m1")

		require.ErrorContains(t, err, "handoff_outcome_tampered", name)
	}
}

func TestOutcomeStoreRejectsInvalidDuplicateAndOutOfSequenceRecords(t *testing.T) {
	store := fixedStore(t)
	for name, outcome := range map[string]Outcome{
		"unknown result": sampleOutcome(1, "approved"),
		"no policy id":   func() Outcome { o := sampleOutcome(1, OutcomePassed); o.PolicyID = ""; return o }(),
		"no gate":        func() Outcome { o := sampleOutcome(1, OutcomePassed); o.GateObserved = ""; return o }(),
		"skipped yet required": func() Outcome {
			o := sampleOutcome(1, OutcomeSkipped)
			o.Required = true
			return o
		}(),
	} {
		_, err := store.Append(outcome)
		require.Error(t, err, name)
	}
	_, err := store.Append(sampleOutcome(2, OutcomePassed))
	require.ErrorContains(t, err, "handoff_outcome_attempt_conflict")
	_, err = store.Append(sampleOutcome(1, OutcomeFailed))
	require.NoError(t, err)
	_, err = store.Append(sampleOutcome(1, OutcomePassed))
	require.ErrorContains(t, err, "handoff_outcome_attempt_conflict", "an attempt can never be replaced")
	_, err = store.Append(sampleOutcome(2, OutcomePassed))
	require.NoError(t, err)
}

func TestOutcomeStoreRejectsMalformedMissionIds(t *testing.T) {
	store := fixedStore(t)
	for _, id := range []string{"", "..", "a/b", `a\b`} {
		o := sampleOutcome(1, OutcomePassed)
		o.MissionID = id
		_, err := store.Append(o)
		require.Error(t, err, id)
	}
}

func TestAuthorizeExecutionAcceptsOnlyPassedOrAuthorizedSkips(t *testing.T) {
	store := fixedStore(t)
	_, err := store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_outcome_missing")

	_, err = store.Append(sampleOutcome(1, OutcomeFailed))
	require.NoError(t, err)
	_, err = store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_outcome_failed")

	_, err = store.Append(sampleOutcome(2, OutcomePassed))
	require.NoError(t, err)
	outcome, err := store.AuthorizeExecution(currentCheck())
	require.NoError(t, err)
	assert.Equal(t, OutcomePassed, outcome.Result)
}

func TestAuthorizeExecutionSkipRequiresTheCurrentFactsToStillAuthorizeIt(t *testing.T) {
	store := fixedStore(t)
	_, err := store.Append(sampleOutcome(1, OutcomeSkipped))
	require.NoError(t, err)
	check := currentCheck()

	_, err = store.AuthorizeExecution(check)
	require.ErrorContains(t, err, "handoff_outcome_skip_not_authorized")

	check.SkipAuthorized = true
	_, err = store.AuthorizeExecution(check)
	require.NoError(t, err)
}

func TestAuthorizeExecutionDeniesStaleCrossMissionPolicyAndReplayedOutcomes(t *testing.T) {
	store := fixedStore(t)
	sealed, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)

	stale := currentCheck()
	stale.PackageDigest = "sha256:amended"
	_, err = store.AuthorizeExecution(stale)
	require.ErrorContains(t, err, "handoff_outcome_stale")

	otherPolicy := currentCheck()
	otherPolicy.PolicyID = "sha256:other-policy"
	_, err = store.AuthorizeExecution(otherPolicy)
	require.ErrorContains(t, err, "handoff_outcome_policy_mismatch")

	// An outcome file copied into another mission's directory is cross-mission.
	other := currentCheck()
	other.MissionID = "m2"
	require.NoError(t, os.MkdirAll(filepath.Join(store.Root, "missions", "handoff", "m2"), 0o755))
	raw, err := os.ReadFile(filepath.Join(store.Root, "missions", "handoff", "m1", attemptFile(1)))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(store.Root, "missions", "handoff", "m2", attemptFile(1)), raw, 0o644))
	_, err = store.AuthorizeExecution(other)
	require.ErrorContains(t, err, "handoff_outcome_cross_mission")

	require.NoError(t, store.Consume(sealed))
	_, err = store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_outcome_replayed")
	require.Error(t, store.Consume(sealed), "consumption is exclusive")
}

func TestOutcomeWithAnUnknownResultNeverAuthorizesEvenWithAValidDigest(t *testing.T) {
	outcome := sampleOutcome(1, "approved")
	outcome.SchemaVersion, outcome.CreatedAt = OutcomeSchemaVersion, "2026-10-01T12:00:00Z"
	digest, err := outcome.digest()
	require.NoError(t, err)
	outcome.Integrity = digest
	store := fixedStore(t)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	raw, err := json.Marshal(outcome)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, attemptFile(1)), raw, 0o644))

	_, err = store.AuthorizeExecution(currentCheck())

	require.ErrorContains(t, err, "handoff_outcome_unknown_result")
}

func TestUnrecognizedOutcomeFilesAreTampering(t *testing.T) {
	store := fixedStore(t)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "attempt-zero.json"), []byte("{}"), 0o644))

	_, err := store.Latest("m1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "handoff_outcome_tampered")
}
