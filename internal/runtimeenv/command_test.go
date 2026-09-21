package runtimeenv

import (
	"context"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForRootDoesNotInheritHostConfiguration(t *testing.T) {
	t.Setenv("OPEN_SPEC_CONFIG", "/outside/config.yaml")
	t.Setenv("XDG_CONFIG_HOME", "/outside/config")

	env := strings.Join(ForRoot(t.TempDir()), "\n")
	require.NotContains(t, env, "OPEN_SPEC_CONFIG")
	require.NotContains(t, env, "/outside/config")
	require.Contains(t, env, "PATH=")
}

func TestCommandUsesRestrictedEnvironment(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	t.Setenv("OPEN_SPEC_CONFIG", "/outside/config.yaml")
	cmd, err := Command(context.Background(), t.TempDir(), "sh", "-c", "test -z \"$OPEN_SPEC_CONFIG\" && test -n \"$PATH\"")
	require.NoError(t, err)
	_, err = cmd.Output()
	require.NoError(t, err)
}

func TestCommandReportsMissingExecutableAsTypedError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := Command(context.Background(), t.TempDir(), "openspec", "context", "--json")
	require.Error(t, err)

	var missing *ExecutableNotFoundError
	require.ErrorAs(t, err, &missing)
	require.Equal(t, "openspec", missing.Name)
	require.ErrorContains(t, err, `resolve provider executable "openspec"`)
}

func TestPlatformEnvAllowsOnlyWindowsStartupVariables(t *testing.T) {
	host := map[string]string{
		"SystemRoot": `C:\Windows`, "TEMP": `C:\Users\u\AppData\Local\Temp`, "TMP": `C:\tmp`,
		"ComSpec": `C:\Windows\System32\cmd.exe`, "PATHEXT": ".COM;.EXE;.CMD",
		"APPDATA": `C:\Users\u\AppData\Roaming`, "NPM_CONFIG_PREFIX": `C:\npm`,
	}
	get := func(k string) string { return host[k] }

	env := strings.Join(platformEnv("windows", get), "\n")
	for _, want := range []string{`SystemRoot=C:\Windows`, "TEMP=", "TMP=", "ComSpec=", "PATHEXT="} {
		require.Contains(t, env, want)
	}
	require.NotContains(t, env, "APPDATA")
	require.NotContains(t, env, "NPM_CONFIG_PREFIX")
}

func TestPlatformEnvIsEmptyOffWindowsAndSkipsUnsetVariables(t *testing.T) {
	get := func(string) string { return "" }
	require.Empty(t, platformEnv("linux", func(string) string { return "x" }))
	require.Empty(t, platformEnv("windows", get))
}

func TestPrivateCommandRunsAbsoluteExecutableWithoutHostPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell script as the private executable")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "node")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\nprintf '%s' \"$PATH\"\n"), 0o755))
	t.Setenv("PATH", "/host/only")

	cmd, err := PrivateCommand(context.Background(), dir, exe)
	require.NoError(t, err)
	out, err := cmd.Output()
	require.NoError(t, err)
	require.Equal(t, dir, string(out), "PATH must be the private runtime directory only")
}

func TestPrivateCommandRejectsRelativeOrMissingExecutable(t *testing.T) {
	_, err := PrivateCommand(context.Background(), t.TempDir(), "node")
	require.Error(t, err)
	_, err = PrivateCommand(context.Background(), t.TempDir(), filepath.Join(t.TempDir(), "absent"))
	var missing *ExecutableNotFoundError
	require.ErrorAs(t, err, &missing)
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(base))
	t.Cleanup(func() { require.NoError(t, os.Chdir(old)) })
	require.NoError(t, os.Mkdir(filepath.Join(base, "rt"), 0o755))
	return base
}

func envValue(env []string, key string) string {
	for _, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			return strings.TrimPrefix(kv, key+"=")
		}
	}
	return ""
}

func TestForRootReturnsAbsolutePathsForRelativeRoot(t *testing.T) {
	base := chdirTemp(t)
	env := ForRoot("rt")
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME"} {
		value := envValue(env, key)
		require.True(t, filepath.IsAbs(value), "%s must be absolute, got %q", key, value)
		require.True(t, strings.HasPrefix(value, filepath.Join(evalSymlinks(t, base), "rt")+string(filepath.Separator)) ||
			strings.HasPrefix(value, filepath.Join(base, "rt")+string(filepath.Separator)), "%s=%q must be under the absolute root", key, value)
	}
}

func evalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

func TestCommandUsesSameAbsoluteRootForDirAndEnv(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	chdirTemp(t)
	cmd, err := Command(context.Background(), "rt", "sh", "-c", "true")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(cmd.Dir), "cmd.Dir must be absolute, got %q", cmd.Dir)
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME"} {
		require.True(t, strings.HasPrefix(envValue(cmd.Env, key), cmd.Dir+string(filepath.Separator)), "%s must be under cmd.Dir", key)
	}
}

func TestCommandWithRelativeRootCreatesNoNestedProviderState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell")
	}
	base := chdirTemp(t)
	cmd, err := Command(context.Background(), "rt", "sh", "-c", `mkdir -p "$XDG_CONFIG_HOME/x"`)
	require.NoError(t, err)
	require.NoError(t, cmd.Run())
	require.DirExists(t, filepath.Join(base, "rt", ".provider-config", "x"))
	require.NoDirExists(t, filepath.Join(base, "rt", "rt"))
}

func TestPrivateCommandUsesAbsoluteRootForRelativeDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell script as the private executable")
	}
	base := chdirTemp(t)
	exe := filepath.Join(base, "node")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\ntrue\n"), 0o755))
	cmd, err := PrivateCommand(context.Background(), "rt", exe)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(cmd.Dir))
	require.True(t, strings.HasPrefix(envValue(cmd.Env, "HOME"), cmd.Dir+string(filepath.Separator)))
}

func TestCommandFailsWhenRelativeRootCannotBeResolved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("removing the current directory is not supported on Windows")
	}
	base := chdirTemp(t)
	require.NoError(t, os.RemoveAll(base))
	if _, err := os.Getwd(); err == nil {
		t.Skip("filepath.Abs still resolves after the working directory is removed on this platform")
	}

	cmd, err := Command(context.Background(), "rt", "sh", "-c", "true")
	require.Error(t, err)
	require.Nil(t, cmd)
	require.ErrorContains(t, err, `resolve provider root "rt"`)

	cmd, err = PrivateCommand(context.Background(), "rt", "/bin/sh")
	require.Error(t, err)
	require.Nil(t, cmd)
	require.ErrorContains(t, err, `resolve provider root "rt"`)
}
