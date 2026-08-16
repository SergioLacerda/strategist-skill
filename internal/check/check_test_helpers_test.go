package check

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Duplicated from cmd/strategist's own cmd_test_helpers_test.go — Go test
// helpers aren't shareable across package boundaries, so this package
// carries its own copies of the helpers check*_test.go files use, mirroring
// how internal/treasurecli duplicated its own subset when it was extracted
// (20260806-treasure-chest-cmd-consolidation).

// attachMissionRun wires a non-nil telemetry.MissionRun into cmd's context, so
// that RunE bodies gated on `telemetryRunFromCmd(cmd) != nil` get exercised.
func attachMissionRun(t *testing.T, cmd *cobra.Command) *telemetry.MissionRun {
	t.Helper()
	run := telemetry.NewMissionRun("test-mission")
	origCtx := cmd.Context()
	t.Cleanup(func() { cmd.SetContext(origCtx) })
	cmd.SetContext(telemetry.WithMissionRun(context.Background(), run))
	return run
}

// freshArtifactDir creates an artifact + manifest pair with no sources
// (= always considered fresh by IsStale).
func freshArtifactDir(t *testing.T) (dir, artifactPath string) {
	t.Helper()
	dir = t.TempDir()
	artifactPath = filepath.Join(dir, "artifact.gz")
	testutil.WriteGzJSON(t, artifactPath, map[string]any{"sources": map[string]int64{}})
	testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{"generated_at": 0})
	return dir, artifactPath
}

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// captureStderr replaces os.Stderr with a pipe and returns whatever was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stderr
	os.Stderr = w
	fn()
	require.NoError(t, w.Close())
	os.Stderr = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// withClosedStdout points os.Stdout at an already-closed file so writes fail,
// exercising output-error branches without a fake io.Writer plumbed through.
func withClosedStdout(t *testing.T, fn func()) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "closed-stdout-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	fn()
}

const minimalPersonaYAML = `id: epic
tone_directive: test tone
phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper
diagnostics:
  pipeline_header: "[Strategist] pipeline=starting mission_id={id} persona=epic\n"
  bootstrap_origin: "[Strategist] profile_path={path} active_yaml={active} reason={reason}\n"
`

// writeMinimalIdentityFiles creates the internal-domain identity files
// (templates/domain/identity/{drift-patterns,what-i-am}.yaml) that checkCmd
// requires (D6 — see check_identity.go). Any hand-built check root fixture
// that doesn't go through minimalCheckRoot must call this too.
//
// drift-patterns.yaml is also one of domain.NormativeRuntimeDefaultFiles()
// (checked for embedded-default parity by validateRuntimeDefaultParity), so
// it must be byte-identical to the real embedded default — an arbitrary
// placeholder would trip a *different* check (runtime_stale_unknown_manifest)
// as an unrelated side effect. what-i-am.yaml is not normative-tracked, so
// any content is fine.
func writeMinimalIdentityFiles(t *testing.T, dir string) {
	t.Helper()
	identityDir := filepath.Join(dir, "templates", "domain", "identity")
	require.NoError(t, os.MkdirAll(identityDir, 0o755))
	driftPatterns, err := embedpkg.Extractor{}.ReadFile("templates/domain/identity/drift-patterns.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(identityDir, "drift-patterns.yaml"), driftPatterns, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(identityDir, "what-i-am.yaml"), []byte("identity: strategist\n"), 0o644))
}

// minimalCheckRoot creates a .strategist/ tree suitable for checkCmd with all
// three slot providers installed plus a valid epic persona.
func minimalCheckRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, provider := range []struct {
		name      string
		riskScore string
	}{
		{"brainstorming", "write_analysis"},
		{"openspec-explore", "write_analysis"},
		{"sdd-ask", "controlled"},
	} {
		provDir := filepath.Join(dir, "skills", provider.name)
		require.NoError(t, os.MkdirAll(provDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(provDir, "skill.yaml"),
			[]byte("id: "+provider.name+"\nrisk_score: "+provider.riskScore+"\n"),
			0o644,
		))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte(minimalPersonaYAML),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	writeMinimalIdentityFiles(t, dir)
	return dir
}

func writeInstallManifestForTest(t *testing.T, root, relPath, hash string) {
	t.Helper()
	manifest := domain.InstallManifest{
		Schema:    "strategist.install-manifest.v1",
		PackageID: "test",
		Files: []domain.InstallManifestFile{{
			Path:   relPath,
			Owner:  domain.RuntimeFileNormative,
			SHA256: hash,
		}},
	}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), data, 0o644))
}
