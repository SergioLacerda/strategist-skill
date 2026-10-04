package mission_test

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestReadCompletionAcceptsExactlyOneObject(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(`{"request_id":"inv_12345678","result":"ok"}`))

	got, err := mission.ReadCompletion(cmd)

	require.NoError(t, err)
	require.Equal(t, domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "ok"}, got)
}

func TestReadCompletionRejectsTrailingObjectsAndData(t *testing.T) {
	for _, raw := range []string{
		`{"request_id":"inv_12345678","result":"ok"}{"request_id":"inv_87654321","result":"extra"}`,
		`{"request_id":"inv_12345678","result":"ok"} trailing`,
		`{"request_id":"inv_12345678","request_id":"inv_87654321","result":"ok"}`,
	} {
		cmd := &cobra.Command{}
		cmd.SetIn(strings.NewReader(raw))
		_, err := mission.ReadCompletion(cmd)
		require.Error(t, err, raw)
	}
}
