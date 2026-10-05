package opportunity_attack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateRejectsInvalidInputsAndEmptyArtifacts(t *testing.T) {
	_, err := Evaluate(Input{})
	require.ErrorContains(t, err, "mission id is required")

	refined := t.TempDir()
	_, err = Evaluate(Input{MissionID: "missing", RefinedPath: refined})
	require.ErrorContains(t, err, "read analysis.md")

	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("\n"), 0o644))
	_, err = Evaluate(Input{MissionID: "empty", RefinedPath: refined})
	require.ErrorContains(t, err, "analysis.md is empty")
}

func TestTreasureChestSourceProjectionBranches(t *testing.T) {
	assert.Equal(t, "source-id", sourceReference(SourceEvidence{ID: "source-id", Path: "source.md"}))
	assert.Equal(t, "source.md", sourceReference(SourceEvidence{Path: "source.md"}))
	assert.Nil(t, sourceEvidence("not a source list"))
	projected := sourceEvidence([]any{
		map[string]any{"id": "source-id", "source_path": "source.md", "coverage_status": "FULL"},
		"ignored",
		map[string]any{"source_path": "path-only", "coverage_status": "full"},
	})
	require.Len(t, projected, 2)
	assert.Equal(t, "full", projected[0].CoverageStatus)
	assert.Equal(t, "path-only", projected[1].Path)
	assert.Equal(t, "path-only", firstUnregisteredSource(projected[1:], map[string]bool{}).SourceRef)
	assert.False(t, firstUnregisteredSource(projected[1:], map[string]bool{"path-only": true}).Warranted)

	_, err := evaluateChestOpportunity("body", t.TempDir(), func(string) (map[string]bool, error) {
		return nil, nil
	})
	assert.ErrorContains(t, err, "frontmatter is missing")
}

func TestAnalysisFrontmatterValidation(t *testing.T) {
	_, err := analysisFrontmatter([]byte("body"))
	require.ErrorContains(t, err, "frontmatter is missing")
	_, err = analysisFrontmatter([]byte("---\nmission_id: m\n"))
	require.ErrorContains(t, err, "frontmatter is unclosed")
	_, err = analysisFrontmatter([]byte("---\n: [\n---\nbody"))
	require.ErrorContains(t, err, "parse analysis frontmatter")

	frontmatter, err := analysisFrontmatter([]byte("---\nmission_id: m\n---\nbody"))
	require.NoError(t, err)
	assert.Equal(t, "m", frontmatter["mission_id"])
}

func TestEvaluateChestOpportunityRejectsMalformedAnalysis(t *testing.T) {
	_, err := evaluateChestOpportunity("not frontmatter", "root", func(string) (map[string]bool, error) {
		return nil, nil
	})
	require.ErrorContains(t, err, "frontmatter is missing")
}

func TestSourceEvidenceUsesPathFallbackAndSkipsInvalidValues(t *testing.T) {
	assert.Nil(t, sourceEvidence("not a list"))
	sources := sourceEvidence([]any{
		"not a mapping",
		map[string]any{"id": 42, "source_path": " source.md ", "coverage_status": "FULL"},
	})
	require.Len(t, sources, 1)
	assert.Equal(t, SourceEvidence{Path: "source.md", CoverageStatus: "full"}, sources[0])
	assert.Equal(t, chestOpportunity{Warranted: true, SourceRef: "source.md"}, firstUnregisteredSource(sources, map[string]bool{}))
}

func TestUpdateAnalysisFrontmatterValidationAndFiltering(t *testing.T) {
	manifest := Manifest{MissionID: "m", Candidates: []Candidate{
		{ID: "OA-ADR-m", Kind: KindADR, Warranted: true},
		{ID: "OA-RUNBOOK-m", Kind: KindRunbook, Warranted: false},
	}}
	_, err := updateAnalysisFrontmatter([]byte("body"), manifest)
	require.ErrorContains(t, err, "frontmatter is missing")
	_, err = updateAnalysisFrontmatter([]byte("---\nmission_id: m\n"), manifest)
	require.ErrorContains(t, err, "frontmatter is unclosed")
	_, err = updateAnalysisFrontmatter([]byte("---\n: [\n---\nbody"), manifest)
	require.ErrorContains(t, err, "parse analysis frontmatter")

	updated, err := updateAnalysisFrontmatter([]byte("---\nmission_id: m\n---\nbody\n"), manifest)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "opportunity_attack:")
	assert.Contains(t, string(updated), "side_quests_approved:")
	assert.Contains(t, string(updated), "body")
}

func TestEnsureAndPersistenceRejectInvalidStores(t *testing.T) {
	refined := t.TempDir()
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m\nmission_status: archivist_done\n---\nnew contract\n",
		"proposal.md": "proposal\n",
		"design.md":   "design\n",
		"tasks.md":    "tasks\n",
	})

	_, err := Ensure(Input{RefinedPath: refined, MissionID: "m"})
	require.ErrorContains(t, err, "package digest is required")

	_, err = Ensure(Input{RefinedPath: refined, MissionID: "m", PackageDigest: "sha256:input"})
	require.ErrorContains(t, err, "strategist root is required")

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "opportunity-attacks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "opportunity-attacks", "m.yaml"), []byte("[invalid"), 0o644))
	_, err = Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m", PackageDigest: "sha256:input"})
	require.ErrorContains(t, err, "parse manifest")
}

func TestEnsurePropagatesEvaluationAndFrontmatterErrors(t *testing.T) {
	missingArtifact := t.TempDir()
	writePackage(t, missingArtifact, map[string]string{
		"analysis.md": "analysis\n",
		"design.md":   "design\n",
		"tasks.md":    "tasks\n",
	})
	_, err := Ensure(Input{RefinedPath: missingArtifact, StrategistRoot: t.TempDir(), MissionID: "missing-proposal", PackageDigest: "sha256:input"})
	require.ErrorContains(t, err, "read proposal.md")

	invalidFrontmatter := t.TempDir()
	writePackage(t, invalidFrontmatter, map[string]string{
		"analysis.md": "analysis without frontmatter\n",
		"proposal.md": "proposal\n",
		"design.md":   "design\n",
		"tasks.md":    "tasks\n",
	})
	_, err = Ensure(Input{RefinedPath: invalidFrontmatter, StrategistRoot: t.TempDir(), MissionID: "bad-frontmatter", PackageDigest: "sha256:input"})
	require.ErrorContains(t, err, "frontmatter is missing")
}

func TestPersistenceAndDigestErrorBranches(t *testing.T) {
	manifest := Manifest{MissionID: "m", PackageDigest: "sha256:one"}
	root := t.TempDir()
	manifestDir := filepath.Join(root, "opportunity-attacks")
	require.NoError(t, os.MkdirAll(manifestDir, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(manifestDir, "m.yaml"), 0o755))
	require.ErrorContains(t, persistManifest(root, manifest), "read existing manifest")
	require.NoError(t, os.Remove(filepath.Join(manifestDir, "m.yaml")))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "m.yaml"), []byte("[invalid"), 0o644))
	require.ErrorContains(t, persistManifest(root, manifest), "parse existing manifest")

	fileRoot := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(fileRoot, []byte("file"), 0o644))
	require.ErrorContains(t, persistManifest(fileRoot, manifest), "create manifest directory")

	manifest.MissionID = "nested/m"
	require.ErrorContains(t, persistManifest(t.TempDir(), manifest), "write manifest")

	require.NoError(t, VerifyDigest(manifest, "sha256:one"))
	for _, pair := range []struct{ manifest, packageDigest string }{
		{manifest: "", packageDigest: "sha256:one"},
		{manifest: "sha256:one", packageDigest: ""},
		{manifest: "sha256:one", packageDigest: "sha256:two"},
	} {
		require.Error(t, VerifyDigest(Manifest{PackageDigest: pair.manifest}, pair.packageDigest))
	}

	_, err := LoadManifest(t.TempDir(), "missing")
	require.Error(t, err)
}

func TestPersistAnalysisAndLoadManifestErrors(t *testing.T) {
	refined := t.TempDir()
	manifest := Manifest{MissionID: "m"}
	require.ErrorContains(t, persistAnalysis(refined, manifest), "read analysis for manifest")

	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("body"), 0o644))
	require.ErrorContains(t, persistAnalysis(refined, manifest), "frontmatter is missing")
	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("---\nmission_id: m\n"), 0o644))
	require.ErrorContains(t, persistAnalysis(refined, manifest), "frontmatter is unclosed")
	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("---\nmission_id: m\n---\nbody\n"), 0o644))
	require.NoError(t, os.Chmod(filepath.Join(refined, "analysis.md"), 0o444))
	require.ErrorContains(t, persistAnalysis(refined, manifest), "persist analysis manifest")
	require.NoError(t, os.Chmod(filepath.Join(refined, "analysis.md"), 0o644))

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "opportunity-attacks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "opportunity-attacks", "bad.yaml"), []byte("[invalid"), 0o644))
	_, err := LoadManifest(root, "bad")
	require.ErrorContains(t, err, "parse manifest")
}
