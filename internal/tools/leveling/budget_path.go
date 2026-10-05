package leveling

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ResolveBudgetLedgerPath validates and anchors the configured budget-event
// ledger below the Strategist root. The role-level ledger remains independent.
func ResolveBudgetLedgerPath(root string, config domain.LevelingConfig) (string, error) {
	if err := config.Validate(); err != nil {
		return "", fmt.Errorf("leveling budget ledger path: %w", err)
	}
	return filepath.Join(root, filepath.FromSlash(config.EffectiveBudgetLedgerPath())), nil
}
