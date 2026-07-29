package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGoverned_NotExist(t *testing.T) {
	t.Parallel()
	result, err := LoadGoverned(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadGoverned_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    title: Chest One\n    path: /some/path\n    trust:\n      tier: T1\n      reviewed_by: user@example.com\n      last_reviewed: '2026-01-01'\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	result, err := LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, result, "chest-one")
	assert.Equal(t, "T1", result["chest-one"].Trust.Tier)
}

func TestLoadGoverned_WithValidGrade(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    path: /some/path\n    trust:\n      tier: T1\n    grade:\n      source_grade: A\n      reuse_value: high\n      implementation_status: implemented\n    open_gaps: [\"missing tests\"]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	result, err := LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, result, "chest-one")
	assert.Equal(t, "A", result["chest-one"].Grade.SourceGrade)
	assert.Equal(t, "high", result["chest-one"].Grade.ReuseValue)
	assert.Equal(t, []string{"missing tests"}, result["chest-one"].OpenGaps)
}

func TestLoadGoverned_WithInvalidGrade(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    path: /some/path\n    trust:\n      tier: T1\n    grade:\n      source_grade: Z\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	_, err := LoadGoverned(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_grade")
}

func TestLoadGoverned_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"),
		[]byte(": not: valID: yaml:\n"), 0o644))

	_, err := LoadGoverned(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chests.yaml")
}
