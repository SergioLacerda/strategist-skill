package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallShimTo_ReadOnlyParent(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	home := t.TempDir()
	require.NoError(t, os.Chmod(home, 0o444))
	t.Cleanup(func() { _ = os.Chmod(home, 0o755) })
	err := installShimTo(context.Background(), home, "", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "mkdir shim dir")
}

func TestInstallShimTo_WriteError(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	home := t.TempDir()
	shimDir := filepath.Join(home, ".claude", "skills", "strategist")
	require.NoError(t, os.MkdirAll(shimDir, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(shimDir, "SKILL.md"), 0o755))
	err := installShimTo(context.Background(), home, "", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "write shim")
}

func TestReadLocalSKILLMD_ReadFileFails(t *testing.T) {
	t.Parallel()
	svc := Service{Extractor: &errReadExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	_, err := svc.readLocalSKILLMD(context.Background(), t.TempDir())
	require.Error(t, err)
	assert.ErrorContains(t, err, "read embedded SKILL.md")
}

type errReadExtractor struct{}

func (e *errReadExtractor) Extract(targetDir string, _ bool) error {
	return os.MkdirAll(targetDir, 0o755)
}

func (e *errReadExtractor) ReadFile(relPath string) ([]byte, error) {
	return nil, fmt.Errorf("errReadExtractor: read error for %s", relPath)
}

func TestInstall_ShimError(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	shimHome := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(shimHome, ".claude"), 0o755))
	require.NoError(t, os.Chmod(filepath.Join(shimHome, ".claude"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(shimHome, ".claude"), 0o755) })

	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: shimHome}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "shim")
}

func TestInstallOptionalShims_GeminiAndCodex(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))

	installOptionalShims(context.Background(), home, "# SKILL", "")

	expectedPaths := []string{
		filepath.Join(home, ".gemini", "skills", "strategist", "SKILL.md"),
		filepath.Join(home, ".gemini", "antigravity", "skills", "strategist", "SKILL.md"),
		filepath.Join(home, ".codex", "skills", "strategist", "SKILL.md"),
	}
	for _, p := range expectedPaths {
		data, err := os.ReadFile(p)
		require.NoError(t, err, "shim should exist at %s", p)
		assert.Contains(t, string(data), "# SKILL", "shim content at %s", p)
	}
}

func TestInstallOptionalShims_SkipsWhenDirAbsent(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	installOptionalShims(context.Background(), home, "# SKILL", "")

	for _, dir := range []string{".gemini", ".codex"} {
		_, err := os.Stat(filepath.Join(home, dir))
		assert.True(t, os.IsNotExist(err), "optional dir %s should not be created", dir)
	}
}

func TestInstall_GitignoreError(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte(""), 0o000))
	t.Cleanup(func() { _ = os.Chmod(gi, 0o644) })

	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "gitignore")
}

func TestInstallGeminiShims_WriteFailureIsLoggedNotFatal(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	// A directory where a file is expected makes this specific shim write fail.
	blocked := filepath.Join(home, ".gemini", "skills", "strategist", "SKILL.md")
	require.NoError(t, os.MkdirAll(blocked, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "antigravity"), 0o755))

	installGeminiShims(filepath.Join(home, ".gemini"), "# SKILL", "")

	// The antigravity shim must still be written despite the other failure.
	data, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity", "skills", "strategist", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "# SKILL")
}

func TestInstallCodexShim_WriteFailureIsLoggedNotFatal(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	blocked := filepath.Join(home, ".codex", "skills", "strategist", "SKILL.md")
	require.NoError(t, os.MkdirAll(blocked, 0o755))

	// Must not panic — the failure is best-effort and only logged.
	installCodexShim(filepath.Join(home, ".codex"), "# SKILL", "")
}

func TestInstallShim_HomeDirError(t *testing.T) {
	clearHomeEnv(t)
	err := installShim(context.Background(), t.TempDir())
	require.Error(t, err)
	assert.ErrorContains(t, err, "home dir")
}

func TestResolveShimPath_HomeDirError(t *testing.T) {
	clearHomeEnv(t)
	s := Service{}
	_, err := s.resolveShimPath("")
	require.Error(t, err)
	assert.ErrorContains(t, err, "home dir")
}

func TestInstallShimStep_ReadSKILLMDFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := Service{Extractor: &errReadExtractor{}}
	_, err := s.installShimStep(context.Background(), dir, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "install: read SKILL.md")
}

func TestInstallShimStep_ResolveShimPathFails(t *testing.T) {
	clearHomeEnv(t)
	dir := t.TempDir()
	s := Service{Extractor: minimalExtractor{}}
	_, err := s.installShimStep(context.Background(), dir, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "resolve shim path")
}

func TestInstallShimStep_InstallShimForFails(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	shimHome := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(shimHome, ".claude"), 0o755))
	require.NoError(t, os.Chmod(filepath.Join(shimHome, ".claude"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(shimHome, ".claude"), 0o755) })

	s := Service{Extractor: minimalExtractor{}, ShimHomeDir: shimHome}
	_, err := s.installShimStep(context.Background(), dir, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "install: shim")
}

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()

	t.Run("no frontmatter returns string unchanged", func(t *testing.T) {
		t.Parallel()
		input := "# SKILL\nsome content\n"
		assert.Equal(t, input, stripFrontmatter(input))
	})

	t.Run("strips frontmatter block", func(t *testing.T) {
		t.Parallel()
		input := "---\nname: strategist\n---\n\n# SKILL\nbody\n"
		want := "# SKILL\nbody\n"
		assert.Equal(t, want, stripFrontmatter(input))
	})

	t.Run("unclosed frontmatter returns string unchanged", func(t *testing.T) {
		t.Parallel()
		input := "---\nname: strategist\nno closing marker\n"
		assert.Equal(t, input, stripFrontmatter(input))
	})
}
