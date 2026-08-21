package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// writeSddFixtures creates .sdd/metadata.json and .sdd/source/governance-core.json
// under dir with the given mandate IDs, all marked type=MANDATE status=required.
func writeSddFixtures(t *testing.T, dir string, mandateIDs []string) {
	t.Helper()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(filepath.Join(sddDir, "source"), 0o755))

	meta := map[string]any{"fingerprints": map[string]any{"combined": "abc123"}}
	metaRaw, _ := json.Marshal(meta)
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), metaRaw, 0o644))

	items := make([]map[string]any, 0, len(mandateIDs))
	for _, id := range mandateIDs {
		items = append(items, map[string]any{"id": id, "type": "MANDATE", "status": "required"})
	}
	coreRaw, _ := json.Marshal(map[string]any{"items": items})
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "source", "governance-core.json"), coreRaw, 0o644))
}

// writeSkillYAML marshals data as YAML and writes it to dir/subpath.
func writeSkillYAML(t *testing.T, dir, subpath string, data map[string]any) string {
	t.Helper()
	full := filepath.Join(dir, subpath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	raw, err := yaml.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(full, raw, 0o644))
	return full
}

// --- stringSlice ---

func TestStringSlice_NestedHit(t *testing.T) {
	m := map[string]any{
		"compliance": map[string]any{
			"mandates": []any{"M001", "M002"},
		},
	}
	assert.Equal(t, []string{"M001", "M002"}, stringSlice(m, "compliance", "mandates"))
}

func TestStringSlice_MissingKey(t *testing.T) {
	m := map[string]any{"compliance": map[string]any{}}
	assert.Nil(t, stringSlice(m, "compliance", "missing"))
}

func TestStringSlice_MissingTopLevel(t *testing.T) {
	assert.Nil(t, stringSlice(map[string]any{}, "compliance", "mandates"))
}

func TestStringSlice_NilMap(t *testing.T) {
	assert.Nil(t, stringSlice(nil, "k"))
}

func TestStringSlice_NotASlice(t *testing.T) {
	m := map[string]any{"key": "not-a-slice"}
	assert.Nil(t, stringSlice(m, "key"))
}

// --- computeComplianceGaps ---

func TestComputeComplianceGaps_FullyCovered(t *testing.T) {
	skill := map[string]any{
		"compliance": map[string]any{
			"mandates": []any{"M001", "M002"},
			"partial":  []any{"M003"},
		},
	}
	report := &SyncReport{MandatesActive: []string{"M001", "M002", "M003", "M004"}}
	computeComplianceGaps(report, skill)
	assert.ElementsMatch(t, []string{"M001", "M002"}, report.MandatesCompliant)
	assert.ElementsMatch(t, []string{"M003"}, report.MandatesPartial)
	assert.ElementsMatch(t, []string{"M004"}, report.MandatesMissing)
}

func TestComputeComplianceGaps_EmptySkill(t *testing.T) {
	report := &SyncReport{MandatesActive: []string{"M001"}}
	computeComplianceGaps(report, map[string]any{})
	assert.Nil(t, report.MandatesCompliant)
	assert.Nil(t, report.MandatesPartial)
	assert.Equal(t, []string{"M001"}, report.MandatesMissing)
}

func TestComputeComplianceGaps_AllCovered(t *testing.T) {
	skill := map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001", "M002"}},
	}
	report := &SyncReport{MandatesActive: []string{"M001", "M002"}}
	computeComplianceGaps(report, skill)
	assert.Empty(t, report.MandatesMissing)
}

// --- applyMissingFields ---

func TestApplyMissingFields_AllMissing(t *testing.T) {
	skill := map[string]any{"name": "test"}
	report := &SyncReport{}
	changed := applyMissingFields(skill, report)
	assert.True(t, changed)
	assert.ElementsMatch(t, []string{"validation_policy", "budget_policy", "telemetry_policy"}, report.FieldsApplied)
}

func TestApplyMissingFields_NoneNeeded(t *testing.T) {
	skill := map[string]any{
		"validation_policy": map[string]any{},
		"budget_policy":     map[string]any{},
		"telemetry_policy":  map[string]any{},
	}
	report := &SyncReport{}
	changed := applyMissingFields(skill, report)
	assert.False(t, changed)
	assert.Empty(t, report.FieldsApplied)
}

func TestApplyMissingFields_PartiallyPresent(t *testing.T) {
	skill := map[string]any{"validation_policy": map[string]any{}}
	report := &SyncReport{}
	changed := applyMissingFields(skill, report)
	assert.True(t, changed)
	assert.Len(t, report.FieldsApplied, 2)
	assert.NotContains(t, report.FieldsApplied, "validation_policy")
}

// --- readSkill ---

func TestReadSkill_Success(t *testing.T) {
	dir := t.TempDir()
	path := writeSkillYAML(t, dir, "skill.yaml", map[string]any{"name": "test-skill"})
	skill, err := readSkill(path)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", skill["name"])
}

func TestReadSkill_NotFound(t *testing.T) {
	_, err := readSkill(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read skill.yaml")
}

func TestReadSkill_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	require.NoError(t, os.WriteFile(path, []byte(": invalid: yaml:\n"), 0o644))
	_, err := readSkill(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse skill.yaml")
}

// --- readGovernance ---

func TestReadGovernance_Success(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001", "M002"})
	fp, active, err := readGovernance(filepath.Join(dir, ".sdd"))
	require.NoError(t, err)
	assert.Equal(t, "abc123", fp)
	assert.ElementsMatch(t, []string{"M001", "M002"}, active)
}

func TestReadGovernance_FallbackFingerprint(t *testing.T) {
	dir := t.TempDir()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(filepath.Join(sddDir, "source"), 0o755))
	// flat governance_fingerprint field (no fingerprints.combined)
	meta := map[string]any{"governance_fingerprint": "fallback-fp"}
	raw, _ := json.Marshal(meta)
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), raw, 0o644))
	coreRaw, _ := json.Marshal(map[string]any{"items": []any{}})
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "source", "governance-core.json"), coreRaw, 0o644))

	fp, _, err := readGovernance(sddDir)
	require.NoError(t, err)
	assert.Equal(t, "fallback-fp", fp)
}

func TestReadGovernance_MetadataNotFound(t *testing.T) {
	dir := t.TempDir()
	_, _, err := readGovernance(filepath.Join(dir, ".sdd"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadGovernance_MetadataReadError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(sddDir, 0o755))
	metaPath := filepath.Join(sddDir, "metadata.json")
	require.NoError(t, os.WriteFile(metaPath, []byte("{}"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(metaPath, 0o644) })

	_, _, err := readGovernance(sddDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read metadata")
}

func TestReadGovernance_InvalidMetadataJSON(t *testing.T) {
	dir := t.TempDir()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(sddDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), []byte("not-json"), 0o644))
	_, _, err := readGovernance(sddDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse metadata")
}

func TestReadGovernance_CoreNotFound(t *testing.T) {
	dir := t.TempDir()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(sddDir, 0o755))
	raw, _ := json.Marshal(map[string]any{"fingerprints": map[string]any{"combined": "x"}})
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), raw, 0o644))
	// source/ dir missing → core file not found
	_, _, err := readGovernance(sddDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "governance-core.json")
}

func TestReadGovernance_InvalidCoreJSON(t *testing.T) {
	dir := t.TempDir()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(filepath.Join(sddDir, "source"), 0o755))
	raw, _ := json.Marshal(map[string]any{"fingerprints": map[string]any{"combined": "x"}})
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), raw, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "source", "governance-core.json"), []byte("not-json"), 0o644))
	_, _, err := readGovernance(sddDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse governance-core.json")
}

// --- RunSync ---

func TestRunSync_NoChanges(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{
		"compliance":        map[string]any{"mandates": []any{"M001"}},
		"validation_policy": map[string]any{},
		"budget_policy":     map[string]any{},
		"telemetry_policy":  map[string]any{},
	})
	report, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), false)
	require.NoError(t, err)
	assert.Empty(t, report.FieldsApplied)
	assert.Empty(t, report.MandatesMissing)
}

func TestRunSync_AppliesFields(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	report, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), false)
	require.NoError(t, err)
	assert.Len(t, report.FieldsApplied, 3)
	data, _ := os.ReadFile(skillPath)
	assert.Contains(t, string(data), "validation_policy")
}

func TestRunSync_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	before, _ := os.ReadFile(skillPath)
	report, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), true)
	require.NoError(t, err)
	assert.True(t, report.DryRun)
	assert.Len(t, report.FieldsApplied, 3)
	after, _ := os.ReadFile(skillPath)
	assert.Equal(t, before, after) // file must not be modified in dry-run
}

func TestRunSync_ReadGovernanceError(t *testing.T) {
	dir := t.TempDir()
	_, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), false)
	require.Error(t, err)
}

func TestRunSync_ReadSkillError(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	// .strategist dir exists but skill.yaml is absent
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))
	_, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill.yaml")
}

func TestRunSync_WriteError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	require.NoError(t, os.Chmod(skillPath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(skillPath, 0o644) })

	_, err := RunSync(filepath.Join(dir, ".strategist"), filepath.Join(dir, ".sdd"), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write skill.yaml")
}
