package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
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
	return binding.Generation, samePackage && sameAdapter
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

func buildLock(old domain.PluginLockFile, source Source, report Report, instanceID, slot string) domain.PluginLockFile {
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
	lock.Bindings = replaceBinding(lock.Bindings, domain.SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: slot, InstalledInstanceID: instanceID,
		Generation: nextGeneration(old, slot), Status: "active", Mode: domain.SlotBindingModeCustom,
	})
	lock.Lock = replaceLockNodes(lock.Lock, source.Package.ID, source.Package.Digest, report.AdapterDigest)
	lock.Lock.SchemaVersion = "strategist-plugin-lock/v1"
	lock.Lock.GraphDigest = plugins.DigestLockNodes(lock.Lock.Nodes)
	lock.Lock.ResolutionID = lock.Lock.GraphDigest
	return lock
}

func existingBinding(bindings []domain.SlotBinding, slot string) (domain.SlotBinding, bool) {
	for _, binding := range bindings {
		if binding.Slot == slot {
			return binding, true
		}
	}
	return domain.SlotBinding{}, false
}

func replaceInstance(instances []domain.InstalledInstance, candidate domain.InstalledInstance) []domain.InstalledInstance {
	for i, instance := range instances {
		if instance.ID == candidate.ID {
			instances[i] = candidate
			return instances
		}
	}
	return append(instances, candidate)
}

func replaceBinding(bindings []domain.SlotBinding, candidate domain.SlotBinding) []domain.SlotBinding {
	for i, binding := range bindings {
		if binding.Slot == candidate.Slot {
			candidate.Generation = binding.Generation + 1
			bindings[i] = candidate
			return bindings
		}
	}
	return append(bindings, candidate)
}

func replaceLockNodes(lock domain.PluginLock, id, packageDigest, adapterDigest string) domain.PluginLock {
	filtered := make([]domain.PluginLockNode, 0, len(lock.Nodes)+2)
	for _, node := range lock.Nodes {
		if (node.ID == id && node.Kind == string(domain.PluginResourcePackage)) || (node.ID == id && node.Kind == string(domain.PluginResourceAdapter)) {
			continue
		}
		filtered = append(filtered, node)
	}
	filtered = append(filtered,
		domain.PluginLockNode{ID: id, Kind: string(domain.PluginResourcePackage), Digest: packageDigest},
		domain.PluginLockNode{ID: id, Kind: string(domain.PluginResourceAdapter), Digest: adapterDigest})
	lock.Nodes = filtered
	return lock
}

func nextGeneration(lock domain.PluginLockFile, slot string) int64 {
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			return binding.Generation + 1
		}
	}
	return 1
}
