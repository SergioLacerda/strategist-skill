package governance

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type fakeSource struct {
	name     string
	snapshot Snapshot
	err      error
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Snapshot(string) (Snapshot, error) {
	if f.err != nil {
		return Snapshot{}, f.err
	}
	return f.snapshot, nil
}

func validFakeSource() fakeSource {
	return fakeSource{
		name: "fixture",
		snapshot: Snapshot{
			SourceID:       "fixture",
			Fingerprint:    "abc123",
			ActiveMandates: []string{"M001", "M002"},
			Validated:      true,
		},
	}
}

func writeSkillYAML(t *testing.T, dir, subpath string, data map[string]any) string {
	t.Helper()
	full := filepath.Join(dir, subpath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	raw, err := yaml.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(full, raw, 0o644))
	return full
}

func TestStringSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data map[string]any
		keys []string
		want []string
	}{
		{name: "nested hit", data: map[string]any{"compliance": map[string]any{"mandates": []any{"M001", "M002"}}}, keys: []string{"compliance", "mandates"}, want: []string{"M001", "M002"}},
		{name: "missing", data: map[string]any{"compliance": map[string]any{}}, keys: []string{"compliance", "mandates"}},
		{name: "not a slice", data: map[string]any{"key": "value"}, keys: []string{"key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stringSlice(tt.data, tt.keys...))
		})
	}
}

func TestComputeComplianceGaps(t *testing.T) {
	skill := map[string]any{"compliance": map[string]any{
		"mandates": []any{"M001", "M002"},
		"partial":  []any{"M003"},
	}}
	report := &SyncReport{MandatesActive: []string{"M001", "M002", "M003", "M004"}}
	computeComplianceGaps(report, skill)
	assert.ElementsMatch(t, []string{"M001", "M002"}, report.MandatesCompliant)
	assert.ElementsMatch(t, []string{"M003"}, report.MandatesPartial)
	assert.Equal(t, []string{"M004"}, report.MandatesMissing)
}

func TestApplyMissingFields(t *testing.T) {
	report := &SyncReport{}
	skill := map[string]any{"name": "test"}
	assert.True(t, applyMissingFields(skill, report))
	assert.ElementsMatch(t, []string{"validation_policy", "budget_policy", "telemetry_policy"}, report.FieldsApplied)

	report = &SyncReport{}
	assert.False(t, applyMissingFields(map[string]any{
		"validation_policy": map[string]any{},
		"budget_policy":     map[string]any{},
		"telemetry_policy":  map[string]any{},
	}, report))
}

func TestReadSkill(t *testing.T) {
	dir := t.TempDir()
	path := writeSkillYAML(t, dir, "skill.yaml", map[string]any{"name": "test-skill"})
	skill, err := readSkill(path)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", skill["name"])

	_, err = readSkill(filepath.Join(dir, "missing.yaml"))
	require.ErrorContains(t, err, "read skill.yaml")
	invalid := filepath.Join(dir, "invalid.yaml")
	require.NoError(t, os.WriteFile(invalid, []byte("not: [valid"), 0o644))
	_, err = readSkill(invalid)
	assert.ErrorContains(t, err, "parse skill.yaml")
}

func TestWriteSyncedSkill_RejectsUnmarshalableValue(t *testing.T) {
	err := writeSyncedSkill(filepath.Join(t.TempDir(), "skill.yaml"), map[string]any{"channel": make(chan int)})
	assert.ErrorContains(t, err, "marshal skill.yaml")
}

func TestRunSync_AppliesFields(t *testing.T) {
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001"}},
	})
	report, err := RunSync(filepath.Join(dir, ".strategist"), validFakeSource(), filepath.Join(dir, "governance"), false)
	require.NoError(t, err)
	assert.Equal(t, "abc123", report.GovernanceFingerprint)
	assert.Equal(t, []string{"M001", "M002"}, report.MandatesActive)
	assert.Equal(t, []string{"M002"}, report.MandatesMissing)
	assert.Len(t, report.FieldsApplied, 3)
	data, err := os.ReadFile(skillPath) //nolint:gosec // test-controlled temp path
	require.NoError(t, err)
	assert.Contains(t, string(data), "validation_policy")
}

func TestRunSync_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	before, err := os.ReadFile(skillPath) //nolint:gosec // test-controlled temp path
	require.NoError(t, err)
	report, err := RunSync(filepath.Join(dir, ".strategist"), validFakeSource(), filepath.Join(dir, "governance"), true)
	require.NoError(t, err)
	assert.True(t, report.DryRun)
	after, err := os.ReadFile(skillPath) //nolint:gosec // test-controlled temp path
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestRunSync_RejectsInvalidSource(t *testing.T) {
	dir := t.TempDir()
	writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	cases := []struct {
		name   string
		source Source
		want   string
	}{
		{name: "nil", want: "source is required"},
		{name: "empty fingerprint", source: fakeSource{name: "fixture", snapshot: Snapshot{Validated: true}}, want: "empty fingerprint"},
		{name: "unvalidated", source: fakeSource{name: "fixture", snapshot: Snapshot{Fingerprint: "fp"}}, want: "unvalidated"},
		{name: "duplicate mandate", source: fakeSource{name: "fixture", snapshot: Snapshot{Fingerprint: "fp", Validated: true, ActiveMandates: []string{"M001", "M001"}}}, want: "duplicate mandate"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunSync(filepath.Join(dir, ".strategist"), tt.source, filepath.Join(dir, "governance"), true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestRunSync_PropagatesSourceError(t *testing.T) {
	dir := t.TempDir()
	writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	_, err := RunSync(filepath.Join(dir, ".strategist"), fakeSource{err: assert.AnError}, filepath.Join(dir, "governance"), true)
	require.ErrorIs(t, err, assert.AnError)
}

func TestRunSync_WriteError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{"name": "s"})
	require.NoError(t, os.Chmod(skillPath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(skillPath, 0o644) })
	_, err := RunSync(filepath.Join(dir, ".strategist"), validFakeSource(), filepath.Join(dir, "governance"), false)
	assert.ErrorContains(t, err, "write skill.yaml")
}
