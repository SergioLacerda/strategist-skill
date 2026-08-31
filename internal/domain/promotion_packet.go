package domain

import "time"

// PromotionPacketAgingThresholdDays is the number of days a pending
// promotion packet can sit without a promote/reject decision before
// EvaluatePromotionPacketAging flags it as aged. See
// .analysis/refined/20260830-skill-gaps-triage/promotion-packet-design.md
// § "What aging means" for the rationale (30 days approximates a monthly
// backlog-grooming cadence, consistent with this repo's own
// occasional/manual periodic-review pattern rather than a continuous one).
const PromotionPacketAgingThresholdDays = 30

// PromotionPacketStatus is a promotion packet's lifecycle position,
// mirroring the proposed/accepted/verified/deprecated status vocabulary
// internal/embed/defaults/jewels.yaml and potions.yaml already use for
// candidate-status data (see ADR-0012), narrowed to the three outcomes a
// promotion packet actually has: it gets promoted into a real artifact,
// rejected, or expires unactioned.
type PromotionPacketStatus string

// Promotion packet status values.
const (
	PromotionPacketStatusPending  PromotionPacketStatus = "pending"
	PromotionPacketStatusPromoted PromotionPacketStatus = "promoted"
	PromotionPacketStatusRejected PromotionPacketStatus = "rejected"
	PromotionPacketStatusExpired  PromotionPacketStatus = "expired"
)

// PromotionPacket is a self-contained candidate for promotion into a
// durable artifact (a docs/runbooks/*.runbook.yaml + its .md, or a
// docs/adr/*.md) that a mission's discovery/refinement pass surfaced as a
// side quest but did not itself write. See the design doc referenced above
// for the full rationale behind this shape and where packets should live
// (.strategist/promotion-packets.yaml, per that doc's recommendation).
//
// This type is intentionally pure data: nothing in this file reads or
// writes a file. The read/write plumbing (a loader/writer, a CLI surface,
// wiring into a mission phase) is deliberately left to a future mission,
// per this task's scope — a design note plus a minimal skeleton, not a
// full pipeline — so that mission can build against a settled shape
// instead of guessing one from scratch.
type PromotionPacket struct {
	// PacketID is a unique, stable identifier for this candidate (e.g. a
	// slug derived from OriginMissionID plus a short description).
	PacketID string

	// OriginMissionID backlinks to the mission whose discovery/refinement
	// pass first surfaced this candidate. This doubles as the "backlink to
	// originating mission" the packet needs — the mission that raised the
	// candidate is the backlink target, so there is no separate field for
	// it.
	OriginMissionID string

	// TriggerReason is why this candidate was raised: the observed
	// condition or pattern that suggests a runbook or ADR is missing.
	TriggerReason string

	// Procedure is the candidate's proposed steps, in the shape of a
	// runbook's analysis/verification list — a first-draft procedure
	// sketch, not yet a curated docs/runbooks/*.runbook.yaml sidecar.
	Procedure []string

	// VerificationChecks are the checks a reviewer should confirm hold
	// before promoting this packet, analogous to a runbook's Checks.
	VerificationChecks []string

	// SuggestedOwner is who the originating mission believes should review
	// or own this candidate (a role name, a team, or "unassigned").
	SuggestedOwner string

	// CreatedAt is when this packet was raised — the aging clock's zero
	// point; see EvaluatePromotionPacketAging.
	CreatedAt time.Time

	// ExpiresAt is an optional, packet-declared hard deadline after which
	// the candidate should be discarded outright, distinct from the
	// generic aging window: aging is a flag that applies to every pending
	// packet regardless of whether it declares its own deadline.
	ExpiresAt *time.Time

	// Status is the packet's current lifecycle position.
	Status PromotionPacketStatus
}

// PromotionPacketAgingDecision reports how long a packet has been open and
// whether that duration has reached PromotionPacketAgingThresholdDays.
type PromotionPacketAgingDecision struct {
	DaysOpen int
	Aged     bool
}

// EvaluatePromotionPacketAging determines whether a packet created at
// createdAt has aged as of today, per PromotionPacketAgingThresholdDays. A
// packet exactly at the threshold (DaysOpen == PromotionPacketAgingThresholdDays)
// counts as aged: "aged" reads naturally as "has reached the threshold,"
// and treating the boundary day as still-fresh would let a packet's flag
// lag depending on what time of day the check happens to run.
//
// This function is pure: it takes both dates as inputs (mirroring
// EvaluateCriticalHit's evidence-in/decision-out shape in
// critical_hit_trigger.go) rather than calling time.Now() itself, so it
// stays deterministic and independently testable.
func EvaluatePromotionPacketAging(createdAt, today time.Time) PromotionPacketAgingDecision {
	daysOpen := int(today.Sub(createdAt).Hours() / 24)
	return PromotionPacketAgingDecision{
		DaysOpen: daysOpen,
		Aged:     daysOpen >= PromotionPacketAgingThresholdDays,
	}
}
