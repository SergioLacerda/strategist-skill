package view_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/mission/gatecriteria"
	missionview "github.com/SergioLacerda/strategist-skill/internal/mission/view"
	"github.com/stretchr/testify/require"
)

func TestViewWithoutACriteriaReportKeepsItsExistingJSONShape(t *testing.T) {
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}})
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "criteria", "additive: a view without the report is byte-compatible")
}

func TestViewCarriesTheCriteriaRowsAndRendersThemFilled(t *testing.T) {
	report := gatecriteria.Report{MissionID: "m-1", Rows: []gatecriteria.Row{
		{Criterion: "route_confidence", State: gatecriteria.StateEvaluated, Value: "85%", Source: "memory/route-decisions.jsonl", Detail: "rota full_pipeline"},
		{Criterion: "provider.archivist_to_gate", State: gatecriteria.StateNotEvaluated, Value: gatecriteria.StateNotEvaluated, Source: "ledger", Detail: "integração não habilitada"},
	}}
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}, Criteria: &report})

	require.NotNil(t, v.Criteria)
	require.Len(t, v.Criteria.Rows, 2)
	var out bytes.Buffer
	require.NoError(t, missionview.RenderHuman(&out, v))
	require.Contains(t, out.String(), "Quality criteria")
	require.Contains(t, out.String(), "route_confidence: 85% [avaliado]")
	require.Contains(t, out.String(), "provider.archivist_to_gate: não avaliado")
}
