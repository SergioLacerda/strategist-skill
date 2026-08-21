package treasure

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oneJewelYAML = `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#widgets"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score:
      value: 55
      reasons: ["recurring across 2 missions"]
`

func writeJewelsFileT(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(content), 0o644))
}

func TestPromoteJewel_Accept(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	require.NoError(t, PromoteJewel(dir, "jewel-1", domain.JewelStatusAccepted, "", time.Now()))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: accepted")
	assert.Contains(t, string(raw), "reviewed_by: human")
}

func TestPromoteJewel_CannotPromoteDeprecated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "x"
    source_refs: ["source#a"]
    trust: T1
    status: deprecated
    reviewed_by: human
`)

	err := PromoteJewel(dir, "jewel-1", domain.JewelStatusAccepted, "", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestPromoteJewel_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	err := PromoteJewel(dir, "does-not-exist", domain.JewelStatusAccepted, "", time.Now())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrJewelNotFound)
}

func TestPromoteJewel_VerifiedAppendsEvidenceRef(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	require.NoError(t, PromoteJewel(dir, "jewel-1", domain.JewelStatusVerified, "EVIDENCE-001", time.Now()))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: verified")
	assert.Contains(t, string(raw), "EVIDENCE-001")
}

func TestMigrateLegacyJewelStatus_RewritesActiveEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "x"
    source_refs: ["source#a"]
    trust: T1
    status: active
    reviewed_by: agent
  - id: jewel-2
    chest_id: source
    kind: pattern
    statement: "y"
    source_refs: ["source#b"]
    trust: T1
    status: accepted
    reviewed_by: human
`)

	migrated, err := MigrateLegacyJewelStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, migrated)

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "status: active")
}

func TestMigrateLegacyJewelStatus_NoActiveEntriesIsNoop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML) // status: proposed, nothing to migrate

	migrated, err := MigrateLegacyJewelStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)
}

func TestMigrateLegacyJewelStatus_NoManifestIsNoop(t *testing.T) {
	t.Parallel()
	migrated, err := MigrateLegacyJewelStatus(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)
}

func TestMigrateLegacyJewelStatusInDocument_NoJewelsSequenceIsNoop(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "schema_version: \"1\"\n")
	migrated, err := MigrateLegacyJewelStatusInDocument(doc)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)
}

func TestMigrateLegacyJewelStatusInDocument_NonMappingRootErrors(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n- b\n")
	_, err := MigrateLegacyJewelStatusInDocument(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a mapping")
}

func TestMigrateLegacyJewelStatusInDocument_NonMappingEntrySkipped(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "schema_version: \"1\"\njewels:\n  - plain-scalar\n")
	migrated, err := MigrateLegacyJewelStatusInDocument(doc)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)
}

func TestReadOrCreateJewelsDocument_CreatesNewWhenMissing(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "jewels.yaml")

	doc, err := ReadOrCreateJewelsDocument(path)
	require.NoError(t, err)
	mapping, err := RootMapping(doc)
	require.NoError(t, err)
	assert.NotNil(t, MappingValue(mapping, "jewels"))
}

func TestReadOrCreateJewelsDocument_ReturnsExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	doc, err := ReadOrCreateJewelsDocument(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	mapping, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(mapping, "jewels")
	require.Len(t, seq.Content, 1)
}

func TestReadOrCreateJewelsDocument_OtherErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory at the manifest path fails with something other than
	// os.ErrNotExist (EISDIR), forcing the propagated-error branch.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "jewels.yaml"), 0o755))

	_, err := ReadOrCreateJewelsDocument(filepath.Join(dir, "jewels.yaml"))
	require.Error(t, err)
}

func TestFindJewelDocument_ReadErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, ": not: valid: yaml:\n")

	_, _, _, err := FindJewelDocument(dir, "jewel-1")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrJewelNotFound)
}

func TestMigrateLegacyJewelStatus_ManifestPathsErrorPropagates(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := MigrateLegacyJewelStatus(dir)
	require.Error(t, err)
}

func TestMigrateLegacyJewelStatus_ReadErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, ": not: valid: yaml:\n")

	_, err := MigrateLegacyJewelStatus(dir)
	require.Error(t, err)
}
