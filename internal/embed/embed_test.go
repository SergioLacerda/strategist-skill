package embed_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_ReadFile(t *testing.T) {
	t.Parallel()
	t.Run("reads embedded file successfully", func(t *testing.T) {
		t.Parallel()
		data, err := embedpkg.Extractor{}.ReadFile("SKILL.md")
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		t.Parallel()
		_, err := embedpkg.Extractor{}.ReadFile("nonexistent/file.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embed: read")
	})

	t.Run("reads template file successfully", func(t *testing.T) {
		t.Parallel()
		data, err := embedpkg.Extractor{}.ReadFile("templates/epic-standalone.yaml")
		require.NoError(t, err)
		assert.Contains(t, string(data), "mode:")
	})

	t.Run("reads embedded default provider manifests", func(t *testing.T) {
		t.Parallel()

		brainstorming, err := embedpkg.Extractor{}.ReadFile("skills/brainstorming/skill.yaml")
		require.NoError(t, err)
		assert.Contains(t, string(brainstorming), "id: brainstorming")
		assert.Contains(t, string(brainstorming), "status: active")
		assert.Contains(t, string(brainstorming), "risk_score: write_analysis")
		assert.Contains(t, string(brainstorming), "provider_class: rankeado")
		assert.Contains(t, string(brainstorming), "canonical_role: ranger")
		assert.Contains(t, string(brainstorming), "auxiliary_tools_allowed:")
		assert.Contains(t, string(brainstorming), "- writing-plans")

		openspecExplore, err := embedpkg.Extractor{}.ReadFile("skills/openspec-explore/skill.yaml")
		require.NoError(t, err)
		assert.Contains(t, string(openspecExplore), "id: openspec-explore")
		assert.Contains(t, string(openspecExplore), "status: active")
		assert.Contains(t, string(openspecExplore), "risk_score: write_analysis")
		assert.Contains(t, string(openspecExplore), "provider_class: rankeado")
		assert.Contains(t, string(openspecExplore), "canonical_role: archivist")
	})
}

func TestExtractor_Extract_ReadOnlyTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("read-only chmod semantics are not reliable for this permission test on Windows")
	}
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

	t.Run("extracted defaults have no legacy idea-capture pipeline entry", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillYAML, err := os.ReadFile(filepath.Join(dir, "skill.yaml"))
		require.NoError(t, err)
		skill := string(skillYAML)
		assert.NotContains(t, skill, "stage: quick_draw_detection")
		assert.NotContains(t, skill, "stage: quick_draw_gate")
		assert.NotContains(t, skill, "write_quick_draw_without_gate")

		// contracts/machine/quick-draw.yaml was renamed to runbook-opportunity.yaml
		// and trimmed to the runbook_opportunity routine only — the idea-capture
		// gate/pipeline is gone; the normalize+append machinery Riposte used to
		// reuse from the old file now lives in riposte.yaml under Riposte's own
		// names.
		_, err = os.Stat(filepath.Join(dir, "contracts", "machine", "quick-draw.yaml"))
		assert.True(t, os.IsNotExist(err), "contracts/machine/quick-draw.yaml should no longer exist")

		runbookOpportunity, err := os.ReadFile(filepath.Join(dir, "contracts", "machine", "runbook-opportunity.yaml"))
		require.NoError(t, err)
		ro := string(runbookOpportunity)
		assert.Contains(t, ro, "runbook_opportunity")
		assert.NotContains(t, ro, "sim: proceed_to_sniper")
		assert.NotContains(t, ro, "ranger_quick_draw:")
		assert.NotContains(t, ro, "archivist_quick_draw:")

		riposte, err := os.ReadFile(filepath.Join(dir, "contracts", "machine", "riposte.yaml"))
		require.NoError(t, err)
		rp := string(riposte)
		assert.Contains(t, rp, "riposte_normalize")
		assert.Contains(t, rp, "riposte_capture")
		assert.NotContains(t, rp, "quick-draw")

		// SKILL.md no longer references the retired idea-capture routing
		skillMD, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		doc := string(skillMD)
		assert.NotContains(t, doc, "Quick Draw")

		pragmatic, err := os.ReadFile(filepath.Join(dir, "personas", "pragmatic.yaml"))
		require.NoError(t, err)
		assert.NotContains(t, string(pragmatic), "quick_draw_gate")
		assert.NotContains(t, string(pragmatic), "quick_draw_success")

		epic, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml"))
		require.NoError(t, err)
		assert.NotContains(t, string(epic), "quick_draw_gate")
		assert.NotContains(t, string(epic), "quick_draw_success")
	})

	t.Run("extracted defaults include opportunist attack and chest scope contracts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillYAML, err := os.ReadFile(filepath.Join(dir, "skill.yaml"))
		require.NoError(t, err)
		skill := string(skillYAML)
		// opportunity_attack is a canonical routine inside each slot (not a standalone pipeline stage)
		assert.Contains(t, skill, "- opportunity_attack")
		assert.Contains(t, skill, "skip_opportunity_attack_routine")
		assert.Contains(t, skill, "suppress_opportunity_attack_feedback")
		assert.Contains(t, skill, "scope_values: [all, discovery, refinement, execution]")

		pragmatic, err := os.ReadFile(filepath.Join(dir, "personas", "pragmatic.yaml"))
		require.NoError(t, err)
		p := string(pragmatic)
		assert.Contains(t, p, "opportunity_detected")
		assert.Contains(t, p, "opportunity_gate")
		assert.Contains(t, p, "Approve? (yes / no / select)")

		epic, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml"))
		require.NoError(t, err)
		e := string(epic)
		assert.Contains(t, e, "opportunity_detected")
		assert.Contains(t, e, "opportunity_gate")
		assert.Contains(t, e, "Approve? (yes / no / select)")
	})

	t.Run("extracted defaults include pipeline bypass hardening", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillYAML, err := os.ReadFile(filepath.Join(dir, "skill.yaml"))
		require.NoError(t, err)
		skill := string(skillYAML)
		assert.Contains(t, skill, "direct_repository_mutation_without_pipeline_evidence")
		assert.Contains(t, skill, "reason=pipeline_bypass_detected")
		assert.Contains(t, skill, "expected_phase={expected_phase}")
		assert.Contains(t, skill, "missing_evidence={missing_evidence}")

		// GAP-ST-06: protocol.md was merged into templates/agent-protocol.md; the
		// hardening rules must survive in the unified document.
		protocol, err := os.ReadFile(filepath.Join(dir, "templates", "agent-protocol.md"))
		require.NoError(t, err)
		assert.Contains(t, string(protocol), "pipeline_bypass_detected")
		assert.Contains(t, string(protocol), "Never mutate the repo without canonical pipeline evidence")

		stub, err := os.ReadFile(filepath.Join(dir, "protocol.md"))
		require.NoError(t, err)
		assert.Contains(t, string(stub), "templates/agent-protocol.md")

		approvalGate, err := os.ReadFile(filepath.Join(dir, "contracts", "machine", "approval-gate.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(approvalGate), "pipeline_bypass_detected")
		assert.Contains(t, string(approvalGate), "missing_evidence=approval_gate:analysis_accepted")
	})

	t.Run("extracted runtime docs preserve runtime-only path contract", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		skillMD, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		doc := string(skillMD)
		assert.Contains(t, doc, "single authoring and generation source")
		assert.Contains(t, doc, "only operational read target")
		assert.Contains(t, doc, "base_path")
		assert.Contains(t, doc, "not a hardcoded `.analysis/`")

		// GAP-ST-06: the runtime-only path contract now lives in the unified
		// templates/agent-protocol.md document.
		protocol, err := os.ReadFile(filepath.Join(dir, "templates", "agent-protocol.md"))
		require.NoError(t, err)
		assert.Contains(t, string(protocol), ".strategist/")
		assert.Contains(t, string(protocol), "base_path")
	})

	t.Run("extracted defaults include ADR language instruction", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir, false))

		// ADR language instruction detail lives in contracts/narrative/07-adr.md
		adrYAML, err := os.ReadFile(filepath.Join(dir, "contracts", "narrative", "07-adr.md"))
		require.NoError(t, err)
		adr := string(adrYAML)
		assert.Contains(t, adr, "pt-BR")
		assert.Contains(t, adr, "en")
		assert.Contains(t, adr, "active.language.docs")

		// SKILL.md itself no longer enumerates contract paths (W4 token-economy fix:
		// contracts/index.yaml is the sole authoritative loading manifest — SKILL.md
		// defers to it instead of restating the narrative load order). The ADR routing
		// reference now lives there.
		skillMD, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		doc := string(skillMD)
		assert.Contains(t, doc, "Archivist")
		assert.Contains(t, doc, "index.yaml")

		indexYAML, err := os.ReadFile(filepath.Join(dir, "contracts", "index.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(indexYAML), "narrative/07-adr.md")
	})
}
