package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildClusters_SortsMultipleClusters(t *testing.T) {
	t.Parallel()
	missions := []ScannedMission{
		{MissionID: "m-zzz-1", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-zzz-2", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-aaa-1", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
		{MissionID: "m-aaa-2", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
	}
	clusters := BuildClusters(missions)
	require.Len(t, clusters, 2)
	assert.Less(t, clusters[0].ID, clusters[1].ID, "clusters must be sorted by ID")
}

func TestClusterID_TruncatesToTwoTags(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cluster-alpha-beta", ClusterID([]string{"alpha", "beta", "gamma"}))
}
