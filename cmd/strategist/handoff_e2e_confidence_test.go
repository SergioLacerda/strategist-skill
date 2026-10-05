package main

import (
	"bytes"
	"log/slog"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/gatecriteria"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/require"
)

// One provider reading must read the same everywhere it is shown: the criteria
// report, the confidence ledger claim, the handoff event and the mission view.
func TestTheProviderConfidenceAgreesAcrossReportLedgerEventAndView(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, err := runPrecheck(t, root)
	require.NoError(t, err)
	_, err = runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Equal(t, 1, sim.Calls(), "one reading serves both boundaries")

	// 1. the criteria report
	report := gatecriteria.Build(gatecriteria.Input{Root: root, BasePath: root + "/../.analysis", MissionID: "m1"})
	gate, sniper := rowOf(t, report, "provider.archivist_to_gate"), rowOf(t, report, "provider.archivist_to_sniper")
	require.Contains(t, gate.Value, "96%")
	require.Contains(t, sniper.Value, "96%", "the delegation recorded on the outcome carries the same number")

	// 2. the confidence ledger
	claims := providerClaims(t, root)
	require.Len(t, claims, 1)
	require.Equal(t, 96, claims[0].ConfidencePercent)

	// 3. the handoff event
	require.Contains(t, logs.String(), "strategist.handoff.criterion_source=provider")
	require.Contains(t, logs.String(), "strategist.handoff.provider_confidence_percent=96")

	// 4. the mission view
	_, status, err := loadMission(root, "m1")
	require.NoError(t, err)
	view, err := missionadapter.LoadView(root, status, "", "role-levels.jsonl", func(r []leveling.Record, _ string) []leveling.Record { return r })
	require.NoError(t, err)
	require.NotNil(t, view.Criteria)
	var viewValues []string
	for _, row := range view.Criteria.Rows {
		if row.Criterion == "provider.archivist_to_gate" || row.Criterion == "provider.archivist_to_sniper" {
			viewValues = append(viewValues, row.Value)
		}
	}
	require.Len(t, viewValues, 2)
	for _, value := range viewValues {
		require.Contains(t, value, "96%")
	}
}

func TestADisabledIntegrationShowsNotEvaluatedEverywhere(t *testing.T) {
	root := informationalMission(t)
	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)

	report := gatecriteria.Build(gatecriteria.Input{Root: root, BasePath: root + "/../.analysis", MissionID: "m1"})
	for _, criterion := range []string{"provider.ranger_to_archivist", "provider.archivist_to_gate", "provider.archivist_to_sniper"} {
		row := rowOf(t, report, criterion)
		require.Equal(t, gatecriteria.StateNotEvaluated, row.State)
		require.Equal(t, gatecriteria.StateNotEvaluated, row.Value)
	}
	require.Empty(t, providerClaims(t, root), "a disabled integration is not a silent gap")
}

func rowOf(t *testing.T, report gatecriteria.Report, criterion string) gatecriteria.Row {
	t.Helper()
	for _, row := range report.Rows {
		if row.Criterion == criterion {
			return row
		}
	}
	t.Fatalf("no row %q", criterion)
	return gatecriteria.Row{}
}
