//go:build spec

package spec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Hardening F-04: a tool or cache path reaches a make recipe from `which`, a
// wildcard, or an operator override, so it may contain spaces (C:/Program Files,
// a user name with a space). An unquoted $(VAR) then splits into several words
// and the recipe fails or runs the wrong thing, which is exactly how an
// unquoted $(PYTHON) broke `make install` on the Windows client.
var pathLikeMakeVars = []string{
	"GOLANGCI_LINT", "GORELEASER", "PYTHON", "GOCACHE", "GOMODCACHE",
	"GOLANGCI_LINT_CACHE", "LOCAL_BIN", "GOPATH_BIN", "GOVULNCHECK", "GOCOGNIT",
}

func TestMakeRecipesQuotePathLikeVariables(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	files := []string{filepath.Join(root, "Makefile")}
	mk, err := filepath.Glob(filepath.Join(root, "make", "*.mk"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, mk...)
	if len(files) < 3 {
		t.Fatalf("expected the Makefile and make/*.mk, found %d files", len(files))
	}

	ref := regexp.MustCompile(`\$\((` + strings.Join(pathLikeMakeVars, "|") + `)\)`)
	for _, file := range files {
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "\t") { // only recipe lines run in a shell
				continue
			}
			for _, loc := range ref.FindAllStringIndex(line, -1) {
				if !insideDoubleQuotes(line, loc[0]) {
					t.Errorf("%s:%d: %s is not double-quoted (breaks on a path with a space): %s",
						filepath.Base(file), i+1, line[loc[0]:loc[1]], strings.TrimSpace(line))
				}
			}
		}
	}
}

// insideDoubleQuotes reports whether byte offset pos of line lies inside a
// double-quoted string, ignoring quotes escaped with a backslash.
func insideDoubleQuotes(line string, pos int) bool {
	open := false
	for i := 0; i < pos; i++ {
		switch {
		case line[i] == '\\':
			i++
		case line[i] == '"':
			open = !open
		}
	}
	return open
}

// Hardening F-05: `go build -o bin/strategist` for Windows writes a file with no
// .exe, which Git Bash runs but cmd and PowerShell cannot resolve. Every target
// that names the binary must take the suffix from one variable (EXE, defaulting
// to `go env GOEXE`), so a Windows host produces and installs strategist.exe.
func TestMakeBinaryNamesCarryThePlatformSuffix(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	targets := []string{"build", "build-standalone", "install", "install-lite", "embed-skills", "embed-skills-check", "standalone-smoke", "compile-skill"}
	cmd := exec.Command("make", append([]string{"-n", "EXE=.exe"}, targets...)...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "bin/strategist.exe") {
		t.Fatalf("no target names bin/strategist.exe under EXE=.exe:\n%s", text)
	}
	bare := regexp.MustCompile(`(^|[^A-Za-z0-9_./-])(\./)?(\$\$HOME/\.local/)?bin/strategist([\s"'|;&)]|$)`)
	for _, line := range strings.Split(text, "\n") {
		if bare.MatchString(line) {
			t.Errorf("binary named without the platform suffix: %s", strings.TrimSpace(line))
		}
	}
}

// Without any override the binary variable must still resolve; an empty
// STRATEGIST_BIN would silently produce `-o ""` and `"./"`.
func TestMakeBinaryVariableResolvesWithoutOverrides(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("make", "-n", "build", "embed-skills-check", "install")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, bad := range []string{`-o ""`, `"./"`, `install -m 755 ""`} {
		if strings.Contains(text, bad) {
			t.Errorf("recipe contains %q: STRATEGIST_BIN resolved to empty", bad)
		}
	}
	if !strings.Contains(text, "bin/strategist") {
		t.Errorf("no recipe names bin/strategist:\n%s", text)
	}
}
