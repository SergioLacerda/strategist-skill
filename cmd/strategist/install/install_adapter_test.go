package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInstaller struct {
	report internalinstall.Report
	err    error
	cfg    domain.InstallConfig
	ctx    context.Context
}

func (f *fakeInstaller) InstallWithReport(ctx context.Context, cfg domain.InstallConfig) (internalinstall.Report, error) {
	f.ctx = ctx
	f.cfg = cfg
	return f.report, f.err
}

func fakeDeps(installer *fakeInstaller) Dependencies {
	return Dependencies{
		ResolveTarget: func(explicit string, global bool) (string, error) {
			if explicit == "" {
				explicit = "resolved"
			}
			if global {
				explicit = "global-" + explicit
			}
			return filepath.Join("/tmp", explicit), nil
		},
		UserHomeDir:    func() (string, error) { return "/tmp/home", nil },
		ServiceFactory: func(string) Installer { return installer },
	}
}

func TestNewBuildsInstallCommandAndMapsFlags(t *testing.T) {
	installer := &fakeInstaller{}
	cmd := New(fakeDeps(installer))
	require.Equal(t, "install", cmd.Name())
	require.NoError(t, cmd.Flags().Set("target", "repo"))
	require.NoError(t, cmd.Flags().Set("silent", "true"))
	require.NoError(t, cmd.Flags().Set("wizard", "true"))
	require.NoError(t, cmd.Flags().Set("global", "true"))
	require.NoError(t, cmd.Flags().Set("force", "true"))
	require.NoError(t, cmd.Flags().Set("allow-downgrade", "true"))
	require.NoError(t, cmd.Flags().Set("strict-compile", "true"))
	require.NoError(t, cmd.Flags().Set("no-shim", "true"))

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Equal(t, filepath.Join("/tmp", "global-repo"), installer.cfg.Target)
	assert.True(t, installer.cfg.Silent)
	assert.True(t, installer.cfg.Wizard)
	assert.True(t, installer.cfg.Global)
	assert.True(t, installer.cfg.Force)
	assert.True(t, installer.cfg.AllowDowngrade)
	assert.True(t, installer.cfg.StrictCompile)
	assert.True(t, installer.cfg.NoShim)
}

func TestRegisterAttachesInstallCommand(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root, fakeDeps(&fakeInstaller{}))
	found, _, err := root.Find([]string{"install"})
	require.NoError(t, err)
	assert.Equal(t, "install", found.Name())
}

func TestRunRejectsConflictingShimFlags(t *testing.T) {
	err := RunForTest(&cobra.Command{Use: "install"}, fakeDeps(&fakeInstaller{}), TestOptions{
		NoShim:   true,
		ShimPath: filepath.Join(t.TempDir(), "SKILL.md"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestRunReportsMissingDependencies(t *testing.T) {
	err := Run(&cobra.Command{Use: "install"}, Dependencies{}, options{})
	require.ErrorContains(t, err, "target resolver is not configured")

	err = Run(&cobra.Command{Use: "install"}, Dependencies{
		ResolveTarget: func(string, bool) (string, error) { return t.TempDir(), nil },
	}, options{})
	require.ErrorContains(t, err, "home directory resolver is not configured")

	err = Run(&cobra.Command{Use: "install"}, Dependencies{
		ResolveTarget: func(string, bool) (string, error) { return t.TempDir(), nil },
		UserHomeDir:   func() (string, error) { return t.TempDir(), nil },
	}, options{})
	require.ErrorContains(t, err, "service factory is not configured")
}

func TestRunRecordsInstallerErrorOnSpan(t *testing.T) {
	errBoom := errors.New("boom")
	err := Run(&cobra.Command{Use: "install"}, fakeDeps(&fakeInstaller{err: errBoom}), options{})
	require.ErrorContains(t, err, "install: boom")
}

func TestRunReturnsBackupWriteError(t *testing.T) {
	installer := &fakeInstaller{report: internalinstall.Report{BackupDir: "/tmp/backup"}}
	cmd := &cobra.Command{Use: "install"}
	cmd.SetOut(failingWriter{})
	err := Run(cmd, fakeDeps(installer), options{})
	require.ErrorContains(t, err, "write output")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestResolveTargetPaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	explicit, err := ResolveTarget("sub", false, nil)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "sub"), explicit)

	home := t.TempDir()
	t.Setenv("HOME", home)
	global, err := ResolveTarget("", true, nil)
	require.NoError(t, err)
	assert.Equal(t, home, global)

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))
	child := filepath.Join(root, "child")
	require.NoError(t, os.MkdirAll(child, 0o755))
	t.Chdir(child)
	discovered, err := ResolveTarget("", false, func(string) (string, string, error) {
		return filepath.Join(root, ".strategist"), root, nil
	})
	require.NoError(t, err)
	assert.Equal(t, root, discovered)

	fallback, err := ResolveTarget("", false, nil)
	require.NoError(t, err)
	assert.Equal(t, child, fallback)
}

func TestCommandContextAndTelemetryHelpers(t *testing.T) {
	key := installTestContextKey{}
	ctx := context.WithValue(context.Background(), key, "value")
	cmd := &cobra.Command{Use: "install"}
	cmd.SetContext(ctx)
	assert.Equal(t, "value", commandContext(cmd).Value(key))

	run := telemetry.NewMissionRun("install-adapter-test")
	runCtx := telemetry.WithMissionRun(context.Background(), run)
	markInstallRun(runCtx, true)
	addMissionLines(runCtx, 2)
	snapshot := run.Snapshot()
	assert.Equal(t, int64(2), snapshot.LinesEmitted)
	assert.NotPanics(t, func() { markInstallRun(context.Background(), false) })

	spanCtx, span := startInstallSpan(runCtx, "/tmp/target")
	span.End()
	assert.Same(t, run, telemetry.MissionRunFromContext(spanCtx))
}

func TestPartialDetectionAndBanners(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, isPartial(dir))
	manifestPath := filepath.Join(dir, ".strategist", ".compiled", ".manifest.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0o755))
	require.NoError(t, os.WriteFile(manifestPath, []byte("x"), 0o644))
	assert.False(t, isPartial(dir))

	full := captureStdout(t, func() { printCompleteBanner("/target", true, false) })
	assert.Contains(t, full, "wizard")
	assert.Contains(t, full, "/target")
	assert.NotContains(t, full, "partial")

	partial := captureStdout(t, func() { printCompleteBanner("/target", false, true) })
	assert.Contains(t, partial, "partial")
	assert.Contains(t, partial, "strategist compile")
	assert.Contains(t, partial, "--strict-compile")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return buf.String()
}

func TestRunPassesCommandContextToService(t *testing.T) {
	installer := &fakeInstaller{}
	key := installTestContextKey{}
	ctx := context.WithValue(context.Background(), key, "kept")
	cmd := &cobra.Command{Use: "install"}
	cmd.SetContext(ctx)
	require.NoError(t, Run(cmd, fakeDeps(installer), options{}))
	assert.Equal(t, "kept", installer.ctx.Value(key))
}

func TestResolveTargetErrors(t *testing.T) {
	_, err := ResolveTarget("", false, func(string) (string, string, error) {
		return "", "", fmt.Errorf("ignored")
	})
	require.NoError(t, err, "discover failures fall back to cwd")

	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	_, err = ResolveTarget("", true, nil)
	require.ErrorContains(t, err, "resolve home dir")
}

// installTestContextKey is a private context key type, so test values never
// collide with keys set by other packages.
type installTestContextKey struct{}
