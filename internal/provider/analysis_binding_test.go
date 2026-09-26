package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

// analysisFixture is a candidate for the discovery or refinement slot whose
// adapter declares the given risk_score line ("" declares none).
func analysisFixture(t *testing.T, slot, role, riskLine string) string {
	t.Helper()
	dir := copyFixture(t)
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [" + slot + "]\nsupported_roles: [" + role + "]\nentrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: [workspace.read]\n" + riskLine
	require.NoError(t, os.WriteFile(filepath.Join(dir, adapterManifestName), []byte(adapter), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte("id: fixture-provider\ncanonical_role: "+role+"\nsupported_slots: ["+slot+"]\nrisk_score: write_analysis\n"), 0o644))
	return dir
}

// Whatever provider add accepts for an analysis slot is accepted by check for its
// risk: the adapter must declare exactly the slot's risk_score contract.
func TestValidateAppliesTheSlotRiskContractAtBindTime(t *testing.T) {
	for _, tc := range []struct {
		slot, role string
	}{{"discovery", "ranger"}, {"refinement", "archivist"}} {
		required := domain.SlotRiskContract[tc.slot]
		t.Run(tc.slot+"/declared", func(t *testing.T) {
			report, err := Validate(analysisFixture(t, tc.slot, tc.role, "risk_score: "+required+"\n"), tc.slot)
			require.NoError(t, err)
			require.True(t, report.Validated, "reasons: %v", reasonCodes(report.Reasons))
		})
		t.Run(tc.slot+"/missing", func(t *testing.T) {
			report, err := Validate(analysisFixture(t, tc.slot, tc.role, ""), tc.slot)
			require.Error(t, err)
			require.Contains(t, reasonCodes(report.Reasons), "analysis_risk_missing")
			require.Contains(t, report.Reasons[len(report.Reasons)-1].Detail, required)
			require.ErrorContains(t, err, required, "the error names the required value")
		})
		t.Run(tc.slot+"/mismatch", func(t *testing.T) {
			report, err := Validate(analysisFixture(t, tc.slot, tc.role, "risk_score: controlled\n"), tc.slot)
			require.Error(t, err)
			require.Contains(t, reasonCodes(report.Reasons), "analysis_risk_mismatch")
		})
	}
}

// The compat view is not the authority check reads for a custom package, so a
// risk that only skill.yaml declares does not satisfy the bind-time rule.
func TestValidateIgnoresARiskOnlyTheCompatViewDeclares(t *testing.T) {
	dir := analysisFixture(t, "refinement", "archivist", "")
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte("id: fixture-provider\ncanonical_role: archivist\nsupported_slots: [refinement]\nrisk_score: write_analysis\n"), 0o644))

	report, err := Validate(dir, "refinement")

	require.Error(t, err)
	require.Contains(t, reasonCodes(report.Reasons), "analysis_risk_missing")
}

func TestValidateWithoutARequestedSlotSkipsTheAnalysisRiskRule(t *testing.T) {
	report, err := Validate(fixturePath(t), "")

	require.NoError(t, err)
	require.NotContains(t, reasonCodes(report.Reasons), "analysis_risk_missing")
}

func TestValidateLeavesTheExecutionRiskRuleUnchanged(t *testing.T) {
	dir := executionFixture(t, sniperRoles, "[workspace.read]", sniperLegacy)

	report, err := Validate(dir, "execution")

	require.NoError(t, err)
	require.True(t, report.Validated)
	require.NotContains(t, reasonCodes(report.Reasons), "analysis_risk_missing")
}
