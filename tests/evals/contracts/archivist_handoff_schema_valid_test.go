//go:build eval

// Scenario D2: the Archivist->Sniper handoff schema itself is present and
// structurally valid YAML with its required_fields key — a pure artifact
// check, no LLM output involved. Reads from internal/embed/defaults/, the
// checked-in canonical source, never from the gitignored .strategist/
// runtime mirror (see .analysis/refined/20260804-test-framework-v2/analysis.md
// known_facts — reading from .strategist/ broke a different test suite
// earlier the same day this mission ran).
package contracts_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestD2_ArchivistHandoffSchemaValid(t *testing.T) {
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "handoff-archivist-to-sniper.schema.yaml")
	res := eval.RunScenario(eval.Scenario{
		ID:          "archivist-handoff-schema-valid",
		Description: "handoff-archivist-to-sniper.schema.yaml exists, parses as YAML, and declares required_fields",
		Input: eval.Input{
			Target: eval.TargetArtifactCheck,
			Params: map[string]any{
				"path":             path,
				"must_parse_yaml":  true,
				"must_contain_key": "required_fields",
			},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}
