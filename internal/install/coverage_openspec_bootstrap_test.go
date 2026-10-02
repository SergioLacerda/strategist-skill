package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubRuntimeCommand(t *testing.T, fn func(context.Context, string, string, ...string) ([]byte, error)) {
	t.Helper()
	original := runRankedRuntimeCommand
	runRankedRuntimeCommand = fn
	t.Cleanup(func() { runRankedRuntimeCommand = original })
}

func TestBootstrapOpenSpecRuntimeFailureModes(t *testing.T) {
	ctx := context.Background()
	exe := rankedExecutable{name: "node", prefix: []string{"openspec.mjs"}}
	valid := domain.WeaponRuntime{Bootstrap: "openspec init", Healthcheck: "openspec context"}

	badBootstrap := valid
	badBootstrap.Bootstrap = "make it so"
	require.ErrorContains(t, bootstrapOpenSpecRuntimeWith(ctx, t.TempDir(), badBootstrap, exe), "invalid bootstrap command")
	badHealth := valid
	badHealth.Healthcheck = "ping"
	require.ErrorContains(t, bootstrapOpenSpecRuntimeWith(ctx, t.TempDir(), badHealth, exe), "invalid healthcheck command")

	existing := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(existing, "config.yaml"), nil, 0o644))
	stubRuntimeCommand(t, func(context.Context, string, string, ...string) ([]byte, error) { return nil, errors.New("exit 1") })
	require.ErrorContains(t, bootstrapOpenSpecRuntimeWith(ctx, existing, valid, exe), "healthcheck failed")

	stubRuntimeCommand(t, func(context.Context, string, string, ...string) ([]byte, error) { return []byte("not json"), nil })
	require.ErrorContains(t, bootstrapOpenSpecRuntimeWith(ctx, existing, valid, exe), "healthcheck failed")

	fresh := filepath.Join(t.TempDir(), "runtime")
	stubRuntimeCommand(t, func(context.Context, string, string, ...string) ([]byte, error) {
		return nil, errors.New("init crashed")
	})
	require.ErrorContains(t, bootstrapOpenSpecRuntimeWith(ctx, fresh, valid, exe), "bootstrap failed")

	stubRuntimeCommand(t, func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil })
	err := bootstrapOpenSpecRuntimeWith(ctx, filepath.Join(t.TempDir(), "runtime2"), valid, exe)
	require.ErrorContains(t, err, "bootstrap completed without")

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, initializeOpenSpecRuntime(ctx, filepath.Join(blocker, "root"), exe, []string{"init"}, []string{"context"}), "create root")
	assert.Equal(t, []string{"openspec.mjs", "a"}, exe.args([]string{"a"}))
}
