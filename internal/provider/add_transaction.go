package provider

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// prepareAddSource validates the requested source and reloads it for binding,
// so a source that changed between validation and add is rejected.
func prepareAddSource(input, slot string) (Source, Report, error) {
	report, err := Validate(input, slot)
	if err != nil {
		return Source{}, report, err
	}
	if err := ensureBindable(report, slot); err != nil {
		return Source{}, report, err
	}
	source, reasons := loadSource(input)
	if len(reasons) > 0 {
		return Source{}, report, fmt.Errorf("provider add: source changed during validation: %s", reasons[0].Code)
	}
	if err := validateProviderID(source.Package.ID); err != nil {
		return Source{}, report, err
	}
	return source, report, nil
}

// addTxn carries one provider add through stage, activate, persist and
// complete, remembering what a rollback must restore.
type addTxn struct {
	root            string
	report          Report
	source          Source
	instanceID      string
	slot            string
	oldLock         domain.PluginLockFile
	oldLockExists   bool
	oldTransactions transactionFile
	tx              domain.PluginTransaction
	target          string
	created         bool
}

func newAddTxn(root string, source Source, report Report, slot string) (*addTxn, error) {
	oldLock, oldLockExists, err := readLock(root)
	if err != nil {
		return nil, err
	}
	oldTransactions, err := readTransactions(root)
	if err != nil {
		return nil, err
	}
	instanceID := source.Package.ID + "@" + source.Package.Version
	return &addTxn{
		root: root, report: report, source: source, instanceID: instanceID, slot: slot,
		oldLock: oldLock, oldLockExists: oldLockExists, oldTransactions: oldTransactions,
		tx: domain.PluginTransaction{SchemaVersion: transactionSchemaVersion, ID: transactionID(instanceID, slot), State: "staging", ToGeneration: nextGeneration(oldLock, slot)},
	}, nil
}

func (a *addTxn) run() (AddResult, error) {
	if err := appendTransaction(a.root, a.oldTransactions, a.tx); err != nil {
		return AddResult{Report: a.report}, err
	}
	newLock, state, err := a.activate()
	if err != nil {
		return a.rollback(state, err)
	}
	if state, err = a.persist(newLock); err != nil {
		return a.rollback(state, err)
	}
	return a.complete(newLock)
}

// activate stages the source and drives it through the lifecycle. The returned
// state names the failed step for the rollback record.
func (a *addTxn) activate() (domain.PluginLockFile, string, error) {
	var err error
	a.target, a.created, err = stageSource(a.root, a.source, a.instanceID)
	if err != nil {
		return domain.PluginLockFile{}, "staging_failed", err
	}
	newLock := buildLock(a.oldLock, a.source, a.report, a.instanceID, a.slot)
	inventory, bindings, err := activateThroughLifecycle(a.oldLock, newLock, a.instanceID, a.slot)
	if err != nil {
		return newLock, "activation_failed", err
	}
	newLock.Inventory = inventory
	newLock.Bindings = bindings
	return newLock, "", nil
}

func (a *addTxn) persist(newLock domain.PluginLockFile) (string, error) {
	if err := writeLock(a.root, newLock); err != nil {
		return "lock_write_failed", err
	}
	a.tx.State = "activated"
	a.tx.FromGeneration = newLockBindingGeneration(a.oldLock, a.slot)
	if err := compileWorkspace(a.root); err != nil {
		return "compile_failed", err
	}
	return "", nil
}

func (a *addTxn) complete(newLock domain.PluginLockFile) (AddResult, error) {
	a.tx.State = "complete"
	if err := appendTransaction(a.root, a.oldTransactions, a.tx); err != nil {
		return AddResult{}, fmt.Errorf("provider add: record completed transaction: %w", err)
	}
	return AddResult{Report: a.report, InstanceID: a.instanceID, BindingGeneration: bindingGeneration(newLock, a.slot), TransactionState: a.tx.State}, nil
}

func (a *addTxn) rollback(state string, cause error) (AddResult, error) {
	return rollbackCandidate(a.root, a.target, a.created, a.oldLock, a.oldLockExists, a.oldTransactions, a.tx, state, cause)
}
