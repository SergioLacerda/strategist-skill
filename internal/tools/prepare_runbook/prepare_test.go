package preparerunbook

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareBuildsReviewableCandidateWithoutWritingCanonicalRunbook(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-1")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writeArtifacts(t, refined, `
---
mission_id: m-1
mission_objective: recover the provider path
---
## procedure
Run go test ./internal/tools/... and record the expected output.
Stop when the expected result is reproduced. Source: provider log; scope: internal/tools.
`)
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	runbooks := filepath.Join(root, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(runbooks, 0o755))
	result, err := Prepare(context.Background(), Input{MissionID: "m-1", CandidateID: "OA-RUNBOOK-m-1", PackageDigest: digest, RefinedPath: refined, RunbookRoot: runbooks})
	require.NoError(t, err)
	assert.Equal(t, StatusPrepared, result.Status)
	assert.Equal(t, ComparisonNovel, result.Comparison.Classification)
	assert.Equal(t, digest, result.SourceDigest)
	assert.NotEmpty(t, result.ProposedRunbook.RunbookID)
	assert.Len(t, result.Provenance, 4)
	_, err = os.Stat(filepath.Join(runbooks, result.ProposedRunbook.RunbookID+".runbook.yaml"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPrepareClassifiesDuplicateAndDoesNotTrustProviderApproval(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-2")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writeArtifacts(t, refined, `
---
mission_id: m-2
mission_objective: recover the provider path
applies_when: [provider recovery]
---
## procedure
Run go test ./internal/tools/...; expected output is green. Stop when verified.
Source: incident report. Scope: internal/tools.
`)
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	runbooks := filepath.Join(root, "runbooks")
	require.NoError(t, os.MkdirAll(runbooks, 0o755))
	// The sidecar is a canonical comparison input, not a destination.
	require.NoError(t, os.WriteFile(filepath.Join(runbooks, "existing.runbook.yaml"), []byte("schema_version: '2'\nrunbook_id: existing\nrunbook_type: operational\nsource_doc: docs/existing.md\napplies_when:\n  - provider recovery\nobjective: recover the provider path\n"), 0o644))
	result, err := Prepare(context.Background(), Input{MissionID: "m-2", CandidateID: "OA-RUNBOOK-m-2", PackageDigest: digest, RefinedPath: refined, RunbookRoot: runbooks, Evaluate: func(context.Context, ProviderRequest) (ProviderReading, error) {
		return ProviderReading{Path: ProviderPathJEV, Approved: true, Confidence: 0.99, Threshold: 0.80}, nil
	}})
	require.NoError(t, err)
	assert.Equal(t, StatusDuplicate, result.Status)
	assert.Equal(t, ComparisonDuplicate, result.Comparison.Classification)
	assert.Equal(t, ProviderPathJEV, result.Provider.Path)
	assert.Contains(t, result.UnresolvedQuestions, "an equivalent canonical Runbook already exists")
}

func TestPrepareReportsInsufficientEvidenceAndStaleDigest(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-3")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writeArtifacts(t, refined, "---\nmission_id: m-3\n---\n## notes\nOnly a topic.\n")
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	result, err := Prepare(context.Background(), Input{MissionID: "m-3", CandidateID: "OA-RUNBOOK-m-3", PackageDigest: digest, RefinedPath: refined, RunbookRoot: root})
	require.NoError(t, err)
	assert.Equal(t, StatusInsufficientEvidence, result.Status)
	assert.NotEmpty(t, result.UnresolvedQuestions)
	_, err = Prepare(context.Background(), Input{MissionID: "m-3", CandidateID: "OA-RUNBOOK-m-3", PackageDigest: "sha256:stale", RefinedPath: refined, RunbookRoot: root})
	assert.ErrorContains(t, err, "stale package digest")
}

func TestPrepareTreatsEmptySelectionAsValidComparisonSetAndMissingRootAsReview(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-4")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writeArtifacts(t, refined, `
---
mission_id: m-4
---
## procedure
Run make check; expected output is green. Stop on failure. Source: CI. Scope: package.
`)
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	result, err := Prepare(context.Background(), Input{MissionID: "m-4", CandidateID: "OA-RUNBOOK-m-4", PackageDigest: digest, RefinedPath: refined, RunbookRoot: filepath.Join(root, "missing"), SelectedRunbooksHint: nil})
	require.NoError(t, err)
	assert.Equal(t, StatusNeedsReview, result.Status)
	assert.Equal(t, ComparisonInsufficient, result.Comparison.Classification)
}

func writeArtifacts(t *testing.T, dir, analysis string) {
	t.Helper()
	files := map[string]string{"analysis.md": analysis, "proposal.md": "## proposal\nThe procedure is proposed.\n", "design.md": "## design\nThe scope is bounded.\n", "tasks.md": "## tasks\n- verify expected output and stop conditions\n"}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}
