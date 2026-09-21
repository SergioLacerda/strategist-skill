package runtimepayload

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// treeComponent pins the directory tree dir of src as one component.
func treeComponent(t *testing.T, src fstest.MapFS, name, goos, dir, dest string) Component {
	t.Helper()
	sum, size, err := TreeDigest(src, dir)
	require.NoError(t, err)
	return Component{Name: name, Version: "1.0.0", OS: goos, Arch: AnyTarget, File: dir, Format: FormatDir, SHA256: sum, Size: size, Dest: dest}
}

func testManifest(components ...Component) Manifest {
	return Manifest{SchemaVersion: SchemaVersion, Provider: "openspec-propose",
		Launcher:   Launcher{Node: map[string]string{"default": "openspec/dist/openspec.mjs"}, Script: "openspec/dist/openspec.mjs"},
		Components: components}
}

// fixture returns a manifest with a per-target component and a shared one, so
// target selection and verification stay covered.
func fixture(t *testing.T) (Manifest, fstest.MapFS) {
	t.Helper()
	src := fstest.MapFS{
		"linux/bin/tool":          {Data: []byte("TOOL"), Mode: 0o755},
		"windows/tool.exe":        {Data: []byte("TOOLEXE")},
		"spec/dist/openspec.mjs":  {Data: []byte("JS")},
		"spec/runtime.build.yaml": {Data: []byte("generated")},
	}
	m := testManifest(
		treeComponent(t, src, "tool", "linux", "linux", "tool"),
		treeComponent(t, src, "tool", "windows", "windows", "tool"),
		treeComponent(t, src, "openspec", AnyTarget, "spec", "openspec"),
	)
	return m, src
}

func TestMaterializeCopiesVerifiedRuntimeForTarget(t *testing.T) {
	m, src := fixture(t)
	dest := filepath.Join(t.TempDir(), "openspec-propose")

	ev, err := Materialize(src, m, "linux", "amd64", dest)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(dest, "tool", "bin", "tool"))
	require.NoFileExists(t, filepath.Join(dest, "tool", "tool.exe"), "only the target's component is copied")
	require.FileExists(t, filepath.Join(dest, "openspec", "dist", "openspec.mjs"))
	require.NoFileExists(t, filepath.Join(dest, "openspec", "runtime.build.yaml"))
	require.Len(t, ev.Components, 2)
	require.NotEmpty(t, ev.Components[0].SHA256)
}

func TestMaterializeSelectsTheWindowsComponent(t *testing.T) {
	m, src := fixture(t)
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "windows", "amd64", dest)

	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dest, "tool", "tool.exe"))
}

func TestMaterializeRejectsUnsupportedTarget(t *testing.T) {
	m, src := fixture(t)
	_, err := Materialize(src, m, "darwin", "arm64", filepath.Join(t.TempDir(), "rt"))
	require.ErrorIs(t, err, ErrTargetUnsupported)
}

func TestMaterializeRejectsMissingPayload(t *testing.T) {
	m, src := fixture(t)
	delete(src, "spec/dist/openspec.mjs")
	delete(src, "spec/runtime.build.yaml")
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrPayloadMissing)
	require.NoDirExists(t, dest, "nothing may be written when verification fails")
}

func TestMaterializeRejectsDigestMismatchBeforeWriting(t *testing.T) {
	m, src := fixture(t)
	src["linux/bin/tool"] = &fstest.MapFile{Data: []byte("tampered")}
	dest := filepath.Join(t.TempDir(), "rt")

	_, err := Materialize(src, m, "linux", "amd64", dest)
	require.ErrorIs(t, err, ErrDigestMismatch)
	require.NoDirExists(t, dest)
}

// A tree entry that is a link, or a name Windows cannot store, is refused and
// leaves nothing behind.
func TestMaterializeRejectsLinksAndReservedNames(t *testing.T) {
	for name, entry := range map[string]fstest.MapFS{
		"symlink":       {"rt/link": {Data: []byte("/etc/passwd"), Mode: fs.ModeSymlink}},
		"reserved name": {"rt/bin/NUL": {Data: []byte("x")}},
	} {
		t.Run(name, func(t *testing.T) {
			entry["rt/ok.txt"] = &fstest.MapFile{Data: []byte("ok")}
			m := testManifest(Component{Name: "openspec", Version: "1", OS: AnyTarget, Arch: AnyTarget, File: "rt", Format: FormatDir, SHA256: strings.Repeat("0", 64), Size: 1, Dest: "openspec"})
			dest := filepath.Join(t.TempDir(), "rt")

			err := copyTree(entry, "rt", dest, &budget{})

			require.ErrorIs(t, err, ErrUnsafeArchive)
			_, materializeErr := Materialize(entry, m, "linux", "amd64", dest+"-full")
			require.Error(t, materializeErr)
			require.NoDirExists(t, dest+"-full")
		})
	}
}

// The write target is re-checked at path level, independently of entryPath.
func TestContainedTargetRejectsEscapes(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{"../evil", "a/../../evil", ".."} {
		_, err := containedTarget(root, rel)
		require.ErrorIs(t, err, ErrUnsafeArchive, rel)
	}
	target, err := containedTarget(root, "dist/openspec.mjs")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "dist", "openspec.mjs"), target)
}

// Only directory trees ship; archive formats are refused at validation.
func TestManifestRejectsArchiveFormats(t *testing.T) {
	m, _ := fixture(t)
	for _, format := range []string{"tar.gz", "zip", ""} {
		bad := m
		bad.Components = []Component{m.Components[2]}
		bad.Components[0].Format = format

		require.ErrorContains(t, bad.Validate(), "unsupported format", format)
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
