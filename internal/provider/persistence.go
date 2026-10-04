package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
)

// AddResult describes a committed local onboarding transaction.
type AddResult struct {
	Report            Report `json:"report" yaml:"report"`
	InstanceID        string `json:"instance_id" yaml:"instance_id"`
	BindingGeneration int64  `json:"binding_generation" yaml:"binding_generation"`
	TransactionState  string `json:"transaction_state" yaml:"transaction_state"`
}

type transactionFile struct {
	SchemaVersion string                     `yaml:"schema_version"`
	Transactions  []domain.PluginTransaction `yaml:"transactions"`
}

const transactionSchemaVersion = "strategist-plugin-transactions/v1"

// Add validates, stages, binds, and compiles a local provider. Existing
// plugins.lock bytes are restored if materialization or compilation fails.
func Add(strategistRoot, input, slot string) (AddResult, error) {
	if strategistRoot == "" {
		return AddResult{}, fmt.Errorf("provider add: strategist root is required")
	}
	source, report, err := prepareAddSource(input, slot)
	if err != nil {
		return AddResult{Report: report}, err
	}
	txn, err := newAddTxn(strategistRoot, source, report, slot)
	if err != nil {
		return AddResult{Report: report}, err
	}
	if generation, bound := alreadyBound(txn.oldLock, source, report, txn.instanceID, slot); bound {
		return AddResult{Report: report, InstanceID: txn.instanceID, BindingGeneration: generation, TransactionState: "complete"}, nil
	}
	return txn.run()
}

// alreadyBound reports whether the slot is already bound to exactly this
// package and adapter, so adding it again is a no-op; it returns the binding's
// generation.
func alreadyBound(lock domain.PluginLockFile, source Source, report Report, instanceID, slot string) (int64, bool) {
	binding, ok := existingBinding(lock.Bindings, slot)
	if !ok || binding.InstalledInstanceID != instanceID {
		return 0, false
	}
	samePackage := lock.NodeDigest(source.Package.ID, string(domain.PluginResourcePackage)) == source.Package.Digest
	sameAdapter := lock.NodeDigest(source.Package.ID, string(domain.PluginResourceAdapter)) == report.AdapterDigest
	// A binding that predates the complete Custom contract is not "already
	// bound": adding again is how an operator replaces it.
	role, _ := domain.DefaultRoleRegistry().RoleForSlot(slot)
	complete := domain.ValidateCustomBinding(lock, binding, role.ID, slot) == nil
	return binding.Generation, samePackage && sameAdapter && complete
}

// removeStage deletes a failed staging directory and returns cause, joined with
// the cleanup failure when the directory could not be removed.
func removeStage(stage string, cause error) error {
	if rmErr := os.RemoveAll(stage); rmErr != nil { //nolint:gosec // stage is derived below the governed root
		return errors.Join(cause, fmt.Errorf("remove staging directory %s: %w", stage, rmErr))
	}
	return cause
}

func stageSource(root string, source Source, instanceID string) (string, bool, error) {
	base := filepath.Join(root, providerDirName)
	stage := filepath.Join(base, ".staging", instanceID+"-"+fmt.Sprint(time.Now().UnixNano()))
	target := filepath.Join(base, instanceID)
	if _, err := os.Stat(target); err == nil {
		return target, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("stat provider target: %w", err)
	}
	if err := copySource(source.Dir, stage); err != nil {
		return "", false, removeStage(stage, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", false, removeStage(stage, fmt.Errorf("create provider directory: %w", err))
	}
	if err := os.Rename(stage, target); err != nil {
		return "", false, removeStage(stage, fmt.Errorf("commit provider materialization: %w", err))
	}
	return target, true, nil
}

// customFacts derives the normalized facts a complete Custom binding needs
// from the validated source, the canonical slot-to-role map and the staged
// local_path connector. Nothing is inferred beyond that: a source with no
// declared entrypoint cannot be bound.
func customFacts(source Source, report Report, slot string) (domain.CustomPackageFacts, error) {
	role, ok := domain.DefaultRoleRegistry().RoleForSlot(slot)
	if !ok {
		return domain.CustomPackageFacts{}, fmt.Errorf("provider add: slot %q has no owning Role", slot)
	}
	entrypoint := ""
	if len(source.Adapter.Entrypoints) > 0 {
		entrypoint = source.Adapter.Entrypoints[0]
	}
	return domain.CustomPackageFacts{
		PackageID: source.Package.ID, PackageVersion: source.Package.Version, Role: role.ID, Slot: slot,
		PackageDigest: source.Package.Digest, AdapterDigest: report.AdapterDigest,
		RuntimeKind: domain.RankedRuntimeHost, ConnectorID: "local_path", Entrypoint: entrypoint,
	}, nil
}

func buildLock(old domain.PluginLockFile, source Source, report Report, instanceID, slot string) (domain.PluginLockFile, error) {
	facts, err := customFacts(source, report, slot)
	if err != nil {
		return domain.PluginLockFile{}, err
	}
	evidence, err := domain.NewCustomBindingEvidence(facts, nextGeneration(old, slot), "active")
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("provider add: %w", err)
	}
	lock := old
	lock.Inventory.Instances = append([]domain.InstalledInstance(nil), old.Inventory.Instances...)
	lock.Bindings = append([]domain.SlotBinding(nil), old.Bindings...)
	lock.Lock.Nodes = append([]domain.PluginLockNode(nil), old.Lock.Nodes...)
	lock.SchemaVersion = domain.PluginLockFileSchemaVersion
	if lock.Inventory.SchemaVersion == "" {
		lock.Inventory.SchemaVersion = "strategist-plugin-inventory/v1"
	}
	lock.Inventory.Instances = replaceInstance(lock.Inventory.Instances, domain.InstalledInstance{
		ID: instanceID, PackageDigest: source.Package.Digest, AdapterDigest: report.AdapterDigest,
		ConnectorID: "local_path", VerificationEvidence: "static_contract_verified", State: "active", LastKnownGood: true,
	})
	for i := range lock.Inventory.Instances {
		lock.Inventory.Instances[i].LastKnownGood = lock.Inventory.Instances[i].ID == instanceID
	}
	lock.Bindings = replaceBinding(lock.Bindings, evidence.Binding)
	lock.Lock = replaceLockNodes(lock.Lock, evidence.Nodes)
	lock.Lock.SchemaVersion = "strategist-plugin-lock/v1"
	lock.Lock.GraphDigest = resolver.DigestLockNodes(lock.Lock.Nodes)
	lock.Lock.ResolutionID = lock.Lock.GraphDigest
	return lock, nil
}

func nextGeneration(lock domain.PluginLockFile, slot string) int64 {
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			return binding.Generation + 1
		}
	}
	return 1
}
