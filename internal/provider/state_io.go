package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func readLock(root string) (domain.PluginLockFile, bool, error) {
	path := filepath.Join(root, "plugins.lock")
	raw, err := os.ReadFile(path) //nolint:gosec // fixed file below the governed root
	if errors.Is(err, os.ErrNotExist) {
		return domain.PluginLockFile{SchemaVersion: domain.PluginLockFileSchemaVersion}, false, nil
	}
	if err != nil {
		return domain.PluginLockFile{}, false, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return domain.PluginLockFile{}, true, fmt.Errorf("parse plugins.lock: %w", err)
	}
	return lock, true, nil
}

func writeLock(root string, lock domain.PluginLockFile) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("marshal plugins.lock: %w", err)
	}
	return atomicWrite(filepath.Join(root, "plugins.lock"), data)
}

func readTransactions(root string) (transactionFile, error) {
	path := filepath.Join(root, transactionFileName)
	raw, err := os.ReadFile(path) //nolint:gosec // fixed file below the governed root
	if errors.Is(err, os.ErrNotExist) {
		return transactionFile{SchemaVersion: transactionSchemaVersion}, nil
	}
	if err != nil {
		return transactionFile{}, fmt.Errorf("read %s: %w", transactionFileName, err)
	}
	var file transactionFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return transactionFile{}, fmt.Errorf("parse %s: %w", transactionFileName, err)
	}
	file.SchemaVersion = transactionSchemaVersion
	return file, nil
}

func appendTransaction(root string, file transactionFile, tx domain.PluginTransaction) error {
	file.SchemaVersion = transactionSchemaVersion
	replaced := false
	for i := range file.Transactions {
		if file.Transactions[i].ID == tx.ID {
			file.Transactions[i], replaced = tx, true
			break
		}
	}
	if !replaced {
		file.Transactions = append(file.Transactions, tx)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", transactionFileName, err)
	}
	return atomicWrite(filepath.Join(root, transactionFileName), data)
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".provider-state-")
	if err != nil {
		return fmt.Errorf("create state temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup after rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck,gosec // preserve the original write error
		return fmt.Errorf("write state temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close state temporary file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("chmod state temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("commit state file: %w", err)
	}
	return nil
}

func rollbackAdd(root string, oldLock domain.PluginLockFile, oldLockExists bool, oldTransactions transactionFile, tx domain.PluginTransaction, state string, cause error) (AddResult, error) {
	return rollbackCandidate(root, "", false, oldLock, oldLockExists, oldTransactions, tx, state, cause)
}

func rollbackCandidate(root, target string, created bool, oldLock domain.PluginLockFile, oldLockExists bool, oldTransactions transactionFile, tx domain.PluginTransaction, state string, cause error) (AddResult, error) {
	cleanupErr := removeCandidate(target, created)
	if err := restoreLock(root, oldLock, oldLockExists); err != nil {
		return AddResult{}, fmt.Errorf("%s; restore plugins.lock: %w", cause, err)
	}
	tx.State = "rolled_back"
	if err := appendTransaction(root, oldTransactions, tx); err != nil {
		return AddResult{}, fmt.Errorf("%s; record rollback: %w", cause, err)
	}
	return AddResult{TransactionState: tx.State}, errors.Join(fmt.Errorf("provider add %s: %w", state, cause), cleanupErr)
}

// removeCandidate deletes a materialization this transaction created; a failed
// removal is returned so the rollback report can include it.
func removeCandidate(target string, created bool) error {
	if !created || target == "" {
		return nil
	}
	if err := os.RemoveAll(target); err != nil { //nolint:gosec // target is the candidate below the governed root
		return fmt.Errorf("remove candidate %s: %w", target, err)
	}
	return nil
}

// restoreLock puts plugins.lock back the way it was before the transaction.
func restoreLock(root string, oldLock domain.PluginLockFile, existed bool) error {
	if existed {
		return writeLock(root, oldLock)
	}
	_ = os.Remove(filepath.Join(root, "plugins.lock")) //nolint:errcheck,gosec // absent before transaction
	return nil
}
