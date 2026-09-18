package runtimeenv

import (
	"context"
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
