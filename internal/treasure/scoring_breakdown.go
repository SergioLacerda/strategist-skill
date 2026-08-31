package treasure

import "time"

// ScoreBreakdown holds independently-inspectable scoring dimensions for a
// mined cluster or gap candidate, alongside an Aggregate kept for backward
// compatibility with callers (ClusterCandidateScore(WithPolicy),
// GapCandidateScore(WithPolicy), JewelScore.Value) that only ever read one
// composite number. Splitting the aggregate into named dimensions is G08's
// fix: a rare-but-important finding (e.g. high SourceTrust/EvidenceQuality,
// low mission-recurrence count) can be inspected on its own instead of being
// buried inside a single structural-recurrence score. See
// .analysis/refined/20260830-skill-gaps-triage/analysis.md Cluster 7 (K13)
// and .analysis/refined/20260830-skill-gaps-triage/tasks.md Task 8.
type ScoreBreakdown struct {
	// RecurrenceScore is the pre-existing structural score: base +
	// missions*weight [+ tags*weight for clusters], clamped to the policy's
	// MaxScore. Identical to what ClusterCandidateScoreWithPolicy /
	// GapCandidateScoreWithPolicy already compute and return today.
	RecurrenceScore int `json:"recurrence_score"`
	// EvidenceQuality reflects how well-graded the candidate's target chest
	// is. Real signal: GovernedChest.Grade.SourceGrade (A/B/C, the same
	// human-reviewed field domain.ChestGrade.SourceGrade models — see
	// internal/domain/chest_grade.go), scaled to 0-100. 0 when the target
	// chest has no recorded source_grade yet (mined clusters/gaps land in
	// the synthetic "mission-history" chest, which is not human-graded by
	// default) — that 0 is a genuine "ungraded" reading, not a fabricated
	// quality score.
	EvidenceQuality int `json:"evidence_quality"`
	// Freshness is a 0-100 recency score derived from the candidate's own
	// GeneratedAt timestamp (Cluster.GeneratedAt / Gap.GeneratedAt, set at
	// mine time by scan_time.go's nowISO): 100 when just generated, decaying
	// linearly to 0 over freshnessWindow. Real signal, not a placeholder —
	// but note today's only call path (RunScanPipeline building candidates
	// immediately after BuildClusters/BuildGaps) always scores a
	// freshly-generated candidate, so it currently reads at or near 100;
	// the decay only shows once a caller re-scores an older, persisted
	// candidate.
	Freshness int `json:"freshness"`
	// SourceTrust reflects the trust tier of the candidate's target chest.
	// Real signal: GovernedChest.Trust.Tier (T0-T3, the same vocabulary
	// internal/domain/jewel_grade.go's tierOrder uses), scaled to 0-100. 0
	// when the target chest has no recorded tier — an honest "unknown
	// trust" reading.
	SourceTrust int `json:"source_trust"`
	// ReuseValue reflects the candidate's target chest's recorded reuse
	// value. Real signal: GovernedChest.Grade.ReuseValue (high/medium/low,
	// the same field domain.ChestGrade.ReuseValue models), scaled to 0-100.
	// 0 when unset.
	ReuseValue int `json:"reuse_value"`
	// Aggregate mirrors the pre-existing single composite score
	// (RecurrenceScore) for backward compatibility with callers that read
	// only one number. It does not yet fold in the other dimensions — no
	// combination formula has been agreed for that, and G08's ask was
	// separable dimensions, not a re-weighted aggregate.
	Aggregate int `json:"aggregate"`
}

// freshnessWindow is the age at which Freshness bottoms out at 0. 30 days is
// a starting default, not a governed policy value yet — ScoringPolicy has no
// override field for it today.
const freshnessWindow = 30 * 24 * time.Hour

// ClusterCandidateBreakdown scores a recurring-mission cluster candidate
// across separable dimensions. chest is the GovernedChest the candidate
// would attach to (typically looked up from LoadGoverned by the caller using
// MissionHistoryChestID); pass the zero value when no governed metadata is
// available — EvidenceQuality/SourceTrust/ReuseValue then read 0 rather than
// a fabricated number.
func ClusterCandidateBreakdown(c Cluster, policy ScoringPolicy, chest GovernedChest) ScoreBreakdown {
	return dimensionalBreakdown(ClusterCandidateScoreWithPolicy(c, policy), c.GeneratedAt, chest, policy)
}

// GapCandidateBreakdown scores an open side-quest gap candidate across
// separable dimensions. See ClusterCandidateBreakdown for the chest
// parameter's contract.
func GapCandidateBreakdown(g Gap, policy ScoringPolicy, chest GovernedChest) ScoreBreakdown {
	return dimensionalBreakdown(GapCandidateScoreWithPolicy(g, policy), g.GeneratedAt, chest, policy)
}

func dimensionalBreakdown(recurrence int, generatedAt string, chest GovernedChest, policy ScoringPolicy) ScoreBreakdown {
	return ScoreBreakdown{
		RecurrenceScore: recurrence,
		EvidenceQuality: clampScore(sourceGradeScore(chest.Grade.SourceGrade), policy.MaxScore),
		Freshness:       clampScore(freshnessScore(generatedAt), policy.MaxScore),
		SourceTrust:     clampScore(trustTierScore(chest.Trust.Tier), policy.MaxScore),
		ReuseValue:      clampScore(reuseValueScore(chest.Grade.ReuseValue), policy.MaxScore),
		Aggregate:       recurrence,
	}
}

// freshnessScore converts an RFC3339 GeneratedAt timestamp into a 0-100
// recency score. Empty or malformed input (no timestamp available) scores 0
// rather than guessing.
func freshnessScore(generatedAt string) int {
	if generatedAt == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return 0
	}
	age := time.Since(t)
	if age <= 0 {
		return 100
	}
	if age >= freshnessWindow {
		return 0
	}
	return int(100 * (1 - float64(age)/float64(freshnessWindow)))
}

// sourceGradeScoreByGrade maps domain.ChestGrade-style source_grade values
// (see internal/domain/chest_grade.go's validSourceGrades) to a 0-100 scale.
var sourceGradeScoreByGrade = map[string]int{"A": 100, "B": 65, "C": 35}

func sourceGradeScore(grade string) int {
	return sourceGradeScoreByGrade[grade] // zero value for "", or any unrecognized grade
}

// reuseValueScoreByValue maps domain.ChestGrade-style reuse_value values
// (see internal/domain/chest_grade.go's validReuseValues) to a 0-100 scale.
var reuseValueScoreByValue = map[string]int{"high": 100, "medium": 60, "low": 20}

func reuseValueScore(value string) int {
	return reuseValueScoreByValue[value] // zero value for "", or any unrecognized value
}

// trustTierScoreByTier maps the T0-T3 trust tier vocabulary (see
// internal/domain/jewel_grade.go's tierOrder) to a 0-100 scale.
var trustTierScoreByTier = map[string]int{"T0": 25, "T1": 50, "T2": 75, "T3": 100}

func trustTierScore(tier string) int {
	return trustTierScoreByTier[tier] // zero value for "", or any unrecognized tier
}
