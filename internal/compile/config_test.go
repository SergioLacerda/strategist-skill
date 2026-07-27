package compile_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fullActiveYAML = "mode: full\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"

// copyTestdata copies the tree rooted at testdata/<fixture> into dst.
func copyTestdata(t testing.TB, fixture, dst string) {
	t.Helper()
	src := filepath.Join("testdata", fixture)
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	require.NoError(t, err)
}

func TestCompileConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t testing.TB, dir string)
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name:  "minimal valid root produces artifact",
			setup: testutil.MinimalRoot,
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
				assert.NotNil(t, a["compiled_at"])
				assert.NotNil(t, a["sources"])
				assertSourceStats(t, a)
				assert.NotNil(t, a["active"])
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, personas, "epic")
				roles, ok := a["roles"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, roles, "default")
			},
		},
		{
			name: "missing active.yaml returns error",
			setup: func(t testing.TB, _ string) {
				t.Helper()
			},
			wantErr: true,
		},
		{
			name: "empty personas and roles dirs are valid",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
			},
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
			},
		},
		{
			name: "testdata: valid-minimal fixture",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				copyTestdata(t, "valid-minimal", dir)
			},
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, personas, "epic")
			},
		},
		{
			name: "non-yaml files in personas dir are ignored",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "README.md"), []byte("# readme"), 0o644))
			},
			check: func(t *testing.T, a map[string]any) {
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.NotContains(t, personas, "README")
			},
		},
		{
			name: "missing roles directory entirely is valid",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
			},
			check: func(t *testing.T, a map[string]any) {
				roles, ok := a["roles"].(map[string]any)
				require.True(t, ok)
				assert.Empty(t, roles)
			},
		},
		{
			name: "invalid roles yaml returns wrapped error",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "bad.yaml"), []byte(": invalid\n  yaml: here"), 0o644))
			},
			wantErr: true,
		},
		{
			name: "legacy roles_config is not preserved in compiled active",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				testutil.MinimalRoot(t, dir)
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "active.yaml"),
					[]byte("mode: full\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				active, ok := a["active"].(map[string]any)
				require.True(t, ok)
				assert.NotContains(t, active, "roles_config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			out := filepath.Join(dir, ".compiled", ".config.gz")
			err := compile.Config(dir, out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out)
			var artifact map[string]any
			testutil.ReadGzJSON(t, out, &artifact)
			if tt.check != nil {
				tt.check(t, artifact)
			}
		})
	}
}

func TestCompileConfig_InjectsPTBRPhaseAnnouncements(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte(fullActiveYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte(`id: epic
tone_directive: be precise
phase_labels:
  discovery: analysis
  refinement: refinement
  execution: execution
diagnostics:
  pipeline_header: "[Strategist] pipeline=starting mission_id={id}"
  bootstrap_origin: "[Strategist] profile_path={path} active_yaml={active} reason={reason}"
phase_announcements:
  en:
    discovery_starting: "Ranger starts."
    discovery_done: "Ranger done."
    refinement_starting: "Archivist starts."
    refinement_done: "Archivist done."
    approval_gate_shown: "Gate shown."
    documentation_starting: "Sniper starts."
    documentation_target_done: "Sniper target done."
    documentation_done: "Sniper done."
`), 0o644))

	out := filepath.Join(dir, ".compiled", ".config.gz")
	require.NoError(t, compile.Config(dir, out))

	var artifact map[string]any
	testutil.ReadGzJSON(t, out, &artifact)
	personas, ok := artifact["personas"].(map[string]any)
	require.True(t, ok)
	epic, ok := personas["epic"].(map[string]any)
	require.True(t, ok)
	pa, ok := epic["phase_announcements"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, pa, "en")
	ptBR, ok := pa["pt-BR"].(map[string]any)
	require.True(t, ok, "expected phase_announcements.pt-BR to be injected")

	for _, key := range []string{
		"discovery_starting", "discovery_done",
		"refinement_starting", "refinement_done",
		"approval_gate_shown",
		"documentation_starting", "documentation_target_done", "documentation_done",
	} {
		value, ok := ptBR[key].(string)
		require.True(t, ok, "missing pt-BR key %q", key)
		assert.NotEmpty(t, value)
	}
}

func TestCompileConfig_PhaseAnnouncementsAbsentWhenPersonaHasNone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	out := filepath.Join(dir, ".compiled", ".config.gz")
	require.NoError(t, compile.Config(dir, out))

	var artifact map[string]any
	testutil.ReadGzJSON(t, out, &artifact)
	personas, ok := artifact["personas"].(map[string]any)
	require.True(t, ok)
	epic, ok := personas["epic"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, epic, "phase_announcements")
}

func TestCompileConfig_InvalidActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: [unclosed"), 0o644))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "compile config")
}

func TestCompileConfig_InvalidPersonaYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "bad.yaml"), []byte(": invalid\n  yaml: here"), 0o644))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
}

func TestCompileConfig_PersonaTypedDecodeFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(fullActiveYAML), 0o644))
	// Valid as a generic map (id decodes to a sequence) but invalid against the
	// typed PersonaConfig struct, whose ID field is a string — triggers the
	// typed re-validation decode error path distinct from persona.Validate() failures.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "badtype.yaml"),
		[]byte("id:\n  - a\n  - b\ntone_directive: valid\n"),
		0o644,
	))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "personas validate")
}

func TestCompileConfig_PersonaFailsValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte(fullActiveYAML),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "noid.yaml"),
		[]byte("name: no-id-persona\nsome_field: value\n"),
		0o644,
	))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "personas")
}
