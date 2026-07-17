package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Whitebox tests targeting uncovered branches in treasure_chest_mutate.go: helper
// functions exercised directly with hand-built inputs, bypassing the full add/remove
// command flow where that's the only way to reach an error branch.

// --- loadChestYAMLDocs ---

func TestLoadChestYAMLDocs_ActiveMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, _, err := loadChestYAMLDocs("op", filepath.Join(dir, "active.yaml"), filepath.Join(dir, "gov.yaml"), filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "op")
}

func TestLoadChestYAMLDocs_GovernedMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	_, _, _, err := loadChestYAMLDocs("op", activePath, filepath.Join(dir, "gov.yaml"), filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "op")
}

func TestLoadChestYAMLDocs_IndexMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	governedPath := filepath.Join(dir, "gov.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(governedPath, []byte("chests: []\n"), 0o644))
	_, _, _, err := loadChestYAMLDocs("op", activePath, governedPath, filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "op")
}

func TestLoadChestYAMLDocs_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	governedPath := filepath.Join(dir, "gov.yaml")
	indexPath := filepath.Join(dir, "idx.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(governedPath, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(indexPath, []byte("sources: []\n"), 0o644))

	activeDoc, governedDoc, indexDoc, err := loadChestYAMLDocs("op", activePath, governedPath, indexPath)
	require.NoError(t, err)
	assert.NotNil(t, activeDoc)
	assert.NotNil(t, governedDoc)
	assert.NotNil(t, indexDoc)
}

// --- applyAddMutations ---

func TestApplyAddMutations_ActiveMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "- a\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := applyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

func TestApplyAddMutations_GovernedMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "- a\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := applyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

func TestApplyAddMutations_IndexMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "- a\n")

	err := applyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

func TestApplyAddMutations_Success(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := applyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.NoError(t, err)
}

// --- applyRemoveMutations ---

func TestApplyRemoveMutations_ActiveError(t *testing.T) {
	t.Parallel()
	docs := chestDocSet{
		active:   mustParseDoc(t, "mode: epic\n"), // no treasure_chests declared
		governed: mustParseDoc(t, "chests: []\n"),
		index:    mustParseDoc(t, "sources: []\n"),
	}
	err := applyRemoveMutations(docs, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestApplyRemoveMutations_GovernedError(t *testing.T) {
	t.Parallel()
	docs := chestDocSet{
		active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		governed: mustParseDoc(t, "schema_version: \"1\"\n"), // no chests declared
		index:    mustParseDoc(t, "sources: []\n"),
	}
	err := applyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestApplyRemoveMutations_IndexError(t *testing.T) {
	t.Parallel()
	docs := chestDocSet{
		active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		governed: mustParseDoc(t, "chests:\n  - id: a\n"),
		index:    mustParseDoc(t, "schema_version: \"1\"\n"), // no sources declared
	}
	err := applyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestApplyRemoveMutations_JewelsError(t *testing.T) {
	t.Parallel()
	docs := chestDocSet{
		active:    mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		governed:  mustParseDoc(t, "chests:\n  - id: a\n"),
		index:     mustParseDoc(t, "sources:\n  - id: a\n"),
		jewels:    mustParseDoc(t, "- a\n"), // not a mapping -> rootMapping error
		hasJewels: true,
	}
	err := applyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestApplyRemoveMutations_Success(t *testing.T) {
	t.Parallel()
	docs := chestDocSet{
		active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		governed: mustParseDoc(t, "chests:\n  - id: a\n"),
		index:    mustParseDoc(t, "sources:\n  - id: a\n"),
	}
	err := applyRemoveMutations(docs, "a")
	require.NoError(t, err)
}

// --- loadRemoveDocs ---

func TestLoadRemoveDocs_ActiveMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := newChestPaths(dir)
	_, err := loadRemoveDocs(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestLoadRemoveDocs_GovernedMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := newChestPaths(dir)
	require.NoError(t, os.WriteFile(p.active, []byte("mode: epic\n"), 0o644))
	_, err := loadRemoveDocs(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestLoadRemoveDocs_IndexMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := newChestPaths(dir)
	require.NoError(t, os.WriteFile(p.active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.governed, []byte("chests: []\n"), 0o644))
	_, err := loadRemoveDocs(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestLoadRemoveDocs_JewelsPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := newChestPaths(dir)
	require.NoError(t, os.WriteFile(p.active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.governed, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.index, []byte("sources: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.jewels, []byte("jewels: []\n"), 0o644))

	docs, err := loadRemoveDocs(p)
	require.NoError(t, err)
	assert.True(t, docs.hasJewels)
	assert.NotNil(t, docs.jewels)
}

func TestLoadRemoveDocs_JewelsAbsentIsNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := newChestPaths(dir)
	require.NoError(t, os.WriteFile(p.active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.governed, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.index, []byte("sources: []\n"), 0o644))

	docs, err := loadRemoveDocs(p)
	require.NoError(t, err)
	assert.False(t, docs.hasJewels)
}

// --- resolveRemoveTarget ---

func TestResolveRemoveTarget_NoPathReturnsIDFlag(t *testing.T) {
	t.Parallel()
	id, err := resolveRemoveTarget(t.TempDir(), "", "flag-id")
	require.NoError(t, err)
	assert.Equal(t, "flag-id", id)
}

func TestResolveRemoveTarget_LoadActiveChestsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valid: yaml:\n"), 0o644))
	_, err := resolveRemoveTarget(dir, "some/path", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestResolveRemoveTarget_NoMatchesFallsBackToIDFlag(t *testing.T) {
	t.Parallel()
	dir := minimalTreasureChestRoot(t)
	id, err := resolveRemoveTarget(dir, "/no/such/path", "flag-id")
	require.NoError(t, err)
	assert.Equal(t, "flag-id", id)
}

func TestResolveRemoveTarget_NoMatchesNoIDFlagErrors(t *testing.T) {
	t.Parallel()
	dir := minimalTreasureChestRoot(t)
	_, err := resolveRemoveTarget(dir, "/no/such/path", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no chest registered")
}

func TestResolveRemoveTarget_MultipleMatchesIsAmbiguous(t *testing.T) {
	t.Parallel()
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source-1
    path: .sdd/source
    scope: all
  - id: source-2
    path: .sdd/source
    scope: all
`), 0o644))

	_, err := resolveRemoveTarget(dir, ".sdd/source", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "source-1")
	assert.Contains(t, err.Error(), "source-2")
}

// --- checkChestIDAvailable ---

func TestCheckChestIDAvailable_LoadActiveChestsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valid: yaml:\n"), 0o644))
	err := checkChestIDAvailable(dir, "any-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

// --- parseTagsFlag ---

func TestParseTagsFlag_AllPartsBlankAfterTrim(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"all"}, parseTagsFlag(" , , "))
}

// --- runTreasureChestAdd / runTreasureChestRemove: resolveStrategistRoot error ---

func TestTreasureChestAdd_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = "" // forces findStrategistRoot(cwd)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

func TestTreasureChestRemove_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = ""
	treasureChestRemoveID = "source"

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

// --- runTreasureChestAdd: loadChestYAMLDocs / writeYAMLNodes error at command level ---

func TestTreasureChestAdd_MissingGovernedFileErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "treasure-chests.yaml")))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir

	err := treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest add")
}

// --- runTreasureChestRemove: loadRemoveDocs / applyRemoveMutations / writeRemoveDocs error at command level ---

func TestTreasureChestRemove_MissingGovernedFileErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "treasure-chests.yaml")))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

func TestTreasureChestRemove_ApplyMutationsErrorAtCommandLevel(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	// active.yaml has no treasure_chests key at all -> removeActiveChestEntry fails
	// inside applyRemoveMutations, reached via --id (skips resolveRemoveTarget's path lookup).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
`), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
	assert.Contains(t, err.Error(), "no treasure_chests declared")
}

func TestTreasureChestRemove_WriteErrorAtCommandLevel(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := minimalTreasureChestRoot(t)
	// Make treasure-chests.yaml unwritable so writeRemoveDocs fails after the
	// active.yaml write already succeeded (partial-write path).
	govPath := filepath.Join(dir, "treasure-chests.yaml")
	require.NoError(t, os.Chmod(govPath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(govPath, 0o644) })

	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest remove")
}

// --- finishChestAdd ---

func TestFinishChestAdd_IndexAfterSuccess(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	orig := treasureChestAddIndexAfter
	t.Cleanup(func() { treasureChestAddIndexAfter = orig })
	treasureChestAddIndexAfter = true

	indexPath := filepath.Join(dir, "knowledge.index.yaml")
	out := captureStdout(t, func() {
		require.NoError(t, finishChestAdd(dir, indexPath))
	})
	assert.Contains(t, out, "index refreshed")

	_, err := os.Stat(filepath.Join(dir, ".compiled"))
	require.NoError(t, err)
}

func TestFinishChestAdd_IndexAfterCompileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := minimalTreasureChestRoot(t)
	orig := treasureChestAddIndexAfter
	t.Cleanup(func() { treasureChestAddIndexAfter = orig })
	treasureChestAddIndexAfter = true

	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(compiledDir, 0o755) })

	indexPath := filepath.Join(dir, "knowledge.index.yaml")
	err := finishChestAdd(dir, indexPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rebuild index")
}
