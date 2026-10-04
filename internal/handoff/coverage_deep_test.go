package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentationTargetContentRejectsMalformedDeclarations(t *testing.T) {
	paths, err := ValidateDocumentationTargetContent([]byte("- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/guide.md"}, paths)
	for _, raw := range []string{
		"- [ ] 1.1 [documentation_target] Write the guide\n",
		"- [ ] 1.1 [documentation_target] Write `/docs/guide.md`\n",
		"- [ ] 1.1 [documentation_target] Write `docs/../guide.md`\n",
		"- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n- [ ] 1.2 [documentation_target] Write `docs/guide.md`\n",
	} {
		_, err := ValidateDocumentationTargetContent([]byte(raw))
		require.Error(t, err)
	}
}

func TestValidateRefinedPackageContentRejectsIncompleteBranches(t *testing.T) {
	_, err := ValidateRefinedPackageContent(map[string][]byte{}, "m-1")
	require.Error(t, err)

	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	files := map[string][]byte{}
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		files[name], err = os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
	}
	delete(files, "proposal.md")
	_, err = ValidateRefinedPackageContent(files, "m-1")
	require.Error(t, err)

	files["proposal.md"] = []byte("proposal\n")
	files["tasks.md"] = []byte("- [ ] 1.1 [unknown_task] do something\n")
	_, err = ValidateRefinedPackageContent(files, "m-1")
	require.Error(t, err)

	files["tasks.md"] = []byte("- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n")
	files["analysis.md"] = []byte("---\nmission_id: m-1\nmission_status: archivist_done\n---\nbody\n")
	_, err = ValidateRefinedPackageContent(files, "m-1")
	require.Error(t, err)
}

func TestValidateRefinedPackageContentRequiresNonEmptySupportFiles(t *testing.T) {
	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	files := make(map[string][]byte)
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		var err error
		files[name], err = os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
	}
	files["design.md"] = nil
	_, err := ValidateRefinedPackageContent(files, "m-1")
	require.Error(t, err)
}

func TestParsePolicyFactsRejectsWrongSchema(t *testing.T) {
	_, err := ParsePolicyFacts(map[string]any{PolicyFactsKey: map[string]any{"schema_version": "wrong"}})
	require.Error(t, err)
}

func TestReadPolicyFactsRejectsMalformedAnalysis(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte("---\nmission_id: m-1\n"), 0o600))
	_, err := readPolicyFacts(dir)
	require.Error(t, err)
}

func TestVerificationMetadataRejectsMalformedDeclarations(t *testing.T) {
	dir := t.TempDir()
	minimalArchivistPackage(t, dir, "m-1")
	analysis, err := os.ReadFile(filepath.Join(dir, "analysis.md"))
	require.NoError(t, err)
	analysis = append(analysis[:len(analysis)-len("---\n\n# Analysis\n")], []byte("handoff_verification: [bad]\n---\n\n# Analysis\n")...)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), analysis, 0o600))
	_, err = ReadVerificationMetadata(dir)
	require.Error(t, err)

	_, err = fencedVerificationMetadata([]byte("```yaml\nhandoff_verification:\n  required: [bad]\n```\n"))
	require.Error(t, err)
}

func TestOutcomeHistoryAndRangerRevisionCorrelation(t *testing.T) {
	store := fixedStore(t)
	_, err := store.Append(sampleOutcome(1, OutcomePassed))
	require.NoError(t, err)
	_, err = store.Append(sampleOutcome(2, OutcomePassed))
	require.NoError(t, err)
	history, err := store.Outcomes("m1")
	require.NoError(t, err)
	require.Len(t, history, 2)

	outcome := Outcome{MissionID: "m1", Transition: TransitionRangerToArchivist, ArtifactDigest: "sha256:artifact", PolicyID: "policy", Result: OutcomePassed}
	check := ExecutionCheck{MissionID: "m1", Transition: TransitionRangerToArchivist, ArtifactDigest: "sha256:artifact", PolicyID: "policy"}
	require.NoError(t, correlateOutcome(outcome, check))
	check.ArtifactDigest = "sha256:other"
	require.Error(t, correlateOutcome(outcome, check))
}

func TestOutcomeBoundaryErrorsWithoutRuntimeState(t *testing.T) {
	store := fixedStore(t)
	_, err := store.OutcomesFor("m1", "unknown-transition")
	require.Error(t, err)
	changed, err := store.InvalidateLatest("m1", "repair")
	require.NoError(t, err)
	assert.False(t, changed)
	_, err = store.Consumed(Outcome{})
	require.Error(t, err)
}
