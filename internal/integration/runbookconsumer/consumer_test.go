package runbookconsumer

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuestionsAreTypedAndAggregateComesFirst(t *testing.T) {
	questions := Questions()
	assert.Equal(t, AggregateKey, questions[0].Key)
	assert.Equal(t, integration.KindChoice, questions[0].Kind)
	assert.Len(t, questions, 5)
}

func TestProjectEnforcesAllowedFieldsAndSize(t *testing.T) {
	request := preparerunbook.ProviderRequest{Fields: map[string]string{"objective": "recover", "verification": "green"}}
	state, err := project(request, []string{"objective"}, 100)
	require.NoError(t, err)
	assert.Contains(t, state, "recover")
	assert.NotContains(t, state, "green")
	_, err = project(request, []string{"verification"}, 2)
	assert.ErrorContains(t, err, "data_policy_denied")
}

func TestReadingUsesLocalThresholdAndKeepsProviderAdvisory(t *testing.T) {
	result := integration.Result{Model: "jev-1.13.0", Answers: map[string]integration.Answer{AggregateKey: {Kind: integration.KindChoice, Choice: OptionSupport, HasConfidence: true, Confidence: 0.80}}}
	reading := readingFromOutcome(outcomeWithResult(result), 0.80)
	assert.Equal(t, preparerunbook.ProviderPathJEV, reading.Path)
	assert.False(t, reading.Approved, "threshold is strict")
	assert.Equal(t, "local_threshold_not_cleared", reading.Reason)

	result.Answers[AggregateKey] = integration.Answer{Kind: integration.KindChoice, Choice: OptionSupport, HasConfidence: true, Confidence: 0.95}
	reading = readingFromOutcome(outcomeWithResult(result), 0.80)
	assert.True(t, reading.Approved)
}

func outcomeWithResult(result integration.Result) mechanism.Outcome {
	return mechanism.Outcome{Result: &result, Decision: policy.Decision{Effective: policy.PathJEV, Preferred: policy.PathJEV, Binding: "sha256:binding"}, Attempted: true}
}
