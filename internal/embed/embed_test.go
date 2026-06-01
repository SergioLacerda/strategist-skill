package embed_test

import (
	"os"
	"path/filepath"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_Extract_ReadOnlyTarget(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	// Make target directory read-only so MkdirAll / WriteFile inside it fails
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := embedpkg.Extractor{}.Extract(dir, false)
	require.Error(t, err)
}

func TestExtractor_Extract(t *testing.T) {
	t.Parallel()
	t.Run("extracts defaults into target directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir, false))

		// Core files that must always be extracted
		expectedFiles := []string{
			"SKILL.md",
			"knowledge.index.yaml",
			"skill.yaml",
			"index.yaml",
		}
		for _, f := range expectedFiles {
			assert.FileExists(t, filepath.Join(dir, f), "expected embedded file: %s", f)
		}

		// Core directories
		expectedDirs := []string{
			"personas",
			"schemas",
			"templates",
		}
		for _, d := range expectedDirs {
			assert.DirExists(t, filepath.Join(dir, d), "expected embedded dir: %s", d)
		}
	})

	t.Run("extract is idempotent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir, false))
		require.NoError(t, ext.Extract(dir, false), "second extract should not fail")
	})

	t.Run("extracted SKILL.md is non-empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))
		data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("extracted defaults include quick_draw pipeline and prompts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillYAML, err := os.ReadFile(filepath.Join(dir, "skill.yaml"))
		require.NoError(t, err)
		skill := string(skillYAML)
		assert.Contains(t, skill, "stage: quick_draw_detection")
		assert.Contains(t, skill, "stage: quick_draw_gate")
		assert.Contains(t, skill, "write_quick_draw_without_gate")
		assert.Contains(t, skill, "<base_path>/todo/<tema>.md")

		// Quick Draw procedure detail lives in contracts/quick-draw.yaml (refactored from SKILL.md)
		quickDraw, err := os.ReadFile(filepath.Join(dir, "contracts", "quick-draw.yaml"))
		require.NoError(t, err)
		qd := string(quickDraw)
		assert.Contains(t, qd, "quick-draw")
		assert.Contains(t, qd, "sim/nao")

		// SKILL.md retains the Quick Draw routing reference
		skillMD, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		doc := string(skillMD)
		assert.Contains(t, doc, "Quick Draw")

		pragmatic, err := os.ReadFile(filepath.Join(dir, "personas", "pragmatic.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(pragmatic), "quick_draw_gate")
		assert.Contains(t, string(pragmatic), "quick_draw_success")

		epic, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(epic), "quick_draw_gate")
		assert.Contains(t, string(epic), "quick_draw_success")
	})

	t.Run("extracted defaults include opportunist attack and chest scope contracts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillYAML, err := os.ReadFile(filepath.Join(dir, "skill.yaml"))
		require.NoError(t, err)
		skill := string(skillYAML)
		assert.Contains(t, skill, "stage: opportunity_gate")
		assert.Contains(t, skill, "condition: opportunity_manifest.count > 0")
		assert.Contains(t, skill, "stage: opportunity_execution")
		assert.Contains(t, skill, "condition: opportunity_gate_granted")
		assert.Contains(t, skill, "skip_opportunity_gate")
		assert.Contains(t, skill, "invoke_opportunity_sniper_without_approval")
		assert.Contains(t, skill, "scope_values: [all, discovery, refinement, execution]")

		pragmatic, err := os.ReadFile(filepath.Join(dir, "personas", "pragmatic.yaml"))
		require.NoError(t, err)
		p := string(pragmatic)
		assert.Contains(t, p, "opportunity_detected")
		assert.Contains(t, p, "opportunity_gate")
		assert.Contains(t, p, "Aprovar? (yes / no / select)")

		epic, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml"))
		require.NoError(t, err)
		e := string(epic)
		assert.Contains(t, e, "opportunity_detected")
		assert.Contains(t, e, "opportunity_gate")
		assert.Contains(t, e, "Aprovar? (yes / no / select)")
	})

	t.Run("extracted defaults include ADR language instruction", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		// ADR language instruction detail lives in contracts/adr.yaml (refactored from SKILL.md)
		adrYAML, err := os.ReadFile(filepath.Join(dir, "contracts", "adr.yaml"))
		require.NoError(t, err)
		adr := string(adrYAML)
		assert.Contains(t, adr, "pt-BR")
		assert.Contains(t, adr, "en")
		assert.Contains(t, adr, "language_source")

		// SKILL.md retains the ADR routing reference
		skillMD, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		doc := string(skillMD)
		assert.Contains(t, doc, "Archivist")
		assert.Contains(t, doc, "contracts/adr.yaml")
	})
}
