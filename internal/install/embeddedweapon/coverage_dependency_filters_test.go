package embeddedweapon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterDependencyResolvedRejectsMissingDependenciesAndComponents(t *testing.T) {
	t.Parallel()
	accepted := []plugincatalog.IngestedSkill{
		{
			ID:      "master",
			Package: domain.PluginPackage{Version: "1.0.0"},
			Adapter: plugincatalog.ExternalSkillAdapter{AuxiliaryTools: []string{"missing-helper"}},
		},
		{
			ID:      "composite",
			Package: domain.PluginPackage{Version: "1.0.0"},
			Adapter: plugincatalog.ExternalSkillAdapter{
				Kind: domain.WeaponKindComposite,
				Composition: &domain.WeaponComposition{
					Strategy:   domain.WeaponCompositionOrderedDAG,
					Components: []domain.WeaponComponent{{ID: "missing-component", Required: true}},
				},
			},
		},
	}
	result := plugincatalog.IngestionResult{}

	got := filterDependencyResolved(accepted, "v1", nil, &result)

	assert.Empty(t, got)
	require.Len(t, result.Rejected, 2)
	assert.Contains(t, result.Rejected[0].Reason, "dependency_unresolved")
	assert.Contains(t, result.Rejected[1].Reason, "dependency_unresolved")
}

func TestResolveCandidatesRecordsResolutionRejection(t *testing.T) {
	t.Parallel()
	result := plugincatalog.IngestionResult{}

	candidates, identities := resolveCandidates([]string{filepath.Join(t.TempDir(), "missing-skill")}, &result)

	assert.Empty(t, candidates)
	assert.Empty(t, identities)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "skill")
}

func TestEmbeddedArtifactComparisonReportsMissingInputs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := compareEmbeddedArtifacts(
		PrepareEmbeddedOptions{DefaultsRoot: dir, LockPath: filepath.Join(dir, "lock.yaml")},
		plugincatalog.IngestionResult{},
		t.TempDir(),
	)
	require.Error(t, err)

	existing := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(existing, []byte("catalog"), 0o644))
	_, err = catalogHasDrifted(existing, filepath.Join(dir, "missing.yaml"))
	require.Error(t, err)
}
