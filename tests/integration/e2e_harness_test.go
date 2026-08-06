//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
)

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

func buildStrategistBinary(t *testing.T) string {
	t.Helper()

	root := repoRoot(t)
	strategistBinaryOnce.Do(func() {
		if err := os.MkdirAll("/tmp/gocache", 0o755); err != nil {
			strategistBinaryErr = err
			return
		}

		buildDir, err := os.MkdirTemp("", "strategist-e2e-bin-*")
		if err != nil {
			strategistBinaryErr = err
			return
		}

		strategistBinaryPath = filepath.Join(buildDir, "strategist")
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
		cmd := exec.Command("go", "build", "-cover", "-covermode=atomic", "-o", strategistBinaryPath, "./cmd/strategist")
		cmd.Dir = root
		cmd.Env = envWithOverrides(map[string]string{
			"GOCACHE": "/tmp/gocache",
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

	home := t.TempDir()
	require.NoError(t, os.MkdirAll("/tmp/gocache", 0o755))

	binary := buildStrategistBinary(t)
	cmd := exec.Command(binary, args...)
	cmd.Dir = workspace
	cmd.Env = envWithOverrides(map[string]string{
		"HOME":       home,
		"GOCACHE":    "/tmp/gocache",
		"GOCOVERDIR": strategistGOCOVERDIR(t),
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

func envWithOverrides(overrides map[string]string) []string {
	base := os.Environ()
	filtered := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := overrides[key]; exists {
			continue
		}
		filtered = append(filtered, entry)
	}
	for key, value := range overrides {
		filtered = append(filtered, key+"="+value)
	}
	return filtered
}
