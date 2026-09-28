package telemetry

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// DeclaredComparison is the result of comparing the claims a handoff declared with
// the records persisted for the mission. Ids are "<agent>/<claim id>", sorted.
type DeclaredComparison struct {
	DeclaredClaims     int
	DeclaredAssertions int
	DeclaredQuestions  int
	PersistedClaims    int
	// PersistedQuestions counts declared questions persisted as reported.
	PersistedQuestions int
	// MissingRecord is true when the summary itself declares no record.
	MissingRecord     bool
	Unpersisted       []string
	PersistedRejected []string
	Mismatched        []string
	UnderOtherAgent   []string
}

// ReviewRecommended reports whether any declared claim is not faithfully persisted.
func (c DeclaredComparison) ReviewRecommended() bool {
	return len(c.Unpersisted)+len(c.PersistedRejected)+len(c.Mismatched)+len(c.UnderOtherAgent) > 0
}

type claimKey struct{ agent, id string }

// CompareDeclaredToPersisted classifies each claim of summary against records,
// keyed by (agent, claim id). It is pure and read-only: a `missing` record never
// counts as persisted, duplicate lines count once, and statements are not compared
// because a persisted record carries none.
func CompareDeclaredToPersisted(summary domain.ConfidenceSummary, records []ConfidenceRecord) DeclaredComparison {
	if summary.MissingRecord {
		return DeclaredComparison{MissingRecord: true}
	}
	byPair, reportedAgents := indexRecords(records)
	out := DeclaredComparison{}
	claims := append(append([]domain.ConfidenceClaim(nil), summary.Claims...), summary.OpenQuestions...)
	for _, claim := range claims {
		out.count(claim)
		out.classify(claim, byPair[claimKey{claim.Agent, claim.ID}], reportedAgents[claim.ID])
	}
	sort.Strings(out.Unpersisted)
	sort.Strings(out.PersistedRejected)
	sort.Strings(out.Mismatched)
	sort.Strings(out.UnderOtherAgent)
	return out
}

func (c *DeclaredComparison) count(claim domain.ConfidenceClaim) {
	c.DeclaredClaims++
	if claim.ClaimKind == domain.ClaimKindQuestion {
		c.DeclaredQuestions++
		return
	}
	c.DeclaredAssertions++
}

// indexRecords groups records by (agent, claim id) and lists, per claim id, the
// agents that reported it. The latest reported record of a pair wins; a rejected
// one is kept only when the pair has no reported record.
func indexRecords(records []ConfidenceRecord) (map[claimKey]pairState, map[string][]string) {
	byPair := make(map[claimKey]pairState)
	reportedAgents := make(map[string][]string)
	for _, record := range records {
		key := claimKey{record.Agent, record.ClaimID}
		state := byPair[key]
		switch record.CoverageStatus {
		case ConfidenceCoverageReported:
			state.reported, state.hasReported = record, true
			reportedAgents[record.ClaimID] = append(reportedAgents[record.ClaimID], record.Agent)
		case ConfidenceCoverageRejected:
			state.rejected = true
		}
		byPair[key] = state
	}
	return byPair, reportedAgents
}

type pairState struct {
	reported    ConfidenceRecord
	hasReported bool
	rejected    bool
}

func (c *DeclaredComparison) classify(claim domain.ConfidenceClaim, state pairState, reportedBy []string) {
	id := claim.Agent + "/" + claim.ID
	switch {
	case state.hasReported && recordMatchesClaim(state.reported, claim):
		c.PersistedClaims++
		if claim.ClaimKind == domain.ClaimKindQuestion {
			c.PersistedQuestions++
		}
	case state.hasReported:
		c.Mismatched = append(c.Mismatched, id)
	case state.rejected:
		c.PersistedRejected = append(c.PersistedRejected, id)
	case len(reportedBy) > 0:
		c.UnderOtherAgent = append(c.UnderOtherAgent, id)
	default:
		c.Unpersisted = append(c.Unpersisted, id)
	}
}

// recordMatchesClaim compares kind, percent, level and correlation key. A claim
// that omits its level is compared at the level its percent derives.
func recordMatchesClaim(record ConfidenceRecord, claim domain.ConfidenceClaim) bool {
	level := claim.ConfidenceLevel
	if level == "" {
		derived, err := domain.ConfidenceLevelForPercent(claim.ConfidencePercent)
		if err != nil {
			return false
		}
		level = derived
	}
	return record.ClaimKind == claim.ClaimKind && record.ConfidencePercent == claim.ConfidencePercent &&
		record.ConfidenceLevel == level && record.CorrelationKey == claim.CorrelationKey
}
