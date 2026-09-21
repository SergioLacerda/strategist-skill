package runtimepayload

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

type entry struct {
	name string
	body string
	mode int64
	link string
}

func tarGz(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.body)), Typeflag: tar.TypeReg}
		if e.link != "" {
			hdr = &tar.Header{Name: e.name, Typeflag: tar.TypeSymlink, Linkname: e.link}
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if e.link == "" {
			_, err := tw.Write([]byte(e.body))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func zipBytes(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		require.NoError(t, err)
		_, err = w.Write([]byte(e.body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// fixture returns a manifest and source FS exercising every supported
// component format (per-target tar.gz/zip archives and a shared archive), so
// the generic materializer's extraction and verification stay covered.
func fixture(t *testing.T) (Manifest, fstest.MapFS) {
	t.Helper()
	nodeUnix := tarGz(t, entry{name: "node-v1/bin/node", body: "NODE", mode: 0o755}, entry{name: "node-v1/LICENSE", body: "lic", mode: 0o644})
	nodeWin := zipBytes(t, entry{name: "node-v1/node.exe", body: "NODEEXE"}, entry{name: "node-v1/LICENSE", body: "lic"})
	spec := tarGz(t, entry{name: "node_modules/@fission-ai/openspec/bin/openspec.js", body: "JS", mode: 0o644})
	m := Manifest{
		SchemaVersion: SchemaVersion,
		Provider:      "openspec-propose",
		Launcher:      Launcher{Node: map[string]string{"default": "node/bin/node", "windows": "node/node.exe"}, Script: "openspec/node_modules/@fission-ai/openspec/bin/openspec.js"},
		Components: []Component{
			{Name: "node", Version: "22.23.2", OS: "linux", Arch: "amd64", File: "node-linux-amd64.tar.gz", Format: FormatTarGz, SHA256: digest(nodeUnix), Size: int64(len(nodeUnix)), Dest: "node", StripComponents: 1},
			{Name: "node", Version: "22.23.2", OS: "windows", Arch: "amd64", File: "node-windows-amd64.zip", Format: FormatZip, SHA256: digest(nodeWin), Size: int64(len(nodeWin)), Dest: "node", StripComponents: 1},
			{Name: "openspec", Version: "1.10.0", OS: AnyTarget, Arch: AnyTarget, File: "openspec.tar.gz", Format: FormatTarGz, SHA256: digest(spec), Size: int64(len(spec)), Dest: "openspec"},
		},
	}
	src := fstest.MapFS{
		"node-linux-amd64.tar.gz": {Data: nodeUnix},
		"node-windows-amd64.zip":  {Data: nodeWin},
		"openspec.tar.gz":         {Data: spec},
	}
	return m, src
}

func TestMaterializeExtractsVerifiedRuntimeForTarget(t *testing.T) {
	m, src := fixture(t)
	dest := filepath.Join(t.TempDir(), "openspec-propose")

	ev, err := Materialize(src, m, "linux", "amd64", dest)
	require.NoError(t, err)

	node, err := os.Stat(filepath.Join(dest, "node", "bin", "node"))
	require.NoError(t, err)
	if runtime.GOOS != "windows" { // Windows file modes carry no execute bit
		require.NotZero(t, node.Mode()&0o111, "executable bit must be preserved")
	}
	require.FileExists(t, filepath.Join(dest, "openspec", "node_modules", "@fission-ai", "openspec", "bin", "openspec.js"))
	require.Len(t, ev.Components, 2)
	require.Equal(t, "22.23.2", ev.Components[0].Version)
	require.NotEmpty(t, ev.Components[0].SHA256)
}

func TestMaterializeZipTargetAndLauncherPaths(t *testing.T) {
	m, src := fixture(t)
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "windows", "amd64", dest)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dest, "node", "node.exe"))

	node, script, err := m.LauncherPaths(dest, "windows")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "node", "node.exe"), node)
	require.Equal(t, filepath.Join(dest, "openspec", "node_modules", "@fission-ai", "openspec", "bin", "openspec.js"), script)
}

func TestMaterializeRejectsUnsupportedTarget(t *testing.T) {
	m, src := fixture(t)
	_, err := Materialize(src, m, "darwin", "arm64", filepath.Join(t.TempDir(), "rt"))
	require.ErrorIs(t, err, ErrTargetUnsupported)
}

func TestMaterializeRejectsMissingPayload(t *testing.T) {
	m, src := fixture(t)
	delete(src, "openspec.tar.gz")
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrPayloadMissing)
	require.NoDirExists(t, dest, "nothing may be written when verification fails")
}

func TestMaterializeRejectsDigestMismatchBeforeWriting(t *testing.T) {
	m, src := fixture(t)
	src["node-linux-amd64.tar.gz"] = &fstest.MapFile{Data: []byte("tampered")}
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrDigestMismatch)
	require.NoDirExists(t, dest)
}

func TestMaterializeRejectsPathTraversalAndSymlinks(t *testing.T) {
	for name, archive := range map[string][]byte{
		"traversal": tarGz(t, entry{name: "../evil", body: "x", mode: 0o644}),
		"absolute":  tarGz(t, entry{name: "/etc/evil", body: "x", mode: 0o644}),
		"symlink":   tarGz(t, entry{name: "link", link: "/etc/passwd"}),
	} {
		t.Run(name, func(t *testing.T) {
			m := Manifest{SchemaVersion: SchemaVersion, Provider: "p", Launcher: Launcher{Node: map[string]string{"default": "n"}, Script: "s"},
				Components: []Component{{Name: "node", Version: "1", OS: AnyTarget, Arch: AnyTarget, File: "a.tar.gz", Format: FormatTarGz, SHA256: digest(archive), Size: int64(len(archive)), Dest: "node"}}}
			src := fstest.MapFS{"a.tar.gz": {Data: archive}}
			base := t.TempDir()
			dest := filepath.Join(base, "rt")

			_, err := Materialize(src, m, "linux", "amd64", dest)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrUnsafeArchive)
			require.NoDirExists(t, dest, "failed extraction must leave nothing behind")
			require.NoFileExists(t, filepath.Join(base, "evil"))
		})
	}
}

func TestParseManifestValidates(t *testing.T) {
	_, err := ParseManifest([]byte("schema_version: nope\n"))
	require.Error(t, err)

	m, _ := fixture(t)
	raw, err := m.Marshal()
	require.NoError(t, err)
	back, err := ParseManifest(raw)
	require.NoError(t, err)
	require.Equal(t, m.Components[0].SHA256, back.Components[0].SHA256)

	m.Components[0].SHA256 = "not-hex"
	raw, err = m.Marshal()
	require.NoError(t, err)
	_, err = ParseManifest(raw)
	require.Error(t, err)
}

func treeFS() fstest.MapFS {
	return fstest.MapFS{
		"rt/dist/openspec.mjs":  {Data: []byte("JS")},
		"rt/schemas/a.yaml":     {Data: []byte("a: 1")},
		"rt/runtime.build.yaml": {Data: []byte("generated")},
	}
}

func TestTreeDigestIsStableOrderIndependentAndSkipsBuildInfo(t *testing.T) {
	first, size, err := TreeDigest(treeFS(), "rt")
	require.NoError(t, err)
	require.Equal(t, int64(len("JS")+len("a: 1")), size, "runtime.build.yaml is not payload")

	again, _, err := TreeDigest(treeFS(), "rt")
	require.NoError(t, err)
	require.Equal(t, first, again)

	changedInfo := treeFS()
	changedInfo["rt/runtime.build.yaml"] = &fstest.MapFile{Data: []byte("different")}
	same, _, err := TreeDigest(changedInfo, "rt")
	require.NoError(t, err)
	require.Equal(t, first, same)

	changed := treeFS()
	changed["rt/dist/openspec.mjs"] = &fstest.MapFile{Data: []byte("JS!")}
	other, _, err := TreeDigest(changed, "rt")
	require.NoError(t, err)
	require.NotEqual(t, first, other)
}

func dirFixture(t *testing.T) (Manifest, fstest.MapFS) {
	t.Helper()
	src := treeFS()
	sum, size, err := TreeDigest(src, "rt")
	require.NoError(t, err)
	m := Manifest{SchemaVersion: SchemaVersion, Provider: "p",
		Launcher:   Launcher{Node: map[string]string{"default": "openspec/dist/openspec.mjs"}, Script: "openspec/dist/openspec.mjs"},
		Components: []Component{{Name: "openspec", Version: "1.13.0", OS: AnyTarget, Arch: AnyTarget, File: "rt", Format: FormatDir, SHA256: sum, Size: size, Dest: "openspec"}}}
	return m, src
}

func TestMaterializeCopiesVerifiedDirectoryComponent(t *testing.T) {
	m, src := dirFixture(t)
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dest, "openspec", "dist", "openspec.mjs"))
	require.FileExists(t, filepath.Join(dest, "openspec", "schemas", "a.yaml"))
	require.NoFileExists(t, filepath.Join(dest, "openspec", "runtime.build.yaml"))
}

func TestMaterializeRejectsTamperedDirectoryComponent(t *testing.T) {
	m, src := dirFixture(t)
	src["rt/dist/openspec.mjs"] = &fstest.MapFile{Data: []byte("evil")}
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrDigestMismatch)
	require.NoDirExists(t, dest)

	missing := fstest.MapFS{}
	_, err = Materialize(missing, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrPayloadMissing)
}

func TestEntryPathRejectsWindowsHostileNames(t *testing.T) {
	for _, name := range []string{
		"CON", "con", "nul.txt", "AUX", "prn.log", "COM1", "com9.exe", "LPT1", "lpt9.txt",
		"dir/CON", "dir/aux.md", "a.", "a ", "dir/b.", "dir/c ", "x/nul",
	} {
		_, _, err := entryPath(name, 0)
		require.ErrorIs(t, err, ErrUnsafeArchive, name)
	}
	for _, name := range []string{"node.exe", "LICENSE", "bin/node", "console.js", "auxiliary/x", "comet", "lpt10", "dir/.hidden", "a.b"} {
		rel, ok, err := entryPath(name, 0)
		require.NoError(t, err, name)
		require.True(t, ok, name)
		require.Equal(t, name, rel)
	}
}

func TestMaterializeRejectsReservedNameEntries(t *testing.T) {
	archive := tarGz(t, entry{name: "bin/NUL", body: "x", mode: 0o644})
	m := Manifest{SchemaVersion: SchemaVersion, Provider: "p", Launcher: Launcher{Node: map[string]string{"default": "n"}, Script: "s"},
		Components: []Component{{Name: "node", Version: "1", OS: AnyTarget, Arch: AnyTarget, File: "a.tar.gz", Format: FormatTarGz, SHA256: digest(archive), Size: int64(len(archive)), Dest: "node"}}}
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(fstest.MapFS{"a.tar.gz": {Data: archive}}, m, "linux", "amd64", dest)

	require.ErrorIs(t, err, ErrUnsafeArchive)
	require.NoDirExists(t, dest)
}
