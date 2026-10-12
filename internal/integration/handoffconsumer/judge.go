package handoffconsumer

import (
	"context"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
)

// Judge is the JEV handoff pre-check as a system under evaluation: it asks the
// same questions through the same Mechanism as a handoff does, and reports the
// provider's own numbers. It applies no threshold and keeps no call budget; the
// calibration decides both from the observations.
type Judge struct {
	Mechanism *mechanism.Mechanism
	Provider  config.Provider
	Name      string
	Version   string
	// Profile is the question profile asked; empty means the default (v1). It may
	// differ from the provider's configured profile, to measure a variant before adopting it.
	Profile string
	Now     func() time.Time
}

func (j Judge) binding() integration.Binding {
	return integration.Binding{Provider: j.Name, Version: j.Version, Model: j.Provider.Model, Consumer: "handoff", Capability: integration.CapHandoffValidate}
}

// Judge projects the fields under the operator's data policy and asks the provider.
func (j Judge) Judge(ctx context.Context, boundary string, fields calibration.Fields) calibration.Verdict {
	transition := TransitionForBoundary(boundary)
	text, err := ProjectFor(j.Profile, transition, MapSource(fields), j.Provider.DataPolicy.AllowedFields, j.Provider.DataPolicy.MaxStateBytes)
	if err != nil {
		state, _ := integration.StateOf(err)
		return calibration.Verdict{State: string(state)}
	}
	started := j.now()
	outcome := j.Mechanism.Call(ctx, j.binding(), integration.Request{
		Correlation: integration.Correlation{MissionID: "dojo-calibration", Consumer: "handoff"},
		Capability:  integration.CapHandoffValidate, State: text, Questions: QuestionsFor(j.Profile, transition),
	})
	if calibration.IsAtomic(j.Profile) {
		return atomicVerdictOf(transition, outcome, j.now().Sub(started))
	}
	return verdictOf(outcome, j.now().Sub(started))
}

// StateMissingProbability marks an answer with no probability of accept, which a calibration cannot use.
const StateMissingProbability = "missing_probability"

// unanswered is the verdict of a call the provider did not serve.
func unanswered(outcome mechanism.Outcome) calibration.Verdict {
	state := outcome.Decision.Reason
	if state == "" {
		state = integration.StateUnavailable
	}
	return calibration.Verdict{State: string(state)}
}

func verdictOf(outcome mechanism.Outcome, latency time.Duration) calibration.Verdict {
	if outcome.Result == nil {
		return unanswered(outcome)
	}
	aggregate, ok := outcome.Result.Answers[AggregateKey]
	pAccept, known := acceptanceConfidence(aggregate)
	if !ok || !known || aggregate.Kind != integration.KindChoice {
		return calibration.Verdict{State: string(integration.StateInvalidResponse), Model: outcome.Result.Model}
	}
	if _, reported := aggregate.Probabilities[OptionAccept]; !reported {
		// A calibration measures the probability of accept. The documented confidence is
		// another quantity, so it is never substituted: the observation is unusable.
		return calibration.Verdict{State: StateMissingProbability, Model: outcome.Result.Model}
	}
	return calibration.Verdict{
		Decision: aggregate.Choice, Probabilities: aggregate.Probabilities, PAccept: pAccept, Confidence: aggregate.Confidence,
		Model: outcome.Result.Model, Latency: latency, InputTokens: outcome.Result.Usage.InputTokens, OutputTokens: outcome.Result.Usage.OutputTokens,
	}
}

func (j Judge) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}
