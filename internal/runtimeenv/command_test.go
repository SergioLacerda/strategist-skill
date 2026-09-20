package runtimeenv

import (
	"context"
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
