package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCatalogDependenciesFlagsMissingAuxiliaryTool proves tasks.md
// Task 2: a master skill (Arma) declaring an auxiliary_tools_allowed
// dependency that is not itself a catalog entry is flagged, not silently
// catalogued as usable — mirrors brainstorming's real, currently-unresolved
// "writing-plans" dependency (see
// .analysis/refined/20260913-role-skill-weapon-taxonomy/analysis.md KF-07).
func TestValidateCatalogDependenciesFlagsMissingAuxiliaryTool(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{
		SchemaVersion: "strategist-plugin-catalog/v2",
		Providers: []pluginCatalogProvider{
			{ID: "master", RiskScore: "write_analysis", AuxiliaryTools: []string{"missing-helper"}},
		},
	}

	violations := ValidateCatalogDependencies(catalog)
	require.Len(t, violations, 1)
	assert.Equal(t, "master", violations[0].ProviderID)
	assert.Equal(t, "missing-helper", violations[0].DependencyID)
	assert.Equal(t, "auxiliary_tools_allowed", violations[0].Source)
	assert.Contains(t, violations[0].String(), "master")
	assert.Contains(t, violations[0].String(), "missing-helper")
}

// TestValidateCatalogDependenciesPassesWhenAuxiliaryToolIsCatalogued proves
// the positive path: once the declared dependency is itself a catalog entry,
// no violation is reported.
func TestValidateCatalogDependenciesPassesWhenAuxiliaryToolIsCatalogued(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{
		SchemaVersion: "strategist-plugin-catalog/v2",
		Providers: []pluginCatalogProvider{
			{ID: "master", RiskScore: "write_analysis", AuxiliaryTools: []string{"helper"}},
			{ID: "helper", RiskScore: "write_analysis"},
		},
	}

	assert.Empty(t, ValidateCatalogDependencies(catalog))
}

// TestValidateCatalogDependenciesFlagsMissingStructuredDependency covers the
// Dependencies (structured, resolver-native) form separately from
// AuxiliaryTools, since a catalog entry may use either.
func TestValidateCatalogDependenciesFlagsMissingStructuredDependency(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{
		SchemaVersion: "strategist-plugin-catalog/v2",
		Providers: []pluginCatalogProvider{
			{ID: "master", RiskScore: "write_analysis", Dependencies: []pluginCatalogDependency{
				{ID: "missing-dep", Kind: "adapter_contract", Constraint: "*"},
			}},
		},
	}

	violations := ValidateCatalogDependencies(catalog)
	require.Len(t, violations, 1)
	assert.Equal(t, "dependencies", violations[0].Source)
}

// TestValidateCatalogDependenciesIgnoresOptionalMissingDependency proves an
// optional structured dependency never blocks cataloguing — only required
// ones are enforced.
func TestValidateCatalogDependenciesIgnoresOptionalMissingDependency(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{
		SchemaVersion: "strategist-plugin-catalog/v2",
		Providers: []pluginCatalogProvider{
			{ID: "master", RiskScore: "write_analysis", Dependencies: []pluginCatalogDependency{
				{ID: "optional-extra", Kind: "adapter_contract", Constraint: "*", Optional: true},
			}},
		},
	}

	assert.Empty(t, ValidateCatalogDependencies(catalog))
}

// TestValidateCatalogDependenciesAgainstRealCatalogHasNoViolations runs the
// validator against the actual embedded catalog.yaml. brainstorming's former
// "writing-plans" auxiliary_tools_allowed gap (KF-07/KF-08 in the taxonomy
// mission's analysis.md) was resolved during the embedded-skill-directory-catalog
// mission's migration to external-skills-source/brainstorming/: the
// declaration was dropped rather than fabricating a writing-plans package
// (see external-skills-source/brainstorming/strategist.yaml) — the entry
// never claimed a discovery-subtype capability that would need it. The
// catalog is expected to be fully dependency-clean today.
func TestValidateCatalogDependenciesAgainstRealCatalogHasNoViolations(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)

	assert.Empty(t, ValidateCatalogDependencies(catalog))
}
