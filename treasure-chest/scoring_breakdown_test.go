package treasure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClusterCandidateBreakdown_ReturnsIndependentDimensions(t *testing.T) {
	t.Parallel()
	c := Cluster{
		ID:            "c1",
		CitedMissions: []string{"m1", "m2"},
		Tags:          []string{"t1", "t2"},
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	chest := GovernedChest{
		ID:    MissionHistoryChestID,
		Trust: GovernedTrust{Tier: "T3"},
		Grade: GovernedGrade{SourceGrade: "A", ReuseValue: "high"},
	}
	policy := DefaultScoringPolicy()

	got := ClusterCandidateBreakdown(c, policy, chest)

	// RecurrenceScore matches the pre-existing single-number score exactly —
	// this dimension is not a new computation, just exposed by name.
	assert.Equal(t, ClusterCandidateScoreWithPolicy(c, policy), got.RecurrenceScore)
	assert.Equal(t, got.RecurrenceScore, got.Aggregate)

	// Each dimension is independently inspectable and, given full governed
	// metadata, non-zero/high — not folded into one opaque number.
	assert.Equal(t, 100, got.EvidenceQuality)
	assert.Equal(t, 100, got.SourceTrust)
	assert.Equal(t, 100, got.ReuseValue)
	assert.InDelta(t, 100, got.Freshness, 1)
}

func TestClusterCandidateBreakdown_UngovernedChestReadsZeroNotFabricated(t *testing.T) {
	t.Parallel()
	c := Cluster{ID: "c1", CitedMissions: []string{"m1", "m2"}, Tags: []string{"t1", "t2"}}
	policy := DefaultScoringPolicy()

	got := ClusterCandidateBreakdown(c, policy, GovernedChest{})

	assert.Zero(t, got.EvidenceQuality)
	assert.Zero(t, got.SourceTrust)
	assert.Zero(t, got.ReuseValue)
	assert.Zero(t, got.Freshness, "no GeneratedAt means no freshness signal, not a guessed one")
	assert.Positive(t, got.RecurrenceScore, "recurrence is unaffected by missing governed metadata")
}

func TestGapCandidateBreakdown_ReturnsIndependentDimensions(t *testing.T) {
	t.Parallel()
	g := Gap{ID: "g1", CitedMissions: []string{"m1"}, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	chest := GovernedChest{
		Trust: GovernedTrust{Tier: "T1"},
		Grade: GovernedGrade{SourceGrade: "B", ReuseValue: "medium"},
	}
	policy := DefaultScoringPolicy()

	got := GapCandidateBreakdown(g, policy, chest)

	assert.Equal(t, GapCandidateScoreWithPolicy(g, policy), got.RecurrenceScore)
	assert.Equal(t, got.RecurrenceScore, got.Aggregate)
	assert.Equal(t, 65, got.EvidenceQuality)
	assert.Equal(t, 50, got.SourceTrust)
	assert.Equal(t, 60, got.ReuseValue)
	assert.InDelta(t, 100, got.Freshness, 1)
}

func TestFreshnessScore_DecaysWithAgeAndHandlesBadInput(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, freshnessScore(""), "empty timestamp is an honest 0, not a guess")
	assert.Equal(t, 0, freshnessScore("not-a-timestamp"))
	assert.Equal(t, 100, freshnessScore(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)),
		"a future/just-now timestamp should not score below 100")

	old := time.Now().Add(-60 * 24 * time.Hour).UTC().Format(time.RFC3339)
	assert.Zero(t, freshnessScore(old), "well past the freshness window should bottom out at 0")

	half := time.Now().Add(-freshnessWindow / 2).UTC().Format(time.RFC3339)
	assert.InDelta(t, 50, freshnessScore(half), 2)
}

func TestSourceGradeReuseValueTrustTierScores_UnrecognizedInputsAreZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 100, sourceGradeScore("A"))
	assert.Equal(t, 65, sourceGradeScore("B"))
	assert.Equal(t, 35, sourceGradeScore("C"))
	assert.Zero(t, sourceGradeScore(""))
	assert.Zero(t, sourceGradeScore("bogus"))

	assert.Equal(t, 100, reuseValueScore("high"))
	assert.Equal(t, 60, reuseValueScore("medium"))
	assert.Equal(t, 20, reuseValueScore("low"))
	assert.Zero(t, reuseValueScore(""))

	assert.Equal(t, 25, trustTierScore("T0"))
	assert.Equal(t, 50, trustTierScore("T1"))
	assert.Equal(t, 75, trustTierScore("T2"))
	assert.Equal(t, 100, trustTierScore("T3"))
	assert.Zero(t, trustTierScore(""))
}
