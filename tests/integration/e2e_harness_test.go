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
)

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
		cmd := exec.Command("go", "build", "-o", strategistBinaryPath, "./cmd/strategist")
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
		"HOME":    home,
		"GOCACHE": "/tmp/gocache",
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
