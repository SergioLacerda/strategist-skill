package leveling_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBudgetLedgerPathUsesIsolatedDefault(t *testing.T) {
	root := t.TempDir()
	path, err := leveling.ResolveBudgetLedgerPath(root, domain.LevelingConfig{})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, domain.DefaultBudgetLedgerPath), path)
}

func TestResolveBudgetLedgerPathAcceptsCustomRelativePath(t *testing.T) {
	root := t.TempDir()
	path, err := leveling.ResolveBudgetLedgerPath(root, domain.LevelingConfig{BudgetLedgerPath: "custom/events.jsonl"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "custom/events.jsonl"), path)
}

func TestResolveBudgetLedgerPathRejectsEscapingPath(t *testing.T) {
	for _, configured := range []string{"/tmp/events.jsonl", "../events.jsonl", "nested/../../events.jsonl"} {
		t.Run(configured, func(t *testing.T) {
			_, err := leveling.ResolveBudgetLedgerPath(t.TempDir(), domain.LevelingConfig{BudgetLedgerPath: configured})
			require.Error(t, err)
			assert.Contains(t, err.Error(), domain.ReasonLevelingBudgetLedgerPathInvalid)
		})
	}
}
