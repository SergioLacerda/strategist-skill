package check

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/mission/bindingdrift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bindingDriftRoot(t *testing.T, refinement string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	body := "mode: epic\nslots:\n  discovery: brainstorming\n  refinement: " + refinement + "\n  execution: sniper\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte(body), 0o600))
	return root
}

func TestBindingDriftAdvisoryReportsASlotChangedSinceTheLastMissionStarted(t *testing.T) {
	root := bindingDriftRoot(t, "openspec-propose")
	require.NoError(t, bindingdrift.Save(root, "m-1", map[string]string{"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper"}, time.Now()))

	advisories := bindingDriftAdvisories(root)
	require.Len(t, advisories, 1)
	assert.Contains(t, advisories[0], "binding_changed")
	assert.Contains(t, advisories[0], "slot=refinement")
	assert.Contains(t, advisories[0], "started_with=writing-plans")
	assert.Contains(t, advisories[0], "now=openspec-propose")
	assert.Contains(t, advisories[0], "m-1")
}

func TestBindingDriftAdvisoryIsSilentWhenNothingChangedOrNoMissionStarted(t *testing.T) {
	root := bindingDriftRoot(t, "writing-plans")
	assert.Empty(t, bindingDriftAdvisories(root), "no mission record yet")
	require.NoError(t, bindingdrift.Save(root, "m-1", map[string]string{"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper"}, time.Now()))
	assert.Empty(t, bindingDriftAdvisories(root), "an unchanged workspace produces no output")
}
