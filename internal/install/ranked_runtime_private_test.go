package install

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/stretchr/testify/require"
)

func TestPrepareRankedProviderRuntimes_UsesEmbeddedOpenSpecWithHostNode(t *testing.T) {
	dir := t.TempDir()
	strategist := filepath.Join(dir, ".strategist")
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", openSpecContract("1.13.0", ""))

	originalFind, originalVersion, originalRun := findHostNode, hostNodeVersion, runRankedRuntimeCommand
	t.Cleanup(func() {
		findHostNode, hostNodeVersion, runRankedRuntimeCommand = originalFind, originalVersion, originalRun
	})
	findHostNode = func(name string) (string, error) {
		require.Equal(t, "node", name)
		return filepath.Join(dir, "node", "node"), nil
	}
	hostNodeVersion = func(context.Context, string, string) ([]byte, error) { return []byte("v20.19.0\n"), nil }
	runRankedRuntimeCommand = func(_ context.Context, commandDir, name string, args ...string) ([]byte, error) {
		require.NotEqual(t, "openspec", name, "the embedded bundle must not resolve openspec from PATH")
		require.True(t, filepath.IsAbs(name))
		require.NotEmpty(t, args)
		require.Contains(t, filepath.ToSlash(args[0]), "weapon-runtime/openspec-propose/openspec/")
		verb := args[1]
		if verb == "init" {
			require.NoError(t, os.MkdirAll(filepath.Join(commandDir, "openspec"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(strategist) + `"}}`), nil
	}

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), strategist))
	require.FileExists(t, filepath.Join(strategist, "weapon-runtime", "openspec-propose", "openspec", "dist", "core", "artifact-graph", "openspec.mjs"))

	raw, err := os.ReadFile(filepath.Join(strategist, domain.RankedRuntimeStatePath))
	require.NoError(t, err)
	var state domain.RankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	require.Equal(t, filepath.Join(dir, "node", "node"), state.Entries[0].Runtime.Node)
}

func TestHostNodeRuntimeRejectsUnsupportedNode(t *testing.T) {
	originalFind, originalVersion := findHostNode, hostNodeVersion
	t.Cleanup(func() { findHostNode, hostNodeVersion = originalFind, originalVersion })
	findHostNode = func(string) (string, error) { return filepath.Join(t.TempDir(), "node"), nil }
	hostNodeVersion = func(context.Context, string, string) ([]byte, error) { return []byte("v20.18.9\n"), nil }

	_, _, err := resolveRankedExecutable(context.Background(), t.TempDir(), "openspec-propose", openSpecContract("1.13.0", ""))
	require.ErrorContains(t, err, "Node >=20.19.0")
}

func openSpecContract(version, node string) domain.RankedRuntimeContract {
	return domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init", Healthcheck: "openspec context --json",
		Version: version, NodeVersion: node,
	}
}

func TestOpenSpecPinMustMatchTheContract(t *testing.T) {
	state := &domain.RankedRuntimeStateRuntime{Components: []domain.RankedRuntimeStateComponent{
		{Name: "openspec", Version: "1.13.0"},
	}}

	require.NoError(t, checkOpenSpecPin(openSpecContract("1.13.0", "22.23.2"), state))
	require.NoError(t, checkOpenSpecPin(openSpecContract("", ""), state), "unpinned contract accepts any bundle")

	err := checkOpenSpecPin(openSpecContract("1.10.0", "22.23.2"), state)
	require.ErrorContains(t, err, domain.ReasonRankedRuntimePinMismatch)
	require.ErrorContains(t, err, "OpenSpec")
}

// D3 regression: the on-disk skill tree carries its own runtime.build.yaml, and
// TreeDigest excludes that file from the digest it publishes, so a tampered
// tree can certify itself. Materialization must therefore read the binary's
// embedded defaults, not the extracted copy.
func TestPrepareRankedProviderRuntimes_IgnoresATamperedOnDiskOpenSpecTree(t *testing.T) {
	dir := t.TempDir()
	strategist := filepath.Join(dir, ".strategist")
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", openSpecContract("1.13.0", ""))

	// A genuinely self-consistent forgery: the certificate carries the real
	// digest of the tampered content, which is possible precisely because
	// TreeDigest excludes BuildInfoFile from what it covers.
	rel := "skills/openspec-propose/runtime"
	tree := filepath.Join(strategist, filepath.FromSlash(rel))
	bundle := filepath.Join(tree, "dist", "core", "artifact-graph")
	require.NoError(t, os.MkdirAll(bundle, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bundle, "openspec.mjs"), []byte("// tampered\n"), 0o644))
	digest, size, err := runtimepayload.TreeDigest(os.DirFS(strategist), rel)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tree, runtimepayload.BuildInfoFile), []byte(fmt.Sprintf(
		"version: 9.9.9\nbundle: dist/core/artifact-graph/openspec.mjs\ntree_sha256: %s\ntree_bytes: %d\n", digest, size)), 0o644))

	// Sanity: the forgery does verify against itself, so the only thing that
	// rejects it is reading the embedded authority instead.
	selfCheck, _, selfErr := runtimepayload.MaterializeOpenSpec(os.DirFS(strategist), filepath.Join(t.TempDir(), "self"))
	require.NoError(t, selfErr, "the forged tree certifies itself")
	selfBody, readErr := os.ReadFile(selfCheck) //nolint:gosec // G304: test-controlled path
	require.NoError(t, readErr)
	require.Contains(t, string(selfBody), "tampered")

	originalFind, originalVersion, originalRun := findHostNode, hostNodeVersion, runRankedRuntimeCommand
	t.Cleanup(func() {
		findHostNode, hostNodeVersion, runRankedRuntimeCommand = originalFind, originalVersion, originalRun
	})
	findHostNode = func(string) (string, error) { return filepath.Join(dir, "node", "node"), nil }
	hostNodeVersion = func(context.Context, string, string) ([]byte, error) { return []byte("v20.19.0\n"), nil }
	runRankedRuntimeCommand = func(_ context.Context, commandDir, _ string, args ...string) ([]byte, error) {
		if args[1] == "init" {
			require.NoError(t, os.MkdirAll(filepath.Join(commandDir, "openspec"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(strategist) + `"}}`), nil
	}

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), strategist))

	materialized := filepath.Join(strategist, "weapon-runtime", "openspec-propose", "openspec", "dist", "core", "artifact-graph", "openspec.mjs")
	body, readErr := os.ReadFile(materialized) //nolint:gosec // G304: test-controlled path
	require.NoError(t, readErr)
	require.NotContains(t, string(body), "tampered", "the forged on-disk tree must never be materialized")

	raw, stateErr := os.ReadFile(filepath.Join(strategist, domain.RankedRuntimeStatePath))
	require.NoError(t, stateErr)
	var state domain.RankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	require.NotEqual(t, "9.9.9", state.Entries[0].Runtime.Components[0].Version,
		"recorded evidence must come from the embedded authority, not the forged certificate")
}
