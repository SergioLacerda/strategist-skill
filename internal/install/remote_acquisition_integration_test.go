package install

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/acquisition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteAcquisitionCanBeManuallyPromotedIntoExistingIngestion(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: acquired-skill\ndescription: acquired\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(source, "strategist.yaml"), []byte("canonical_role: ranger\nroles: [ranger]\nkind: atomic\nsupported_slots: [discovery]\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: test\n"), 0o644))
	runRemoteTestGitNoOutput(t, source, "init", "--quiet")
	runRemoteTestGitNoOutput(t, source, "config", "user.email", "test@example.invalid")
	runRemoteTestGitNoOutput(t, source, "config", "user.name", "test")
	runRemoteTestGitNoOutput(t, source, "add", "SKILL.md", "strategist.yaml")
	runRemoteTestGitNoOutput(t, source, "commit", "--quiet", "-m", "fixture")
	commit := runRemoteTestGit(t, source, "rev-parse", "HEAD")
	bare := filepath.Join(t.TempDir(), "acquired.git")
	runRemoteTestGitNoOutput(t, "", "clone", "--bare", "--quiet", source, bare)

	acquired, err := acquisition.Acquire(context.Background(), acquisition.Request{
		Source: acquisition.SourceGit, Repository: bare, Commit: commit, PackageID: "acquired-skill",
		AllowNetwork: true, QuarantineRoot: t.TempDir(),
	})
	require.NoError(t, err)

	// This copy models the explicit human promotion step. It intentionally
	// excludes acquisition evidence and READY from the package input.
	promotedRoot := t.TempDir()
	promotedPackage := filepath.Join(promotedRoot, "acquired-skill")
	copyRemotePackage(t, acquired.QuarantinePath, promotedPackage)
	result, err := IngestExternalSkills(promotedRoot, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Equal(t, "acquired-skill", result.Ingested[0].ID)
	assert.FileExists(t, filepath.Join(acquired.QuarantinePath, "acquisition.json"))
	assert.NoFileExists(t, filepath.Join(promotedRoot, "catalog.yaml"))
	assert.NoFileExists(t, filepath.Join(promotedRoot, "external-skills-source.lock.yaml"))
}

func copyRemotePackage(t *testing.T, source, destination string) {
	t.Helper()
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		return copyRemotePackageEntry(source, destination, path, info, err)
	}))
}

func copyRemotePackageEntry(source, destination, path string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	rel, err := filepath.Rel(source, path)
	if err != nil {
		return err
	}
	if rel == "." || rel == "acquisition.json" || rel == "READY" {
		return nil
	}
	target := filepath.Join(destination, rel)
	if info.IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	return copyRemotePackageFile(path, target)
}

func copyRemotePackageFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o644)
}

func runRemoteTestGitNoOutput(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), output)
}

func runRemoteTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}
