package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteKnowledgeIndexSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		initialKI   string
		cfg         domain.WizardConfig
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "no chest path — file unchanged",
			initialKI:   "sources: []\n",
			cfg:         domain.WizardConfig{TreasureChestPath: ""},
			wantContain: []string{"sources: []"},
		},
		{
			name:      "chest path replaces placeholder",
			initialKI: "sources: []\n",
			cfg:       domain.WizardConfig{TreasureChestPath: ".sdd/source"},
			wantContain: []string{
				"id: source",
				"path: .sdd/source",
				"tags: [all]",
			},
			wantAbsent: []string{"sources: []"},
		},
		{
			name:      "preserves surrounding comments",
			initialKI: "# preamble\nsources: []\n# postamble\n",
			cfg:       domain.WizardConfig{TreasureChestPath: "docs/specs"},
			wantContain: []string{
				"# preamble",
				"id: specs",
				"# postamble",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			kiPath := filepath.Join(dir, "knowledge.index.yaml")
			require.NoError(t, os.WriteFile(kiPath, []byte(tt.initialKI), 0o644))
			require.NoError(t, writeKnowledgeIndexSource(dir, tt.cfg))
			data, err := os.ReadFile(kiPath)
			require.NoError(t, err)
			s := string(data)
			for _, want := range tt.wantContain {
				assert.Contains(t, s, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, s, absent)
			}
		})
	}
}

// TestWriteKnowledgeIndexSource_AlreadyConfigured is the regression guard for the
// 2026-07-26 install rollback bug: a second `strategist install --wizard` run
// against a workspace whose knowledge.index.yaml was already substituted by a
// prior run must no-op successfully, not error and roll back the install.
func TestWriteKnowledgeIndexSource_AlreadyConfigured(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	initial := "sources:\n  - id: already-set\n"
	require.NoError(t, os.WriteFile(kiPath, []byte(initial), 0o644))
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.NoError(t, err)
	data, readErr := os.ReadFile(kiPath)
	require.NoError(t, readErr)
	assert.Equal(t, initial, string(data), "already-configured file must be left unchanged")
}

// TestWriteKnowledgeIndexSource_CorruptedTemplate proves the original template-drift
// guard survives the idempotency fix: a file with neither the placeholder nor the
// bare "sources:" key present is genuine corruption, not a legitimate re-run.
func TestWriteKnowledgeIndexSource_CorruptedTemplate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("# empty\nunrelated: true\n"), 0o644))
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.ErrorContains(t, err, `placeholder "sources: []" not found`)
	assert.ErrorContains(t, err, `"sources:" key absent`)
}

// TestWriteKnowledgeIndexSource_SecondRunIsIdempotent drives the function twice in
// sequence — the exact shape of the reported bug (wizard run #1 then wizard run #2
// on the same workspace) — and asserts neither call errors and the second is a
// true no-op on top of the first's result.
func TestWriteKnowledgeIndexSource_SecondRunIsIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))
	cfg := domain.WizardConfig{TreasureChestPath: ".sdd/source"}

	require.NoError(t, writeKnowledgeIndexSource(dir, cfg), "first run must substitute")
	afterFirst, err := os.ReadFile(kiPath)
	require.NoError(t, err)
	assert.Contains(t, string(afterFirst), "id: source")

	require.NoError(t, writeKnowledgeIndexSource(dir, cfg), "second run must not error")
	afterSecond, err := os.ReadFile(kiPath)
	require.NoError(t, err)
	assert.Equal(t, string(afterFirst), string(afterSecond), "second run must be a no-op")
}

func TestWriteKnowledgeIndexSource_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "read knowledge.index.yaml")
}

func TestWriteKnowledgeIndexSource_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))
	// Atomic writes go through a temp file + rename in the same directory, so the
	// failure mode that blocks them is a read-only directory, not a read-only file.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "write knowledge.index.yaml")
}

func TestWriteTreasureChestManifest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		initialTC   string
		cfg         domain.WizardConfig
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "no chest path — file unchanged",
			initialTC:   "chests: []\n",
			cfg:         domain.WizardConfig{TreasureChestPath: ""},
			wantContain: []string{"chests: []"},
		},
		{
			name:      "chest path replaces placeholder",
			initialTC: "chests: []\n",
			cfg:       domain.WizardConfig{TreasureChestPath: ".sdd/source"},
			wantContain: []string{
				"id: source",
				"title: source",
				"path: .sdd/source",
				"tier: T1",
			},
			wantAbsent: []string{"chests: []"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tcPath := filepath.Join(dir, "treasure-chests.yaml")
			require.NoError(t, os.WriteFile(tcPath, []byte(tt.initialTC), 0o644))
			require.NoError(t, writeTreasureChestManifest(dir, tt.cfg))
			data, err := os.ReadFile(tcPath)
			require.NoError(t, err)
			s := string(data)
			for _, want := range tt.wantContain {
				assert.Contains(t, s, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, s, absent)
			}
		})
	}
}

func TestWriteTreasureChestManifest_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := writeTreasureChestManifest(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "read treasure-chests.yaml")
}

// TestWriteTreasureChestManifest_AlreadyConfigured mirrors
// TestWriteKnowledgeIndexSource_AlreadyConfigured for treasure-chests.yaml — same
// bug class, same fix shape.
func TestWriteTreasureChestManifest_AlreadyConfigured(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tcPath := filepath.Join(dir, "treasure-chests.yaml")
	initial := "chests:\n  - id: already-set\n"
	require.NoError(t, os.WriteFile(tcPath, []byte(initial), 0o644))
	err := writeTreasureChestManifest(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.NoError(t, err)
	data, readErr := os.ReadFile(tcPath)
	require.NoError(t, readErr)
	assert.Equal(t, initial, string(data), "already-configured file must be left unchanged")
}

// TestWriteTreasureChestManifest_CorruptedTemplate mirrors
// TestWriteKnowledgeIndexSource_CorruptedTemplate for treasure-chests.yaml.
func TestWriteTreasureChestManifest_CorruptedTemplate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tcPath := filepath.Join(dir, "treasure-chests.yaml")
	require.NoError(t, os.WriteFile(tcPath, []byte("# empty\nunrelated: true\n"), 0o644))
	err := writeTreasureChestManifest(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.ErrorContains(t, err, `placeholder "chests: []" not found`)
	assert.ErrorContains(t, err, `"chests:" key absent`)
}

func TestWriteTreasureChestManifest_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	tcPath := filepath.Join(dir, "treasure-chests.yaml")
	require.NoError(t, os.WriteFile(tcPath, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := writeTreasureChestManifest(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "write treasure-chests.yaml")
}

func TestWriteActiveYAMLBytes_LockWriteFailureIsNonBlocking(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory at the lock path makes integrity.WriteLock fail while
	// active.yaml itself still gets written — the failure is only warned about.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, configLockName), 0o755))

	err := writeActiveYAMLBytes(dir, []byte("mode: full\n"))
	require.NoError(t, err, "lock failure must not block active.yaml write")

	content, readErr := os.ReadFile(filepath.Join(dir, activeYAMLName))
	require.NoError(t, readErr)
	assert.Equal(t, "mode: full\n", string(content))
}

// TestWriteTreasureChestManifest_SecondRunIsIdempotent mirrors
// TestWriteKnowledgeIndexSource_SecondRunIsIdempotent for treasure-chests.yaml.
func TestWriteTreasureChestManifest_SecondRunIsIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tcPath := filepath.Join(dir, "treasure-chests.yaml")
	require.NoError(t, os.WriteFile(tcPath, []byte("chests: []\n"), 0o644))
	cfg := domain.WizardConfig{TreasureChestPath: ".sdd/source"}

	require.NoError(t, writeTreasureChestManifest(dir, cfg), "first run must substitute")
	afterFirst, err := os.ReadFile(tcPath)
	require.NoError(t, err)
	assert.Contains(t, string(afterFirst), "id: source")

	require.NoError(t, writeTreasureChestManifest(dir, cfg), "second run must not error")
	afterSecond, err := os.ReadFile(tcPath)
	require.NoError(t, err)
	assert.Equal(t, string(afterFirst), string(afterSecond), "second run must be a no-op")
}

func TestTreasureChestID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want string
	}{
		{".sdd/source", "source"},
		{"source", "source"},
		{"/absolute/path/to/chest", "chest"},
		{"trailing/slash/", "slash"},
		{"nodslash", "nodslash"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, treasureChestID(tt.path))
		})
	}
}

func TestWriteActiveYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         domain.WizardConfig
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "full mode with custom slots",
			cfg: domain.WizardConfig{
				Mode:               "pragmatic",
				BasePath:           ".analysis",
				UILanguage:         "en",
				DocLanguage:        "en",
				ChatLanguage:       "pt-BR",
				CodeLanguage:       "en",
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "openspec-explore",
				ExecutionProvider:  "sniper",
			},
			wantContain: []string{
				"mode: pragmatic",
				"base_path: .analysis",
				"language:",
				"  ui: en",
				"  docs: en",
				"  chat: pt-BR",
				"  code: en",
				"discovery: brainstorming",
				"refinement: openspec-explore",
				"execution: sniper",
			},
			wantAbsent: []string{"roles_config", "execution_mode", "git_persistence_mode", "adr_enabled"},
		},
		{
			name: "with treasure chest path",
			cfg: domain.WizardConfig{
				Mode:               "pragmatic",
				BasePath:           ".analysis",
				UILanguage:         "en",
				DocLanguage:        "en",
				ChatLanguage:       "pt-BR",
				CodeLanguage:       "en",
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "openspec-explore",
				ExecutionProvider:  "sniper",
				TreasureChestPath:  ".sdd/source",
			},
			wantContain: []string{
				"treasure_chests:",
				"id: source",
				"path: .sdd/source",
				"scope: all",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, writeActiveYAML(dir, tt.cfg))
			data, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
			require.NoError(t, err)
			s := string(data)
			for _, want := range tt.wantContain {
				assert.Contains(t, s, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, s, absent)
			}
		})
	}
}

func TestWriteActiveYAML_DoesNotEmitExecutionMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := domain.WizardConfig{
		Mode:               "epic",
		BasePath:           ".analysis",
		UILanguage:         "en",
		DocLanguage:        "en",
		ChatLanguage:       "en",
		CodeLanguage:       "en",
		DiscoveryProvider:  "brainstorming",
		RefinementProvider: "openspec-explore",
		ExecutionProvider:  "sdd-ask",
	}
	require.NoError(t, writeActiveYAML(dir, cfg))
	data, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, "execution_mode")
	assert.NotContains(t, s, "git_persistence_mode")
	assert.NotContains(t, s, "adr_enabled")
}
