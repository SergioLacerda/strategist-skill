package runbookconsumer

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
)

// Questions returns the bounded criteria sent to the confidence evaluator.
func Questions() []integration.Question {
	return []integration.Question{
		{Key: AggregateKey, Kind: integration.KindChoice, Instructions: "Does this candidate describe a reusable, novel or materially extended and sufficiently evidenced Runbook within the stated scope?", Options: []integration.Option{{Key: OptionSupport, Description: "The candidate is reusable and sufficiently supported."}, {Key: OptionReview, Description: "The candidate needs human review before any materialization."}, {Key: OptionReject, Description: "The candidate is not supported."}}},
		{Key: "reusable_procedure", Kind: integration.KindNoul, Instructions: "The proposed procedure is reproducible."},
		{Key: "novel_or_extension", Kind: integration.KindNoul, Instructions: "The candidate is novel or a material extension rather than a duplicate."},
		{Key: "evidence_sufficiency", Kind: integration.KindNoul, Instructions: "Commands, expected outputs, stop conditions and provenance are evidenced."},
		{Key: "scope_bounded", Kind: integration.KindNoul, Instructions: "The candidate stays within its declared scope."},
	}
}

func project(request preparerunbook.ProviderRequest, allowed []string, maxBytes int) (string, error) {
	var b strings.Builder
	for _, field := range []string{"objective", "applies_when", "preconditions", "verification", "comparison"} {
		if !contains(allowed, field) || strings.TrimSpace(request.Fields[field]) == "" {
			continue
		}
		fmt.Fprintf(&b, "## %s\n%s\n\n", field, strings.TrimSpace(request.Fields[field]))
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("%s: no allowed Runbook field can be sent", integration.StateDataPolicyDenied)
	}
	if maxBytes > 0 && b.Len() > maxBytes {
		return "", fmt.Errorf("%s: projected state exceeds the data policy size limit", integration.StateDataPolicyDenied)
	}
	return b.String(), nil
}

func readingFromOutcome(outcome mechanism.Outcome, threshold float64) preparerunbook.ProviderReading {
	reading := preparerunbook.ProviderReading{Path: preparerunbook.ProviderPathMain, Threshold: threshold, Attempted: outcome.Attempted, BindingDigest: outcome.Decision.Binding}
	if outcome.Result == nil {
		return unavailableReading(reading, outcome)
	}
	reading.Path = preparerunbook.ProviderPathJEV
	reading.Model = outcome.Result.Model
	answer, ok := outcome.Result.Answers[AggregateKey]
	if !ok || answer.Kind != integration.KindChoice {
		return invalidReading(reading)
	}
	reading.Confidence = answerConfidence(answer)
	reading.Approved = answer.Choice == OptionSupport && reading.Confidence > threshold
	if !reading.Approved {
		reading.Reason = "local_threshold_not_cleared"
	}
	return reading
}

func unavailableReading(reading preparerunbook.ProviderReading, outcome mechanism.Outcome) preparerunbook.ProviderReading {
	reading.Reason = string(outcome.Decision.Reason)
	if reading.Reason == "" {
		reading.Reason = string(integration.StateUnavailable)
	}
	return reading
}

func invalidReading(reading preparerunbook.ProviderReading) preparerunbook.ProviderReading {
	reading.Path = preparerunbook.ProviderPathMain
	reading.Reason = string(integration.StateInvalidResponse)
	return reading
}

func answerConfidence(answer integration.Answer) float64 {
	confidence, present := answer.Probabilities[OptionSupport]
	if present || !answer.HasConfidence {
		return confidence
	}
	if answer.Choice == OptionSupport {
		return answer.Confidence
	}
	return 1 - answer.Confidence
}

func fallback(reason string) preparerunbook.ProviderReading {
	return preparerunbook.ProviderReading{Path: preparerunbook.ProviderPathMain, Reason: reason}
}

func readingReason(reading preparerunbook.ProviderReading) integration.State {
	if reading.Reason == "" {
		return ""
	}
	return integration.State(reading.Reason)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
