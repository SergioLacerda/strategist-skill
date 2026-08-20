// Package governancebridge defines Strategist's pluggable interface for
// delegating policy enforcement to an external governance system, without
// Strategist knowing that provider's internal logic (item 4 of the
// governança-plugável document; UNC-02 resolution: this is a new package —
// internal/governance stays intact as the concrete .sdd/ adapter, see
// internal/governance/bridge_adapter.go).
//
// A nil GovernanceBridge is the common, fully-supported case: Strategist
// runs standalone with no external governance configured (acceptance check
// 6.1 — absence of external governance must never break the pipeline).
package governancebridge

import "context"

// GovernanceRequest describes one policy question Strategist asks an
// external governance provider before or during a mission phase.
type GovernanceRequest struct {
	MissionID     string
	Phase         string
	Action        string
	CorrelationID string
}

// GovernanceDecision is the provider's answer. It is read-only advisory
// input to Strategist — evaluating a GovernanceDecision never itself mutates
// Strategist runtime state (acceptance check 6.7: no concurrent
// auto-correction between Strategist and external governance). Authority
// identifies who made the decision, using the closed taxonomy in
// internal/telemetry (telemetry.AuthorityStrategistLocal or
// telemetry.AuthorityExternal(providerID)).
type GovernanceDecision struct {
	Allowed       bool
	Reason        string
	PolicyID      string
	Authority     string
	CorrelationID string
}

// GovernanceBridge lets Strategist delegate a policy question to an external
// governance system. Evaluate MUST be read-only with respect to Strategist's
// own runtime and workspace state — it answers a question, it does not act
// on the answer. Strategist itself decides what to do with the returned
// GovernanceDecision (acceptance check 6.7).
type GovernanceBridge interface {
	Evaluate(ctx context.Context, request GovernanceRequest) (GovernanceDecision, error)
}
