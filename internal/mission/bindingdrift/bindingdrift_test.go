package bindingdrift

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeActive(t *testing.T, root, refinement string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(root, 0o755))
	body := "mode: epic\nslots:\n  discovery: brainstorming\n  refinement: " + refinement + "\n  execution: sniper\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte(body), 0o600))
}

func TestCurrentSlotsReadsTheSlotsBlock(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	writeActive(t, root, "writing-plans")
	slots, err := CurrentSlots(root)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper"}, slots)
}

func TestCurrentSlotsOfAMissingFileIsEmptyNotAnError(t *testing.T) {
	slots, err := CurrentSlots(filepath.Join(t.TempDir(), ".strategist"))
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestSaveAndLoadRoundTripAndNeverOverwrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	first := map[string]string{"refinement": "writing-plans"}
	require.NoError(t, Save(root, "m1", first, time.Unix(100, 0)))
	require.NoError(t, Save(root, "m1", map[string]string{"refinement": "openspec-propose"}, time.Unix(200, 0)))

	rec, found, err := Load(root, "m1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, first, rec.Slots, "the bindings a mission started with are immutable")
}

func TestLoadOfAMissionWithoutARecord(t *testing.T) {
	_, found, err := Load(filepath.Join(t.TempDir(), ".strategist"), "nope")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestDiffReportsOnlyChangedSlotsSorted(t *testing.T) {
	start := map[string]string{"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper"}
	now := map[string]string{"discovery": "brainstorming", "refinement": "openspec-propose", "execution": "sniper"}
	changes := Diff(start, now)
	require.Len(t, changes, 1)
	assert.Equal(t, Change{Slot: "refinement", StartedWith: "writing-plans", Now: "openspec-propose"}, changes[0])
	assert.Contains(t, changes[0].String(), "refinement")
	assert.Contains(t, changes[0].String(), "writing-plans")
	assert.Contains(t, changes[0].String(), "openspec-propose")
	assert.Empty(t, Diff(start, start), "an unchanged workspace produces no output")
}

func TestDiffTreatsAnAddedOrRemovedSlotAsAChange(t *testing.T) {
	assert.Len(t, Diff(map[string]string{"a": "x"}, map[string]string{}), 1)
	assert.Len(t, Diff(map[string]string{}, map[string]string{"a": "x"}), 1)
	assert.Empty(t, Diff(nil, nil))
}

func TestLatestReturnsTheNewestRecord(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, Save(root, "old", map[string]string{"refinement": "a"}, time.Unix(100, 0)))
	require.NoError(t, Save(root, "new", map[string]string{"refinement": "b"}, time.Unix(300, 0)))
	rec, found, err := Latest(root)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "new", rec.MissionID)
}

func TestSaveRejectsAnUnsafeMissionID(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.Error(t, Save(root, "../escape", map[string]string{"a": "b"}, time.Now()))
}

func TestCurrentSlotsRejectsAMalformedConfiguration(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("slots: [unterminated"), 0o600))
	_, err := CurrentSlots(root)
	require.Error(t, err)
}

func TestCurrentSlotsOfAConfigurationWithoutSlotsIsEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\n"), 0o600))
	slots, err := CurrentSlots(root)
	require.NoError(t, err)
	assert.NotNil(t, slots)
	assert.Empty(t, slots)
}

func TestLoadRejectsAMalformedRecordButLatestSkipsIt(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, Save(root, "good", map[string]string{"refinement": "a"}, time.Unix(100, 0)))
	dir := filepath.Join(root, filepath.FromSlash(recordsRelDir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o600))

	_, _, err := Load(root, "bad")
	require.Error(t, err)

	rec, found, err := Latest(root)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "good", rec.MissionID, "one bad record must not hide the others")
}

func TestLatestWithNoRecordsIsNotFound(t *testing.T) {
	_, found, err := Latest(filepath.Join(t.TempDir(), ".strategist"))
	require.NoError(t, err)
	assert.False(t, found)
}

func TestSaveFailsWhenTheRecordDirectoryCannotBeCreated(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "missions")
	require.NoError(t, os.WriteFile(blocker, []byte("a file where a directory is needed"), 0o600))
	require.Error(t, Save(base, "m1", map[string]string{"a": "b"}, time.Now()))
}

func TestChangeStringShowsNoneForAnEmptySide(t *testing.T) {
	assert.Contains(t, Change{Slot: "x", Now: "y"}.String(), "started_with=(none)")
	assert.Contains(t, Change{Slot: "x", StartedWith: "y"}.String(), "now=(none)")
}

func TestLoadReportsARecordPathThatIsNotAReadableFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.FromSlash(recordsRelDir), "m1.json"), 0o755))
	_, _, err := Load(root, "m1")
	require.Error(t, err)
}

func TestLoadRejectsAnUnsafeMissionID(t *testing.T) {
	_, _, err := Load(filepath.Join(t.TempDir(), ".strategist"), "../escape")
	require.Error(t, err)
}
