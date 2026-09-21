package install

import (
	"context"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBootstrapOpenSpecRuntime_UsesContainedCommandsAndDirectConfig(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	root := filepath.Join(t.TempDir(), ".strategist", "openspec")
	binDir := t.TempDir()
	script := filepath.Join(binDir, "openspec")
	scriptBody := `#!/bin/sh
set -eu
printf '%s|%s|%s\n' "$PWD" "$*" "${OPEN_SPEC_CONFIG:-}" >> "$PWD/command.log"
if [ "$1" = "init" ]; then
  mkdir -p "$PWD/openspec"
  printf 'schema: spec-driven\n' > "$PWD/openspec/config.yaml"
  exit 0
fi
printf '{"root":{"path":"%s"},"members":[],"status":[]}\n' "$(dirname "$PWD")"
`
	require.NoError(t, os.WriteFile(script, []byte(scriptBody), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OPEN_SPEC_CONFIG", filepath.Join(t.TempDir(), "foreign-config.yaml"))

	runtime := domain.RankedRuntimeContract{
		Kind:        domain.RankedRuntimeOpenSpecRoot,
		Root:        ".strategist/openspec",
		Bootstrap:   "openspec init --profile core --tools codex",
		Healthcheck: "openspec context --json",
	}
	require.NoError(t, bootstrapOpenSpecRuntime(context.Background(), root, runtime))
	require.FileExists(t, filepath.Join(root, "config.yaml"))
	assertNoNestedOpenSpecRoot(t, root)

	initLog, err := os.ReadFile(filepath.Join(filepath.Dir(root), "command.log"))
	require.NoError(t, err)
	healthLog, err := os.ReadFile(filepath.Join(root, "command.log"))
	require.NoError(t, err)
	initLines := strings.Split(strings.TrimSpace(string(initLog)), "\n")
	healthLines := strings.Split(strings.TrimSpace(string(healthLog)), "\n")
	require.Len(t, initLines, 1)
	require.Len(t, healthLines, 1)
	require.Contains(t, initLines[0], filepath.Dir(root)+"|init --profile core --tools codex|")
	require.Contains(t, healthLines[0], root+"|context --json|")
	require.NotContains(t, string(initLog), "foreign-config.yaml")
	require.NotContains(t, string(healthLog), "foreign-config.yaml")
}

func assertNoNestedOpenSpecRoot(t *testing.T, root string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, "openspec", "config.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
