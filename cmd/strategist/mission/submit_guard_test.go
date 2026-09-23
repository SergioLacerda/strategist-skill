package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTasks(t *testing.T, base, mission, body string) {
	t.Helper()
	dir := filepath.Join(base, "refined", mission)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(body), 0o600))
}

// Analysis-only terminal events must not drop documentation targets that
// Sniper is supposed to materialize.
func TestRequireAnalysisOnlyPackage(t *testing.T) {
	base := t.TempDir()
	writeTasks(t, base, "docs", "- [ ] 1.1 [documentation_target] write the guide\n")
	writeTasks(t, base, "code", "- [ ] 1.1 [implementation_handoff] change code\n")

	for _, event := range []domain.MissionEngineEvent{domain.MissionEventGateApprovedAnalysisOnly, domain.MissionEventHandoffNotApplicable} {
		err := requireAnalysisOnlyPackage(base, "docs", event)
		require.Error(t, err, event)
		assert.Contains(t, err.Error(), "documentation_target")
		require.NoError(t, requireAnalysisOnlyPackage(base, "code", event), event)
		require.NoError(t, requireAnalysisOnlyPackage(base, "absent", event), "no refined package declares no target")
	}
	require.NoError(t, requireAnalysisOnlyPackage(base, "docs", domain.MissionEventGateApproved), "other events are not guarded")
}
