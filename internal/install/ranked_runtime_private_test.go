package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/stretchr/testify/require"
)

func tgz(t *testing.T, name, body string, mode int64) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}))
	_, err := tw.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

// fakeNode stands in for the pinned Node: it emulates `openspec init` and
// `openspec context --json` so the whole private-runtime path runs for real,
// with no openspec on PATH. The subprocess PATH is only the private runtime
// directory, so the script may use shell builtins and `command -p` only.
const fakeNode = `#!/bin/sh
case "$2" in
  init) command -p mkdir -p openspec && printf 'schema: spec-driven\n' > openspec/config.yaml ;;
  context) printf '{"root":{"path":"%s"}}' "${PWD%/*}" ;;
esac
`

func registerFakePayload(t *testing.T) {
	t.Helper()
	nodeArc, nodeSum := tgz(t, "bin/node", fakeNode, 0o755)
	specArc, specSum := tgz(t, "openspec.js", "// stub", 0o644)
	m := runtimepayload.Manifest{
		SchemaVersion: runtimepayload.SchemaVersion, Provider: "openspec-propose",
		Launcher: runtimepayload.Launcher{Node: map[string]string{"default": "node/bin/node"}, Script: "openspec/openspec.js"},
		Components: []runtimepayload.Component{
			{Name: "node", Version: "22.0.0", OS: runtimepayload.AnyTarget, Arch: runtimepayload.AnyTarget, File: "node.tgz", Format: runtimepayload.FormatTarGz, SHA256: nodeSum, Size: int64(len(nodeArc)), Dest: "node"},
			{Name: "openspec", Version: "1.10.0", OS: runtimepayload.AnyTarget, Arch: runtimepayload.AnyTarget, File: "openspec.tgz", Format: runtimepayload.FormatTarGz, SHA256: specSum, Size: int64(len(specArc)), Dest: "openspec"},
		},
	}
	src := fstest.MapFS{"node.tgz": {Data: nodeArc}, "openspec.tgz": {Data: specArc}}
	original := payloadSource
	t.Cleanup(func() { payloadSource = original })
	payloadSource = func() (runtimepayload.Embedded, bool) { return runtimepayload.Embedded{Manifest: m, FS: src}, true }
}

func TestPrepareRankedProviderRuntimes_UsesPrivateRuntimeWithoutHostOpenSpec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Node is a POSIX shell script")
	}
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init --profile core --tools codex", Healthcheck: "openspec context --json",
	})
	registerFakePayload(t)
	t.Setenv("PATH", t.TempDir())

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist")))

	require.FileExists(t, filepath.Join(dir, ".strategist", "openspec", "config.yaml"))
	require.FileExists(t, filepath.Join(dir, ".strategist", "weapon-runtime", "openspec-propose", "node", "bin", "node"))

	raw, err := os.ReadFile(filepath.Join(dir, ".strategist", rankedRuntimeStatePath))
	require.NoError(t, err)
	var state rankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	require.Len(t, state.Entries, 1)
	private := state.Entries[0].Runtime
	require.NotNil(t, private)
	require.Equal(t, "weapon-runtime/openspec-propose/node/bin/node", private.Node)
	require.Equal(t, "weapon-runtime/openspec-propose/openspec/openspec.js", private.Script)
	require.Len(t, private.Components, 2)
	require.Equal(t, "1.10.0", private.Components[1].Version)
}

func TestPrepareRankedProviderRuntimes_CorruptPayloadFailsClosedWithoutFallback(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init --profile core --tools codex", Healthcheck: "openspec context --json",
	})
	registerFakePayload(t)
	original := payloadSource
	payloadSource = func() (runtimepayload.Embedded, bool) {
		e, ok := original()
		e.FS = fstest.MapFS{"node.tgz": {Data: []byte("tampered")}, "openspec.tgz": {Data: []byte("x")}}
		return e, ok
	}
	// A working host openspec must NOT rescue a bad private payload.
	called := false
	orig := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = orig })
	runRankedRuntimeCommand = func(context.Context, string, string, ...string) ([]byte, error) { called = true; return nil, nil }

	err := prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist"))
	require.ErrorIs(t, err, runtimepayload.ErrDigestMismatch)
	require.False(t, called, "no provider command may run after a payload failure")
	require.NoDirExists(t, filepath.Join(dir, ".strategist", "weapon-runtime", "openspec-propose"))
}
