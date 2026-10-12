package opportunity_attack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureEvaluatesCandidatesIndependentlyAndPersistsGateManifest(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, ".analysis", "refined", "m-1")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-1\nmission_status: archivist_done\nsources_consulted:\n  - id: external-guide\n    coverage_status: full\n---\n# analysis\n",
		"proposal.md": "# proposal\nIntroduce a new contract and a verified command with expected output.\n",
		"design.md":   "# design\nThe rejected alternative is recorded as a trade-off.\n",
		"tasks.md":    "# tasks\n- [ ] 1.1 [implementation_handoff] implement it\n",
	})
	// Treasure Chest registry files are intentionally irrelevant to OA. Invalid
	// contents prove the evaluator does not read or parse them.
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("not: [valid"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("not: [valid"), 0o644))

	digestBefore, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	manifest, err := Ensure(Input{
		RefinedPath: refined, StrategistRoot: root, MissionID: "m-1", PackageDigest: digestBefore,
		ChestIndex: func(string) (map[string]bool, error) { return map[string]bool{}, nil },
	})
	require.NoError(t, err)
	require.Len(t, manifest.Candidates, 3)
	assert.True(t, manifest.Candidates[0].Warranted)
	assert.True(t, manifest.Candidates[1].Warranted)
	assert.True(t, manifest.Candidates[2].Warranted)
	assert.Equal(t, "external-guide", manifest.Candidates[2].SourceRef)
	assert.Equal(t, []string{"OA-ADR-m-1", "OA-RUNBOOK-m-1", "OA-CHEST-m-1"}, manifestIDs(manifest))

	loaded, err := LoadManifest(root, "m-1")
	require.NoError(t, err)
	assert.Equal(t, manifest.PackageDigest, loaded.PackageDigest)
	raw, err := os.ReadFile(filepath.Join(refined, "analysis.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "opportunity_attack:")
	assert.Contains(t, string(raw), "side_quests_approved:")
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	assert.Equal(t, manifest.PackageDigest, digest, "manifest must bind the persisted package identity")
}

func TestAttachRunbookPreparationIsDigestBoundAndKeepsCandidateProposed(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-prepare")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-prepare\n---\nresolution steps and expected output\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "tasks\n",
	})
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	manifest, err := Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-prepare", PackageDigest: digest})
	require.NoError(t, err)
	result := preparerunbook.Result{SchemaVersion: preparerunbook.SchemaVersion, Status: preparerunbook.StatusPrepared, MissionID: "m-prepare", CandidateID: "OA-RUNBOOK-m-prepare", PackageDigest: manifest.PackageDigest, SourceDigest: manifest.PackageDigest}
	updated, err := AttachRunbookPreparation(root, "m-prepare", manifest.PackageDigest, result)
	require.NoError(t, err)
	assert.Equal(t, StatusProposed, updated.Candidates[1].Status)
	assert.Equal(t, preparerunbook.StatusPrepared, updated.Candidates[1].RunbookPreparation.Status)
	_, err = AttachRunbookPreparation(root, "m-prepare", "sha256:stale", result)
	assert.ErrorContains(t, err, "digest mismatch")
}

func TestEvaluateKeepsFalseCandidatesAndDoesNotTreatSelectionHintAsEvidence(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-2")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-2\nmission_status: archivist_done\nselected_runbooks_hint: [deploy]\nsources_consulted: []\n---\n# analysis\nThe selection hint is informational only.\n",
		"proposal.md": "# proposal\nKeep the current behavior.\n",
		"design.md":   "# design\nNo architecture or execution procedure is introduced.\n",
		"tasks.md":    "# tasks\n- [ ] 1.1 [analysis_artifact] record evidence\n",
	})
	evaluation, err := Evaluate(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-2"})
	require.NoError(t, err)
	assert.Equal(t, Evaluation{}, evaluation)
	manifest, err := Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-2", PackageDigest: "sha256:none"})
	require.NoError(t, err)
	require.Len(t, manifest.Candidates, 3)
	for _, candidate := range manifest.Candidates {
		assert.False(t, candidate.Warranted, candidate.ID)
	}
	raw, err := os.ReadFile(filepath.Join(refined, "analysis.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "side_quests_approved: []")
}

func TestEvaluateTreasureChestOpportunityUsesIndependentIndexBoundary(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-chest")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-chest\nsources_consulted:\n  - id: registered\n    source_path: registered.md\n    coverage_status: full\n  - id: uncovered\n    source_path: uncovered.md\n    coverage_status: full\n---\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "tasks\n",
	})
	seenRoot := ""
	evaluation, err := Evaluate(Input{
		RefinedPath: refined, StrategistRoot: root, MissionID: "m-chest",
		ChestIndex: func(gotRoot string) (map[string]bool, error) {
			seenRoot = gotRoot
			return map[string]bool{"registered": true}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, root, seenRoot)
	assert.True(t, evaluation.Chest)
	assert.Equal(t, "uncovered", evaluation.ChestSource)
}

func TestEvaluateTreasureChestDoesNotTreatPartialOrRegisteredSourcesAsOpportunities(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "m-no-chest")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-no-chest\nsources_consulted:\n  - id: partial\n    coverage_status: partial\n  - id: known\n    coverage_status: full\n---\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "tasks\n",
	})
	evaluation, err := Evaluate(Input{
		RefinedPath: refined, StrategistRoot: root, MissionID: "m-no-chest",
		ChestIndex: func(string) (map[string]bool, error) { return map[string]bool{"known": true}, nil },
	})
	require.NoError(t, err)
	assert.False(t, evaluation.Chest)
	assert.Empty(t, evaluation.ChestSource)
}

func TestEvaluateTreasureChestIndexErrorsAreExplicit(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "m-index-error")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-index-error\nsources_consulted: []\n---\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "tasks\n",
	})
	_, err := Evaluate(Input{
		RefinedPath: refined, StrategistRoot: root, MissionID: "m-index-error",
		ChestIndex: func(string) (map[string]bool, error) { return nil, assert.AnError },
	})
	assert.ErrorContains(t, err, "load Treasure Chest index")
}

func TestEnsureReplacesAChangedPackageAndAllowsReplay(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-4")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-4\nmission_status: archivist_done\nsources_consulted: []\n---\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "- [ ] 1 [analysis_artifact] evidence\n",
	})
	first, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	_, err = Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-4", PackageDigest: first})
	require.NoError(t, err)
	current, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	_, err = Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-4", PackageDigest: current})
	require.NoError(t, err)
	_, err = Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-4", PackageDigest: "sha256:changed"})
	require.NoError(t, err)
	manifest, err := LoadManifest(root, "m-4")
	require.NoError(t, err)
	assert.NotEmpty(t, manifest.PackageDigest)
	require.Error(t, VerifyDigest(manifest, "sha256:changed"))
}

func TestEnsureKeepsDigestStableWithPolicyFacts(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "facts")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: facts\nmission_status: archivist_done\nhandoff_policy_facts:\n  schema_version: strategist-handoff-policy-facts/v1\n  mandatory_constraints: []\n  unresolved_questions: [\"which store?\"]\n  forbidden_scope: []\n  destructive_operation_possible: false\n  security_sensitive_task: false\n  informational_only: false\nsources_consulted: []\n---\n\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "- [ ] 1 [analysis_artifact] evidence\n",
	})
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	manifest, err := Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "facts", PackageDigest: digest})
	require.NoError(t, err)
	current, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	assert.Equal(t, manifest.PackageDigest, current)
	_, err = Ensure(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "facts", PackageDigest: current})
	require.NoError(t, err)
}

func TestEvaluateIgnoresTreasureChestRegistryFiles(t *testing.T) {
	root := t.TempDir()
	refined := filepath.Join(root, "refined", "m-3")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	writePackage(t, refined, map[string]string{
		"analysis.md": "---\nmission_id: m-3\nmission_status: archivist_done\nsources_consulted:\n  - id: source-1\n    coverage_status: partial\n  - id: source-2\n    coverage_status: full\n---\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "- [ ] 1 [analysis_artifact] evidence\n",
	})
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources: [unterminated"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("chests: [unterminated"), 0o644))
	evaluation, err := Evaluate(Input{RefinedPath: refined, StrategistRoot: root, MissionID: "m-3"})
	require.NoError(t, err)
	assert.Equal(t, Evaluation{}, evaluation)
}

func writePackage(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(strings.TrimLeft(content, "\n")), 0o644))
	}
}

func manifestIDs(manifest Manifest) []string {
	ids := make([]string, 0, len(manifest.Candidates))
	for _, candidate := range manifest.Candidates {
		ids = append(ids, candidate.ID)
	}
	return ids
}
