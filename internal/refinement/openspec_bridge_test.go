package refinement

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopConfidenceRecorder(domain.ConfidenceClaim, []domain.Evidence) error { return nil }

func TestNormalizeOpenSpecPublishesCanonicalPackageAndPromotesPending(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	mission := "m-1"
	change := "20260917-change"
	pending := filepath.Join(base, "pending", mission+"-analysis.md")
	changeDir := filepath.Join(runtime, "changes", change)
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(changeDir, "specs", "private"), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte(`---
schema_version: strategist-ranger-discovery/v1
mission_id: m-1
mission_status: archivist_pending
sources_consulted: []
---

## mission_objective
objective
## known_facts
facts
## confidence_summary
summary
## handoff
handoff
`), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), canonicalTestArtifact(name, ""), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "specs", "private", "spec.md"), []byte("private\n"), 0o644))

	var recorded domain.ConfidenceClaim
	result, err := NormalizeOpenSpec(OpenSpecInput{
		MissionID: mission, BasePath: base, RuntimeRoot: runtime, ChangeID: change,
		PendingAnalysisPath: pending,
		RecordConfidence: func(claim domain.ConfidenceClaim, _ []domain.Evidence) error {
			recorded = claim
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "archivist", recorded.Agent)
	assert.Equal(t, "archivist-openspec-normalization", recorded.CorrelationKey)
	assertFile(t, filepath.Join(result.RefinedPath, "analysis.md"), "mission_status: archivist_done", "provider_change_id: "+change)
	assertFile(t, filepath.Join(result.RefinedPath, "proposal.md"), "# proposal.md")
	assertFile(t, filepath.Join(result.RefinedPath, "design.md"), "# design.md")
	assertFile(t, filepath.Join(result.RefinedPath, "tasks.md"), "[task_type: analysis_artifact]")
	_, err = os.Stat(pending)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(result.RefinedPath, "specs"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(changeDir)
	require.ErrorIs(t, err, os.ErrNotExist, "a published change leaves the active list")
	archived, err := filepath.Glob(filepath.Join(runtime, "changes", "archive", "*-"+change))
	require.NoError(t, err)
	assert.Len(t, archived, 1, "it is moved to changes/archive/<date>-<id>")
}

func TestNormalizeOpenSpecRejectsEscapingAndPartialOutput(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\n---\n# analysis\n"), 0o644))
	_, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "../escape", PendingAnalysisPath: pending, RecordConfidence: noopConfidenceRecorder})
	require.Error(t, err)

	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("proposal"), 0o644))
	_, err = NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change", PendingAnalysisPath: pending, RecordConfidence: noopConfidenceRecorder})
	require.ErrorContains(t, err, "incomplete change")
	_, err = os.Stat(filepath.Join(base, "refined", "m-1"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestNormalizeOpenSpecRejectsMissionIdentityMismatch(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: another\n---\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), canonicalTestArtifact(name, ""), 0o644))
	}
	_, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change", PendingAnalysisPath: pending, RecordConfidence: noopConfidenceRecorder})
	require.ErrorContains(t, err, "mission_id does not match")
}

func TestNormalizeOpenSpecRejectsMalformedDocumentationTargetBeforePublication(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\nmission_status: archivist_pending\n---\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), []byte(name), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] 1.1 [documentation_target] Write the guide\n"), 0o644))

	_, err := NormalizeOpenSpec(OpenSpecInput{
		MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change",
		PendingAnalysisPath: pending, RecordConfidence: noopConfidenceRecorder,
	})

	require.ErrorContains(t, err, "validate refined package")
	_, statErr := os.Stat(filepath.Join(base, "refined", "m-1"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(changeDir)
	require.NoError(t, statErr, "provider change remains active for repair")
	_, statErr = os.Stat(pending)
	require.NoError(t, statErr, "pending analysis remains available")
}

func TestNormalizeOpenSpecRejectsAnUnclassifiedTaskBeforePublication(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\nmission_status: archivist_pending\n---\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), canonicalTestArtifact(name, ""), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] 1.1 publish without classification\n"), 0o644))

	_, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change", PendingAnalysisPath: pending, RecordConfidence: noopConfidenceRecorder})

	require.ErrorContains(t, err, "task has no explicit classification")
	_, statErr := os.Stat(filepath.Join(base, "refined", "m-1"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(changeDir)
	require.NoError(t, statErr, "the provider change remains repairable")
}

func TestNormalizeOpenSpecFailsBeforePublicationWhenArchivistConfidenceCannotBeRecorded(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\nmission_status: archivist_pending\n---\n"), 0o600))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), canonicalTestArtifact(name, ""), 0o600))
	}

	_, err := NormalizeOpenSpec(OpenSpecInput{
		MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change",
		PendingAnalysisPath: pending,
		RecordConfidence: func(domain.ConfidenceClaim, []domain.Evidence) error {
			return errors.New("history unavailable")
		},
	})

	require.ErrorContains(t, err, "record Archivist confidence")
	_, statErr := os.Stat(filepath.Join(base, "refined", "m-1"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(changeDir)
	require.NoError(t, statErr, "provider change must remain active for a safe retry")
}

func canonicalTestArtifact(name, suffix string) []byte {
	if name == "tasks.md" {
		return []byte("- [ ] 1.1 [task_type: analysis_artifact] record evidence " + suffix + "\n")
	}
	return []byte("# " + name + " " + suffix + "\n")
}

func assertFile(t *testing.T, path string, fragments ...string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, fragment := range fragments {
		require.Contains(t, string(raw), fragment, "missing %q in %s", fragment, path)
	}
}
