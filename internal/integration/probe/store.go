package probe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Load reads the last observation. A missing file is "never probed".
func Load(path string) (Record, bool, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("read probe record: %w", err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, false, fmt.Errorf("parse probe record: %w", err)
	}
	return record, true, nil
}

// Save writes the observation with owner-only permissions.
func Save(path string, record Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode probe record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create probe directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write probe record: %w", err)
	}
	return nil
}
