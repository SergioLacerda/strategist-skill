package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sampleOutcomeDigest is the integrity digest the pre-delegation code produced
// for sampleOutcome(1, passed) at the fixed clock. It must never change: an
// outcome without a delegation is byte-identical to what it was before the field
// existed, so every recorded outcome still verifies (the change is additive).
const sampleOutcomeDigest = "sha256:66e82611a1db81de1237647a4905cd943672868fc3ade14284371dd1683e2a53"

var delegationClock = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func validDelegation() *Delegation {
	return &Delegation{
		Provider: "jev", Model: "jev-1.13.0", BindingDigest: "sha256:binding", Capability: "handoff.validate",
		Criterion: "handoff_acceptable", Subject: "sha256:pkg", Checks: []string{CheckInputOutputConformance},
		Confidence: 0.95, Threshold: 0.90, InputTokens: 392, OutputTokens: 65,
	}
}

func delegatedOutcome(attempt int, result string) Outcome {
	outcome := sampleOutcome(attempt, result)
	outcome.Delegation = validDelegation()
	return outcome
}

func TestOutcomeWithoutDelegationKeepsItsIntegrityDigest(t *testing.T) {
	sealed, err := sampleOutcome(1, OutcomePassed).Seal(delegationClock)
	require.NoError(t, err)
	require.Equal(t, sampleOutcomeDigest, sealed.Integrity)
	raw, err := json.Marshal(sealed)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "delegation", "an absent delegation adds nothing to the record")
}

func TestDelegationVocabularyIsClosed(t *testing.T) {
	require.Equal(t, []string{CheckInputOutputConformance}, DelegableChecks())
	require.ElementsMatch(t,
		[]string{CheckArtifactIntegrity, CheckApprovalGate, CheckAuthorization, CheckPolicyFacts, CheckReceiverComprehension},
		NeverDelegatedChecks())
	for _, never := range NeverDelegatedChecks() {
		require.NotContains(t, DelegableChecks(), never)
	}
}

func TestSealAcceptsAnApprovedDelegationAndCoversItWithIntegrity(t *testing.T) {
	sealed, err := delegatedOutcome(1, OutcomePassed).Seal(delegationClock)
	require.NoError(t, err)
	require.NoError(t, sealed.VerifyIntegrity())
	require.NotEqual(t, sampleOutcomeDigest, sealed.Integrity, "the delegation changes the integrity digest")
	require.True(t, sealed.Delegation.Approved())
}

func TestSealRejectsAnInvalidDelegationFailingClosed(t *testing.T) {
	for name, edit := range map[string]func(*Outcome){
		"no provider":           func(o *Outcome) { o.Delegation.Provider = "" },
		"no model":              func(o *Outcome) { o.Delegation.Model = "" },
		"no binding digest":     func(o *Outcome) { o.Delegation.BindingDigest = "" },
		"no capability":         func(o *Outcome) { o.Delegation.Capability = "" },
		"no criterion":          func(o *Outcome) { o.Delegation.Criterion = "" },
		"no subject":            func(o *Outcome) { o.Delegation.Subject = "" },
		"another revision":      func(o *Outcome) { o.Delegation.Subject = "sha256:other" },
		"no checks":             func(o *Outcome) { o.Delegation.Checks = nil },
		"confidence above one":  func(o *Outcome) { o.Delegation.Confidence = 1.01 },
		"negative confidence":   func(o *Outcome) { o.Delegation.Confidence = -0.1 },
		"threshold zero":        func(o *Outcome) { o.Delegation.Threshold = 0 },
		"threshold above one":   func(o *Outcome) { o.Delegation.Threshold = 1.5 },
		"exactly the threshold": func(o *Outcome) { o.Delegation.Confidence, o.Delegation.Threshold = 0.90, 0.90 },
		"below the threshold":   func(o *Outcome) { o.Delegation.Confidence = 0.40 },
		"negative usage":        func(o *Outcome) { o.Delegation.InputTokens = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			outcome := delegatedOutcome(1, OutcomePassed)
			edit(&outcome)
			_, err := outcome.Seal(delegationClock)
			require.ErrorContains(t, err, "handoff_delegation_invalid")
		})
	}
}

func TestSealRejectsEveryCheckThatMayNotBeDelegated(t *testing.T) {
	forbidden := append(NeverDelegatedChecks(), "something_unknown")
	for _, check := range forbidden {
		t.Run(check, func(t *testing.T) {
			outcome := delegatedOutcome(1, OutcomePassed)
			outcome.Delegation.Checks = []string{CheckInputOutputConformance, check}
			_, err := outcome.Seal(delegationClock)
			require.ErrorContains(t, err, "handoff_delegation_forbidden_check")
		})
	}
}

func TestDelegationTamperingIsDetected(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"confidence raised":  func(m map[string]any) { m["delegation"].(map[string]any)["confidence"] = 0.99 },
		"model swapped":      func(m map[string]any) { m["delegation"].(map[string]any)["model"] = "jev-9" },
		"checks widened":     func(m map[string]any) { m["delegation"].(map[string]any)["checks"] = []any{"approval_gate"} },
		"delegation removed": func(m map[string]any) { delete(m, "delegation") },
	} {
		t.Run(name, func(t *testing.T) {
			store := fixedStore(t)
			_, err := store.Append(delegatedOutcome(1, OutcomePassed))
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
			require.ErrorContains(t, err, "handoff_outcome_tampered")
		})
	}
}

func TestADelegatedOutcomeIsSingleUse(t *testing.T) {
	store := fixedStore(t)
	sealed, err := store.Append(delegatedOutcome(1, OutcomePassed))
	require.NoError(t, err)

	authorized, err := store.AuthorizeExecution(currentCheck())
	require.NoError(t, err)
	require.Equal(t, sealed.Integrity, authorized.Integrity)
	require.NoError(t, store.Consume(authorized))

	_, err = store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_outcome_replayed")
}

func TestAuthorizeDeniesADelegatedOutcomeThatBypassedSealing(t *testing.T) {
	store := fixedStore(t)
	forged := delegatedOutcome(1, OutcomePassed)
	forged.Delegation.Checks = []string{CheckApprovalGate}
	forged.SchemaVersion, forged.CreatedAt = OutcomeSchemaVersion, delegationClock.Format(time.RFC3339Nano)
	digest, err := forged.digest()
	require.NoError(t, err)
	forged.Integrity = digest
	raw, err := json.Marshal(forged)
	require.NoError(t, err)
	dir := filepath.Join(store.Root, "missions", "handoff", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, attemptFile(1)), append(raw, '\n'), 0o644))

	_, err = store.AuthorizeExecution(currentCheck())
	require.ErrorContains(t, err, "handoff_delegation_forbidden_check", "a record with a valid digest but a forbidden delegated check never authorizes execution")
}

func TestHandoffContractNamesTheSameDelegationVocabularyAsTheCode(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "contracts", "machine", "handoff-contract.yaml"))
	require.NoError(t, err)
	contract := string(raw)
	require.Contains(t, contract, "delegated_validation:")
	for _, check := range append(DelegableChecks(), NeverDelegatedChecks()...) {
		require.Contains(t, contract, check, "contract must name the check")
	}
}

func TestASkippedOutcomeMayCarryTheConformanceDelegation(t *testing.T) {
	outcome := delegatedOutcome(1, OutcomeSkipped)
	outcome.Required = false
	sealed, err := outcome.Seal(delegationClock)
	require.NoError(t, err)
	require.NoError(t, sealed.VerifyIntegrity())
}

func TestRangerOutcomeDelegationIsBoundToTheArtifactDigest(t *testing.T) {
	outcome := Outcome{
		MissionID: "m1", Transition: TransitionRangerToArchivist, ArtifactDigest: "sha256:artifact", Attempt: 1, Result: OutcomePassed,
		PolicyID: "p", Required: true, GateObserved: "ranger_normalized",
	}
	bound := validDelegation()
	bound.Subject = "sha256:artifact"
	outcome.Delegation = bound
	_, err := outcome.Seal(delegationClock)
	require.NoError(t, err)

	outcome.Delegation = validDelegation() // subject is the package digest of another transition
	_, err = outcome.Seal(delegationClock)
	require.ErrorContains(t, err, "handoff_delegation_invalid")
}

func TestWithDelegationDropsAnUnfitDelegationInsteadOfFailing(t *testing.T) {
	base := sampleOutcome(1, OutcomePassed)

	attached := base.WithDelegation(validDelegation())
	require.NotNil(t, attached.Delegation)

	wrongRevision := validDelegation()
	wrongRevision.Subject = "sha256:stale"
	require.Nil(t, base.WithDelegation(wrongRevision).Delegation, "a stale delegation is dropped: main serves the handoff")
	require.Nil(t, base.WithDelegation(nil).Delegation)
	require.Nil(t, base.Delegation, "the original outcome is not mutated")
}
