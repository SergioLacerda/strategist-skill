package preparerunbook

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSidecar(id string) []byte {
	return []byte("schema_version: '2'\nrunbook_id: " + id + "\nrunbook_type: operational\nsource_doc: docs/" + id + ".md\napplies_when:\n  - flaky test\nobjective: fix flaky test\n")
}

func TestCompareHelpersCoverInventoryOrderingAndRelations(t *testing.T) {
	candidate := runbook.Runbook{Objective: "Fix flaky test", AppliesWhen: []string{"flaky test"}}
	assert.Equal(t, ComparisonDuplicate, nextComparison(ComparisonNovel, "duplicate"))
	assert.Equal(t, ComparisonExtension, nextComparison(ComparisonNovel, "extension"))
	assert.Equal(t, ComparisonExtension, nextComparison(ComparisonExtension, "extension"))
	assert.Equal(t, ComparisonNovel, nextComparison(ComparisonNovel, "none"))
	assert.Equal(t, "duplicate", relationBetween(candidate, runbook.Runbook{Objective: " fix   flaky test ", AppliesWhen: []string{"flaky test"}}))
	assert.Equal(t, "extension", relationBetween(candidate, runbook.Runbook{Objective: "Fix CI test failure", AppliesWhen: []string{"ci_test_failure"}}))
	assert.Equal(t, "none", relationBetween(candidate, runbook.Runbook{Objective: "Database cleanup"}))

	paths := []string{"a.runbook.yaml", "provider.runbook.yaml", "other.runbook.yaml"}
	assert.Equal(t, paths, prioritizePaths(paths, nil))
	assert.Equal(t, []string{"provider.runbook.yaml", "a.runbook.yaml", "other.runbook.yaml"}, prioritizePaths(paths, []string{"provider", "provider"}))
	ordered := appendHintMatches(nil, paths, "runbook", map[string]bool{})
	assert.Len(t, ordered, 3)
	assert.Equal(t, paths, appendUnmatched(ordered[:1], paths, map[string]bool{paths[0]: true}))

	root := t.TempDir()
	valid := filepath.Join(root, "valid.runbook.yaml")
	require.NoError(t, os.WriteFile(valid, validSidecar("valid"), 0o644))
	ref, relation, reason := comparePath(valid, candidate)
	assert.Empty(t, reason)
	assert.Equal(t, "duplicate", relation)
	assert.Equal(t, "valid", ref.RunbookID)
	_, _, reason = comparePath(filepath.Join(root, "missing.runbook.yaml"), candidate)
	assert.NotEmpty(t, reason)
	invalid := filepath.Join(root, "invalid.runbook.yaml")
	require.NoError(t, os.WriteFile(invalid, []byte("not: a runbook"), 0o644))
	_, _, reason = comparePath(invalid, candidate)
	assert.NotEmpty(t, reason)
	require.NoError(t, os.Remove(invalid))

	pathsFound, err := inventoryPaths(root, 1)
	require.NoError(t, err)
	assert.Len(t, pathsFound, 1)
	_, err = inventoryPaths(filepath.Join(root, "missing"), 1)
	require.Error(t, err)
	assert.Equal(t, "Runbook root is not configured", runbookRootReason(" "))
	assert.Equal(t, "Runbook root is not readable", runbookRootReason(filepath.Join(root, "missing")))
	file := filepath.Join(root, "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	assert.Equal(t, "Runbook root is not a directory", runbookRootReason(file))
	assert.Empty(t, runbookRootReason(root))

	comparison, err := compare(Input{RunbookRoot: root, MaxInventory: 2, SelectedRunbooksHint: []string{"valid"}}, candidate)
	require.NoError(t, err)
	assert.Equal(t, []string{"valid"}, comparison.SelectedHints)
	assert.Equal(t, ComparisonDuplicate, comparison.Classification)
	comparison, err = compare(Input{}, candidate)
	require.NoError(t, err)
	assert.Equal(t, ComparisonInsufficient, comparison.Classification)
}

func TestCompareInventoryAndWalkerErrorsAreReported(t *testing.T) {
	_, err := inventoryPaths(filepath.Join(t.TempDir(), "missing"), 0)
	require.Error(t, err)
	assert.Equal(t, "Runbook inventory entry could not be read", compareInventory([]string{filepath.Join(t.TempDir(), "missing")}, runbook.Runbook{}).Reason)
	paths := []string{}
	callback := collectInventoryPath(&paths, 8)
	require.EqualError(t, callback("", nil, errors.New("walk failed")), "walk failed")
	assert.NoError(t, callback("x.txt", fileEntry(false, "x.txt"), nil))
	assert.NoError(t, callback("x.runbook.yaml", fileEntry(false, "x.runbook.yaml"), nil))
	assert.Equal(t, []string{"x.runbook.yaml"}, paths)
}

func fileEntry(_ bool, name string) fakeDirEntry {
	return fakeDirEntry{name: name}
}

type fakeDirEntry struct{ name string }

func (e fakeDirEntry) Name() string             { return e.name }
func (fakeDirEntry) IsDir() bool                { return false }
func (fakeDirEntry) Type() os.FileMode          { return 0 }
func (fakeDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("not needed") }

func TestPreparationHelpersCoverCandidateAndProviderBranches(t *testing.T) {
	root := t.TempDir()
	artifactPath := filepath.Join(root, "analysis.md")
	require.NoError(t, os.WriteFile(artifactPath, []byte("analysis"), 0o644))
	artifacts := []artifact{
		{path: artifactPath, text: "procedure verification stop condition provenance scope", front: map[string]any{"objective": "front objective", "applies_when": []any{"front signal"}}},
	}
	evidence := Evidence{StopConditions: true}
	candidate := buildCandidate(Input{MissionID: "mission-1", PackageDigest: "sha256:1234567890123456"}, artifacts, "analytical runbook", evidence)
	assert.Equal(t, runbook.RunbookTypeAnalytical, candidate.RunbookType)
	assert.Equal(t, "front objective", candidate.Objective)
	assert.NotEmpty(t, candidate.DecisionGates)
	assert.NotEmpty(t, candidate.Verification)

	defaults := buildCandidate(Input{MissionID: "mission 2", PackageDigest: "not-a-digest"}, nil, "## objective\nheading objective\n", Evidence{})
	assert.Contains(t, defaults.Objective, "heading objective")
	assert.NotEmpty(t, defaults.AppliesWhen)
	assert.NotEmpty(t, defaults.Verification)
	assert.NotEmpty(t, defaults.RunbookID)

	assert.Equal(t, "front objective", firstFrontString(artifacts, "missing", "objective"))
	assert.Empty(t, firstFrontString(artifacts, "missing"))
	assert.Equal(t, []string{"front signal"}, firstFrontStrings(artifacts, "applies_when"))
	assert.Nil(t, firstFrontStrings(artifacts, "missing"))

	assert.Empty(t, provenance([]artifact{{path: filepath.Join(root, "missing")}}))
	assert.Len(t, provenance(artifacts), 1)

	assert.Equal(t, ProviderPathMain, evaluateProvider(context.Background(), Input{}, "digest", candidate, Comparison{}).Path)
	reading := evaluateProvider(context.Background(), Input{Evaluate: func(context.Context, ProviderRequest) (ProviderReading, error) {
		return ProviderReading{}, errors.New("provider down")
	}}, "digest", candidate, Comparison{})
	assert.Contains(t, reading.Reason, "jev_error")
	reading = evaluateProvider(context.Background(), Input{Evaluate: func(context.Context, ProviderRequest) (ProviderReading, error) {
		return ProviderReading{Path: ProviderPathJEV}, nil
	}}, "digest", candidate, Comparison{Classification: ComparisonNovel})
	assert.Equal(t, ProviderPathJEV, reading.Path)
}

func TestPreparationInputAndParsingFailures(t *testing.T) {
	inputs := []Input{
		{CandidateID: "candidate", PackageDigest: "digest", RefinedPath: "path"},
		{MissionID: "mission", PackageDigest: "digest", RefinedPath: "path"},
		{MissionID: "mission", CandidateID: "candidate", RefinedPath: "path"},
		{MissionID: "mission", CandidateID: "candidate", PackageDigest: "digest"},
	}
	for _, input := range inputs {
		_, err := Prepare(context.Background(), input)
		require.Error(t, err)
	}
	assert.Equal(t, []string{"procedure", "reproducible", "expected_outputs", "stop_conditions", "provenance", "scope"}, (Evidence{}).Missing())
	assert.Empty(t, (Evidence{Procedure: true, Reproducible: true, ExpectedOutputs: true, StopConditions: true, Provenance: true, Scope: true}).Missing())

	assert.Empty(t, firstHeadingText("# title", "objective"))
	assert.Equal(t, "value", firstHeadingText("## objective\nvalue\n", "objective"))
	assert.Empty(t, firstHeadingText("## objective\nvalue", "other"))
	front, body := splitFrontmatter("plain text")
	assert.Empty(t, front)
	assert.Equal(t, "plain text", body)
	front, body = splitFrontmatter("---\nmissing")
	assert.Empty(t, front)
	assert.Equal(t, "---\nmissing", body)
	front, body = splitFrontmatter("---\nobjective: value\n---\nbody")
	assert.Equal(t, "value", front["objective"])
	assert.Equal(t, "body", body)
	front, body = splitFrontmatter("---\n[invalid\n---\nbody")
	assert.Empty(t, front)
	assert.Equal(t, "body", body)

	assert.Equal(t, []string{"A", "B"}, frontStrings([]any{" A ", 1, "B"}))
	assert.Equal(t, []string{"plain"}, frontStrings(" plain "))
	assert.Nil(t, frontStrings(" "))
	assert.Equal(t, []string{"A", "B"}, nonEmptyStrings([]any{" A ", "", 2, "B"}))
	assert.Empty(t, signalNames("no known signal"))
	assert.Equal(t, []string{"Provide deterministic evidence for scope"}, missingQuestions([]string{"scope"}))
}

func TestTextAndReasonHelpersCoverRemainingBranches(t *testing.T) {
	assert.Equal(t, []string{"Run verification"}, matchingLines("Run verification\nRun verification", "verification"))
	assert.Equal(t, "clean**", func() string {
		value, ok := matchingLine("- **clean**", []string{"clean"})
		assert.True(t, ok)
		return value
	}())
	_, ok := matchingLine("  ", []string{"clean"})
	assert.False(t, ok)
	assert.Equal(t, "a", normalize(" A  "))
	assert.Equal(t, "a|b", normalizeList([]string{"B", " a "}))
	assert.Equal(t, []string{"a", "b"}, unique([]string{"a", "a", "b"}, 8))
	assert.Equal(t, "mission-1", slug("Mission 1!"))
	assert.Equal(t, "abcdefabcdef", shortDigest("sha256:abcdefabcdef1234"))
	assert.Equal(t, "not-a-digest", shortDigest("not a digest"))
	assert.True(t, hasAny("contains procedure", "procedure"))
	assert.False(t, hasAny("empty", "missing"))
	for _, kind := range []ComparisonKind{ComparisonDuplicate, ComparisonExtension, ComparisonNovel, ComparisonInsufficient, "unknown"} {
		assert.NotEmpty(t, comparisonReason(kind))
	}
}

func TestReadArtifactsAndPrepareProviderSuccess(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("---\nobjective: value\n---\ncontent"), 0o644))
	}
	artifacts, err := readArtifacts(root)
	require.NoError(t, err)
	assert.Len(t, artifacts, 4)
	assert.Contains(t, joinArtifacts(artifacts), "content")

	empty := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(empty, []byte("  "), 0o644))
	_, err = readArtifacts(filepath.Dir(empty))
	require.Error(t, err)
	require.NoError(t, os.Remove(filepath.Join(filepath.Dir(empty), "analysis.md")))
	_, err = readArtifacts(filepath.Dir(empty))
	require.Error(t, err)

	input := Input{MissionID: "mission", CandidateID: "candidate", PackageDigest: "sha256:wrong", RefinedPath: root, RunbookRoot: filepath.Join(root, "inventory")}
	_, err = Prepare(context.Background(), input)
	require.Error(t, err)
	assert.Equal(t, []string{"hint"}, selectedRunbookHints(Input{SelectedRunbooksHint: []string{"hint"}}, artifacts))
	assert.Equal(t, []string{"value"}, selectedRunbookHints(Input{}, []artifact{{front: map[string]any{"selected_runbooks_hint": "value"}}}))
}
