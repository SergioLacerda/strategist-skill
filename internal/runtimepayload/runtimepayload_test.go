package runtimepayload

import (
	"errors"
	"io/fs"
	"os"
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

func TestRegisterAndDefaultOpenSpec(t *testing.T) {
	RegisterOpenSpec(nil)
	_, ok := DefaultOpenSpec()
	require.False(t, ok)

	src := fstest.MapFS{}
	RegisterOpenSpec(src)
	got, ok := DefaultOpenSpec()
	require.True(t, ok)
	require.Equal(t, src, got)

	// Reset
	RegisterOpenSpec(nil)
}

func TestEmbeddedOpenSpecVersion_Errors(t *testing.T) {
	missingFS := fstest.MapFS{}
	_, err := EmbeddedOpenSpecVersion(missingFS)
	require.ErrorIs(t, err, ErrPayloadMissing)

	badYamlFS := fstest.MapFS{
		openSpecTreeDir + "/" + BuildInfoFile: {Data: []byte("version: [invalid")},
	}
	_, err = EmbeddedOpenSpecVersion(badYamlFS)
	require.Error(t, err)

	emptyVerFS := fstest.MapFS{
		openSpecTreeDir + "/" + BuildInfoFile: {Data: []byte("version: '  '")},
	}
	_, err = EmbeddedOpenSpecVersion(emptyVerFS)
	require.ErrorContains(t, err, "no version")
}

func TestMaterializeOpenSpec_Errors(t *testing.T) {
	missingFS := fstest.MapFS{}
	_, _, err := MaterializeOpenSpec(missingFS, t.TempDir())
	require.ErrorIs(t, err, ErrPayloadMissing)
}

func TestVerifyMaterializedOpenSpec_TreeDigestError(t *testing.T) {
	err := VerifyMaterializedOpenSpec(filepath.Join(t.TempDir(), "nonexistent"), "somehash")
	require.Error(t, err)
}

func TestBudgetLimits(t *testing.T) {
	b := &budget{files: maxExtractedFiles}
	require.ErrorIs(t, b.file(), ErrUnsafeArchive)

	bBytes := &budget{bytes: maxExtractedBytes}
	buf := strings.NewReader("hello")
	err := copyToFile(filepath.Join(t.TempDir(), "f.txt"), 0o644, buf, bBytes)
	require.ErrorIs(t, err, ErrUnsafeArchive)
}

func TestEntryPerm(t *testing.T) {
	require.Equal(t, fs.FileMode(0o755), entryPerm(0o755))
	require.Equal(t, fs.FileMode(0o644), entryPerm(0o644))
}

func TestManifestValidationEdgeCases(t *testing.T) {
	m := Manifest{
		SchemaVersion: SchemaVersion,
		Provider:      "",
		Launcher:      Launcher{Node: map[string]string{"default": "app.js"}, Script: "app.js"},
		Components: []Component{
			{Name: "c", Version: "1", OS: AnyTarget, Arch: AnyTarget, File: "f", Format: FormatDir, SHA256: strings.Repeat("a", 64), Size: 10, Dest: "d"},
		},
	}
	require.ErrorContains(t, m.Validate(), "provider and components are required")

	m.Provider = "p"
	m.Launcher.Node = map[string]string{}
	require.ErrorContains(t, m.Validate(), "launcher needs node.default")

	m.Launcher.Node = map[string]string{"default": "/absolute/path"}
	require.ErrorContains(t, m.Validate(), "must be clean and relative")

	m.Launcher.Node = map[string]string{"default": "app.js"}
	m.Components[0].Name = ""
	require.ErrorContains(t, m.Validate(), "name, version, os and arch are required")

	m.Components[0].Name = "c"
	m.Components[0].File = "."
	require.ErrorContains(t, m.Validate(), "file \".\" must be a clean relative path")

	m.Components[0].File = "f"
	m.Components[0].Size = 0
	require.ErrorContains(t, m.Validate(), "size must be positive")

	m.Components[0].Size = 10
	m.Components[0].Dest = "/dest"
	require.ErrorContains(t, m.Validate(), "dest \"/dest\" must be clean and relative")
}

func TestEntryPathStripAndTraversal(t *testing.T) {
	_, ok, err := entryPath("a/b", 2)
	require.NoError(t, err)
	require.False(t, ok)

	_, _, err = entryPath("a/../..", 1)
	require.ErrorIs(t, err, ErrUnsafeArchive)

	_, _, err = entryPath("", 0)
	require.ErrorIs(t, err, ErrUnsafeArchive)

	_, _, err = entryPath("/abs", 0)
	require.ErrorIs(t, err, ErrUnsafeArchive)

	_, _, err = entryPath("drive:path", 0)
	require.ErrorIs(t, err, ErrUnsafeArchive)
}

type customErrCloser struct {
	err error
}

func (c customErrCloser) Close() error {
	return c.err
}

type errReader struct{}

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("read failure")
}

func TestCloseInto(t *testing.T) {
	var err1 error
	closeInto(customErrCloser{err: nil}, "test1", &err1)
	require.NoError(t, err1)

	var err2 error
	closeInto(customErrCloser{err: errors.New("close failed")}, "test2", &err2)
	require.ErrorContains(t, err2, "close test2: close failed")

	initialErr := errors.New("initial failure")
	err3 := initialErr
	closeInto(customErrCloser{err: errors.New("close failed")}, "test3", &err3)
	require.Equal(t, initialErr, err3)
}

func TestCleanupStaging_OnExtractionFailure(t *testing.T) {
	src := fstest.MapFS{
		"spec/link": {Data: []byte("/etc/passwd"), Mode: fs.ModeSymlink},
	}
	m := testManifest(Component{
		Name: "openspec", Version: "1", OS: AnyTarget, Arch: AnyTarget,
		File: "spec", Format: FormatDir, SHA256: strings.Repeat("0", 64), Size: 1, Dest: "openspec",
	})
	dest := filepath.Join(t.TempDir(), "target")
	err := activate(src, m.Components, dest)
	require.Error(t, err)
	require.NoDirExists(t, dest+".staging", "staging directory must be cleaned up on failure")
}

func TestCopyToFile_Errors(t *testing.T) {
	tempDir := t.TempDir()

	filePathAsDir := filepath.Join(tempDir, "file_blocking_dir")
	require.NoError(t, os.WriteFile(filePathAsDir, []byte("block"), 0o644))
	targetUnderFile := filepath.Join(filePathAsDir, "sub.txt")
	err := copyToFile(targetUnderFile, 0o644, strings.NewReader("data"), &budget{})
	require.Error(t, err)
	require.ErrorContains(t, err, "create")

	targetFile := filepath.Join(tempDir, "read_err.txt")
	err = copyToFile(targetFile, 0o644, errReader{}, &budget{})
	require.Error(t, err)
	require.ErrorContains(t, err, "write")
}

func TestWriteEntry_ErrorPaths(t *testing.T) {
	tempDir := t.TempDir()
	b := &budget{}

	filePathAsDir := filepath.Join(tempDir, "file_parent")
	require.NoError(t, os.WriteFile(filePathAsDir, []byte("block"), 0o644))
	blockedName := filepath.Join("file_parent", "child.txt")
	err := writeEntry(tempDir, blockedName, 0, 0o644, strings.NewReader("hi"), b)
	require.Error(t, err)

	limitBudget := &budget{files: maxExtractedFiles}
	err = writeEntry(tempDir, "ok.txt", 0, 0o644, strings.NewReader("hi"), limitBudget)
	require.ErrorIs(t, err, ErrUnsafeArchive)

	err = writeEntry(tempDir, "../evil.txt", 0, 0o644, strings.NewReader("hi"), b)
	require.ErrorIs(t, err, ErrUnsafeArchive)
}

func TestReadYAMLErrors(t *testing.T) {
	var into struct {
		Field string `yaml:"field"`
	}
	missingFS := fstest.MapFS{}
	err := readYAML(missingFS, "nonexistent.yaml", &into)
	require.ErrorIs(t, err, ErrPayloadMissing)

	badFS := fstest.MapFS{
		"bad.yaml": {Data: []byte("field: [unclosed list")},
	}
	err = readYAML(badFS, "bad.yaml", &into)
	require.Error(t, err)
	require.ErrorContains(t, err, "parse bad.yaml")
}

func TestWindowsHostileSegment_StemCheck(t *testing.T) {
	hostileNames := []string{
		"CON.txt", "con.log", "NUL.dat", "AUX.yaml", "PRN.csv",
		"COM1.bin", "com9.exe", "LPT1.dll", "lpt9.sh",
	}
	for _, name := range hostileNames {
		_, _, err := entryPath(name, 0)
		require.ErrorIs(t, err, ErrUnsafeArchive, name)
	}
}

func TestManifest_Marshal(t *testing.T) {
	m, _ := fixture(t)
	raw, err := m.Marshal()
	require.NoError(t, err)
	require.Contains(t, string(raw), "schema_version")
}

type readErrFS struct {
	fstest.MapFS
}

func (m readErrFS) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, "fail.txt") {
		return nil, errors.New("open failure")
	}
	return m.MapFS.Open(name)
}

func (m readErrFS) ReadFile(name string) ([]byte, error) {
	if strings.HasSuffix(name, "fail.txt") {
		return nil, errors.New("read failure")
	}
	return m.MapFS.ReadFile(name)
}

func TestTreeDigest_ReadError(t *testing.T) {
	sys := readErrFS{
		MapFS: fstest.MapFS{
			"rt/fail.txt": {Data: []byte("fail")},
		},
	}
	_, _, err := TreeDigest(sys, "rt")
	require.Error(t, err)
	require.ErrorContains(t, err, "read fail.txt")

	b := &budget{}
	err = copyTreeFile(sys, "rt", t.TempDir(), "fail.txt", b)
	require.Error(t, err)
	require.ErrorContains(t, err, "open fail.txt")
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	_, err := ParseManifest([]byte(":\n  - : : invalid yaml"))
	require.Error(t, err)
	require.ErrorContains(t, err, "parse runtime payload manifest")
}

func TestMaterializeOpenSpec_MaterializeError(t *testing.T) {
	badFS := fstest.MapFS{
		openSpecTreeDir + "/" + BuildInfoFile: {
			Data: []byte("version: '1.0.0'\nbundle: 'openspec.mjs'\ntree_sha256: 'badsha'\ntree_bytes: 10\n"),
		},
		openSpecTreeDir + "/openspec.mjs": {Data: []byte("console.log('hi');")},
	}
	_, _, err := MaterializeOpenSpec(badFS, t.TempDir())
	require.Error(t, err)
}

func TestWindowsHostileSegment_EdgeCases(t *testing.T) {
	require.False(t, windowsHostileSegment(""))
	require.True(t, windowsHostileSegment("file."))
	require.True(t, windowsHostileSegment("file "))
	for _, res := range []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM9", "LPT1", "LPT9"} {
		require.True(t, windowsHostileSegment(res), res)
		require.True(t, windowsHostileSegment(strings.ToLower(res)), res)
		require.True(t, windowsHostileSegment(res+".ext"), res+".ext")
	}
	require.False(t, windowsHostileSegment("normal.txt"))
}

func TestWriteEntry_StripFully(t *testing.T) {
	b := &budget{}
	err := writeEntry(t.TempDir(), "a/b", 5, 0o644, strings.NewReader("hi"), b)
	require.NoError(t, err)
}

func TestSwapIn_Error(t *testing.T) {
	tempDir := t.TempDir()
	staging := filepath.Join(tempDir, "staging")
	require.NoError(t, os.MkdirAll(staging, 0o755))

	badDest := filepath.Join(tempDir, "blocking_file", "dest")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "blocking_file"), []byte("file"), 0o644))

	err := swapIn(staging, badDest)
	require.Error(t, err)
}

func TestEmbeddedOpenSpecVersion_Success(t *testing.T) {
	goodFS := fstest.MapFS{
		openSpecTreeDir + "/" + BuildInfoFile: {
			Data: []byte("version: '1.13.0'\nbundle: 'openspec.mjs'\ntree_sha256: 'abc'\ntree_bytes: 100\n"),
		},
	}
	ver, err := EmbeddedOpenSpecVersion(goodFS)
	require.NoError(t, err)
	require.Equal(t, "1.13.0", ver)
}

func TestCleanupStaging_RemoveError(t *testing.T) {
	tempDir := t.TempDir()
	staging := filepath.Join(tempDir, "read_only_staging")
	sub := filepath.Join(staging, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "file"), []byte("x"), 0o444))
	require.NoError(t, os.Chmod(sub, 0o555))
	require.NoError(t, os.Chmod(staging, 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(staging, 0o755)
		_ = os.Chmod(sub, 0o755)
	})

	cause := errors.New("original cause")
	err := cleanupStaging(staging, cause)
	require.Error(t, err)
	require.ErrorIs(t, err, cause)
}

func TestEntryPath_DotEdgeCase(t *testing.T) {
	_, _, err := entryPath("dir/.", 1)
	require.ErrorIs(t, err, ErrUnsafeArchive)
}
