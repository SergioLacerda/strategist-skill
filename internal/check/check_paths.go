package check

import (
	"fmt"
	"path/filepath"
)

// absoluteRoot gives every downstream consumer of --root one canonical
// absolute spelling, so relative roots cannot diverge from the absolute paths
// providers report.
func absoluteRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve absolute root: %w", err)
	}
	return absolute, nil
}
