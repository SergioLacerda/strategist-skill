package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/opportunity_attack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureOpportunityAttackRunsOnlyAtRefinementDoneAndPersistsManifest(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, ".analysis", "refined", "m-1")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	for name, content := range map[string]string{
		"analysis.md": "---\nmission_id: m-1\nmission_status: archivist_done\nsources_consulted: []\n---\nbody\n",
		"proposal.md": "new contract\n", "design.md": "design\n", "tasks.md": "- [ ] 1 [implementation_handoff] task\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(refined, name), []byte(content), 0o644))
	}

	require.NoError(t, ensureOpportunityAttack(LifecycleDependencies{}, root, filepath.Join(root, ".analysis"), "m-1", domain.MissionEventGateApproved))
	_, err := opportunity_attack.LoadManifest(root, "m-1")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, ensureOpportunityAttack(LifecycleDependencies{}, root, filepath.Join(root, ".analysis"), "m-1", domain.MissionEventRefinementDone))
	manifest, err := opportunity_attack.LoadManifest(root, "m-1")
	require.NoError(t, err)
	assert.Equal(t, "m-1", manifest.MissionID)
	assert.True(t, manifest.Candidates[0].Warranted)
}

func TestLoadOpportunityChestIndexUsesConfiguredKnowledgeIndex(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("knowledge_index_path: indexes/knowledge.yaml\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "indexes"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "indexes", "knowledge.yaml"), []byte("sources:\n  - id: configured-source\n  - id: second-source\n"), 0o644))
	index, err := loadOpportunityChestIndex(root)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"configured-source": true, "second-source": true}, index)
}

func TestLoadOpportunityChestIndexMissingIndexIsEmptyAndMalformedIndexFails(t *testing.T) {
	root := t.TempDir()
	index, err := loadOpportunityChestIndex(root)
	require.NoError(t, err)
	assert.Empty(t, index)
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources: [unterminated\n"), 0o644))
	_, err = loadOpportunityChestIndex(root)
	assert.ErrorContains(t, err, "parse knowledge index")
}
