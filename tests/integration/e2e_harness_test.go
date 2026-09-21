//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type cliResult struct {
	args     []string
	stdout   string
	stderr   string
	exitCode int
}

var (
	strategistBinaryOnce sync.Once
	strategistBinaryPath string
	strategistBinaryErr  error

	strategistCoverDirOnce sync.Once
	strategistCoverDirPath string
	strategistCoverDirErr  error

	strategistGoCacheOnce sync.Once
	strategistGoCachePath string
	strategistGoCacheErr  error

	strategistCLIRunMu sync.Mutex
)

// integrationEnvAllowlist contains only host values needed to locate tools,
// preserve platform behavior, and use pre-provisioned Go dependencies. Product
// configuration and credentials must never leak into a CLI integration test.
var integrationEnvAllowlist = []string{
	"PATH", "TMPDIR", "TMP", "TEMP", "SystemRoot", "WINDIR", "PATHEXT",
	"LANG", "LC_ALL", "TZ", "GOTOOLCHAIN", "GOPATH", "GOPROXY", "GOSUMDB", "GONOSUMDB",
	"GONOPROXY", "GOPRIVATE", "GOMODCACHE", "GOFLAGS", "CGO_ENABLED", "CC", "CXX",
}

// strategistGOCOVERDIR returns the directory every runStrategistCLI subprocess
// writes GOCOVERDIR binary coverage counters into, so that time spent inside
// the compiled strategist binary (built with -cover below) counts toward
// this package's `-coverpkg=./internal/...` measurement. Without this, a
// subprocess run is invisible to `go test`'s own coverage instrumentation —
// see docs/integration-coverage-gaps.md and
// .analysis/refined/20260805-integration-coverage-mapping/analysis.md for
// why that gap existed and how it was found.
//
// scripts/test-style-report.sh sets STRATEGIST_E2E_GOCOVERDIR to a directory
// it also passes to `go test -args -test.gocoverdir=...`, so the test
// binary's own in-process coverage (compile_test.go, install_test.go, etc.)
// and every subprocess binary's coverage merge into one directory, mergeable
// with a single `go tool covdata textfmt` call. A plain `go test` run
// without that env var still works — it falls back to a scratch temp dir,
// which simply never gets merged into a profile (coverage number is
// unaffected, only STRATEGIST_E2E_GOCOVERDIR-driven runs measure it).
func strategistGOCOVERDIR(t *testing.T) string {
	t.Helper()

	strategistCoverDirOnce.Do(func() {
		dir := os.Getenv("STRATEGIST_E2E_GOCOVERDIR")
		if dir == "" {
			var err error
			dir, err = os.MkdirTemp("", "strategist-e2e-covdata-*")
			if err != nil {
				strategistCoverDirErr = err
				return
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			strategistCoverDirErr = err
			return
		}
		strategistCoverDirPath = dir
	})

	require.NoError(t, strategistCoverDirErr)
	return strategistCoverDirPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func strategistGoCache(t *testing.T) string {
	t.Helper()

	strategistGoCacheOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "strategist-e2e-gocache")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			strategistGoCacheErr = err
			return
		}
		strategistGoCachePath = dir
	})

	require.NoError(t, strategistGoCacheErr)
	return strategistGoCachePath
}

func buildStrategistBinary(t *testing.T) string {
	t.Helper()

	root := repoRoot(t)
	goCache := strategistGoCache(t)
	strategistBinaryOnce.Do(func() {
		buildDir, err := os.MkdirTemp("", "strategist-e2e-bin-*")
		if err != nil {
			strategistBinaryErr = err
			return
		}

		binaryName := "strategist"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		strategistBinaryPath = filepath.Join(buildDir, binaryName)
		// -covermode=atomic must match what `go test -race` forces on the
		// test binary's own instrumentation — go tool covdata refuses to
		// merge "set"-mode and "atomic"-mode counter files from the same
		// GOCOVERDIR ("counter mode clash"), which silently produced a
		// 0-line merged profile before this was pinned explicitly.
		// Deliberately no -coverpkg here: `go build -coverpkg=./internal/...`
		// (excluding package main, cmd/strategist, from the covered set)
		// silently produces a binary that writes zero GOCOVERDIR files on
		// exit — confirmed by direct experiment, not documented behavior.
		// Instead this instruments the whole main module (including
		// cmd/strategist) and scripts/test-style-report.sh's textfmt step
		// filters the merged profile back down to internal/... lines only,
		// to keep the metric's defined scope.
		goPathOutput, err := exec.Command("go", "env", "GOPATH").Output()
		if err != nil {
			strategistBinaryErr = fmt.Errorf("resolve GOPATH: %w", err)
			return
		}
		cmd := exec.Command("go", "build", "-cover", "-covermode=atomic", "-o", strategistBinaryPath, "./cmd/strategist")
		cmd.Dir = root
		cmd.Env = hermeticEnv(map[string]string{
			"GOCACHE": goCache,
			"GOPATH":  strings.TrimSpace(string(goPathOutput)),
		})

		output, err := cmd.CombinedOutput()
		if err != nil {
			strategistBinaryErr = fmt.Errorf("build strategist: %w\n%s", err, string(output))
		}
	})

	require.NoError(t, strategistBinaryErr)
	return strategistBinaryPath
}

func runStrategistCLI(t *testing.T, workspace string, args ...string) cliResult {
	t.Helper()
	return runStrategistCLIWithEnv(t, workspace, nil, args...)
}

func runStrategistCLIWithEnv(t *testing.T, workspace string, extraEnv map[string]string, args ...string) cliResult {
	return runStrategistCLIWithInput(t, workspace, extraEnv, "", args...)
}

func runStrategistCLIWithInput(t *testing.T, workspace string, extraEnv map[string]string, input string, args ...string) cliResult {
	t.Helper()

	home := t.TempDir()
	xdgConfig := t.TempDir()
	xdgCache := t.TempDir()
	xdgData := t.TempDir()
	goCache := strategistGoCache(t)

	binary := buildStrategistBinary(t)
	cmd := exec.Command(binary, args...)
	cmd.Dir = workspace
	cmd.Env = hermeticEnv(map[string]string{
		"HOME":            home,
		"USERPROFILE":     home,
		"HOMEDRIVE":       "",
		"HOMEPATH":        "",
		"XDG_CONFIG_HOME": xdgConfig,
		"XDG_CACHE_HOME":  xdgCache,
		"XDG_DATA_HOME":   xdgData,
		"GOCACHE":         goCache,
		"GOCOVERDIR":      strategistGOCOVERDIR(t),
	})
	if len(extraEnv) > 0 {
		cmd.Env = hermeticEnvWithBase(cmd.Env, extraEnv)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	strategistCLIRunMu.Lock()
	err := cmd.Run()
	strategistCLIRunMu.Unlock()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run strategist %s: %v", strings.Join(args, " "), err)
		}
	}

	return cliResult{
		args:     append([]string(nil), args...),
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}

func (r cliResult) output() string {
	return r.stdout + r.stderr
}

func hermeticEnv(overrides map[string]string) []string {
	base := make(map[string]string, len(integrationEnvAllowlist))
	for _, key := range integrationEnvAllowlist {
		if value, ok := os.LookupEnv(key); ok {
			base[key] = value
		}
	}
	return appendEnvOverrides(base, overrides)
}

func hermeticEnvWithBase(baseEnv []string, overrides map[string]string) []string {
	base := make(map[string]string, len(baseEnv)+len(overrides))
	for _, entry := range baseEnv {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			base[key] = value
		}
	}
	return appendEnvOverrides(base, overrides)
}

func appendEnvOverrides(base, overrides map[string]string) []string {
	for key, value := range overrides {
		for existing := range base {
			if strings.EqualFold(existing, key) {
				delete(base, existing)
			}
		}
		base[key] = value
	}
	keys := make([]string, 0, len(base))
	for key := range base {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+base[key])
	}
	return env
}

// withHostOpenSpec returns extra environment whose PATH carries an executable
// named openspec, standing in for a machine that has the OpenSpec CLI.
//
// The harness builds the strategist binary WITHOUT the embedded runtime, and a
// provider that declares a runtime now (correctly) blocks readiness when there
// is no executable for it anywhere. Tests that only need a ready workspace use
// this instead of relying on the runner having OpenSpec installed; the missing
// runtime itself is covered by its own tests and by the standalone smoke.
func withHostOpenSpec(t *testing.T) map[string]string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stands in for openspec with a POSIX shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openspec"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil { //nolint:gosec // test stub must be executable
		t.Fatalf("write openspec stub: %v", err)
	}
	return map[string]string{"PATH": dir + string(os.PathListSeparator) + os.Getenv("PATH")}
}
