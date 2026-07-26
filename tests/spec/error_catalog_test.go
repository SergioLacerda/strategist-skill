//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// W7b: contracts/machine/errors.yaml is the single normative catalog for
// error/blocked-state tokens. These tests freeze its parity with the rest of
// the corpus and with the drift-pattern catalog.

type errorCatalogFixture struct {
	Errors []struct {
		Token string `yaml:"token"`
		Emit  string `yaml:"emit"`
	} `yaml:"errors"`
	DriftIndex []string `yaml:"drift_index"`
}

func loadErrorCatalog(t *testing.T, root string) errorCatalogFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "contracts", "machine", "errors.yaml"))
	if err != nil {
		t.Fatalf("read errors.yaml: %v", err)
	}
	var cat errorCatalogFixture
	if err := yaml.Unmarshal(data, &cat); err != nil {
		t.Fatalf("parse errors.yaml: %v", err)
	}
	if len(cat.Errors) == 0 || len(cat.DriftIndex) == 0 {
		t.Fatalf("errors.yaml catalog is empty (errors=%d drift_index=%d)", len(cat.Errors), len(cat.DriftIndex))
	}
	return cat
}

// Every drift id in the catalog index must exist in identity/drift-patterns.yaml
// and vice versa — the two files must never diverge.
func TestErrorCatalogDriftIndexMatchesDriftPatterns(t *testing.T) {
	t.Parallel()

	root := filepath.Join(repoRoot(t), "internal", "embed", "defaults")
	cat := loadErrorCatalog(t, root)

	data, err := os.ReadFile(filepath.Join(root, "templates", "domain", "identity", "drift-patterns.yaml"))
	if err != nil {
		t.Fatalf("read drift-patterns.yaml: %v", err)
	}
	var patterns struct {
		Patterns []struct {
			ID string `yaml:"id"`
		} `yaml:"patterns"`
	}
	if err := yaml.Unmarshal(data, &patterns); err != nil {
		t.Fatalf("parse drift-patterns.yaml: %v", err)
	}

	fromPatterns := make(map[string]bool)
	for _, p := range patterns.Patterns {
		fromPatterns[p.ID] = true
	}
	fromCatalog := make(map[string]bool)
	for _, id := range cat.DriftIndex {
		fromCatalog[id] = true
	}

	for id := range fromPatterns {
		if !fromCatalog[id] {
			t.Errorf("drift pattern %q missing from errors.yaml drift_index", id)
		}
	}
	for id := range fromCatalog {
		if !fromPatterns[id] {
			t.Errorf("errors.yaml drift_index lists %q which is not in drift-patterns.yaml", id)
		}
	}
}

// Every `error=<token>` emitted anywhere in the shipped corpus must resolve to a
// catalog entry — no undocumented error tokens.
func TestCorpusErrorTokensResolveToCatalog(t *testing.T) {
	t.Parallel()

	root := filepath.Join(repoRoot(t), "internal", "embed", "defaults")
	cat := loadErrorCatalog(t, root)

	known := make(map[string]bool)
	for _, e := range cat.Errors {
		known[e.Token] = true
	}
	for _, id := range cat.DriftIndex {
		known[id] = true // drift=<id> lines are cataloged in drift-patterns.yaml
	}

	re := regexp.MustCompile(`(?:error|drift)=([a-z_]+)`)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".yaml" {
			return nil
		}
		if filepath.Base(path) == "errors.yaml" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			if !known[m[1]] {
				t.Errorf("%s emits %q which has no entry in contracts/machine/errors.yaml", path, m[0])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk corpus: %v", err)
	}
}

// No orphan catalog entries: every cataloged token must be used somewhere
// outside the catalog itself (corpus, Go code, or tests).
func TestErrorCatalogHasNoOrphanTokens(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	cat := loadErrorCatalog(t, filepath.Join(root, "internal", "embed", "defaults"))

	var corpus strings.Builder
	for _, dir := range []string{
		filepath.Join(root, "internal"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "tests"),
	} {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if filepath.Base(path) == "errors.yaml" {
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".md", ".yaml":
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				corpus.Write(data)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	all := corpus.String()
	for _, e := range cat.Errors {
		if !strings.Contains(all, e.Token) {
			t.Errorf("catalog token %q is not referenced anywhere outside errors.yaml — orphan entry", e.Token)
		}
	}
}
