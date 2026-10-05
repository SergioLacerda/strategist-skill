//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLiveTargetsAreDetachedFromEveryBuildChain pins the rule that nothing a plain
// make target or a CI workflow does can call an external provider: the live
// targets are not in .PHONY, are a dependency of nothing, refuse to run without an
// explicit STRATEGIST_LIVE_CONSENT=yes, and no workflow mentions them. The
// sanctioned path to a live check is the DOJO, behind a consent term.
func TestLiveTargetsAreDetachedFromEveryBuildChain(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	files := []string{filepath.Join(root, "Makefile")}
	mk, err := filepath.Glob(filepath.Join(root, "make", "*.mk"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, mk...)

	phony := regexp.MustCompile(`(?m)^\.PHONY:(?:[^\n]*\\\n)*[^\n]*`)
	dependency := regexp.MustCompile(`(?m)^[A-Za-z][\w.-]*\s*:[^\n=]*\blive-`)
	for _, file := range files {
		content := readFile(t, file)
		for _, block := range phony.FindAllString(content, -1) {
			if strings.Contains(block, "live") {
				t.Errorf("%s: a live target is listed in .PHONY: %s", file, block)
			}
		}
		for _, line := range dependency.FindAllString(content, -1) {
			if !strings.HasPrefix(line, "live-") {
				t.Errorf("%s: a target depends on a live target: %q", file, line)
			}
		}
	}

	goMk := readFile(t, filepath.Join(root, "make", "go.mk"))
	for _, target := range []string{"live-jev", "live-calibration"} {
		recipe := goMk[strings.Index(goMk, "\n"+target+":"):]
		recipe = recipe[:strings.Index(recipe[1:], "\n\n")+1]
		if !strings.Contains(recipe, `"$(STRATEGIST_LIVE_CONSENT)" = "yes"`) {
			t.Errorf("%s must refuse to run without STRATEGIST_LIVE_CONSENT=yes", target)
		}
	}

	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, workflow := range workflows {
		raw, readErr := os.ReadFile(workflow)
		if readErr != nil {
			continue
		}
		for _, needle := range []string{"live-jev", "live-calibration", "tags=live", "-tags live", "STRATEGIST_LIVE_CONSENT"} {
			if strings.Contains(string(raw), needle) {
				t.Errorf("%s mentions %q: CI must never run a live check", workflow, needle)
			}
		}
	}
}
