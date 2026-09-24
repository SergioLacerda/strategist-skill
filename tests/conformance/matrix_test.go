package conformance_test

import (
	"context"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/conformance"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	pluginconformance "github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/client-matrix.yaml
var matrixFixture embed.FS

func TestClientMatrixIsValidAndReportsStructuralEvidenceDeterministically(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)

	available := map[string]bool{
		"codex": true, "claude": true, "gemini-antigravity": true, "governance-bridge": true,
	}
	one, err := matrix.Evaluate(available)
	require.NoError(t, err)
	two, err := matrix.Evaluate(available)
	require.NoError(t, err)

	assert.Equal(t, one, two)
	assert.Equal(t, 4, one.ClientCount)
	assert.Equal(t, 5, one.RowCount)
	assert.Contains(t, one.MatrixDigest, "sha256:")
	for _, result := range one.Results {
		if result.RowID == "codex-live-probe" {
			assert.False(t, result.Passed)
			assert.Equal(t, "live_probe_required", result.Reason)
		}
	}
	live := findResult(one.Results, "codex-live-probe")
	assert.False(t, live.Passed)
	assert.Equal(t, "live_probe_required", live.Reason)

	report, err := conformance.MarshalReport(one)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(report, &decoded))
	assert.InDelta(t, float64(5), decoded["row_count"], 0)
}

func findResult(results []conformance.RowResult, rowID string) conformance.RowResult {
	for _, result := range results {
		if result.RowID == rowID {
			return result
		}
	}
	return conformance.RowResult{}
}

func TestClientMatrixRejectsUnclassifiedClientAndUnavailableSurface(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)
	matrix.Rows[0].Client = "unknown-client"
	_, err = matrix.Evaluate(map[string]bool{"codex": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unclassified client")

	matrix.Rows[0].Client = "codex"
	_, err = matrix.Evaluate(map[string]bool{"codex": false, "claude": true, "gemini-antigravity": true, "governance-bridge": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestClientMatrixComparesIdentityBearingFieldsExactly(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)
	expected := matrix.Rows[0]
	actual := expected
	require.NoError(t, conformance.CompareIdentity(expected, actual))
	actual.AuthorityOwner = "plugins.lock"
	require.Error(t, conformance.CompareIdentity(expected, actual))
}

func TestClientAdapterSmokeUsesHermeticRootsAndContextAuthority(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	require.NoError(t, os.WriteFile(filepath.Join(project, "b.md"), []byte("b"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "a.md"), []byte("a"), 0o600))

	first, err := domain.MaterializeContext(project, []domain.ContextReference{
		{Ref: "b.md", Kind: "context"}, {Ref: "a.md", Kind: "context"},
	}, 4, 20)
	require.NoError(t, err)
	second, err := domain.MaterializeContext(project, []domain.ContextReference{
		{Ref: "a.md", Kind: "context"}, {Ref: "b.md", Kind: "context"},
	}, 4, 20)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, "workspace:a.md", first.References[0].Provenance)
	assert.NotContains(t, first.References[0].Provenance, home)
}

func TestClientMatrixRejectsMalformedFixture(t *testing.T) {
	_, err := conformance.DecodeMatrix([]byte("schema_version: x\nclients: []\nrows: []\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one row")
}

func TestClientMatrixKeepsStaticAndUnsupportedEvidenceFailClosed(t *testing.T) {
	unsupported := connectors.UnsupportedConnector{IDValue: "external", ConnectorAPIVersion: "v1"}
	probe := unsupported.Probe(context.TODO(), domain.InstalledInstance{ID: "provider"}, "invoke")
	assert.Equal(t, domain.ReadinessUnsupported, probe.Status)
	assert.Equal(t, pluginconformance.EvidenceUnsupported, pluginconformance.StateForReadiness(probe.Status))

	native := connectors.NativeRuntimeConnector{ConnectorID: "native", ConnectorAPIVersion: "v1"}
	staticProbe := native.Probe(context.TODO(), domain.InstalledInstance{ID: "provider"}, "invoke")
	assert.Equal(t, domain.ReadinessUnknown, staticProbe.Status)
	assert.Equal(t, pluginconformance.EvidenceUnknown, pluginconformance.StateForReadiness(staticProbe.Status))
}

type conformanceDiscoveryConnector struct {
	connectors.UnsupportedConnector
	result connectors.ConnectorResult
}

func (c conformanceDiscoveryConnector) Capabilities(context.Context) connectors.RuntimeCapabilities {
	return connectors.RuntimeCapabilities{ConnectorID: c.IDValue, CanInvoke: true}
}

func (c conformanceDiscoveryConnector) Invoke(context.Context, connectors.InvocationEnvelope) connectors.ConnectorResult {
	return c.result
}

func TestDiscoveryConformanceRequiresHostInvocationEvidence(t *testing.T) {
	request := provider.DiscoveryWeaponRequest{
		MissionID:    "conformance-mission",
		Role:         "ranger",
		Slot:         string(domain.SlotDiscovery),
		ProviderID:   "brainstorming",
		ArtifactPath: ".analysis/pending/conformance-mission.md",
	}
	connector := conformanceDiscoveryConnector{
		UnsupportedConnector: connectors.UnsupportedConnector{IDValue: "host-loader"},
		result: connectors.ConnectorResult{
			Status:             domain.ReadinessReady,
			ProviderID:         "brainstorming",
			InvocationEvidence: "host-conformance-run",
			Artifact:           []byte("# Findings\n\nUntrusted result."),
		},
	}
	artifact, err := provider.InvokeDiscoveryViaConnector(context.Background(), request, domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-1")
	require.NoError(t, err)
	assert.Equal(t, "host-conformance-run", artifact.InvocationEvidence)
	assert.Contains(t, string(artifact.Content), "mission_status: ranger_pending")

	_, err = provider.InvokeDiscoveryViaConnector(context.Background(), request, domain.InstalledInstance{ID: "brainstorming"}, connectors.NativeRuntimeConnector{ConnectorID: "native", ConnectorAPIVersion: "v1"}, nil, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
}

func TestLiveProbeRequiresCertifiedEvidence(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)
	var live conformance.Row
	for _, row := range matrix.Rows {
		if row.EvidenceTier == conformance.EvidenceLive {
			live = row
			break
		}
	}
	require.NotEmpty(t, live.ID)
	for _, state := range []conformance.EvidenceState{conformance.StateUnknown, conformance.StateUnsupported, conformance.StateFailed, conformance.StateBlocked} {
		result, err := conformance.EvaluateLiveProbe(live, state)
		require.NoError(t, err)
		assert.False(t, result.Passed)
	}
	result, err := conformance.EvaluateLiveProbe(live, conformance.StateCertified)
	require.NoError(t, err)
	assert.True(t, result.Passed)
}

func TestClientMatrixConsolidatesExplicitLiveProbeEvidenceFailClosed(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)
	available := map[string]bool{
		"codex": true, "claude": true, "gemini-antigravity": true, "governance-bridge": true,
	}

	tests := []struct {
		name     string
		observed map[string]conformance.EvidenceState
		passed   bool
		reason   string
	}{
		{name: "missing", observed: nil, passed: false, reason: "live_probe_required"},
		{name: "certified", observed: map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateCertified}, passed: true, reason: "live_probe_verified"},
		{name: "unknown", observed: map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateUnknown}, passed: false, reason: "live_probe_not_ready"},
		{name: "unsupported", observed: map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateUnsupported}, passed: false, reason: "live_probe_not_ready"},
		{name: "failed", observed: map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateFailed}, passed: false, reason: "live_probe_not_ready"},
		{name: "blocked", observed: map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateBlocked}, passed: false, reason: "live_probe_not_ready"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := matrix.EvaluateWithLiveProbes(available, test.observed)
			require.NoError(t, err)
			result := findResult(report.Results, "codex-live-probe")
			assert.Equal(t, test.passed, result.Passed)
			assert.Equal(t, test.reason, result.Reason)
		})
	}
}

func TestClientMatrixRejectsUnknownLiveProbeRow(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)

	_, err = matrix.EvaluateWithLiveProbes(map[string]bool{
		"codex": true, "claude": true, "gemini-antigravity": true, "governance-bridge": true,
	}, map[string]conformance.EvidenceState{"missing-live-row": conformance.StateCertified})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown row "missing-live-row"`)
}

func TestClientMatrixLiveProbeReportIsDeterministic(t *testing.T) {
	data, err := matrixFixture.ReadFile("testdata/client-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)
	available := map[string]bool{
		"codex": true, "claude": true, "gemini-antigravity": true, "governance-bridge": true,
	}
	probes := map[string]conformance.EvidenceState{"codex-live-probe": conformance.StateCertified}

	one, err := matrix.EvaluateWithLiveProbes(available, probes)
	require.NoError(t, err)
	two, err := matrix.EvaluateWithLiveProbes(available, probes)
	require.NoError(t, err)
	assert.Equal(t, one, two)
}

func TestInvocationEnvelopeSerializationPreservesIdentity(t *testing.T) {
	envelope, err := domain.ComposeInvocationEnvelope(domain.ComposeInvocationRequest{
		MissionID:       "matrix-mission",
		Phase:           domain.PhaseDiscovery,
		OutputSchemaRef: "ranger-to-archivist/v1",
		Plan:            domain.RoleInvocationPlan{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming"},
		Components: []domain.InvocationComponent{{
			Ref: "roles/ranger.yaml", Kind: "role", Phase: domain.PhaseDiscovery,
			Digest: "sha256:role", Selected: true,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, domain.InvocationEnvelopeSchemaVersion, envelope.SchemaVersion)
	assert.Equal(t, "ranger", envelope.Role)
	assert.Equal(t, "discovery", envelope.Slot)
	assert.Equal(t, "brainstorming", envelope.Provider)
	assert.Equal(t, []string{"roles/ranger.yaml"}, envelope.RequiredContextRefs)
}

func TestRankedPlanRequiresCertifiedRoleAffinityAndProviderIdentity(t *testing.T) {
	binding := domain.SlotBinding{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: domain.SlotBindingModeRanked}
	stamp := domain.CatalogRankedStamp{
		ID: "brainstorming", Ranked: true, CertificationDigest: "sha256:certified",
		Roles: []string{"ranger"}, ConformanceLevel: "C1",
	}
	plan, err := domain.NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotBindingModeRanked, plan.Mode)

	stamp.ID = "other"
	_, err = domain.NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp)
	require.Error(t, err)
	stamp.ID = binding.InstalledInstanceID
	stamp.Roles = []string{"archivist"}
	_, err = domain.NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp)
	require.Error(t, err)
}

func TestContextMaterializationRejectsEscapingAndAbsoluteReferences(t *testing.T) {
	root := t.TempDir()
	cases := []string{"../outside.md", filepath.Join(root, "absolute.md")}
	for _, ref := range cases {
		_, err := domain.MaterializeContext(root, []domain.ContextReference{{Ref: ref, Kind: "context"}}, 4, 20)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid relative reference")
	}
}
