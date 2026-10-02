package rolevalidation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// readCatalogRankedStamp reads the materialized plugins/catalog.yaml under
// root and returns providerID's certification stamp, the same tolerant-read
// shape as readLock/readRoleMap in this package.
func readCatalogRankedStamp(root, providerID string) (domain.CatalogRankedStamp, bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("read plugins/catalog.yaml: %w", err)
	}
	stamp, ok, err := domain.FindCatalogRankedStamp(raw, providerID)
	if err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("find catalog ranked stamp: %w", err)
	}
	return stamp, ok, nil
}
