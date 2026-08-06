package treasure

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterJewels_DefaultExcludesDeprecatedAndSorts(t *testing.T) {
	t.Parallel()
	jewels := map[string][]Jewel{
		"b": {
			{ID: "jewel-2", ChestID: "b", Status: domain.JewelStatusDeprecated},
			{ID: "jewel-1", ChestID: "b", Status: domain.JewelStatusAccepted},
		},
		"a": {
			{ID: "jewel-3", ChestID: "a", Status: domain.JewelStatusProposed},
		},
	}

	got := FilterJewels(jewels, JewelFilter{})

	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].ChestID)
	assert.Equal(t, "jewel-3", got[0].ID)
	assert.Equal(t, "b", got[1].ChestID)
	assert.Equal(t, "jewel-1", got[1].ID)
}

func TestFilterJewels_AllIncludesDeprecated(t *testing.T) {
	t.Parallel()
	jewels := map[string][]Jewel{
		"source": {
			{ID: "jewel-1", ChestID: "source", Status: domain.JewelStatusDeprecated},
		},
	}

	got := FilterJewels(jewels, JewelFilter{Status: "all"})

	require.Len(t, got, 1)
	assert.Equal(t, "jewel-1", got[0].ID)
}

func TestFilterJewels_ChestAndStatus(t *testing.T) {
	t.Parallel()
	jewels := map[string][]Jewel{
		"a": {{ID: "jewel-a", ChestID: "a", Status: domain.JewelStatusProposed}},
		"b": {{ID: "jewel-b", ChestID: "b", Status: domain.JewelStatusProposed}},
	}

	got := FilterJewels(jewels, JewelFilter{
		ChestID: "b",
		Status:  domain.JewelStatusProposed,
	})

	require.Len(t, got, 1)
	assert.Equal(t, "jewel-b", got[0].ID)
}

func TestProposedJewels(t *testing.T) {
	t.Parallel()
	jewels := map[string][]Jewel{
		"source": {
			{ID: "jewel-1", ChestID: "source", Status: domain.JewelStatusProposed},
			{ID: "jewel-2", ChestID: "source", Status: domain.JewelStatusAccepted},
		},
	}

	got := ProposedJewels(jewels)
	require.Len(t, got, 1)
	assert.Equal(t, "jewel-1", got[0].ID)
}

func TestFindJewel(t *testing.T) {
	t.Parallel()
	jewels := map[string][]Jewel{
		"source": {{ID: "jewel-1", ChestID: "source"}},
	}

	got, ok := FindJewel(jewels, "jewel-1")
	require.True(t, ok)
	assert.Equal(t, "source", got.ChestID)

	_, ok = FindJewel(jewels, "missing")
	assert.False(t, ok)
}

func TestParseItemIDs(t *testing.T) {
	t.Parallel()

	got := ParseItemIDs(" jewel-1, jewel-2 ", "jewel-2", ",", "jewel-3")

	assert.Equal(t, []string{"jewel-1", "jewel-2", "jewel-3"}, got)
}

func TestCandidateScoresUseDefaultPolicy(t *testing.T) {
	t.Parallel()

	cluster := Cluster{CitedMissions: []string{"a", "b"}, Tags: []string{"cache", "widget"}}
	gap := Gap{CitedMissions: []string{"a", "b"}}

	assert.Equal(t, 70, ClusterCandidateScore(cluster))
	assert.Equal(t, 60, GapCandidateScore(gap))
}

func TestCandidateScoresUseCustomPolicy(t *testing.T) {
	t.Parallel()

	policy := ScoringPolicy{
		ClusterBase:          10,
		ClusterMissionWeight: 20,
		ClusterTagWeight:     1,
		GapBase:              5,
		GapMissionWeight:     25,
		MaxScore:             50,
	}

	cluster := Cluster{CitedMissions: []string{"a", "b"}, Tags: []string{"cache", "widget"}}
	gap := Gap{CitedMissions: []string{"a", "b", "c"}}

	assert.Equal(t, 50, ClusterCandidateScoreWithPolicy(cluster, policy))
	assert.Equal(t, 50, GapCandidateScoreWithPolicy(gap, policy))
}
