//go:build spec

package spec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Hardening F-11: the developer environment caused three failures in one thread
// (a golangci-lint built with an older Go than go.mod, a small temp dir filled
// by read-only module caches, a stale strategist earlier on PATH). `make doctor`
// names each fault and its remedy; these tests pin that with simulated faults.

func runDoctor(t *testing.T, fakeBin map[string]string, env ...string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range fakeBin {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", "scripts/doctor.sh")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), append([]string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")}, env...)...)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run doctor: %v\n%s", err, out)
	}
	return string(out), code
}

func TestDoctorFailsWhenGolangciLintWasBuiltWithAnOlderGoThanGoMod(t *testing.T) {
	t.Parallel()
	out, code := runDoctor(t, map[string]string{
		"golangci-lint": `echo "golangci-lint has version 2.12.2 built with go1.20.1 from (unknown) on (unknown)"`,
	})
	if code == 0 {
		t.Fatalf("doctor must fail for an outdated golangci-lint build:\n%s", out)
	}
	for _, want := range []string{"FAIL", "golangci-lint", "go1.20.1", "go install github.com/golangci/golangci-lint"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorFailsWhenPythonIsNotUsable(t *testing.T) {
	t.Parallel()
	// A Windows Store stub or a broken shim exists on PATH but cannot run.
	out, code := runDoctor(t, map[string]string{"python3": `exit 9`})
	if code == 0 || !strings.Contains(out, "python") || !strings.Contains(out, "FAIL") {
		t.Fatalf("doctor must fail for an unusable python (exit=%d):\n%s", code, out)
	}
}

func TestDoctorWarnsButPassesOnLowTempSpace(t *testing.T) {
	t.Parallel()
	out, _ := runDoctor(t, nil, "DOCTOR_MIN_TMP_MB=999999999")
	if !strings.Contains(out, "WARN") || !strings.Contains(out, "temp") {
		t.Fatalf("doctor must warn about low temp space:\n%s", out)
	}
	if strings.Contains(out, "FAIL  temp") {
		t.Errorf("low temp space is a warning, not a failure:\n%s", out)
	}
}

func TestDoctorReportsAStaleStrategistOnPath(t *testing.T) {
	t.Parallel()
	out, _ := runDoctor(t, map[string]string{
		"strategist": `if [ "$1" = "version" ]; then echo V1.0.0; echo "runtime payload: none (openspec resolved from PATH)"; fi`,
	})
	if !strings.Contains(out, "runtime payload: none") && !strings.Contains(out, "no embedded runtime") {
		t.Fatalf("doctor must flag a strategist without the embedded runtime:\n%s", out)
	}
}

// Hardening F-11 (task 2.4): any Go edit changes the catalog's host_api_digest,
// so a commit that touches Go or the embedded defaults without regenerating the
// catalog fails the drift test later, in CI. The pre-commit hook runs the drift
// check only when such files are staged, and names the remedy.

func runDriftHook(t *testing.T, staged, checkCmd string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", "scripts/hooks/check-embedded-drift.sh")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "STAGED_FILES="+staged, "EMBED_CHECK_CMD="+checkCmd)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run hook script: %v\n%s", err, out)
	}
	return string(out), code
}

func TestDriftHookSkipsWhenNothingRelevantIsStaged(t *testing.T) {
	t.Parallel()
	out, code := runDriftHook(t, "README.md\ndocs/guide.md", "exit 1")
	if code != 0 || !strings.Contains(out, "skipped") {
		t.Fatalf("docs-only commits must not run the drift check (exit=%d):\n%s", code, out)
	}
}

func TestDriftHookBlocksAndNamesTheRemedyWhenGoChangesDrift(t *testing.T) {
	t.Parallel()
	for _, staged := range []string{"internal/install/x.go", "cmd/strategist/main.go", "external-skills-source/openspec-propose/SKILL.md", "internal/embed/defaults/plugins/catalog.yaml"} {
		out, code := runDriftHook(t, staged, "exit 1")
		if code == 0 || !strings.Contains(out, "make embed-skills") {
			t.Errorf("%s: drift must block and name `make embed-skills` (exit=%d):\n%s", staged, code, out)
		}
	}
}

func TestDriftHookPassesWhenTheCatalogIsCurrent(t *testing.T) {
	t.Parallel()
	out, code := runDriftHook(t, "internal/install/x.go", "exit 0")
	if code != 0 {
		t.Fatalf("a current catalog must pass (exit=%d):\n%s", code, out)
	}
}
