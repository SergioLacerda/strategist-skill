package mission_test

import (
	"bytes"
	"testing"

	"github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteResultPreservesHumanAndJSONEnvelopes(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	status := domain.MissionEngineStatus{MissionID: "output-mission", Phase: domain.PhaseBootstrap, State: domain.StateInit}

	require.NoError(t, mission.WriteResult(cmd, false, status))
	assert.Equal(t, "mission_id=output-mission phase=BOOTSTRAP state=INIT\n", out.String())

	out.Reset()
	require.NoError(t, mission.WriteResult(cmd, true, status))
	assert.Contains(t, out.String(), `"mission_id":"output-mission"`)

	out.Reset()
	require.NoError(t, mission.WriteResult(cmd, false, map[string]string{"status": "ok"}))
	assert.Contains(t, out.String(), `"status":"ok"`)
}

func TestWriteResultReportsWriterFailures(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{})
	status := domain.MissionEngineStatus{MissionID: "writer-error", Phase: "discovery", State: "active"}

	err := mission.WriteResult(cmd, false, status)
	require.ErrorContains(t, err, "write mission result")
	err = mission.WriteResult(cmd, true, status)
	require.ErrorContains(t, err, "encode mission result")
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, assert.AnError }
