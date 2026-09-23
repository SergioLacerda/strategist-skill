package initiative

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// directoryLedger returns a Runtime whose ledger path is a directory: the lock
// file can still be created next to it, but every read of the ledger fails, so
// each *Locked method reaches its LatestAdvice / ReadRecords error branch.
func directoryLedger(t *testing.T) Runtime {
	t.Helper()
	ledger := filepath.Join(t.TempDir(), "ledger-dir")
	require.NoError(t, os.MkdirAll(ledger, 0o750))
	return Runtime{Advisor: Advisor{Policy: DefaultPolicy()}, LedgerFile: ledger}
}

func newTestRuntime(t *testing.T) Runtime {
	t.Helper()
	runtime, err := NewRuntime(t.TempDir(), DefaultPolicy())
	require.NoError(t, err)
	return runtime
}

func initialInput(role string) AdviceInput {
	return AdviceInput{MissionID: "m", Role: role, RunID: "r", Trigger: TriggerInitial}
}

func reevalInput(role string) AdviceInput {
	return AdviceInput{MissionID: "m", Role: role, RunID: "r", Trigger: TriggerHandoffChallenged}
}

func TestNewRuntimeRejectsAnInvalidPolicy(t *testing.T) {
	_, err := NewRuntime(t.TempDir(), Policy{})

	require.Error(t, err)
}

func TestRuntimeRequiresAValidPolicyAndALedgerPath(t *testing.T) {
	noLedger := Runtime{Advisor: Advisor{Policy: DefaultPolicy()}}
	noPolicy := Runtime{LedgerFile: filepath.Join(t.TempDir(), "l.jsonl")}

	for _, runtime := range []Runtime{noLedger, noPolicy} {
		_, _, enterErr := runtime.EnterRole(initialInput("ranger"))
		_, reevalErr := runtime.Reevaluate(reevalInput("ranger"))
		_, resultErr := runtime.RecordResult(validAdvice(), Result{})

		require.Error(t, enterErr)
		require.Error(t, reevalErr)
		require.Error(t, resultErr)
	}
	_, _, err := noLedger.EnterRole(initialInput("ranger"))
	require.ErrorContains(t, err, "ledger path is required")
}

func TestEnterRoleRequiresTheInitialTrigger(t *testing.T) {
	_, _, err := newTestRuntime(t).EnterRole(reevalInput("ranger"))

	require.ErrorContains(t, err, "role entry requires initial trigger")
}

func TestEnterRolePropagatesAdviseAndLedgerErrors(t *testing.T) {
	_, _, adviseErr := newTestRuntime(t).EnterRole(initialInput("not-a-role"))
	require.ErrorContains(t, adviseErr, "is not in policy")

	_, _, readErr := directoryLedger(t).EnterRole(initialInput("ranger"))
	require.ErrorContains(t, readErr, "read ledger")
}

func TestEnterRoleFailsWhenTheLedgerDirectoryCannotBeCreated(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	runtime := Runtime{Advisor: Advisor{Policy: DefaultPolicy()}, LedgerFile: filepath.Join(blocker, "sub", "l.jsonl")}

	_, _, err := runtime.EnterRole(initialInput("ranger"))

	require.ErrorContains(t, err, "create ledger directory")
}

func TestEnterRoleFailsWhenTheLockFileCannotBeOpened(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "l.jsonl")
	require.NoError(t, os.MkdirAll(ledger+".lock", 0o750))
	runtime := Runtime{Advisor: Advisor{Policy: DefaultPolicy()}, LedgerFile: ledger}

	_, _, err := runtime.EnterRole(initialInput("ranger"))

	require.ErrorContains(t, err, "open ledger lock")
}

func TestEnterRoleFailsWhenTheLedgerIsNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	ledger := filepath.Join(t.TempDir(), "l.jsonl")
	require.NoError(t, os.WriteFile(ledger, nil, 0o400))
	runtime := Runtime{Advisor: Advisor{Policy: DefaultPolicy()}, LedgerFile: ledger}

	_, _, err := runtime.EnterRole(initialInput("ranger"))

	require.ErrorContains(t, err, "open ledger")
}

func TestReevaluateRejectsInitialAndDisabledTriggers(t *testing.T) {
	_, initialErr := newTestRuntime(t).Reevaluate(initialInput("ranger"))
	require.ErrorContains(t, initialErr, "requires a non-initial trigger")

	policy := DefaultPolicy()
	policy.Triggers = []Trigger{TriggerScopeChanged}
	runtime, err := NewRuntime(t.TempDir(), policy)
	require.NoError(t, err)
	_, disabledErr := runtime.Reevaluate(reevalInput("ranger"))
	require.ErrorContains(t, disabledErr, "is not enabled by policy")
}

func TestReevaluateRequiresPriorAdvice(t *testing.T) {
	_, err := newTestRuntime(t).Reevaluate(reevalInput("ranger"))

	require.ErrorContains(t, err, "cannot re-evaluate without prior advice")
}

func TestReevaluatePropagatesLedgerReadErrors(t *testing.T) {
	_, err := directoryLedger(t).Reevaluate(reevalInput("ranger"))

	require.ErrorContains(t, err, "read ledger")
}

func TestRecordResultRequiresPersistedAdvice(t *testing.T) {
	runtime := newTestRuntime(t)
	advice, err := runtime.Advisor.Advise(initialInput("ranger"))
	require.NoError(t, err)

	_, err = runtime.RecordResult(advice, validResultForAdvice(advice))

	require.ErrorContains(t, err, "cannot record result without persisted advice")
}

func TestRecordResultPropagatesLedgerReadErrors(t *testing.T) {
	_, err := directoryLedger(t).RecordResult(validAdvice(), Result{})

	require.ErrorContains(t, err, "read ledger")
}

func TestRecordResultRejectsAResultThatDoesNotMatchTheAdvice(t *testing.T) {
	runtime := newTestRuntime(t)
	advice, _, err := runtime.EnterRole(initialInput("ranger"))
	require.NoError(t, err)
	result := validResultForAdvice(advice)
	result.MissionID = "another-mission"

	_, err = runtime.RecordResult(advice, result)

	require.Error(t, err)
}

func TestRecordResultRejectsAdviceFromAnotherPolicy(t *testing.T) {
	runtime := newTestRuntime(t)
	advice, _, err := runtime.EnterRole(initialInput("ranger"))
	require.NoError(t, err)
	advice.PolicyVersion = "99"

	_, err = runtime.RecordResult(advice, validResultForAdvice(advice))

	require.ErrorContains(t, err, "advice policy identity does not match active policy")
}

func TestValidatePersistedAdviceRejectsAdviceThatDiffersFromTheLedger(t *testing.T) {
	policy := DefaultPolicy()
	candidate := validAdvice()
	candidate.PolicyVersion, candidate.PolicyDigest = policy.Version, policy.Digest()
	persisted := candidate
	persisted.AdviceID = "another"

	require.NoError(t, validatePersistedAdvice(candidate, candidate, policy))
	require.ErrorContains(t, validatePersistedAdvice(candidate, persisted, policy), "does not match persisted latest advice")
}

func TestAdviseRejectsATriggerThePolicyDoesNotEnable(t *testing.T) {
	policy := DefaultPolicy()
	policy.Triggers = []Trigger{TriggerScopeChanged}

	_, err := Advisor{Policy: policy}.Advise(AdviceInput{
		MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerHandoffChallenged, Supersedes: "prior",
	})

	require.ErrorContains(t, err, "is not enabled by policy")
}

func TestValidateAdviceIdentityRejectsAnUnknownTrigger(t *testing.T) {
	err := validateAdviceIdentity(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: Trigger("bogus")}, "ranger")

	require.ErrorContains(t, err, "unknown trigger")
}

func TestValidCheckStatusOnlyAcceptsTheDeclaredStatuses(t *testing.T) {
	for _, status := range []CheckStatus{CheckSatisfied, CheckPartial, CheckBlocked, CheckNotApplicable} {
		require.True(t, validCheckStatus(status), status)
	}
	require.False(t, validCheckStatus(CheckStatus("bogus")))
}

func TestAdviceSequencePropagatesLedgerReadErrors(t *testing.T) {
	_, err := adviceSequence(directoryLedger(t).LedgerFile, "m", "ranger", "r")

	require.Error(t, err)
}

func TestWriteLedgerLineReportsAFailedWrite(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is not available")
	}

	err := writeLedgerLine("/dev/full", []byte("{}"))

	require.ErrorContains(t, err, "write ledger")
}

func TestCloseLedgerReportsACloseFailureWithoutMaskingAnEarlierError(t *testing.T) {
	closed, err := os.CreateTemp(t.TempDir(), "ledger-*")
	require.NoError(t, err)
	require.NoError(t, closed.Close())

	var fresh error
	closeLedger(closed, &fresh)
	require.ErrorContains(t, fresh, "close ledger")

	earlier := os.ErrInvalid
	keep := earlier
	closeLedger(closed, &keep)
	require.Same(t, earlier, keep)
}

func TestLockHelpersReportFailuresOnAClosedFile(t *testing.T) {
	closed, err := os.CreateTemp(t.TempDir(), "lock-*")
	require.NoError(t, err)
	require.NoError(t, closed.Close())

	require.ErrorContains(t, lockLedgerFile(closed), "lock ledger")
	require.ErrorContains(t, unlockLedgerFile(closed), "unlock ledger")
	require.Error(t, closeLedgerFile(closed, true))
}
