package acquisition

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestValidateImmutableInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{"short commit", func() error { return ValidateCommit("abc") }, "commit_invalid"},
		{"uppercase commit", func() error { return ValidateCommit(strings.ToUpper(testCommit)) }, "commit_invalid"},
		{"escaping package", func() error { return ValidatePackageID("../skill") }, "package_id_invalid"},
		{"whitespace package", func() error { return ValidatePackageID("skill name") }, "package_id_invalid"},
		{"mutable repository", func() error { return ValidateRepository("main") }, "repository_invalid"},
		{"unsupported source", func() error {
			return validateRequest(Request{Source: "archive", Repository: "https://github.com/a/b", Commit: testCommit, PackageID: "skill", AllowNetwork: true})
		}, "source_unsupported"},
		{"network opt-in", func() error {
			return validateRequest(Request{Source: SourceGit, Repository: "https://github.com/a/b", Commit: testCommit, PackageID: "skill"})
		}, "network_opt_in_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			var coded Error
			require.ErrorAs(t, err, &coded)
			assert.Equal(t, tt.want, coded.Code)
		})
	}
}

func TestValidateRepositoryAcceptsSupportedForms(t *testing.T) {
	t.Parallel()
	for _, repository := range []string{
		"https://github.com/SergioLacerda/skill-for-hire",
		"ssh://git@github.com/SergioLacerda/skill-for-hire",
		"git@github.com:SergioLacerda/skill-for-hire.git",
		filepath.Join(t.TempDir(), "skill-for-hire.git"),
	} {
		require.NoError(t, ValidateRepository(repository), repository)
	}
}

func TestExtractArchiveRejectsUnsafeEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		add  func(*tar.Writer) error
		want string
	}{
		{"traversal", func(w *tar.Writer) error { return writeTarFile(w, "../escape", []byte("bad")) }, "archive_path_invalid"},
		{"absolute", func(w *tar.Writer) error { return writeTarFile(w, "/escape", []byte("bad")) }, "archive_path_invalid"},
		{"duplicate", func(w *tar.Writer) error {
			if err := writeTarFile(w, "SKILL.md", []byte("one")); err != nil {
				return err
			}
			return writeTarFile(w, "SKILL.md", []byte("two"))
		}, "archive_duplicate_entry"},
		{"symlink", func(w *tar.Writer) error {
			return w.WriteHeader(&tar.Header{Name: "SKILL.md", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
		}, "unsafe_entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := buildTar(t, tt.add)
			root := t.TempDir()
			err := extractArchive(root, archive)
			var coded Error
			require.ErrorAs(t, err, &coded)
			assert.Equal(t, tt.want, coded.Code)
		})
	}
}

func TestAcquireWritesManifestAndReadyOnlyAfterSuccessfulValidation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runner := fakeGitRunner{archive: validPackageArchive(t), resolved: testCommit}
	result, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: "https://github.com/example/skill-for-hire", Commit: testCommit,
		PackageID: "skill-for-hire/ranger", AllowNetwork: true, QuarantineRoot: root, Runner: runner,
	})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(result.QuarantinePath, "SKILL.md"))
	assert.FileExists(t, filepath.Join(result.QuarantinePath, "strategist.yaml"))
	assert.FileExists(t, filepath.Join(result.QuarantinePath, acquisitionManifestName))
	assert.FileExists(t, filepath.Join(result.QuarantinePath, readyMarkerName))
	assert.Equal(t, "package_shape_validated;trust_pending", result.Manifest.Validation)
	assert.NotEmpty(t, result.Manifest.TreeDigest)
}

func TestAcquireRollsBackIncompleteTransaction(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: "https://github.com/example/skill-for-hire", Commit: testCommit,
		PackageID: "skill", AllowNetwork: true, QuarantineRoot: root,
		Runner: fakeGitRunner{err: errors.New("fixture failed")},
	})
	require.Error(t, err)
	entries, readErr := os.ReadDir(root)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestAcquireRejectsManagedQuarantineRoot(t *testing.T) {
	t.Parallel()
	managed := t.TempDir()
	_, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: "https://github.com/example/skill-for-hire", Commit: testCommit,
		PackageID: "skill", AllowNetwork: true, QuarantineRoot: filepath.Join(managed, "nested"),
		ForbiddenRoots: []string{managed}, Runner: fakeGitRunner{},
	})
	var coded Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, "quarantine_root_forbidden", coded.Code)
}

func TestAcquireRejectsSymlinkedManagedQuarantineRoot(t *testing.T) {
	t.Parallel()
	managed := t.TempDir()
	link := filepath.Join(t.TempDir(), "quarantine-link")
	if err := os.Symlink(managed, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: "https://github.com/example/skill-for-hire", Commit: testCommit,
		PackageID: "skill", AllowNetwork: true, QuarantineRoot: link,
		ForbiddenRoots: []string{managed}, Runner: fakeGitRunner{},
	})
	var coded Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, "quarantine_root_forbidden", coded.Code)
}

func TestCommandGitRunnerUsesArgumentSafePinnedCommands(t *testing.T) {
	t.Parallel()
	command := &recordingCommandRunner{responses: map[string][]byte{
		"rev-parse FETCH_HEAD^{commit}":      []byte(testCommit + "\n"),
		"archive --format=tar " + testCommit: validPackageArchive(t),
	}}
	runner := CommandGitRunner{Command: command}
	archive, resolved, err := runner.FetchArchive(context.Background(), t.TempDir(), "https://github.com/example/skill-for-hire", testCommit, true)
	require.NoError(t, err)
	assert.Equal(t, testCommit, resolved)
	assert.NotEmpty(t, archive)
	assert.Equal(t, []string{
		"init --quiet",
		"remote add origin https://github.com/example/skill-for-hire",
		"fetch --no-tags --depth=1 origin " + testCommit,
		"rev-parse FETCH_HEAD^{commit}",
		"archive --format=tar " + testCommit,
	}, command.calls)
}

func TestCommandGitRunnerRequiresNetworkOptInBeforeCommands(t *testing.T) {
	t.Parallel()
	command := &recordingCommandRunner{}
	_, _, err := (CommandGitRunner{Command: command}).FetchArchive(context.Background(), t.TempDir(), "https://github.com/example/skill-for-hire", testCommit, false)
	var coded Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, "network_opt_in_required", coded.Code)
	assert.Empty(t, command.calls)
}

func TestAcquireLocalBareRepositoryVerifiesExactCommitWithoutLiveNetwork(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	runGit(t, source, "init", "--quiet")
	runGit(t, source, "config", "user.email", "test@example.invalid")
	runGit(t, source, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: local-skill\n---\nbody\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(source, "strategist.yaml"), []byte("roles: [ranger]\n"), 0o644))
	runGit(t, source, "add", "SKILL.md", "strategist.yaml")
	runGit(t, source, "commit", "--quiet", "-m", "fixture")
	commit := runGit(t, source, "rev-parse", "HEAD")
	bare := filepath.Join(t.TempDir(), "fixture.git")
	runGit(t, "", "clone", "--bare", "--quiet", source, bare)

	result, err := Acquire(context.Background(), Request{
		Source: SourceGit, Repository: bare, Commit: commit, PackageID: "local-skill",
		AllowNetwork: true, QuarantineRoot: t.TempDir(),
	})
	require.NoError(t, err)
	assert.Equal(t, commit, result.Manifest.Commit)
	assert.FileExists(t, filepath.Join(result.QuarantinePath, readyMarkerName))
}

func TestRunAcquireRemoteDoesNotStartWithoutNetworkOptIn(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	command := &recordingCommandRunner{}
	err := runCLIForTest(context.Background(), root, command)
	var coded Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, "network_opt_in_required", coded.Code)
	assert.Empty(t, command.calls)
	entries, readErr := os.ReadDir(root)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func validPackageArchive(t *testing.T) []byte {
	t.Helper()
	return buildTar(t, func(w *tar.Writer) error {
		if err := writeTarFile(w, "SKILL.md", []byte("---\nname: ranger\n---\n# Ranger\n")); err != nil {
			return err
		}
		return writeTarFile(w, "strategist.yaml", []byte("roles: [ranger]\n"))
	})
}

func buildTar(t *testing.T, add func(*tar.Writer) error) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	require.NoError(t, add(writer))
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func writeTarFile(writer *tar.Writer, name string, data []byte) error {
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}

type fakeGitRunner struct {
	archive  []byte
	resolved string
	err      error
}

func (f fakeGitRunner) FetchArchive(context.Context, string, string, string, bool) ([]byte, string, error) {
	if f.err != nil {
		return nil, "", f.err
	}
	return f.archive, f.resolved, nil
}

type recordingCommandRunner struct {
	responses map[string][]byte
	calls     []string
}

func (r *recordingCommandRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	if response, ok := r.responses[key]; ok {
		return response, nil
	}
	if strings.HasPrefix(key, "archive ") {
		return nil, errors.New("unexpected archive command")
	}
	return nil, nil
}

func runCLIForTest(ctx context.Context, root string, command CommandRunner) error {
	_, err := Acquire(ctx, Request{
		Source: SourceGit, Repository: "https://github.com/example/skill-for-hire", Commit: testCommit,
		PackageID: "skill", QuarantineRoot: filepath.Join(root, "quarantine"), Runner: CommandGitRunner{Command: command},
	})
	return err
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}
