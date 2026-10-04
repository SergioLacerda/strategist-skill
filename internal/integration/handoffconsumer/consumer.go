package handoffconsumer

import (
	"context"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
)

// Status is the outcome of one pre-check for the agent.
type Status string

// Pre-check statuses.
const (
	// StatusApproved: the provider's confidence cleared the threshold; the
	// delegation can be recorded and the agent skips its own conformance check.
	StatusApproved Status = "approved"
	// StatusSignal: the provider answered but did not approve; main proceeds and
	// the agent receives the unsatisfied criteria as repair hints.
	StatusSignal Status = "signal"
	// StatusFallback: the provider did not serve the call; main proceeds.
	StatusFallback Status = "fallback"
	// StatusDisabled: no integration is enabled; main proceeds silently.
	StatusDisabled Status = "disabled"
)

// Request is one pre-check: which handoff, which revision, and where its
// contract fields can be read.
type Request struct {
	MissionID  string
	Transition string
	// Subject is the digest of the revision under check; the delegation is bound to it.
	Subject string
	Source  Source
}

// Report is what the caller learns. Delegation is set only for StatusApproved.
type Report struct {
	Status     Status
	Reason     integration.State
	Decision   policy.Decision
	Confidence float64
	Threshold  float64
	Delegation *handoff.Delegation
	Hints      []Hint
	Attempted  bool
}

// Consumer runs pre-checks through the Integration Mechanism.
type Consumer struct {
	Mechanism *mechanism.Mechanism
	Provider  config.Provider
	Name      string
	Version   string
	Ledger    Ledger
	Now       func() time.Time
}

func (c *Consumer) binding() integration.Binding {
	return integration.Binding{Provider: c.Name, Version: c.Version, Model: c.Provider.Model, Consumer: "handoff", Capability: integration.CapHandoffValidate}
}

// Precheck asks the provider whether the artifact conforms and applies the local
// approval rule. It never blocks: every failure becomes a fallback to main.
func (c *Consumer) Precheck(ctx context.Context, request Request) Report {
	questions := Questions(request.Transition)
	if !c.Provider.Enabled || len(questions) == 0 {
		return Report{Status: StatusDisabled, Reason: integration.StateDisabled}
	}
	if state := c.refuse(request); state != "" {
		return c.fallback(request, state)
	}
	text, err := Project(request.Transition, request.Source, c.Provider.DataPolicy.AllowedFields, c.Provider.DataPolicy.MaxStateBytes)
	if err != nil {
		state, _ := integration.StateOf(err)
		return c.fallback(request, state)
	}
	started := c.now()
	outcome := c.Mechanism.Call(ctx, c.binding(), integration.Request{
		Correlation: integration.Correlation{MissionID: request.MissionID, Consumer: "handoff"},
		Capability:  integration.CapHandoffValidate, State: text, Questions: questions,
	})
	report := c.report(request, outcome)
	c.record(request, report, outcome, c.now().Sub(started))
	return report
}

// refuse returns a state when the call budget is spent or cannot be read. The
// budget is independent of the handoff's own attempts and counts only calls made.
func (c *Consumer) refuse(request Request) integration.State {
	count, err := c.Ledger.Count(request.MissionID, request.Transition)
	switch {
	case err != nil:
		return integration.StateUnavailable
	case count >= c.Provider.Budget():
		return integration.StateRateLimited
	}
	return ""
}

func (c *Consumer) fallback(request Request, state integration.State) Report {
	decision := policy.Select(state, c.binding())
	report := Report{Status: StatusFallback, Reason: state, Decision: decision}
	c.record(request, report, mechanism.Outcome{Decision: decision}, 0)
	return report
}

func (c *Consumer) report(request Request, outcome mechanism.Outcome) Report {
	report := Report{Reason: outcome.Decision.Reason, Decision: outcome.Decision, Attempted: outcome.Attempted, Threshold: c.Provider.Threshold()}
	if outcome.Result == nil {
		report.Status = StatusFallback
		if outcome.Decision.Reason == integration.StateDisabled {
			report.Status = StatusDisabled
		}
		return report
	}
	verdict := Evaluate(request.Transition, *outcome.Result, report.Threshold)
	report.Confidence, report.Hints = verdict.Confidence, verdict.Hints
	if !verdict.Approved {
		report.Status = StatusSignal
		return report
	}
	report.Status = StatusApproved
	report.Delegation = &handoff.Delegation{
		Provider: c.Name, Model: outcome.Result.Model, BindingDigest: outcome.Decision.Binding,
		Capability: string(integration.CapHandoffValidate), Criterion: AggregateKey, Subject: request.Subject,
		Checks: handoff.DelegableChecks(), Confidence: verdict.Confidence, Threshold: report.Threshold,
		InputTokens: outcome.Result.Usage.InputTokens, OutputTokens: outcome.Result.Usage.OutputTokens,
	}
	return report
}

func (c *Consumer) record(request Request, report Report, outcome mechanism.Outcome, latency time.Duration) {
	record := Record{
		At: c.now().UTC().Format(time.RFC3339Nano), MissionID: request.MissionID, Transition: request.Transition, Subject: request.Subject,
		PreferredPath: string(policy.PathJEV), EffectivePath: string(outcome.Decision.Effective), Reason: string(report.Reason),
		FellBack: outcome.Decision.FellBack, Suspect: outcome.Decision.Suspect, Binding: outcome.Decision.Binding,
		Attempted: outcome.Attempted, Approved: report.Status == StatusApproved, Confidence: report.Confidence,
		Threshold: report.Threshold, LatencyMS: latency.Milliseconds(),
	}
	if outcome.Result != nil {
		record.Model = outcome.Result.Model
		record.InputTokens, record.OutputTokens = outcome.Result.Usage.InputTokens, outcome.Result.Usage.OutputTokens
	}
	if err := c.Ledger.Append(record); err != nil {
		return // observability only: a ledger failure never blocks a handoff
	}
}

func (c *Consumer) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}
