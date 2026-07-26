package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

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

// --- version ---

// --- validate ---

// minimalValidateRoot creates a .strategist/-like tree suitable for validateCmd:
// active.yaml, personas/pragmatic.yaml, roles/default.yaml.
func minimalValidateRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "pragmatic.yaml"),
		[]byte("id: pragmatic\ntone_directive: precise\nphase_labels:\n  discovery: analysis\n  refinement: refinement\n  execution: execution\ndiagnostics:\n  pipeline_header: \"[Strategist] pipeline=starting\"\n  bootstrap_origin: \"[Strategist] profile_path={path}\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\nrefinement: archivist\nexecution: caveman\n"), 0o644))
	return dir
}

// --- dojo ---

func setupDojoScenario(t *testing.T, scenario, criteria, runContent string) string {
	t.Helper()
	dir := t.TempDir()

	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	scenarioDir := filepath.Join(dir, ".analysis", "dojo", scenario)
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte(criteria), 0o644))

	if runContent != "" {
		runDir := filepath.Join(dir, ".analysis", "dojo", "run", "todo")
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte(runContent), 0o644))
	}
	return strategistRoot
}

// --- check ---

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
// (templates/domain/identity/{drift-patterns,what-i-am}.yaml) that checkCmd now
// requires (D6 — see check_identity.go). Any hand-built check root fixture that
// doesn't go through minimalCheckRoot must call this too.
//
// drift-patterns.yaml is also one of domain.NormativeRuntimeDefaultFiles() (checked
// for embedded-default parity by validateRuntimeDefaultParity), so it must be
// byte-identical to the real embedded default — an arbitrary placeholder would
// trip a *different* check (runtime_stale_unknown_manifest) as an unrelated
// side effect. what-i-am.yaml is not normative-tracked, so any content is fine.

// writeMinimalIdentityFiles creates the internal-domain identity files
// (templates/domain/identity/{drift-patterns,what-i-am}.yaml) that checkCmd now
// requires (D6 — see check_identity.go). Any hand-built check root fixture that
// doesn't go through minimalCheckRoot must call this too.
//
// drift-patterns.yaml is also one of domain.NormativeRuntimeDefaultFiles() (checked
// for embedded-default parity by validateRuntimeDefaultParity), so it must be
// byte-identical to the real embedded default — an arbitrary placeholder would
// trip a *different* check (runtime_stale_unknown_manifest) as an unrelated
// side effect. what-i-am.yaml is not normative-tracked, so any content is fine.
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
