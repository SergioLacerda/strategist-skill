//go:build eval

// Scenario D3: the progress-event contract itself is present and
// structurally valid YAML with its event_format key.
package contracts_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestD3_ProgressEventSchemaValid(t *testing.T) {
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "progress-contract.yaml")
	res := eval.RunScenario(eval.Scenario{
		ID:          "progress-event-schema-valid",
		Description: "progress-contract.yaml exists, parses as YAML, and declares event_format",
		Input: eval.Input{
			Target: eval.TargetArtifactCheck,
			Params: map[string]any{
				"path":             path,
				"must_parse_yaml":  true,
				"must_contain_key": "event_format",
			},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}
