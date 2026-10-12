//go:build spec

package spec_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type makefileInventory struct {
	phony   map[string][]string
	recipes map[string][]string
}

var makeTargetToken = regexp.MustCompile("^[A-Za-z0-9_./%+@-]+$")

func parseMakefileInventory(sources map[string]string) (makefileInventory, error) {
	inventory := makefileInventory{
		phony:   make(map[string][]string),
		recipes: make(map[string][]string),
	}
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		lines := strings.Split(strings.ReplaceAll(sources[path], "\r\n", "\n"), "\n")
		for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
			line := strings.TrimSpace(lines[lineNumber])
			if strings.HasPrefix(line, ".PHONY:") {
				phony, nextLine := parsePhonyBlock(lines, lineNumber)
				for _, target := range phony {
					inventory.phony[target] = append(inventory.phony[target], fmt.Sprintf("%s:%d", path, lineNumber+1))
				}
				lineNumber = nextLine
				continue
			}
			if strings.HasPrefix(lines[lineNumber], " ") || strings.HasPrefix(lines[lineNumber], "\t") {
				continue
			}
			targets, ok := parseRecipeTargets(line)
			if !ok {
				continue
			}
			for _, target := range targets {
				inventory.recipes[target] = append(inventory.recipes[target], fmt.Sprintf("%s:%d", path, lineNumber+1))
			}
		}
	}
	return inventory, nil
}

func parsePhonyBlock(lines []string, start int) ([]string, int) {
	var targets []string
	lineNumber := start
	for ; lineNumber < len(lines); lineNumber++ {
		line := strings.TrimSpace(lines[lineNumber])
		if lineNumber == start {
			line = strings.TrimSpace(strings.TrimPrefix(line, ".PHONY:"))
		}
		line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		targets = append(targets, strings.Fields(line)...)
		if !strings.HasSuffix(strings.TrimSpace(lines[lineNumber]), "\\") {
			break
		}
	}
	return targets, lineNumber
}

func parseRecipeTargets(line string) ([]string, bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 || strings.HasPrefix(line, ".") {
		return nil, false
	}
	if colon+1 < len(line) && strings.ContainsRune("=?!+", rune(line[colon+1])) {
		return nil, false
	}
	left := strings.TrimSpace(line[:colon])
	if left == "" {
		return nil, false
	}
	targets := strings.Fields(left)
	for _, target := range targets {
		if !makeTargetToken.MatchString(target) {
			return nil, false
		}
	}
	return targets, true
}

func readMakefileSources(t *testing.T, root string) map[string]string {
	t.Helper()
	sources := map[string]string{}
	sources["Makefile"] = readFile(t, filepath.Join(root, "Makefile"))
	entries, err := os.ReadDir(filepath.Join(root, "make"))
	if err != nil {
		t.Fatalf("read make directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".mk" {
			continue
		}
		relative := filepath.ToSlash(filepath.Join("make", entry.Name()))
		sources[relative] = readFile(t, filepath.Join(root, relative))
	}
	return sources
}

func assertUniqueOwnership(t *testing.T, kind string, entries map[string][]string) {
	t.Helper()
	for target, locations := range entries {
		if len(locations) > 1 {
			t.Errorf("%s target %q declared more than once: %s", kind, target, strings.Join(locations, ", "))
		}
	}
}

func TestMakefileInventoryParsesSplitPhonyBlocksAndDetectsDuplicates(t *testing.T) {
	t.Parallel()

	inventory, err := parseMakefileInventory(map[string]string{
		"Makefile":        strings.Join([]string{".PHONY: alpha \\", " beta", "alpha:", "\t@true", ""}, "\n"),
		"make/quality.mk": strings.Join([]string{".PHONY: gamma", "gamma:", "\t@true", ""}, "\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := inventory.phony["alpha"]; len(got) != 1 || got[0] != "Makefile:1" {
		t.Fatalf("alpha phony locations = %#v, want [Makefile:1]", got)
	}
	if got := inventory.phony["beta"]; len(got) != 1 || got[0] != "Makefile:1" {
		t.Fatalf("beta phony locations = %#v, want [Makefile:1]", got)
	}

	duplicate, err := parseMakefileInventory(map[string]string{
		"Makefile":   ".PHONY: duplicate\n",
		"make/go.mk": ".PHONY: duplicate\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := duplicate.phony["duplicate"]; len(got) != 2 {
		t.Fatalf("duplicate phony locations = %#v, want two locations", got)
	}
}

func TestMakefilePhonyOwnershipAndPublicTargetClassification(t *testing.T) {
	t.Parallel()

	inventory, err := parseMakefileInventory(readMakefileSources(t, repoRoot(t)))
	if err != nil {
		t.Fatal(err)
	}
	assertUniqueOwnership(t, ".PHONY", inventory.phony)
	assertUniqueOwnership(t, "recipe", inventory.recipes)

	for target, owner := range map[string]string{
		"ci-lint":            "Makefile:",
		"ci-test":            "Makefile:",
		"ci":                 "Makefile:",
		"treasure-chest-tmp": "make/go.mk:",
		"embed-skills":       "make/release.mk:",
		"embed-skills-check": "make/release.mk:",
	} {
		locations := inventory.recipes[target]
		if len(locations) != 1 || !strings.HasPrefix(locations[0], owner) {
			t.Errorf("recipe %q owned at %v, want one location under %s", target, locations, owner)
		}
		if len(inventory.phony[target]) != 1 {
			t.Errorf(".PHONY %q declared at %v, want one location", target, inventory.phony[target])
		}
	}

	for _, target := range []string{"live-jev", "live-calibration", "eval-promptfoo"} {
		if len(inventory.phony[target]) != 0 {
			t.Errorf("deliberate live/manual target %q must not be .PHONY: %v", target, inventory.phony[target])
		}
		if len(inventory.recipes[target]) != 1 {
			t.Errorf("deliberate live/manual target %q must have one recipe: %v", target, inventory.recipes[target])
		}
	}
}

func TestMakefileRetainsSemanticSubmakeBoundaries(t *testing.T) {
	t.Parallel()

	quality := readFile(t, filepath.Join(repoRoot(t), "make", "quality.mk"))
	goMake := readFile(t, filepath.Join(repoRoot(t), "make", "go.mk"))
	governance := readFile(t, filepath.Join(repoRoot(t), "make", "governance.mk"))

	for _, boundary := range []struct {
		name   string
		source string
		needle string
	}{
		{"lint complexity report", quality, "@$(MAKE) complexity-report"},
		{"lint file-size report", quality, "@$(MAKE) go-file-size-report"},
		{"lint-fix formatting check", quality, "@$(MAKE) fmt-check"},
		{"validate-all expansion", goMake, "$(MAKE) validate-expanded"},
		{"docs generation", governance, "$(MAKE) docs-generate"},
		{"coverage drift", governance, "@$(MAKE) coverage-docs-drift-check"},
	} {
		if !strings.Contains(boundary.source, boundary.needle) {
			t.Errorf("%s boundary missing %q", boundary.name, boundary.needle)
		}
	}
}

func TestDocsGeneratorListIsExplicitAndOrdered(t *testing.T) {
	t.Parallel()

	source := readFile(t, filepath.Join(repoRoot(t), "make", "docs.mk"))
	if strings.Contains(source, "wildcard") || strings.Contains(source, "generate-*.sh") {
		t.Fatal("docs generator composition must not discover scripts through a filename glob")
	}
	ordered := []string{
		"scripts/generate-command-tree.sh",
		"scripts/generate-schema-index.sh",
		"scripts/generate-contract-index.sh",
		"scripts/generate-event-catalog.sh",
		"scripts/generate-taxonomy-inventory.sh",
		"scripts/generate-quality-budgets.sh",
		"scripts/generate-coverage-policy.sh",
		"scripts/generate-eval-scenarios.sh",
	}
	previous := -1
	for _, generator := range ordered {
		index := strings.Index(source, generator)
		if index < 0 {
			t.Fatalf("docs generator list missing %q", generator)
		}
		if index <= previous {
			t.Fatalf("docs generator %q is out of order", generator)
		}
		previous = index
	}
	if !strings.Contains(source, "DOC_GENERATOR_SCRIPTS :=") {
		t.Fatal("docs generators must have one named Make variable")
	}
}
