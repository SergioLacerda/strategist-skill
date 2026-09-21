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
		Token      string `yaml:"token"`
		Emit       string `yaml:"emit"`
		EnforcedBy string `yaml:"enforced_by"`
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

// T3 (deep-analysis implementation review, 2026-07-26): every token classifies as
// enforced_by machine_enforced, machine_observed, or agent_only (2026-08-30:
// migrated from the old binary|agent vocabulary — see
// .analysis/refined/20260830-skill-gaps-triage/tasks.md Task 4, G03), and
// machine_enforced/machine_observed tokens must be literally, reachably
// emitted by non-test Go code — a symbol name or doctrine-text mention referencing
// the token is not sufficient (that was GAP-2's misclassification in the original
// Pass 0 inventory: extension-unfiltered grep counted embed/defaults docs as Go).
// TestEnforcedByTagsUseUnifiedVocabulary (enforcement_vocabulary_test.go) checks
// the same three-value vocabulary across other tagged contract files.
func TestErrorCatalogEnforcedByIsAccurate(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	cat := loadErrorCatalog(t, filepath.Join(root, "internal", "embed", "defaults"))

	var goCorpus strings.Builder
	for _, dir := range []string{filepath.Join(root, "internal"), filepath.Join(root, "cmd")} {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if strings.HasSuffix(path, "_test.go") || filepath.Ext(path) != ".go" {
				return nil
			}
			if strings.Contains(filepath.ToSlash(path), "internal/embed/defaults/") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			goCorpus.Write(data)
			goCorpus.WriteByte('\n')
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	code := goCorpus.String()

	for _, e := range cat.Errors {
		switch e.EnforcedBy {
		case "machine_enforced", "machine_observed":
			literal := "error=" + e.Token
			reasonLiteral := "reason=" + e.Token
			// Readiness blockers are emitted as a ReasonCode field or a Reason*
			// constant rather than inside message text; both are literal,
			// reachable emissions by non-test Go code.
			fieldLiteral := regexp.MustCompile(`Reason[A-Za-z]*(:|\s*=)\s*"` + regexp.QuoteMeta(e.Token) + `"`)
			if !strings.Contains(code, literal) && !strings.Contains(code, reasonLiteral) && !fieldLiteral.MatchString(code) {
				t.Errorf("catalog token %q is enforced_by: %s but no non-test .go file "+
					"under internal/ or cmd/ (excluding embed defaults) literally emits %q or %q",
					e.Token, e.EnforcedBy, literal, reasonLiteral)
			}
		case "agent_only":
			// no reachability requirement — doctrine-only tokens
		default:
			t.Errorf("catalog token %q has invalid or missing enforced_by (got %q, want machine_enforced|machine_observed|agent_only)", e.Token, e.EnforcedBy)
		}
	}
}

// Hardening F-03: every ranked_* reason code Go can emit that blocks readiness
// must have a catalog entry with a remedy, so an operator is never left with a
// bare token. Informational (non-blocking) codes are listed explicitly.
func TestRankedBlockingReasonCodesAreCatalogedWithAnAction(t *testing.T) {
	t.Parallel()

	informational := map[string]string{
		"ranked_certification_verified": "success detail, never blocks",
		"ranked_runtime_healthy":        "success detail, never blocks",
		"ranked_runtime_not_required":   "provider declares no runtime",
	}
	emitted := rankedReasonCodesInGo(t, repoRoot(t))
	if len(emitted) < 10 {
		t.Fatalf("scan found only %d ranked_* reason codes; the scanner is broken", len(emitted))
	}

	data := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "errors.yaml"))
	var cat struct {
		Errors []struct {
			Token  string `yaml:"token"`
			Action string `yaml:"action"`
		} `yaml:"errors"`
	}
	if err := yaml.Unmarshal([]byte(data), &cat); err != nil {
		t.Fatalf("parse errors.yaml: %v", err)
	}
	actions := map[string]string{}
	for _, e := range cat.Errors {
		actions[e.Token] = strings.TrimSpace(e.Action)
	}

	for code := range emitted {
		if _, ok := informational[code]; ok {
			continue
		}
		action, cataloged := actions[code]
		if !cataloged {
			t.Errorf("reason code %q is emitted by Go but has no entry in errors.yaml", code)
		} else if action == "" {
			t.Errorf("reason code %q is cataloged without an action (remedy)", code)
		}
	}
}

var (
	rankedReasonLiteral = regexp.MustCompile(`ReasonCode:\s*"(ranked_[a-z_]+)"`)
	rankedReasonConst   = regexp.MustCompile(`Reason[A-Za-z]*\s*=\s*"(ranked_[a-z_]+)"`)
)

func rankedReasonCodesInGo(t *testing.T, root string) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, re := range []*regexp.Regexp{rankedReasonLiteral, rankedReasonConst} {
				for _, m := range re.FindAllStringSubmatch(string(body), -1) {
					found[m[1]] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", dir, err)
		}
	}
	return found
}
