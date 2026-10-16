package rankedbinding

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestBootstrapOpenSpecRuntime_UsesContainedCommandsAndDirectConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist", "openspec")
	binDir := t.TempDir()
	script := testutil.BuildGoExecutableAt(t, binDir, "openspec", `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	cwd, _ := os.Getwd()
	log, _ := os.OpenFile(filepath.Join(cwd, "command.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if log != nil {
		_, _ = fmt.Fprintf(log, "%s|%s|%s\n", cwd, strings.Join(os.Args[1:], " "), os.Getenv("OPEN_SPEC_CONFIG"))
		_ = log.Close()
	}
	if len(os.Args) > 1 && os.Args[1] == "init" {
		_ = os.MkdirAll(filepath.Join(cwd, "openspec"), 0o755)
		_ = os.WriteFile(filepath.Join(cwd, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644)
		return
	}
	_, _ = fmt.Fprintf(os.Stdout, "{\"root\":{\"path\":%q},\"members\":[],\"status\":[]}", filepath.Dir(cwd))
}
`)
	t.Setenv("OPEN_SPEC_CONFIG", filepath.Join(t.TempDir(), "foreign-config.yaml"))

	runtime := domain.WeaponRuntime{
		Kind:        domain.RankedRuntimeOpenSpecRoot,
		Root:        ".strategist/openspec",
		Bootstrap:   "openspec init --profile core --tools codex",
		Healthcheck: "openspec context --json",
	}
	require.NoError(t, BootstrapOpenSpecRuntimeWith(context.Background(), root, runtime, RankedExecutable{Name: script}))
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
