package acquisition

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationAndErrorBranches(t *testing.T) {
	assert.Equal(t, "acquisition_empty", (Error{Code: "empty"}).Error())
	assert.Equal(t, "acquisition_invalid: detail", (Error{Code: "invalid", Detail: "detail"}).Error())

	for _, repository := range []string{"", " https://example.com/repo", "-repo", "https://", "ftp://example.com/repo", "https://@example.com/repo", "https://example.com/repo name", "https://example.com/repo\nname"} {
		assertCodedError(t, ValidateRepository(repository), "repository_invalid")
	}
	for _, id := range []string{"", " skill", "skill ", "skill name", "skill\nname", "../skill", "/skill", strings.Repeat("x", 257)} {
		assertCodedError(t, ValidatePackageID(id), "package_id_invalid")
	}
	for _, commit := range []string{"", strings.Repeat("a", 39), strings.Repeat("a", 41), strings.Repeat("g", 40)} {
		assertCodedError(t, ValidateCommit(commit), "commit_invalid")
	}

	assert.NoError(t, ValidateRepository("https://github.com/example/repo"))
	assert.NoError(t, ValidateRepository("ssh://git@example.com/repo"))
	assert.NoError(t, ValidateRepository("git@example.com:repo"))
	assert.NoError(t, ValidateRepository(filepath.Join(t.TempDir(), "repo")))

	assertCodedError(t, validateRequest(Request{}), "source_unsupported")
	assertCodedError(t, validateRequest(Request{Source: SourceGit}), "repository_invalid")
	assertCodedError(t, validateRequest(Request{Source: SourceGit, Repository: "https://example.com/repo"}), "commit_invalid")
	assertCodedError(t, validateRequest(Request{Source: SourceGit, Repository: "https://example.com/repo", Commit: testCommit}), "package_id_invalid")
	assertCodedError(t, validateRequest(Request{Source: SourceGit, Repository: "https://example.com/repo", Commit: testCommit, PackageID: "skill"}), "network_opt_in_required")
}

func TestArchiveFailureBranches(t *testing.T) {
	assertCodedError(t, extractArchive(t.TempDir(), []byte("not a tar")), "archive_invalid")
	assertCodedError(t, extractArchive(t.TempDir(), make([]byte, maxArchiveBytes+1)), "archive_too_large")
	_, err := cleanRelativeArchivePath("")
	assertCodedError(t, err, "archive_path_invalid")
	_, err = cleanRelativeArchivePath("bad\x00name")
	assertCodedError(t, err, "archive_path_invalid")
	_, err = cleanRelativeArchivePath("../name")
	assertCodedError(t, err, "archive_path_invalid")
	_, err = cleanRelativeArchivePath("folder/../../name")
	assertCodedError(t, err, "archive_path_invalid")

	for _, entry := range []struct {
		name string
		make func(*tar.Writer) error
		want string
	}{
		{"hardlink", func(w *tar.Writer) error {
			return w.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "SKILL.md"})
		}, "unsafe_entry"},
		{"unsupported", func(w *tar.Writer) error { return w.WriteHeader(&tar.Header{Name: "device", Typeflag: tar.TypeChar}) }, "unsafe_entry"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			assertCodedError(t, extractArchive(t.TempDir(), buildTar(t, entry.make)), entry.want)
		})
	}

	root := t.TempDir()
	err = extractArchiveEntry(root, tar.NewReader(bytes.NewReader(nil)), &tar.Header{Name: "../outside", Typeflag: tar.TypeDir}, "../outside")
	assertCodedError(t, err, "archive_path_invalid")
	err = extractArchiveEntry(root, tar.NewReader(bytes.NewReader(nil)), &tar.Header{Name: "large", Typeflag: tar.TypeReg, Size: maxArchiveBytes + 1}, "large")
	assertCodedError(t, err, "archive_file_too_large")
	err = extractRegularFile(filepath.Join(root, "short"), tar.NewReader(bytes.NewReader(nil)), 1, "short")
	assertCodedError(t, err, "archive_invalid")
	require.NoError(t, os.WriteFile(filepath.Join(root, "dir"), []byte("x"), 0o600))
	assertCodedError(t, extractArchiveEntry(root, tar.NewReader(bytes.NewReader(nil)), &tar.Header{Name: "dir/file", Typeflag: tar.TypeReg, Size: 0}, "dir/file"), "archive_write_failed")
	require.NoError(t, os.WriteFile(filepath.Join(root, "existing"), []byte("x"), 0o600))
	assertCodedError(t, extractArchiveEntry(root, tar.NewReader(bytes.NewReader(nil)), &tar.Header{Name: "existing", Typeflag: tar.TypeReg, Size: 0}, "existing"), "archive_write_failed")
	assertCodedError(t, makeArchiveDirectory(filepath.Join(root, "existing", "child")), "archive_write_failed")
	assertCodedError(t, validateArchiveFileSize(-1, "negative"), "archive_file_too_large")
	_, err = nextArchivePath(&tar.Header{Name: "x"}, map[string]struct{}{}, new(int))
	assertCodedError(t, err, "")
	files := maxArchiveFiles
	_, err = nextArchivePath(&tar.Header{Name: "x"}, map[string]struct{}{}, &files)
	assertCodedError(t, err, "archive_too_many_files")

	assert.True(t, isPAXHeader(&tar.Header{Typeflag: tar.TypeXHeader}))
	assert.True(t, isPAXHeader(&tar.Header{Typeflag: tar.TypeXGlobalHeader}))
	assert.False(t, isPAXHeader(&tar.Header{Typeflag: tar.TypeReg}))
	longName := strings.Repeat("a", 120)
	require.NoError(t, extractArchive(t.TempDir(), buildTar(t, func(w *tar.Writer) error {
		return writeTarFile(w, longName, []byte("pax"))
	})))
}

func TestDigestAndManifestFailureBranches(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("data"), 0o640))
	require.NoError(t, os.Mkdir(filepath.Join(root, "folder"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, acquisitionManifestName), []byte("ignored"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, readyMarkerName), []byte("ignored"), 0o600))
	digest, files, err := deterministicTreeDigest(root)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(digest, "sha256:"))
	assert.Len(t, files, 2)

	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "file.txt"), link); err == nil {
		_, _, err = deterministicTreeDigest(root)
		assertCodedError(t, err, "digest_failed")
		assert.NoError(t, os.Remove(link))
	}
	_, _, err = deterministicTreeDigest(filepath.Join(root, "missing"))
	require.Error(t, err)

	_, err = collectTreeEntry(root, filepath.Join(root, "file.txt"), nil, errors.New("walk failed"))
	require.Error(t, err)
	entry, err := collectTreeEntry(root, root, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entry)
	for _, name := range []string{acquisitionManifestName, readyMarkerName} {
		entry, err = collectTreeEntry(root, filepath.Join(root, name), fakeEntry{name: name}, nil)
		require.NoError(t, err)
		assert.Nil(t, entry)
	}

	_, err = treeEntryFromDirent("missing", "missing", fakeEntry{mode: os.ModeSymlink})
	require.Error(t, err)
	_, err = treeEntryFromDirent("missing", "missing", fakeEntry{infoErr: errors.New("inspect failed")})
	require.Error(t, err)
	entry, err = treeEntryFromDirent("folder", "folder", fakeEntry{info: fileInfo{mode: os.ModeDir | 0o750}})
	require.NoError(t, err)
	assert.Equal(t, byte('d'), entry.Kind)
	_, err = treeEntryFromDirent("device", "device", fakeEntry{info: fileInfo{mode: os.ModeNamedPipe}})
	assertCodedError(t, err, "unsafe_entry")
	_, err = treeEntryFromDirent("missing", "missing", fakeEntry{info: fileInfo{mode: 0o600}})
	require.Error(t, err)

	manifest := newManifest(Request{Source: SourceGit, Repository: "repo", Commit: testCommit, PackageID: "skill"}, "sha256:digest", files, time.Unix(0, 0))
	require.NoError(t, validateManifest(manifest))
	manifest.Source = "archive"
	require.Error(t, validateManifest(manifest))
	require.Error(t, writeManifest(filepath.Join(root, "missing"), manifest))
	_, err = createManifestTemp(filepath.Join(root, "missing"), []byte("data"))
	require.Error(t, err)
	require.NoError(t, writeManifest(root, manifest))
	removeManifestTemp(filepath.Join(root, "missing"))
	require.NoError(t, writeReady(root))
	require.Error(t, writeReady(filepath.Join(root, "missing")))
	manifestPath := filepath.Join(root, acquisitionManifestName)
	require.NoError(t, os.Remove(manifestPath))
	require.NoError(t, os.Mkdir(manifestPath, 0o700))
	assert.Error(t, writeManifest(root, manifest))
}

func TestGitAndQuarantineFailureBranches(t *testing.T) {
	work := t.TempDir()
	fileWork := filepath.Join(work, "file")
	require.NoError(t, os.WriteFile(fileWork, []byte("x"), 0o600))
	_, err := createGitWorkDir(fileWork)
	assertCodedError(t, err, "git_workspace_failed")

	for _, failAt := range []string{"init", "remote", "fetch"} {
		runner := &failingCommandRunner{failAt: failAt}
		assertCodedError(t, initializeGitWorkDir(context.Background(), runner, work, "repo", testCommit), "git_command_failed")
	}
	for _, test := range []struct {
		name   string
		output []byte
		failAt string
		want   string
	}{
		{"resolve command", []byte{}, "rev-parse", "git_command_failed"},
		{"resolve value", []byte("not-a-commit"), "", "commit_resolution_invalid"},
		{"archive command", []byte(testCommit), "archive", "git_command_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &failingCommandRunner{failAt: test.failAt, resolve: test.output}
			_, _, err := archiveResolvedCommit(context.Background(), runner, work)
			assertCodedError(t, err, test.want)
		})
	}
	_, _, err = (CommandGitRunner{Command: &failingCommandRunner{failAt: "init"}}).FetchArchive(context.Background(), work, "repo", testCommit, true)
	assertCodedError(t, err, "git_command_failed")
	_, _, err = (CommandGitRunner{Command: &failingCommandRunner{}}).FetchArchive(context.Background(), fileWork, "repo", testCommit, true)
	assertCodedError(t, err, "git_workspace_failed")
	_, err = OSCommandRunner{}.Run(context.Background(), filepath.Join(work, "missing"), "version")
	require.Error(t, err)
	_, err = OSCommandRunner{}.Run(context.Background(), work, "definitely-not-a-git-command")
	require.Error(t, err)

	_, err = createAcquisitionTransaction(filepath.Join(work, "missing"))
	assertCodedError(t, err, "quarantine_create_failed")
	_, err = prepareQuarantineRoot(filepath.Join(work, "missing", "nested"), nil)
	assertCodedError(t, err, "")
	managed := t.TempDir()
	assertCodedError(t, rejectUnderForbiddenRoot(filepath.Join(managed, "nested"), []string{"", managed}), "quarantine_root_forbidden")
	assert.NoError(t, rejectUnderForbiddenRoot(filepath.Join(work, "outside"), []string{managed}))
	assert.NoError(t, rejectUnderForbiddenRootEntry(filepath.Join(work, "outside"), managed))
	assert.True(t, isWithinRoot("."))
	assert.True(t, isWithinRoot("child"))
	assert.False(t, isWithinRoot(".."))
	assert.False(t, isWithinRoot(filepath.Join("..", "child")))

	link := filepath.Join(work, "root-link")
	if err := os.Symlink(managed, link); err == nil {
		resolved, resolveErr := canonicalPath(link)
		require.NoError(t, resolveErr)
		assert.Equal(t, managed, resolved)
	}

	shape := t.TempDir()
	assertCodedError(t, validatePackageShape(shape), "package_shape_invalid")
	require.NoError(t, os.WriteFile(filepath.Join(shape, "SKILL.md"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(shape, "strategist.yaml"), nil, 0o600))
	assertCodedError(t, validatePackageShape(shape), "package_shape_invalid")
	require.NoError(t, os.WriteFile(filepath.Join(shape, "strategist.yaml"), []byte("x"), 0o600))
	require.NoError(t, os.Remove(filepath.Join(shape, "SKILL.md")))
	require.NoError(t, os.Mkdir(filepath.Join(shape, "SKILL.md"), 0o700))
	assertCodedError(t, validatePackageShape(shape), "package_shape_invalid")

	_, err = finishAcquisition(context.Background(), Request{Commit: testCommit}, t.TempDir())
	assertCodedError(t, err, "network_opt_in_required")
	transaction := t.TempDir()
	_, err = finishAcquisition(context.Background(), Request{Commit: testCommit, Runner: fakeGitRunner{archive: validPackageArchive(t), resolved: strings.Repeat("a", 40)}}, transaction)
	assertCodedError(t, err, "commit_mismatch")
}

func TestFinalizeAcquisitionFailureBranches(t *testing.T) {
	validTransaction := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "strategist.yaml"), []byte("role: ranger"), 0o600))
		return root
	}
	request := Request{Source: SourceGit, Repository: "https://example.com/repo", Commit: testCommit, PackageID: "skill"}
	invalidManifest := validTransaction(t)
	_, err := finalizeAcquisition(Request{Source: "archive", Commit: testCommit, PackageID: "skill"}, invalidManifest)
	assertCodedError(t, err, "manifest_invalid")

	manifestFailure := validTransaction(t)
	require.NoError(t, os.Mkdir(filepath.Join(manifestFailure, acquisitionManifestName), 0o700))
	_, err = finalizeAcquisition(request, manifestFailure)
	assertCodedError(t, err, "manifest_write_failed")

	readyFailure := validTransaction(t)
	require.NoError(t, os.Mkdir(filepath.Join(readyFailure, readyMarkerName), 0o700))
	_, err = finalizeAcquisition(request, readyFailure)
	assertCodedError(t, err, "ready_write_failed")

	digestFailure := validTransaction(t)
	if err := os.Symlink(filepath.Join(digestFailure, "missing"), filepath.Join(digestFailure, "link")); err == nil {
		_, err = finalizeAcquisition(request, digestFailure)
		assertCodedError(t, err, "digest_failed")
	}
}

func TestAcquireRejectsUnusableQuarantineRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(root, []byte("not a directory"), 0o600))
	_, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: "https://example.com/repo", Commit: testCommit,
		PackageID: "skill", AllowNetwork: true, QuarantineRoot: root, Runner: fakeGitRunner{},
	})
	assertCodedError(t, err, "quarantine_root_invalid")
}

func TestRemainingFilesystemAndTransactionBranches(t *testing.T) {
	root := t.TempDir()
	assert.NoError(t, makeArchiveDirectory(filepath.Join(root, "nested", "directory")))
	assert.NoError(t, extractArchiveEntry(root, tar.NewReader(bytes.NewReader(nil)), &tar.Header{Name: "empty", Typeflag: tar.TypeDir}, "empty"))
	assertCodedError(t, ensureContained(root, filepath.Join(filepath.Dir(root), "outside")), "archive_path_invalid")

	temp, err := createManifestTemp(root, []byte("temporary"))
	require.NoError(t, err)
	removeManifestTemp(temp)
	assert.NoFileExists(t, temp)
	directoryTemp := filepath.Join(root, "temporary-directory")
	require.NoError(t, os.Mkdir(directoryTemp, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directoryTemp, "file"), []byte("x"), 0o600))
	removeManifestTemp(directoryTemp)

	transaction := filepath.Join(root, "transaction")
	require.NoError(t, os.Mkdir(transaction, 0o700))
	resultErr := errors.New("failed")
	removeFailedAcquisition(transaction, &resultErr)
	assert.NoDirExists(t, transaction)
	var noErr error
	removeFailedAcquisition(transaction, &noErr)

	_, err = prepareQuarantineRoot("", nil)
	require.NoError(t, err)
	fileRoot := filepath.Join(root, "root-file")
	require.NoError(t, os.WriteFile(fileRoot, []byte("x"), 0o600))
	_, err = prepareQuarantineRoot(fileRoot, nil)
	assertCodedError(t, err, "quarantine_root_invalid")

	_, err = finishAcquisition(context.Background(), Request{
		Commit: testCommit,
		Runner: fakeGitRunner{archive: []byte("invalid tar"), resolved: testCommit},
	}, t.TempDir())
	assertCodedError(t, err, "archive_invalid")
	_, err = finishAcquisition(context.Background(), Request{
		Commit: testCommit,
		Runner: fakeGitRunner{archive: buildTar(t, func(w *tar.Writer) error {
			return writeTarFile(w, "SKILL.md", []byte("only one file"))
		}), resolved: testCommit},
	}, t.TempDir())
	assertCodedError(t, err, "package_shape_invalid")
}

func assertCodedError(t *testing.T, err error, code string) {
	t.Helper()
	if code == "" {
		assert.NoError(t, err)
		return
	}
	var coded Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, code, coded.Code)
}

type fakeEntry struct {
	name    string
	mode    os.FileMode
	info    os.FileInfo
	infoErr error
}

func (e fakeEntry) Name() string      { return e.name }
func (e fakeEntry) IsDir() bool       { return e.mode.IsDir() }
func (e fakeEntry) Type() os.FileMode { return e.mode }
func (e fakeEntry) Info() (os.FileInfo, error) {
	if e.infoErr != nil {
		return nil, e.infoErr
	}
	if e.info != nil {
		return e.info, nil
	}
	return fileInfo{mode: e.mode}, nil
}

type fileInfo struct {
	mode os.FileMode
}

func (i fileInfo) Name() string       { return "fixture" }
func (i fileInfo) Size() int64        { return 0 }
func (i fileInfo) Mode() os.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return time.Time{} }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return nil }

type failingCommandRunner struct {
	failAt  string
	resolve []byte
}

func (r *failingCommandRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	if err := r.commandFailure(command); err != nil {
		return nil, err
	}
	if command == "rev-parse FETCH_HEAD^{commit}" {
		return r.resolvedCommit()
	}
	if strings.HasPrefix(command, "archive ") && r.failAt == "archive" {
		return nil, errors.New("archive failed")
	}
	return nil, nil
}

func (r *failingCommandRunner) commandFailure(command string) error {
	switch {
	case r.failAt == "init" && command == "init --quiet":
		return errors.New("init failed")
	case r.failAt == "remote" && strings.HasPrefix(command, "remote "):
		return errors.New("remote failed")
	case r.failAt == "fetch" && strings.HasPrefix(command, "fetch "):
		return errors.New("fetch failed")
	default:
		return nil
	}
}

func (r *failingCommandRunner) resolvedCommit() ([]byte, error) {
	if r.failAt == "rev-parse" {
		return nil, errors.New("resolve failed")
	}
	if r.resolve != nil {
		return r.resolve, nil
	}
	return []byte(testCommit), nil
}

var _ fs.DirEntry = fakeEntry{}
