package lifecycle_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	instanceActive    = "inst-active"
	instanceCandidate = "inst-candidate"
)

func TestLifecycleStagesProbesAndActivatesWithCASGeneration(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{
		SchemaVersion:       "strategist-plugin-binding/v1",
		Slot:                "refinement",
		InstalledInstanceID: instanceActive,
		Generation:          7,
		Status:              "enabled",
	}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))
	require.NoError(t, store.Activate(tx.ID, 7))

	binding, ok := store.Binding("refinement")
	require.True(t, ok)
	assert.Equal(t, instanceCandidate, binding.InstalledInstanceID)
	assert.Equal(t, int64(8), binding.Generation)

	active, _ := store.Instance(instanceCandidate)
	old, _ := store.Instance(instanceActive)
	assert.Equal(t, lifecycle.StateActive, active.State)
	assert.True(t, active.LastKnownGood)
	assert.False(t, old.LastKnownGood)
	assert.Equal(t, lifecycle.StateComplete, store.Transaction(tx.ID).State)
	assert.Equal(t, []string{
		lifecycle.StateResolved,
		lifecycle.StateStaged,
		lifecycle.StateProbed,
		lifecycle.StateActive,
		lifecycle.StateComplete,
	}, store.Transaction(tx.ID).JournalStates())
}

func TestLifecycleRejectsActivateBeforeProbe(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	err = store.Activate(tx.ID, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")
}

func TestLifecycleRejectsCASGenerationConflict(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 3, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))

	err = store.Activate(tx.ID, 2)
	require.Error(t, err)
	require.ErrorContains(t, err, "binding_generation_conflict")
}

func TestLifecycleRollsBackFailedProbeToLastKnownGood(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 5, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, false))

	err = store.Activate(tx.ID, 5)
	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")

	require.NoError(t, store.Rollback(tx.ID))
	binding, _ := store.Binding("refinement")
	candidate, _ := store.Instance(instanceCandidate)
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
	assert.Equal(t, lifecycle.StateFailed, candidate.State)
	assert.Equal(t, lifecycle.StateRolledBack, store.Transaction(tx.ID).State)
}

func TestLifecycleRecoveryRollsBackIncompleteTransaction(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateStaged},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 4, Status: "enabled"}}
	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	require.NoError(t, store.Recover())

	binding, _ := store.Binding("refinement")
	candidate, _ := store.Instance(instanceCandidate)
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
	assert.Equal(t, lifecycle.StateFailed, candidate.State)
	assert.Equal(t, lifecycle.StateRolledBack, store.Transaction(tx.ID).State)
}

func TestLifecycleRollbackUsesLastKnownGoodWhenRecordedTargetMissing(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateStaged},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "missing-old", Generation: 2, Status: "enabled"}}
	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	require.NoError(t, store.Rollback(tx.ID))

	binding, _ := store.Binding("refinement")
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
}

func TestLifecycleUninstallBlockedWhenBoundOrReferenced(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateResolved},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}
	store.Dependents[instanceCandidate] = []string{"dep-1"}

	err := store.Uninstall(instanceActive)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_bound_instance")

	err = store.Uninstall(instanceCandidate)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_required_dependent")
}

func TestLifecycleQuarantineDisablesBoundInstance(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 3, Status: "enabled"}}

	require.NoError(t, store.Quarantine(instanceActive, "malicious_adapter_fixture"))

	instance, _ := store.Instance(instanceActive)
	binding, _ := store.Binding("refinement")
	assert.Equal(t, lifecycle.StateQuarantined, instance.State)
	assert.False(t, instance.LastKnownGood)
	assert.Equal(t, "quarantined", binding.Status)
	assert.Equal(t, int64(4), binding.Generation)

	err := store.Uninstall(instanceActive)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_bound_instance")
}

func TestLifecycleBeginResumesExistingTransaction(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	first, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(first.ID))

	resumed, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	assert.Equal(t, lifecycle.StateStaged, resumed.State, "Begin on an existing id resumes it as-is, not a fresh Resolved transaction")
}

func TestLifecycleBeginRejectsMissingCandidateOrBinding(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	_, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.Error(t, err)
	require.ErrorContains(t, err, "candidate_instance_missing")

	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	_, err = store.Begin("tx-2", "refinement", instanceCandidate)
	require.Error(t, err)
	require.ErrorContains(t, err, "binding_missing")
}

func TestLifecycleStageIsIdempotentWhenAlreadyPastResolved(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Stage(tx.ID), "Stage is idempotent once already Staged")
}

func TestLifecycleStageRejectsInvalidState(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, false)) // drives the transaction to StateFailed

	err = store.Stage(tx.ID)
	require.Error(t, err)
	require.ErrorContains(t, err, "stage_invalid_state")
}

func TestLifecycleProbeIsIdempotentWithSameResult(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))
	require.NoError(t, store.Probe(tx.ID, true), "Probe is idempotent when repeated with the same result")
}

func TestLifecycleProbeRejectsInvalidState(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)

	err = store.Probe(tx.ID, true)
	require.Error(t, err)
	require.ErrorContains(t, err, "probe_invalid_state")
}

func TestLifecycleActivateIsIdempotentOnceComplete(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))
	require.NoError(t, store.Activate(tx.ID, 1))

	require.NoError(t, store.Activate(tx.ID, 1), "Activate is idempotent once the transaction is Complete")
}

func TestLifecycleRollbackIsIdempotentOnceComplete(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))
	require.NoError(t, store.Activate(tx.ID, 1))

	require.NoError(t, store.Rollback(tx.ID), "Rollback on a Complete transaction is a no-op, not an error")
	assert.Equal(t, lifecycle.StateComplete, store.Transaction(tx.ID).State, "an already-Complete transaction is left untouched by Rollback")
}

func TestLifecycleRecoverySkipsAlreadyCompleteTransaction(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "inst-complete", State: lifecycle.StateVerified},
		{ID: instanceCandidate, State: lifecycle.StateStaged},
	}
	store.Bindings = []domain.SlotBinding{
		{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"},
		{Slot: "execution", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"},
	}

	completed, err := store.Begin("tx-complete", "refinement", "inst-complete")
	require.NoError(t, err)
	require.NoError(t, store.Stage(completed.ID))
	require.NoError(t, store.Probe(completed.ID, true))
	require.NoError(t, store.Activate(completed.ID, 1))

	incomplete, err := store.Begin("tx-incomplete", "execution", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(incomplete.ID))

	require.NoError(t, store.Recover())

	assert.Equal(t, lifecycle.StateComplete, store.Transaction(completed.ID).State, "Recover must not touch an already-Complete transaction")
	assert.Equal(t, lifecycle.StateRolledBack, store.Transaction(incomplete.ID).State)
}

func TestLifecycleUninstallRemovesUnboundUnreferencedInstance(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateResolved}}

	require.NoError(t, store.Uninstall(instanceCandidate))

	_, ok := store.Instance(instanceCandidate)
	assert.False(t, ok, "Uninstall removes the instance from inventory")
}

func TestLifecycleUninstallMissingInstance(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	err := store.Uninstall("does-not-exist")
	require.Error(t, err)
	require.ErrorContains(t, err, "instance_missing")
}

func TestLifecycleQuarantineMissingInstance(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	err := store.Quarantine("does-not-exist", "reason")
	require.Error(t, err)
	require.ErrorContains(t, err, "instance_missing")
}

func TestLifecycleTransactionReturnsZeroValueForUnknownID(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	assert.Equal(t, lifecycle.Transaction{}, store.Transaction("does-not-exist"))
}

func TestLifecycleBindingReturnsFalseForUnknownSlot(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	_, ok := store.Binding("does-not-exist")
	assert.False(t, ok)
}

func TestLifecycleOperationsRejectUnknownTransactionID(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()

	require.ErrorContains(t, store.Stage("does-not-exist"), "transaction_missing")
	require.ErrorContains(t, store.Probe("does-not-exist", true), "transaction_missing")
	require.ErrorContains(t, store.Activate("does-not-exist", 1), "transaction_missing")
	require.ErrorContains(t, store.Rollback("does-not-exist"), "transaction_missing")
}

func TestLifecycleStageRejectsCandidateRemovedAfterBegin(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)

	// The candidate instance disappears from inventory between Begin and
	// Stage — exercises transitionCandidate's own candidate_instance_missing
	// check, distinct from the one Begin already performs.
	store.Inventory.Instances = nil

	err = store.Stage(tx.ID)
	require.Error(t, err)
	require.ErrorContains(t, err, "candidate_instance_missing")
}

func TestLifecycleRollbackWithNoLastKnownGoodLeavesBindingUnset(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateStaged}}
	// InstalledInstanceID intentionally references an instance that does not
	// exist, and no instance in inventory has LastKnownGood set — Rollback
	// falls through lastKnownGoodInstanceID() to its empty-string "none
	// found" branch, and rollbackTarget stays "".
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "missing-old", Generation: 2, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	require.NoError(t, store.Rollback(tx.ID))

	binding, _ := store.Binding("refinement")
	assert.Equal(t, "missing-old", binding.InstalledInstanceID, "no rollback target was applied, binding is left as-is")
}

func TestLifecycleActivateRejectsBindingRemovedAfterBegin(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))

	// The binding disappears between Begin and Activate — exercises
	// Activate's own binding_missing check, distinct from Begin's.
	store.Bindings = nil

	err = store.Activate(tx.ID, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "binding_missing")
}

func TestLifecycleDeprecateMarksBoundInstanceWithoutDeletingIt(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 9, Status: "enabled"}}

	require.NoError(t, store.Deprecate(instanceActive, "upstream_eol"))

	instance, _ := store.Instance(instanceActive)
	binding, _ := store.Binding("refinement")
	assert.Equal(t, lifecycle.StateDeprecated, instance.State)
	assert.False(t, instance.LastKnownGood)
	assert.Equal(t, "deprecated", binding.Status)
	assert.Equal(t, int64(10), binding.Generation)
	assert.Equal(t, "upstream_eol", instance.VerificationEvidence)
}
