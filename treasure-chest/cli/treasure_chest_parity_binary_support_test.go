package treasurecli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type parityProcessResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func buildStrategistForParityTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	bin := filepath.Join(t.TempDir(), "strategist")
	cache := os.Getenv("GOCACHE")
	if cache == "" {
		cache = "/tmp/strategist-treasure-binary-cache"
	}

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/strategist")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GOCACHE="+cache)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build strategist: %s", output)
	return bin
}

func runParityProcess(t *testing.T, bin string, args ...string) parityProcessResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run strategist: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return parityProcessResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}

func assertParitySuccess(t *testing.T, result parityProcessResult) {
	t.Helper()
	require.Equal(t, 0, result.exitCode, "stderr: %s", result.stderr)
	if result.stderr != "" {
		t.Errorf("unexpected stderr: %s", result.stderr)
	}
}

func readParityFile(t *testing.T, root, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, name))
	require.NoError(t, err)
	return string(raw)
}
