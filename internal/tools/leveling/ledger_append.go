package leveling

import (
	"errors"
	"fmt"
	"os"
)

func appendJSONLine(path string, encoded []byte, label string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // path is resolved below .strategist
	if err != nil {
		return fmt.Errorf("leveling: open %s ledger: %w", label, err)
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return errors.Join(fmt.Errorf("leveling: write %s record: %w", label, err), f.Close())
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("leveling: close %s ledger: %w", label, err)
	}
	return nil
}
