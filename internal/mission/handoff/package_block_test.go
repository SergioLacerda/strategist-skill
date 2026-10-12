package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writePackage(t *testing.T, analysis string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o644))
	return dir
}

func TestReadPackageFrontmatterBlockDecodesATypedBlock(t *testing.T) {
	dir := writePackage(t, "---\nmission_id: m\nblock:\n  name: x\n  count: 2\n---\n\nbody\n")
	var block struct {
		Name  string `yaml:"name"`
		Count int    `yaml:"count"`
	}
	found, err := ReadPackageFrontmatterBlock(dir, "block", &block)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "x", block.Name)
	require.Equal(t, 2, block.Count)
}

func TestReadPackageFrontmatterBlockReportsAbsenceAndBadInput(t *testing.T) {
	dir := writePackage(t, "---\nmission_id: m\n---\n\nbody\n")
	var block map[string]any
	found, err := ReadPackageFrontmatterBlock(dir, "block", &block)
	require.NoError(t, err)
	require.False(t, found, "an absent block is reported, never invented")

	typed := writePackage(t, "---\nblock: [1, 2]\n---\n\nbody\n")
	var wrong struct {
		Name string `yaml:"name"`
	}
	found, err = ReadPackageFrontmatterBlock(typed, "block", &wrong)
	require.Error(t, err)
	require.True(t, found, "the block exists but does not decode")

	_, err = ReadPackageFrontmatterBlock(t.TempDir(), "block", &block)
	require.Error(t, err)
	_, err = ReadPackageFrontmatterBlock(writePackage(t, "no frontmatter\n"), "block", &block)
	require.Error(t, err)
}
