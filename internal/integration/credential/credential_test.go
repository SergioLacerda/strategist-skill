package credential

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
)

func writeEnv(t *testing.T, content string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
	return path
}

func TestSecretNeverPrintsItsValue(t *testing.T) {
	secret := Secret{value: "sk-synthetic"}
	require.Equal(t, "sk-synthetic", secret.Reveal())
	for _, rendered := range []string{secret.String(), fmt.Sprintf("%v", secret), fmt.Sprintf("%+v", secret), fmt.Sprintf("%#v", secret)} {
		require.NotContains(t, rendered, "sk-synthetic")
	}
}

func TestResolveFromEnvironmentReference(t *testing.T) {
	env := Env{Getenv: func(name string) string {
		if name == "TYPESAFE_API_KEY" {
			return "sk-env"
		}
		return ""
	}}
	secret, err := Resolve("env:TYPESAFE_API_KEY", env)
	require.NoError(t, err)
	require.Equal(t, "sk-env", secret.Reveal())
}

func TestResolveFromDotenvDoesNotTouchTheProcessEnvironment(t *testing.T) {
	const name = "STRATEGIST_TEST_JEV_KEY"
	path := writeEnv(t, "# comment\nOTHER=1\nexport "+name+"=\"sk-dotenv\"\n", 0o600)

	secret, err := Resolve("dotenv:"+path+"#"+name, Env{})
	require.NoError(t, err)
	require.Equal(t, "sk-dotenv", secret.Reveal())

	require.Empty(t, os.Getenv(name), "the key is never exported with os.Setenv")
	for _, entry := range os.Environ() {
		require.NotContains(t, entry, "sk-dotenv", "a child built from os.Environ() cannot inherit the key")
	}
}

func TestResolveReportsCredentialMissing(t *testing.T) {
	path := writeEnv(t, "OTHER=1\n", 0o600)
	for name, ref := range map[string]string{
		"unset env":        "env:ABSENT_VAR",
		"missing file":     "dotenv:" + filepath.Join(t.TempDir(), "nope") + "#X",
		"missing variable": "dotenv:" + path + "#ABSENT",
		"empty value":      "dotenv:" + writeEnv(t, "X=\n", 0o600) + "#X",
		"malformed":        "plain-text",
		"no variable":      "dotenv:" + path,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(ref, Env{Getenv: func(string) string { return "" }})
			state, ok := integration.StateOf(err)
			require.True(t, ok)
			require.Equal(t, integration.StateCredentialMissing, state)
			require.NotContains(t, err.Error(), "sk-")
		})
	}
}

func TestDotenvParserHasNoShellExpansion(t *testing.T) {
	values := ParseDotenv("A=$HOME\nB='single # kept'\nC=plain # trailing\n\nbad line\n")
	require.Equal(t, "$HOME", values["A"])
	require.Equal(t, "single # kept", values["B"])
	require.Equal(t, "plain", values["C"])
	require.NotContains(t, values, "bad line")
}

func TestHygieneFlagsLoosePermissionsAndMissingIgnore(t *testing.T) {
	loose := writeEnv(t, "X=1\n", 0o644)
	findings := Hygiene(loose, func(string) (bool, error) { return false, nil })
	if runtime.GOOS != "windows" {
		require.Contains(t, findings, FindingLoosePermissions)
	}
	require.Contains(t, findings, FindingNotIgnored)

	tight := writeEnv(t, "X=1\n", 0o600)
	require.Empty(t, Hygiene(tight, func(string) (bool, error) { return true, nil }))

	unknown := Hygiene(tight, func(string) (bool, error) { return false, errors.New("not a repository") })
	require.Contains(t, unknown, FindingIgnoreUnknown)
	require.NotContains(t, unknown, FindingNotIgnored)
}

func TestWorkspaceEnvResolvesRelativeDotenvAgainstTheWorkspace(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, ".env"), []byte("K=sk-workspace\n"), 0o600))

	secret, err := Resolve("dotenv:.env#K", WorkspaceEnv(workspace))
	require.NoError(t, err)
	require.Equal(t, "sk-workspace", secret.Reveal())

	_, err = Resolve("dotenv:.env#K", WorkspaceEnv(t.TempDir()))
	require.Error(t, err, "another workspace has no such file")
}
