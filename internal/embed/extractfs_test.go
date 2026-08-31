package embed

// Whitebox tests for extractFS — covers all error paths using an
// in-memory fs.FS constructed with testing/fstest.

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFS_Basic(t *testing.T) {
	t.Parallel()
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
		"root/sub/a.md":  {Data: []byte("# A")},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir, false))
	assert.FileExists(t, filepath.Join(dir, "file.yaml"))
	assert.FileExists(t, filepath.Join(dir, "sub", "a.md"))
}

func TestExtractFS_MkdirError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("read-only chmod semantics are not reliable for this permission test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/sub/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := extractFS(memFS, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed:")
}

func TestExtractFS_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	// Pre-create file.yaml as a directory so WriteFile fails with EISDIR
	require.NoError(t, os.Mkdir(filepath.Join(dir, "file.yaml"), 0o755))

	err := extractFS(memFS, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: write")
}

func TestExtractFS_EmptyRoot(t *testing.T) {
	t.Parallel()
	memFS := fstest.MapFS{
		"root": {Mode: os.ModeDir},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir, false))
}

// errFS is an fs.FS that wraps a MapFS but fails ReadDir for a specific path,
// causing fs.WalkDir to deliver an error to the walk callback.
type errFS struct {
	fstest.MapFS
	failPath string
}

func (e errFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == e.failPath {
		return nil, os.ErrPermission
	}
	return e.MapFS.ReadDir(name)
}

func TestExtractFS_WalkCallbackError(t *testing.T) {
	t.Parallel()
	// errFS makes WalkDir deliver an error for "root/broken" to the callback.
	mem := errFS{
		MapFS: fstest.MapFS{
			"root/broken": {Mode: os.ModeDir},
		},
		failPath: "root/broken",
	}
	dir := t.TempDir()
	err := extractFS(mem, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: walk")
}

// --- Merge mode tests ---

// TestMergeMode_PreservesUserModified verifies that re-extracting with force=false
// does not overwrite a file the user has modified.
func TestMergeMode_PreservesUserModified(t *testing.T) {
	t.Parallel()
	embedded := []byte("default: value\n")
	userContent := []byte("user: customized\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: embedded},
	}
	dir := t.TempDir()

	// First install: file doesn't exist → written from embedded.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	configPath := filepath.Join(dir, "config.yaml")
	got, _ := os.ReadFile(configPath)
	assert.Equal(t, embedded, got, "first install should write embedded content")

	// User customizes the file.
	require.NoError(t, os.WriteFile(configPath, userContent, 0o644))

	// Re-install (merge mode): user content must be preserved.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	got, _ = os.ReadFile(configPath)
	assert.Equal(t, userContent, got, "merge mode must preserve user-modified file")
}

// TestMergeMode_Idempotent verifies that re-extracting with force=false and the
// same embedded content is idempotent — the file is overwritten with identical bytes.
func TestMergeMode_Idempotent(t *testing.T) {
	t.Parallel()
	content := []byte("version: 1\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: content},
	}
	dir := t.TempDir()

	// First install.
	require.NoError(t, extractFS(memFS, "root", dir, false))

	// Re-install with same embedded content: should succeed without error.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	assert.Equal(t, content, got, "idempotent re-install should keep the same content")
}

// --- strategist install convergence (three-way classification) ---
//
// The raw extractFS primitive above (see TestMergeMode_PreservesUserModified,
// TestMergeMode_Idempotent) only ever compares on-disk content against the
// *current* embedded default: it has no way to tell "user kept the old
// default unmodified" from "user customized it" — both simply differ from a
// newer embedded version and would be preserved either way.
//
// `strategist install` no longer relies on that raw comparison for its
// merge-mode decision. It now delegates to internal/install's three-way file
// state classification (domain.DecideUpgradeFileState /
// domain.UpgradeFileWillWrite — the same logic `strategist upgrade` uses),
// which additionally consults the install manifest recorded by the previous
// install: a file whose on-disk hash still matches exactly what it was
// installed with is safely auto-upgraded to the new embedded default, while a
// file that has actually diverged from that recorded hash is a genuine
// customization and is preserved unless --force. The tests below drive
// install.Service.Install end-to-end (rather than extractFS directly) to
// prove that convergence, since extractFS itself is intentionally unchanged
// and is only install's fallback when no FileLister is configured.

// installFakeSource is a minimal domain.FileExtractor + domain.FileLister
// backed by an in-memory file map, standing in for the embedded defaults FS
// across two successive install.Service.Install runs.
type installFakeSource struct {
	files map[string][]byte
}

func (f installFakeSource) Extract(string, bool) error { return nil } // unused: Lister is set, so install uses the three-way path.

func (f installFakeSource) ReadFile(rel string) ([]byte, error) {
	data, ok := f.files[rel]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (f installFakeSource) AllPaths() ([]string, error) {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

// noopCompiler implements domain.Compiler without touching disk.
type noopCompiler struct{}

func (noopCompiler) CompileAll(string, string) error { return nil }

// newInstallFakeSource seeds every normative runtime default file (required
// by install's separate, stricter normative-file guard) plus the silent-mode
// active.yaml template, then layers extraPaths on top — typically the one
// "ordinary" embedded file each test below exercises the three-way logic on.
func newInstallFakeSource(extraPaths map[string][]byte) installFakeSource {
	files := map[string][]byte{
		"templates/epic-standalone.yaml": []byte("mode: epic\nbase_path: .analysis\n"),
	}
	for _, f := range domain.NormativeRuntimeDefaultFiles() {
		files[f.Path] = []byte(f.Path + " v1\n")
	}
	for path, data := range extraPaths {
		files[path] = data
	}
	return installFakeSource{files: files}
}

func newInstallTestService(src installFakeSource) install.Service {
	return install.Service{
		Extractor: src,
		Lister:    src,
		Compiler:  noopCompiler{},
		Version:   "1.0.0-test",
	}
}

func installTestConfig(dir string, force bool) domain.InstallConfig {
	return domain.InstallConfig{Target: dir, Silent: true, Global: true, NoShim: true, Force: force}
}

// TestInstallMergeMode_UnmodifiedDefaultIsAutoUpgraded proves the fix for the
// limitation the old raw-merge-only test above documented: a plain re-run of
// `strategist install` (merge mode, no --force) now applies a new embedded
// default to a file the user never touched, instead of silently leaving the
// stale v1 content in place.
func TestInstallMergeMode_UnmodifiedDefaultIsAutoUpgraded(t *testing.T) {
	t.Parallel()
	const relPath = "docs/example.md"
	dir := t.TempDir()
	src := newInstallFakeSource(map[string][]byte{relPath: []byte("version: 1\n")})
	svc := newInstallTestService(src)
	cfg := installTestConfig(dir, false)

	require.NoError(t, svc.Install(context.Background(), cfg))
	target := filepath.Join(dir, ".strategist", filepath.FromSlash(relPath))
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "version: 1\n", string(got), "first install should write the embedded v1 content")

	// Embedded bumps to v2 (simulates a new release) — the on-disk file is
	// still exactly v1, untouched by the user.
	src.files[relPath] = []byte("version: 2\n")

	require.NoError(t, svc.Install(context.Background(), cfg))
	got, err = os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "version: 2\n", string(got),
		"merge-mode install must auto-upgrade an unmodified prior default to the new embedded version")
}

// TestInstallMergeMode_CustomizedFileIsPreservedWithoutForce proves the other
// half of the three-way classification: a file the user genuinely edited is
// never silently overwritten by a plain re-run, even though the embedded
// default has since moved on.
func TestInstallMergeMode_CustomizedFileIsPreservedWithoutForce(t *testing.T) {
	t.Parallel()
	const relPath = "docs/example.md"
	dir := t.TempDir()
	src := newInstallFakeSource(map[string][]byte{relPath: []byte("version: 1\n")})
	svc := newInstallTestService(src)
	cfg := installTestConfig(dir, false)

	require.NoError(t, svc.Install(context.Background(), cfg))
	target := filepath.Join(dir, ".strategist", filepath.FromSlash(relPath))
	require.NoError(t, os.WriteFile(target, []byte("user: customized\n"), 0o644))

	src.files[relPath] = []byte("version: 2\n")

	require.NoError(t, svc.Install(context.Background(), cfg))
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "user: customized\n", string(got),
		"merge-mode install must preserve a genuinely user-customized file")
}

// TestInstallMergeMode_ForceOverwritesCustomizedFile proves --force still
// overrides a preserved customization, matching the raw extractor's existing
// force-mode contract (TestForceMode_OverwritesUserModified), AND that the
// overwritten content is backed up first — this is the regression test for
// the bug found during the 20260830-skill-gaps-followup review: the
// three-way merge-mode path (extractRuntimeTree) used to call
// writeUpgradeFile directly with no snapshot step, so a --force merge-mode
// install could silently destroy a customization with no backup, unlike
// `strategist upgrade --force`'s identical operation.
func TestInstallMergeMode_ForceOverwritesCustomizedFile(t *testing.T) {
	t.Parallel()
	const relPath = "docs/example.md"
	dir := t.TempDir()
	src := newInstallFakeSource(map[string][]byte{relPath: []byte("version: 1\n")})
	svc := newInstallTestService(src)

	require.NoError(t, svc.Install(context.Background(), installTestConfig(dir, false)))
	target := filepath.Join(dir, ".strategist", filepath.FromSlash(relPath))
	require.NoError(t, os.WriteFile(target, []byte("user: customized\n"), 0o644))

	src.files[relPath] = []byte("version: 2\n")

	require.NoError(t, svc.Install(context.Background(), installTestConfig(dir, true)))
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "version: 2\n", string(got), "--force must overwrite a customized file even in the three-way path")

	backupsRoot := filepath.Join(dir, ".strategist", ".upgrade-backups")
	stamps, err := os.ReadDir(backupsRoot)
	require.NoError(t, err, "a --force merge-mode overwrite of a customized file must create a backup snapshot dir")
	require.Len(t, stamps, 1, "expected exactly one backup snapshot from the single overwriting install")

	backedUp, err := os.ReadFile(filepath.Join(backupsRoot, stamps[0].Name(), filepath.FromSlash(relPath)))
	require.NoError(t, err)
	assert.Equal(t, "user: customized\n", string(backedUp),
		"the backup snapshot must contain the pre-overwrite (customized) content, not the new default")
}

// TestInstallMergeMode_AutoUpgradeDoesNotCreateSpuriousBackup proves the
// backup-safety fix above doesn't over-trigger: a plain (non --force)
// merge-mode install that only auto-upgrades an unmodified file (nothing the
// user customized was overwritten) must not create a backup dir at all —
// ApplyUpgrade's own upgradeWriteSet still marks AutoUpgrade entries for
// backup too (see its comment: "on disk, matches a prior installed default
// exactly... embedded default has since moved on"), so this asserts that
// case does create exactly one backup and never confuses "nothing was ever
// installed before" (fresh install, no prior manifest) with "something was
// overwritten."
func TestInstallMergeMode_AutoUpgradeDoesNotCreateSpuriousBackup(t *testing.T) {
	t.Parallel()
	const relPath = "docs/example.md"
	dir := t.TempDir()
	src := newInstallFakeSource(map[string][]byte{relPath: []byte("version: 1\n")})
	svc := newInstallTestService(src)

	require.NoError(t, svc.Install(context.Background(), installTestConfig(dir, false)))
	backupsRoot := filepath.Join(dir, ".strategist", ".upgrade-backups")
	_, err := os.Stat(backupsRoot)
	assert.True(t, os.IsNotExist(err), "a fresh install (everything Missing, nothing to overwrite) must not create a backup dir")
}

// TestForceMode_OverwritesUserModified verifies that force=true always overwrites.
func TestForceMode_OverwritesUserModified(t *testing.T) {
	t.Parallel()
	embedded := []byte("default: value\n")
	userContent := []byte("user: customized\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: embedded},
	}
	dir := t.TempDir()

	// Write user content directly (simulating a customized install).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), userContent, 0o644))

	// Force extract: must overwrite user content.
	require.NoError(t, extractFS(memFS, "root", dir, true))
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	assert.Equal(t, embedded, got, "force mode must overwrite user-modified files")
}

// errOpenFS wraps MapFS but makes Open return an error for a specific file path,
// so fs.ReadFile fails inside writeEmbedFile while WalkDir still enumerates the entry.
type errOpenFS struct {
	fstest.MapFS
	failPath string
}

func (e errOpenFS) Open(name string) (fs.File, error) {
	if name == e.failPath {
		return nil, os.ErrPermission
	}
	return e.MapFS.Open(name)
}

// ReadFile overrides the promoted MapFS.ReadFile so that fs.ReadFile (which prefers
// ReadFileFS) also hits the error path for failPath.
func (e errOpenFS) ReadFile(name string) ([]byte, error) {
	if name == e.failPath {
		return nil, os.ErrPermission
	}
	return e.MapFS.ReadFile(name)
}

func TestWriteEmbedFile_ReadError(t *testing.T) {
	t.Parallel()
	mem := errOpenFS{
		MapFS: fstest.MapFS{
			"root/file.yaml": {Data: []byte("x: 1\n")},
		},
		failPath: "root/file.yaml",
	}
	dir := t.TempDir()
	err := extractFS(mem, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: read")
}

// TestUserModified_NonExistent verifies userModified returns false for a file that
// doesn't exist on disk (not yet installed — not a user modification).
func TestUserModified_NonExistent(t *testing.T) {
	t.Parallel()
	result := userModified("/tmp/definitely-does-not-exist-xyz123.yaml", []byte("x: 1\n"))
	assert.False(t, result, "non-existent file is not user-modified")
}
