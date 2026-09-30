package plugins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scaffoldFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "demo")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\nmetadata:\n  version: \"1.0.0\"\n---\n"), 0o644))
	return dir
}

func runScaffold(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewScaffoldSidecar()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestScaffoldSidecarCommandGeneratesThenChecks(t *testing.T) {
	dir := scaffoldFixture(t)

	out, err := runScaffold(t, dir, "--role", "ranger", "--slot", "discovery")
	require.NoError(t, err)
	assert.Contains(t, out, "scaffold-sidecar: created")

	out, err = runScaffold(t, dir, "--role", "ranger", "--slot", "discovery", "--check")
	require.NoError(t, err)
	assert.Contains(t, out, "scaffold-sidecar: unchanged")
}

func TestScaffoldSidecarCommandSurfacesTheFailureCode(t *testing.T) {
	dir := scaffoldFixture(t)

	_, err := runScaffold(t, dir, "--role", "ranger", "--slot", "refinement")

	require.ErrorContains(t, err, "role_slot_mismatch")
}

func TestScaffoldSidecarCommandRequiresExactlyOnePackageDir(t *testing.T) {
	_, err := runScaffold(t, "--role", "ranger", "--slot", "discovery")

	require.Error(t, err)
}
