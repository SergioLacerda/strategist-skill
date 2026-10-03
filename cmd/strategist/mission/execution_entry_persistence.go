package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func cleanupExecutionEntryTemp(tmp *os.File) error {
	var cleanupErrs []error
	if err := tmp.Close(); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("close temporary execution entry: %w", err))
	}
	if err := os.Remove(tmp.Name()); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("remove temporary execution entry: %w", err))
	}
	return errors.Join(cleanupErrs...)
}

func removeExecutionEntryTemp(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove temporary execution entry: %w", err)
	}
	return nil
}

func saveNewExecutionEntry(path string, e executionEntry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal execution entry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create execution-entry directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-execution-entry-")
	if err != nil {
		return fmt.Errorf("create temporary execution entry: %w", err)
	}
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("write execution entry: %w", errors.Join(err, cleanupExecutionEntryTemp(tmp)))
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary execution entry: %w", errors.Join(err, removeExecutionEntryTemp(tmp.Name())))
	}
	if err := executionEntryLink(tmp.Name(), path); err != nil {
		return fmt.Errorf("link execution entry: %w", errors.Join(err, removeExecutionEntryTemp(tmp.Name())))
	}
	if err := removeExecutionEntryTemp(tmp.Name()); err != nil {
		return err
	}
	return nil
}

func saveExecutionEntry(root string, e *executionEntry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal execution entry: %w", err)
	}
	path := executionEntryPath(root, e.MissionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create execution-entry directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-execution-entry-")
	if err != nil {
		return fmt.Errorf("create temporary execution entry: %w", err)
	}
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("write execution entry: %w", errors.Join(err, cleanupExecutionEntryTemp(tmp)))
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary execution entry: %w", errors.Join(err, removeExecutionEntryTemp(tmp.Name())))
	}
	if err := executionEntryRename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename execution entry: %w", errors.Join(err, removeExecutionEntryTemp(tmp.Name())))
	}
	return nil
}
