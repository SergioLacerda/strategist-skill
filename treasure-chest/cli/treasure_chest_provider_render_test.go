package treasurecli

import (
	"bytes"
	"io"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func providerRenderProjection() jeweler.Projection {
	return jeweler.Projection{
		Identity: jeweler.ProviderIdentity{
			Package: "treasure-chest-tmp", Version: "0.1.0", API: "knowledge-api/v1",
			Digest: "sha256:test", Provenance: "test-only",
		},
		CapturedAt:  "2026-10-11T12:00:00Z",
		Freshness:   jeweler.FreshnessFresh,
		Readiness:   jeweler.Readiness{Status: "ready", Reason: "activation_evidence_verified"},
		Items:       []jeweler.Item{{Kind: "potion", ID: "fix-timeout", Applicability: "accepted", Reason: "matched"}},
		Limitations: []string{"result_limit_applied"},
	}
}

func TestRenderTreasureChestProviderQuerySupportsTableJSONAndRejectsUnknownFormat(t *testing.T) {
	projection := providerRenderProjection()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, renderTreasureChestProviderQuery(cmd, outputFormatTable, projection))
	require.Contains(t, out.String(), "provider:    treasure-chest-tmp@0.1.0")
	require.Contains(t, out.String(), "limitation:  result_limit_applied")

	out.Reset()
	require.NoError(t, renderTreasureChestProviderQuery(cmd, outputFormatJSON, projection))
	require.Contains(t, out.String(), `"contract_version"`)

	err := renderTreasureChestProviderQuery(cmd, "yaml", projection)
	require.EqualError(t, err, `treasure-chest provider query: unknown --format "yaml" (want table or json)`)
}

func TestRenderTreasureChestProviderMaintainSupportsTableJSONAndEmptyDetail(t *testing.T) {
	result := jeweler.MaintenanceResult{
		Operation: "index",
		Readiness: jeweler.Readiness{Status: "ready", Reason: "maintenance_command_completed"},
		Detail:    "indexed 2 entries",
	}
	var out bytes.Buffer
	require.NoError(t, renderMaintenanceResult(&out, outputFormatTable, result))
	require.Equal(t, "operation: index\nreadiness: ready (maintenance_command_completed)\ndetail:    indexed 2 entries\n", out.String())

	out.Reset()
	result.Detail = ""
	require.NoError(t, renderMaintenanceResult(&out, outputFormatTable, result))
	require.NotContains(t, out.String(), "detail:")

	out.Reset()
	require.NoError(t, renderMaintenanceResult(&out, outputFormatJSON, result))
	require.Contains(t, out.String(), `"operation":"index"`)

	err := renderMaintenanceResult(&out, "yaml", result)
	require.EqualError(t, err, `treasure-chest provider maintain: unknown --format "yaml" (want table or json)`)
}

func TestProviderQueryRenderReportsWriterFailure(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(providerFailWriter{})
	err := renderTreasureChestProviderQuery(cmd, outputFormatTable, providerRenderProjection())
	require.Error(t, err)
	require.Contains(t, err.Error(), "render provider identity")
}

type providerFailWriter struct{}

func (providerFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
