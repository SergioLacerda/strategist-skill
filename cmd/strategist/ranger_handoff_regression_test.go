package main

// Ranger handoff command-boundary regression coverage.

import (
	"bytes"
	"testing"

	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRangerHandoffEvaluateOptionsValidateRequiredPairs(t *testing.T) {
	require.ErrorContains(t, validateRangerHandoffOptions(rangerHandoffEvaluateOptions{}), "--mission-id")
	require.ErrorContains(t, validateRangerHandoffOptions(rangerHandoffEvaluateOptions{MissionID: "m1", Challenges: "challenges.yaml"}), "--challenges and --ack")
	require.NoError(t, validateRangerHandoffOptions(rangerHandoffEvaluateOptions{MissionID: "m1"}))
}

func TestRangerHandoffInputWithoutChallengesIsEmpty(t *testing.T) {
	got, err := rangerHandoffInput(rangerHandoffEvaluateOptions{})
	require.NoError(t, err)
	assert.Equal(t, livemission.RangerHandoffInput{}, got)
}

func TestPrintRangerHandoffEvaluationRendersSummaryWithoutChallengeResult(t *testing.T) {
	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

	require.NoError(t, printRangerHandoffEvaluation(cmd, livemission.RangerHandoffResult{}))
	assert.Contains(t, output.String(), "transition: ranger_to_archivist")
	assert.Contains(t, output.String(), "required: false")
}
