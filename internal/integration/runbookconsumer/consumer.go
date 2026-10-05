// Package runbookconsumer is the bounded, advisory JEV consumer for prepared
// Runbook candidates. It projects only the candidate review fields and never
// authorizes canonical materialization.
package runbookconsumer

import (
	"context"
	"log/slog"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
)

const (
	// ConsumerName identifies the Runbook preparation consumer.
	ConsumerName = "runbook_prepare"
	// Transition identifies the Runbook preparation transition.
	Transition = "runbook_prepare"
	// AggregateKey is the provider answer key for candidate support.
	AggregateKey = "runbook_candidate_supported"
	// OptionSupport is the provider answer for a supported candidate.
	OptionSupport = "support"
	// OptionReview is the provider answer for a candidate needing review.
	OptionReview = "review"
	// OptionReject is the provider answer for an unsupported candidate.
	OptionReject = "reject"
)

// Consumer is the bounded, advisory Runbook provider consumer.
type Consumer struct {
	Mechanism *mechanism.Mechanism
	Provider  config.Provider
	Name      string
	Version   string
	Ledger    handoffconsumer.Ledger
	Now       func() time.Time
}

func (c *Consumer) binding() integration.Binding {
	name := c.Name
	if name == "" {
		name = "jev"
	}
	version := c.Version
	if version == "" {
		version = "1"
	}
	return integration.Binding{Provider: name, Version: version, Model: c.Provider.Model, Consumer: ConsumerName, Capability: integration.CapConfidenceEvaluate}
}

// Evaluate uses JEV when the complete mechanism preflight permits it. Every
// failure is returned as a main-path reading; callers continue deterministically.
func (c *Consumer) Evaluate(ctx context.Context, request preparerunbook.ProviderRequest) (preparerunbook.ProviderReading, error) {
	reading := preparerunbook.ProviderReading{Path: preparerunbook.ProviderPathMain, Reason: string(integration.StateDisabled)}
	if !c.providerEnabled() {
		return reading, nil
	}
	if reading := c.rateLimited(request.MissionID); reading != nil {
		return *reading, nil
	}
	state, err := project(request, c.Provider.DataPolicy.AllowedFields, c.Provider.DataPolicy.MaxStateBytes)
	if err != nil {
		return fallback(state), nil
	}
	outcome := c.Mechanism.Call(ctx, c.binding(), integration.Request{
		Correlation: integration.Correlation{MissionID: request.MissionID, Consumer: ConsumerName},
		Capability:  integration.CapConfidenceEvaluate, State: state, Questions: Questions(),
	})
	reading = readingFromOutcome(outcome, c.Provider.Threshold())
	c.recordOutcome(request, outcome, reading)
	return reading, nil
}

func (c *Consumer) providerEnabled() bool {
	return c.Mechanism != nil && c.Provider.Enabled
}

func (c *Consumer) rateLimited(missionID string) *preparerunbook.ProviderReading {
	if c.Ledger.Path == "" {
		return nil
	}
	count, err := c.Ledger.Count(missionID, Transition)
	if err != nil {
		reading := fallback("ledger_unavailable")
		return &reading
	}
	if count >= c.Provider.Budget() {
		reading := fallback(string(integration.StateRateLimited))
		return &reading
	}
	return nil
}

func (c *Consumer) recordOutcome(request preparerunbook.ProviderRequest, outcome mechanism.Outcome, reading preparerunbook.ProviderReading) {
	if c.Ledger.Path == "" {
		return
	}
	record := handoffconsumer.Record{MissionID: request.MissionID, Transition: Transition, Subject: request.Subject, PreferredPath: string(outcome.Decision.Preferred), EffectivePath: string(outcome.Decision.Effective), Reason: string(readingReason(reading)), FellBack: outcome.Decision.FellBack, Suspect: outcome.Decision.Suspect, Binding: outcome.Decision.Binding, Attempted: outcome.Attempted, Approved: reading.Approved, Confidence: reading.Confidence, Threshold: reading.Threshold, Provider: c.binding().Provider, Answered: outcome.Result != nil}
	if outcome.Result != nil {
		record.Model = outcome.Result.Model
		record.InputTokens = outcome.Result.Usage.InputTokens
		record.OutputTokens = outcome.Result.Usage.OutputTokens
	}
	if err := c.Ledger.Append(record); err != nil {
		slog.Warn("runbook provider reading was not recorded", "mission_id", request.MissionID, "reason", err)
	}
}
