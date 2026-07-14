//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeHappyPathActiveYAML(t *testing.T, strategistDir string) {
	t.Helper()

	activeYAML := []byte(`mode: epic
base_path: .analysis
roles_config: roles/default.yaml
knowledge_index_path: knowledge.index.yaml
language:
  ui: pt-BR
  docs: en
  chat: pt-BR
  code: en
adr_enabled: true
mission_mode: entrega_revisada
escopo_done: entrega
aplicar_alteracoes: false

slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask

treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`)
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), activeYAML, 0o644))
}

func TestE2E_CLI_InstallCompileValidateCheckStale(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")

	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())
	assert.Contains(t, install.output(), "[Strategist] install complete")
	assert.FileExists(t, filepath.Join(strategistDir, "active.yaml"))
	assert.FileExists(t, filepath.Join(strategistDir, "SKILL.md"))
	assert.FileExists(t, filepath.Join(strategistDir, "knowledge.index.yaml"))

	writeHappyPathActiveYAML(t, strategistDir)

	compile := runStrategistCLI(t, workspace, "compile", "--root", strategistDir)
	require.Equal(t, 0, compile.exitCode, compile.output())
	assert.Contains(t, compile.output(), "[Strategist] compile complete")
	assert.FileExists(t, filepath.Join(strategistDir, ".compiled", ".config.gz"))
	assert.FileExists(t, filepath.Join(strategistDir, ".compiled", ".domain.gz"))
	assert.FileExists(t, filepath.Join(strategistDir, ".compiled", ".index.gz"))
	assert.FileExists(t, filepath.Join(strategistDir, ".compiled", ".manifest.gz"))

	var compiledConfig map[string]any
	testutil.ReadGzJSON(t, filepath.Join(strategistDir, ".compiled", ".config.gz"), &compiledConfig)
	active, ok := compiledConfig["active"].(map[string]any)
	require.True(t, ok, "compiled active must be an object")
	assert.Equal(t, "epic", active["mode"])
	assert.NotEmpty(t, active["roles_config"])

	var compiledDomain map[string]any
	testutil.ReadGzJSON(t, filepath.Join(strategistDir, ".compiled", ".domain.gz"), &compiledDomain)
	loadAlways, ok := compiledDomain["load_always"].(map[string]any)
	require.True(t, ok, "compiled domain load_always must be an object")
	assert.NotEmpty(t, loadAlways)

	checkFresh := runStrategistCLI(t, workspace, "check-stale", filepath.Join(strategistDir, ".compiled", ".config.gz"))
	require.Equal(t, 0, checkFresh.exitCode, checkFresh.output())
	assert.Contains(t, strings.ToLower(checkFresh.output()), "stale=false")
}

func TestE2E_CLI_ValidateOnMinimalRoot(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(strategistDir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(strategistDir, "roles"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte("mode: epic\nroles_config: default\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "personas", "epic.yaml"), testutil.ValidMinimalPersonaYAML(), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "roles", "default.yaml"), []byte("discovery: brainstorming\nrefinement: archivist\nexecution: caveman\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))

	validate := runStrategistCLI(t, workspace, "validate", "--root", strategistDir)
	require.Equal(t, 0, validate.exitCode, validate.output())
	assert.Contains(t, validate.output(), "[Strategist] validate OK")
}

func TestE2E_CLI_InstallForceRewritesActiveYAML(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")

	firstInstall := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, firstInstall.exitCode, firstInstall.output())

	originalTemplate, err := embedpkg.Extractor{}.ReadFile("templates/epic-standalone.yaml")
	require.NoError(t, err)

	customContent := append(append([]byte{}, originalTemplate...), []byte("\ncustom_note: keep-me\n")...)
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), customContent, 0o644))

	secondInstall := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, secondInstall.exitCode, secondInstall.output())
	afterNoForce, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(afterNoForce), "custom_note: keep-me")

	forcedInstall := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent", "--force")
	require.Equal(t, 0, forcedInstall.exitCode, forcedInstall.output())
	afterForce, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	assert.NotContains(t, string(afterForce), "custom_note: keep-me")
	assert.Contains(t, string(afterForce), "mode: epic")
	assert.Contains(t, string(afterForce), "base_path: .analysis")
}

func TestE2E_CLI_TreasureChestsFlowIntoCompiledConfig(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")

	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())
	writeHappyPathActiveYAML(t, strategistDir)

	compile := runStrategistCLI(t, workspace, "compile", "--root", strategistDir)
	require.Equal(t, 0, compile.exitCode, compile.output())

	var compiledConfig map[string]any
	testutil.ReadGzJSON(t, filepath.Join(strategistDir, ".compiled", ".config.gz"), &compiledConfig)
	active, ok := compiledConfig["active"].(map[string]any)
	require.True(t, ok, "compiled active must be an object")

	treasureChests, ok := active["treasure_chests"].([]any)
	require.True(t, ok, "compiled active must carry treasure_chests")
	require.NotEmpty(t, treasureChests)

	first, ok := treasureChests[0].(map[string]any)
	require.True(t, ok, "treasure chest entry must be an object")
	assert.Equal(t, "source", first["id"])
	assert.Equal(t, ".sdd/source", first["path"])
	assert.Equal(t, "all", first["scope"])
}
