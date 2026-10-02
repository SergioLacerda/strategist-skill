package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateSubmitArtifactsRequiresNormalizedDiscoveryArtifact(t *testing.T) {
	base := t.TempDir()

	err := validateSubmitArtifacts(base, "m-discovery", domain.MissionEventDiscoveryDone)

	require.ErrorContains(t, err, "discovery artifact validation failed")
	require.ErrorContains(t, err, "no such file or directory")
}

func TestValidateSubmitArtifactsRejectsMalformedDiscoveryArtifact(t *testing.T) {
	base := t.TempDir()
	pending := filepath.Join(base, "pending")
	require.NoError(t, os.MkdirAll(pending, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(pending, "m-discovery-analysis.md"), []byte("---\nmission_id: m-discovery\nmission_status: ranger_pending\nsources_consulted: []\n---\n\n## mission_objective\n"), 0o600))

	err := validateSubmitArtifacts(base, "m-discovery", domain.MissionEventDiscoveryDone)

	require.ErrorContains(t, err, "discovery artifact validation failed")
	require.ErrorContains(t, err, "missing section")
}

func TestValidateSubmitArtifactsRequiresCompleteRefinedPackage(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "refined", "m-refined"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(base, "refined", "m-refined", "analysis.md"), []byte("---\nmission_id: m-refined\nmission_status: archivist_done\n---\n\nanalysis\n"), 0o600))

	err := validateSubmitArtifacts(base, "m-refined", domain.MissionEventRefinementDone)

	require.ErrorContains(t, err, "refined package validation failed")
	require.ErrorContains(t, err, "proposal.md")
}

func TestValidateSubmitArtifactsAcceptsCompleteRefinedPackageForBothGateKinds(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "refined", "m-gate")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte("---\nmission_id: m-gate\nmission_status: archivist_done\n---\n\nanalysis\n"), 0o600))
	for _, name := range []string{"proposal.md", "design.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] 1.1 [task_type: implementation_handoff] change code\n"), 0o600))

	for _, event := range []domain.MissionEngineEvent{domain.MissionEventGateApproved, domain.MissionEventGateApprovedAnalysisOnly} {
		require.NoError(t, validateSubmitArtifacts(base, "m-gate", event), event)
	}
}
