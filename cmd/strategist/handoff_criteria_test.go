package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
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

func seedStageUsage(t *testing.T, root string) {
	t.Helper()
	tokens := func(v int64) *int64 { return &v }
	for _, r := range []struct {
		id, phase string
		in, out   int64
	}{{"a", "DISCOVERY", 100, 11}, {"b", "REFINEMENT", 200, 22}} {
		_, err := telemetry.AppendMissionUsageRecord(filepath.Join(root, "memory", "mission-usage.jsonl"), telemetry.MissionUsageRecord{
			SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: r.id, MissionID: "m1", Phase: r.phase,
			Role: "ranger", InvocationID: "inv_" + r.id, Attempt: 1, UsageScope: "invocation", RecordKind: "incremental",
			TokensInput: tokens(r.in), TokensOutput: tokens(r.out), MeasurementSource: "agent_report",
			MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:00:00Z",
		})
		require.NoError(t, err)
	}
}

func TestCriteriaPrintsTheStageTokenBlockAfterTheRows(t *testing.T) {
	root := informationalMission(t)
	seedStageUsage(t, root)
	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	tail := lines[len(lines)-3:]
	require.Equal(t, "tokens estimados:", tail[0])
	require.Contains(t, tail[1], "discovery")
	require.Contains(t, tail[1], "(111) -> refining")
	require.Contains(t, tail[1], "(222) -> delivering")
	require.Contains(t, tail[1], "(—)")
	require.Contains(t, tail[2], "total")
	require.Contains(t, tail[2], "(333)")
	require.Contains(t, out, "token_usage.by_stage: discovery 111; refining 222; delivering —; total 333")
}

func TestCriteriaStageTokenBlockSaysNoUsageWhenNothingWasReported(t *testing.T) {
	root := informationalMission(t)
	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Contains(t, out, "tokens estimados:")
	require.Contains(t, out, "nenhum uso reportado")
	require.NotContains(t, out, "(0)", "an unobserved volume is never zero")
}

func TestCriteriaJSONCarriesTheStageTokens(t *testing.T) {
	root := informationalMission(t)
	seedStageUsage(t, root)
	out, err := runCriteria(t, handoffCriteriaOptions{Root: root, MissionID: "m1", JSON: true})
	require.NoError(t, err)
	var parsed struct {
		TokenStages struct {
			Availability string `json:"availability"`
			Stages       []struct {
				Stage   string `json:"stage"`
				Total   *int64 `json:"total"`
				Pending bool   `json:"pending"`
			} `json:"stages"`
		} `json:"token_stages"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Equal(t, "available", parsed.TokenStages.Availability)
	require.Len(t, parsed.TokenStages.Stages, 3)
	require.Equal(t, int64(111), *parsed.TokenStages.Stages[0].Total)
	require.Nil(t, parsed.TokenStages.Stages[2].Total)
	require.True(t, parsed.TokenStages.Stages[2].Pending)
}
