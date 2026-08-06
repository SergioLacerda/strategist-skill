package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- scoring half ---

func TestClusterCandidateScoreWithPolicy(t *testing.T) {
	t.Parallel()
	c := Cluster{ID: "c1", CitedMissions: []string{"m1", "m2"}, Tags: []string{"t1", "t2"}}
	policy := DefaultScoringPolicy()

	got := ClusterCandidateScoreWithPolicy(c, policy)
	assert.Positive(t, got)
	assert.LessOrEqual(t, got, policy.MaxScore)
	assert.Equal(t, ClusterCandidateScore(c), got)
}

func TestClusterCandidateScoreWithPolicy_ClampsToMaxScore(t *testing.T) {
	t.Parallel()
	c := Cluster{ID: "c1", CitedMissions: make([]string, 50), Tags: make([]string, 50)}
	policy := DefaultScoringPolicy()

	got := ClusterCandidateScoreWithPolicy(c, policy)
	assert.Equal(t, policy.MaxScore, got)
}

func TestGapCandidateScoreWithPolicy(t *testing.T) {
	t.Parallel()
	g := Gap{ID: "g1", CitedMissions: []string{"m1"}}
	policy := DefaultScoringPolicy()

	got := GapCandidateScoreWithPolicy(g, policy)
	assert.Positive(t, got)
	assert.LessOrEqual(t, got, policy.MaxScore)
	assert.Equal(t, GapCandidateScore(g), got)
}

func TestBuildJewelCandidates_ClustersAndGaps(t *testing.T) {
	t.Parallel()
	clusters := []Cluster{{ID: "c1", CitedMissions: []string{"m1"}, Tags: []string{"t1"}}}
	gaps := []Gap{{ID: "g1", CitedMissions: []string{"m1"}}}

	got := BuildJewelCandidates(clusters, gaps)
	require.Len(t, got, 2)
	assert.Equal(t, "jewel-c1", got[0].ID)
	assert.Equal(t, MissionHistoryChestID, got[0].ChestID)
	assert.Equal(t, domain.JewelStatusProposed, got[0].Status)
	assert.Equal(t, "jewel-gap-g1", got[1].ID)
	assert.Equal(t, "gap", got[1].Kind)
}

func TestBuildJewelCandidates_EmptyInputsYieldsEmpty(t *testing.T) {
	t.Parallel()
	got := BuildJewelCandidates(nil, nil)
	assert.Empty(t, got)
}

// --- persistence half ---

func TestWriteProposedJewels_WritesNewCandidates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	candidates := []Jewel{
		{ID: "jewel-c1", ChestID: MissionHistoryChestID, Kind: "pattern", Statement: "x",
			SourceRefs: []string{"mission-history#m1"}, Trust: "T2", Status: domain.JewelStatusProposed, ReviewedBy: "agent"},
	}

	written, skipped, err := WriteProposedJewels(dir, candidates)
	require.NoError(t, err)
	assert.Equal(t, 1, written)
	assert.Equal(t, 0, skipped)

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", MissionHistoryChestID+".yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: jewel-c1")
	assert.Contains(t, string(raw), "status: proposed")
}

func TestWriteProposedJewels_SkipsExistingIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML) // contains jewel-1, chest_id: source
	candidates := []Jewel{
		{ID: "jewel-1", ChestID: "source", Kind: "pattern", Statement: "dup",
			SourceRefs: []string{"source#a"}, Trust: "T2", Status: domain.JewelStatusProposed, ReviewedBy: "agent"},
	}

	written, skipped, err := WriteProposedJewels(dir, candidates)
	require.NoError(t, err)
	assert.Equal(t, 0, written)
	assert.Equal(t, 1, skipped)
}

func TestWriteProposedJewels_EmptyCandidatesNoop(t *testing.T) {
	t.Parallel()
	written, skipped, err := WriteProposedJewels(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, written)
	assert.Equal(t, 0, skipped)
}

func TestExistingJewelIDs_NoManifestsIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := ExistingJewelIDs(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestExistingJewelIDs_CollectsAcrossManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	got, err := ExistingJewelIDs(dir)
	require.NoError(t, err)
	assert.True(t, got["jewel-1"])
}
