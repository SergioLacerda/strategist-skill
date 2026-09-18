package rolevalidation

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestRoleWeaponCriticalMutants is a deterministic mutation gate for the
// authorization predicates. Each case mutates one persisted/catalog input;
// accepting one would mean that the corresponding fail-closed guard survived.
func TestRoleWeaponCriticalMutants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bindings string
		catalog  string
		active   domain.ActiveConfig
	}{
		{
			name: "active provider identity changed",
			bindings: `
  - slot: discovery
    installed_instance_id: openspec-explore
  - slot: refinement
    installed_instance_id: openspec-propose
`,
			active: domain.ActiveConfig{Slots: map[string]string{
				"discovery": "brainstorming", "refinement": "openspec-propose",
			}},
		},
		{
			name: "duplicate slot binding added",
			bindings: `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
`,
			active: domain.ActiveConfig{Slots: map[string]string{
				"discovery": "brainstorming", "refinement": "openspec-propose",
			}},
		},
		{
			name: "binding mode changed to unknown value",
			bindings: `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: bypass
  - slot: refinement
    installed_instance_id: openspec-propose
`,
			active: domain.ActiveConfig{Slots: map[string]string{
				"discovery": "brainstorming", "refinement": "openspec-propose",
			}},
		},
		{
			name: "ranked certification removed",
			bindings: `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`,
			catalog: `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
`,
			active: domain.ActiveConfig{Slots: map[string]string{
				"discovery": "brainstorming", "refinement": "openspec-propose",
			}},
		},
		{
			name: "ranked role affinity changed",
			bindings: `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`,
			catalog: `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: brainstorming
    canonical_role: archivist
    roles: [archivist]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
`,
			active: domain.ActiveConfig{Slots: map[string]string{
				"discovery": "brainstorming", "refinement": "openspec-propose",
			}},
		},
	}

	killed := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeValidationRoot(t, tt.bindings)
			if tt.catalog != "" {
				writeRankedCatalogFile(t, root, tt.catalog)
			}
			failures := ValidateRuntimeBindings(root, tt.active)
			require.NotEmpty(t, failures, "critical mutant survived: %s", tt.name)
			killed++
		})
	}
	t.Logf("critical mutants killed: %d/%d", killed, len(tests))
}
