package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGaps_DuplicateIDAcrossMissionsAppendsCitation(t *testing.T) {
	t.Parallel()
	missions := []ScannedMission{
		{MissionID: "mission-a", SQs: []SQEntry{{ID: "SQ-1", Status: "sq_pending"}}},
		{MissionID: "mission-b", SQs: []SQEntry{{ID: "SQ-1", Status: "sq_pending"}}},
	}
	gaps := BuildGaps(missions)
	require.Len(t, gaps, 1)
	assert.Equal(t, []string{"mission-a", "mission-b"}, gaps[0].CitedMissions)
}

func TestBuildGaps_NonPendingStatusSkipped(t *testing.T) {
	t.Parallel()
	missions := []ScannedMission{
		{MissionID: "mission-a", SQs: []SQEntry{{ID: "SQ-1", Status: "sq_backlog"}}},
	}
	gaps := BuildGaps(missions)
	assert.Empty(t, gaps, "a side quest that isn't sq_pending must never become a gap")
}
