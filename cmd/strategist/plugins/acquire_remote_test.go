package plugins

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/acquisition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const acquireRemoteTestCommit = "0123456789abcdef0123456789abcdef01234567"

func TestRunAcquireRemoteRequiresExplicitNetworkOptIn(t *testing.T) {
	t.Parallel()
	runner := &testAcquireRunner{}
	err := RunAcquireRemote(context.Background(), &bytes.Buffer{}, acquisition.Request{
		Source: acquisition.SourceGit, Repository: "https://github.com/example/skill-for-hire",
		Commit: acquireRemoteTestCommit, PackageID: "skill", QuarantineRoot: t.TempDir(), Runner: runner,
	})
	var coded acquisition.Error
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, "network_opt_in_required", coded.Code)
	assert.False(t, runner.called)
}

func TestRunAcquireRemoteReportsQuarantineWithoutPromotion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runner := &testAcquireRunner{archive: testAcquireArchive(t), resolved: acquireRemoteTestCommit}
	var output bytes.Buffer
	err := RunAcquireRemote(context.Background(), &output, acquisition.Request{
		Source: acquisition.SourceGit, Repository: "https://github.com/example/skill-for-hire",
		Commit: acquireRemoteTestCommit, PackageID: "skill", AllowNetwork: true,
		QuarantineRoot: root, Runner: runner,
	})
	require.NoError(t, err)
	assert.Contains(t, output.String(), "quarantined")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	quarantine := filepath.Join(root, entries[0].Name())
	assert.FileExists(t, filepath.Join(quarantine, "READY"))
	assert.NoDirExists(t, filepath.Join(quarantine, ".strategist"))
	assert.NotContains(t, output.String(), "prepare-embedded")
}

type testAcquireRunner struct {
	archive  []byte
	resolved string
	err      error
	called   bool
}

func (r *testAcquireRunner) FetchArchive(_ context.Context, _ string, _ string, _ string, _ bool) ([]byte, string, error) {
	r.called = true
	if r.err != nil {
		return nil, "", r.err
	}
	return r.archive, r.resolved, nil
}

func testAcquireArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for name, data := range map[string][]byte{
		"SKILL.md":        []byte("---\nname: skill\n---\nbody\n"),
		"strategist.yaml": []byte("roles: [ranger]\n"),
	} {
		require.NoError(t, writer.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}))
		_, err := writer.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
