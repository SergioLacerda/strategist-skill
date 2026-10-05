package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func runCriteria(t *testing.T, opts handoffCriteriaOptions) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := runHandoffCriteria(cmd, opts)
	return out.String(), err
}

func TestCriteriaPrintsEveryRowFilledEvenForAMissionWithNoRecords(t *testing.T) {
	root := informationalMission(t)
	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)

	for _, criterion := range []string{"route_confidence", "claim_confidence.ranger", "confidence_review", "critic", "mission_quality", "provider.archivist_to_gate"} {
		require.Contains(t, out, criterion)
	}
	require.Contains(t, out, "não avaliado")
	require.NotContains(t, out, ": \n", "no value is ever blank")
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		require.NotEmpty(t, strings.TrimSpace(line))
	}
}

func TestCriteriaShowsTheGateReadingAfterAPrecheck(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)
	_, err := runPrecheck(t, root)
	require.NoError(t, err)

	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Contains(t, out, "provider.archivist_to_gate: JEV confiança: 96% (limiar 80%) — aprovado, não calibrado")
	require.Contains(t, out, "handoff.archivist_to_sniper: não avaliado", "only after the gate")
	require.Equal(t, 1, sim.Calls(), "the report is read-only and never calls the provider")
}

func TestCriteriaJSONHasNoEmptyField(t *testing.T) {
	root := informationalMission(t)
	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1", JSON: true})
	require.NoError(t, err)
	var parsed struct {
		MissionID string           `json:"mission_id"`
		Rows      []map[string]any `json:"rows"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Equal(t, "m1", parsed.MissionID)
	require.NotEmpty(t, parsed.Rows)
	for _, row := range parsed.Rows {
		for field, value := range row {
			require.NotEmpty(t, value, "%v.%s", row["criterion"], field)
		}
	}
}

func TestCriteriaNeedsAMissionAndARoot(t *testing.T) {
	_, err := runCriteria(t, handoffCriteriaOptions{})
	require.ErrorContains(t, err, "--mission-id")
	_, err = runCriteria(t, handoffCriteriaOptions{Root: t.TempDir() + "/absent", MissionID: "m1"})
	require.Error(t, err)
}
