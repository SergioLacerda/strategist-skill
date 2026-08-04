package eval

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func runArtifactCheckScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	path := paramString(p, "path")

	raw, err := os.ReadFile(filepath.Clean(path)) //nolint:gosec // G304: eval scenario paths are test-authored, not runtime input
	if err != nil {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertArtifactExists,
			Message:       "artifact not found",
			Expected:      path,
			Actual:        err.Error(),
		})
		return
	}
	// Phase 2 (20260804-eval-fake-provider): content assertions run against
	// the fixture's raw content regardless of must_parse_yaml, since most
	// fixtures here are markdown with YAML frontmatter, not pure YAML.
	evaluateContentAssertions(string(raw), s.Assertions, res)

	if !paramBool(p, "must_parse_yaml") {
		return
	}

	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		res.Violations = append(res.Violations, Violation{
			Message:  "artifact is not valid YAML",
			Expected: path,
			Actual:   err.Error(),
		})
		return
	}
	if key := paramString(p, "must_contain_key"); key != "" {
		if _, ok := doc[key]; !ok {
			res.Violations = append(res.Violations, Violation{
				Message:  "artifact missing required top-level key",
				Expected: key,
				Actual:   fmt.Sprintf("keys present: %v", mapKeys(doc)),
			})
		}
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
