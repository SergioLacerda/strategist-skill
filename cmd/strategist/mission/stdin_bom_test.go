package mission_test

import (
	"strings"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const utf8BOM = "\xEF\xBB\xBF"

// A route decision written by Windows PowerShell 5.1 carries a UTF-8 byte
// order mark; it must be recorded exactly like the same decision without it.
func TestRoute_AcceptsByteOrderMarkPrefixedDecision(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	decision := `{"mission_id":"m-bom","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.9,"evidence_state":"requires_discovery","discovery_subtype":"diagnostic","fallback_route":"full_pipeline"}`

	cmd := mission.NewRoute(lifecycleDeps(t))
	cmd.SetIn(strings.NewReader(utf8BOM + decision))
	cmd.SetArgs([]string{"--root", root, "--mission-id", "m-bom"})

	require.NoError(t, cmd.Execute())
}

func TestReadCompletion_AcceptsByteOrderMarkPrefixedObject(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(utf8BOM + `{"request_id":"inv_1","result":"ok"}`))

	completion, err := mission.ReadCompletion(cmd)

	require.NoError(t, err)
	assert.Equal(t, "inv_1", completion.RequestID)
}
