package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// executionFixture builds a custom-Sniper candidate: a Weapon that asks to be
// bound to the execution slot, with the given adapter roles line, requested
// permissions and legacy skill.yaml body.
func executionFixture(t *testing.T, rolesLine, permissions, legacy string) string {
	t.Helper()
	dir := copyFixture(t)
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [execution]\n" + rolesLine + "entrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: " + permissions + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, adapterManifestName), []byte(adapter), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte(legacy), 0o644))
	return dir
}

const (
	sniperRoles  = "supported_roles: [sniper]\n"
	sniperLegacy = "id: fixture-provider\ncanonical_role: sniper\nsupported_slots: [execution]\nrisk_score: controlled\n"
)

func TestValidateAcceptsDocumentationOnlyCustomSniper(t *testing.T) {
	dir := executionFixture(t, sniperRoles, "[workspace.read, analysis.write, docs.write]", sniperLegacy)

	report, err := Validate(dir, "execution")

	require.NoError(t, err)
	require.True(t, report.Validated, "reasons: %v", reasonCodes(report.Reasons))
}

func TestValidateRejectsCustomSniperWithoutSniperRoleAffinity(t *testing.T) {
	dir := executionFixture(t, "", "[workspace.read]", "id: fixture-provider\nsupported_slots: [execution]\n")

	report, err := Validate(dir, "execution")

	require.Error(t, err)
	require.Contains(t, reasonCodes(report.Reasons), "execution_role_affinity_missing")
}

func TestValidateRejectsCustomSniperRequestingBeyondDocumentation(t *testing.T) {
	for _, permission := range []string{"source.write", "network.access", "subprocess.exec", "secret.access", "external_app.access"} {
		t.Run(permission, func(t *testing.T) {
			dir := executionFixture(t, sniperRoles, "[workspace.read, "+permission+"]", sniperLegacy)

			report, err := Validate(dir, "execution")

			require.Error(t, err)
			require.Contains(t, reasonCodes(report.Reasons), "execution_permission_exceeds_documentation")
			require.Contains(t, report.Reasons[len(report.Reasons)-1].Detail, permission)
		})
	}
}

func TestValidateRejectsCustomSniperDeclaringRiskBelowControlled(t *testing.T) {
	legacy := "id: fixture-provider\ncanonical_role: sniper\nsupported_slots: [execution]\nrisk_score: write_analysis\n"
	dir := executionFixture(t, sniperRoles, "[workspace.read]", legacy)

	report, err := Validate(dir, "execution")

	require.Error(t, err)
	require.Contains(t, reasonCodes(report.Reasons), "execution_risk_below_controlled")
}

func TestExecutionRulesDoNotChangeOtherSlots(t *testing.T) {
	dir := copyFixture(t)
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [refinement]\nsupported_roles: [archivist]\nentrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: [workspace.read, source.write]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, adapterManifestName), []byte(adapter), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte("id: fixture-provider\ncanonical_role: archivist\nsupported_slots: [refinement]\nrisk_score: write_analysis\n"), 0o644))

	report, err := Validate(dir, "refinement")

	require.NoError(t, err)
	require.True(t, report.Validated, "reasons: %v", reasonCodes(report.Reasons))
}

func TestAddRecordsCustomSniperBindingAsCustomMode(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, copySource(filepath.Join("..", "embed", "defaults"), root))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: ranger\n  refinement: archivist\n  execution: sniper\n"), 0o644))
	dir := executionFixture(t, sniperRoles, "[workspace.read, docs.write]", sniperLegacy)

	result, err := Add(root, dir, "execution")

	require.NoError(t, err)
	require.Equal(t, "complete", result.TransactionState)
	lockRaw, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, readErr)
	require.Contains(t, string(lockRaw), "mode: custom", "a custom Sniper is never recorded as the ranked default")
}

// risk_score and scratch_root live in adapter.yaml, the Strategist-owned
// compatibility layer; the compat skill.yaml only backs them up.
func adapterWith(t *testing.T, extra string) string {
	t.Helper()
	dir := copyFixture(t)
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [execution]\nsupported_roles: [sniper]\nentrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: [workspace.read]\n" + extra
	require.NoError(t, os.WriteFile(filepath.Join(dir, adapterManifestName), []byte(adapter), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte("id: fixture-provider\ncanonical_role: sniper\nsupported_slots: [execution]\n"), 0o644))
	return dir
}

func TestAdapterDeclaresRiskScoreAndScratchRoot(t *testing.T) {
	dir := adapterWith(t, "risk_score: controlled\nscratch_root: runtime\n")

	report, err := Validate(dir, "execution")

	require.NoError(t, err)
	require.True(t, report.Validated, "reasons: %v", reasonCodes(report.Reasons))
}

func TestExecutionRiskComesFromTheAdapterAndBeatsTheCompatView(t *testing.T) {
	dir := adapterWith(t, "risk_score: write_analysis\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte(sniperLegacy), 0o644)) // compat view says controlled

	report, err := Validate(dir, "execution")

	require.Error(t, err)
	require.Contains(t, reasonCodes(report.Reasons), "execution_risk_below_controlled", "the adapter is the authority")
}

func TestAdapterRejectsAnUnknownRiskScoreOrScratchRoot(t *testing.T) {
	for _, extra := range []string{"risk_score: reckless\n", "scratch_root: elsewhere\n"} {
		report, err := Validate(adapterWith(t, extra), "execution")

		require.Error(t, err, extra)
		require.Contains(t, reasonCodes(report.Reasons), "adapter_contract_invalid", extra)
	}
}
