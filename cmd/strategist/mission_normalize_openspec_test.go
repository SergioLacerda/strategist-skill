package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath(t *testing.T) {
	projectRoot := t.TempDir()
	fallback := filepath.Join(projectRoot, ".strategist", "openspec")

	assert.Equal(t, fallback, resolvePath("", fallback, projectRoot))
	absolute := filepath.Join(t.TempDir(), "custom")
	assert.Equal(t, absolute, resolvePath(absolute, fallback, projectRoot))
	assert.Equal(t, filepath.Join(projectRoot, "custom", "path"), resolvePath("custom/path", fallback, projectRoot))
}

func TestRecordNormalizeConfidencePersistsArchivistClaim(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\n"), 0o600))
	percent := 100
	evidence := []domain.Evidence{{
		ID: "E-1", SourceRef: "openspec/changes/c-1/tasks.md",
		Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh,
		ConfidencePercent: &percent,
	}}
	claim := domain.ConfidenceClaim{
		ID: "A-1", Statement: "ready", Agent: telemetry.ConfidenceAgentArchivist,
		CorrelationKey: "archivist-openspec-normalization", ClaimKind: domain.ClaimKindAssertion,
		ConfidencePercent: percent, ConfidenceLevel: domain.ConfidenceHigh,
		EvidenceIDs: []string{"E-1"}, EvidenceClasses: []string{domain.EvidenceClassExplicit},
		CalibrationStatus: domain.CalibrationNoSample,
	}

	require.NoError(t, recordNormalizeConfidence(missionadapter.NormalizeOptions{Root: root, MissionID: "m-1"}, claim, evidence))
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, telemetry.ConfidenceAgentArchivist, records[0].Agent)
	assert.Equal(t, telemetry.ConfidenceCoverageReported, records[0].CoverageStatus)
}

func TestResolveNormalizePathsReportsMissingActiveConfig(t *testing.T) {
	_, _, _, err := resolveNormalizePaths(missionadapter.NormalizeOptions{Root: t.TempDir(), MissionID: "mission-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve active base path")
}

func TestNormalizePublicationAndMissionLoadUseTheActiveRuntime(t *testing.T) {
	_, root := workspaceWithRoot(t)
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	require.NoError(t, saveMission(root, status))

	loaded, err := loadNormalizeMission(missionadapter.NormalizeOptions{Root: root, MissionID: status.MissionID})
	require.NoError(t, err)
	assert.Equal(t, status, loaded)

	publishedAt := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, recordNormalizePublication(missionadapter.NormalizeOptions{Root: root}, refinement.PackagePublication{
		MissionID: "m-1", ProviderChangeID: "c-1", SourceDigest: "sha256:source", PackageDigest: "sha256:package", PublishedAt: publishedAt,
	}))
	records, err := telemetry.ReadRefinedPackagePublications(telemetry.RefinedPackagePublicationHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "c-1", records[0].ProviderChangeID)
	assert.Equal(t, publishedAt, records[0].PublishedAt)
}
