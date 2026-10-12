package install

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readAuditRecords(t *testing.T, strategistDir string) []AuditRecord {
	t.Helper()
	f, err := os.Open(filepath.Join(strategistDir, "memory", "install-events.jsonl"))
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck // read-only test file
	var records []AuditRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec AuditRecord
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &rec))
		records = append(records, rec)
	}
	return records
}

func TestInstallAppendsAnAuditRecordWithBeforeAndAfterSlots(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true, ResetConfig: true}))

	records := readAuditRecords(t, strategistDir)
	require.NotEmpty(t, records)
	last := records[len(records)-1]
	assert.Equal(t, "install", last.Command)
	assert.Equal(t, "silent", last.Mode)
	assert.True(t, last.Force)
	assert.True(t, last.ResetConfig)
	assert.Equal(t, "writing-plans", last.SlotsBefore["refinement"], "the binding that was reset must be attributable")
	assert.NotEqual(t, "writing-plans", last.SlotsAfter["refinement"])
	assert.True(t, last.ConfigChanged)
	assert.NotEmpty(t, last.At)
}

func TestForcedInstallRecordsThatTheConfigurationWasNotChanged(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true}))

	records := readAuditRecords(t, strategistDir)
	last := records[len(records)-1]
	assert.True(t, last.Force)
	assert.False(t, last.ResetConfig)
	assert.False(t, last.ConfigChanged)
	assert.Equal(t, last.SlotsBefore, last.SlotsAfter)
}

func TestAuditRecordCarriesNoFileContentOrPaths(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	raw, err := os.ReadFile(filepath.Join(strategistDir, "memory", "install-events.jsonl"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), dir, "no absolute path may appear")
	assert.NotContains(t, string(raw), "base_path", "no file content may appear")
}

func TestReadSlotBindingsReturnsNilForMalformedActiveYAML(t *testing.T) {
	t.Parallel()
	strategistDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, names.ActiveYAMLName), []byte("slots: [broken"), 0o600))
	assert.Nil(t, readSlotBindings(strategistDir))
}
