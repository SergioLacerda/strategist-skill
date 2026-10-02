package providence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/governance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeProvidenceFixture(t *testing.T, root string, metadata any, core any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "source"), 0o755))
	metadataRaw, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "metadata.json"), metadataRaw, 0o644))
	coreRaw, err := json.Marshal(core)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "source", "governance-core.json"), coreRaw, 0o644))
}

func TestSnapshot_ValidAndDeterministic(t *testing.T) {
	root := t.TempDir()
	writeProvidenceFixture(t, root,
		map[string]any{"fingerprints": map[string]any{"combined": "6b2505c4"}},
		map[string]any{"items": []map[string]any{
			{"id": "M020", "type": "MANDATE", "status": "required"},
			{"id": "G001", "type": "GUIDELINE", "status": "required"},
			{"id": "M001", "type": "MANDATE", "status": "required"},
		}})

	source := New()
	assert.Equal(t, "providence", source.Name())
	snapshot, err := source.Snapshot(root)
	require.NoError(t, err)
	assert.Equal(t, "providence", snapshot.SourceID)
	assert.Equal(t, "6b2505c4", snapshot.Fingerprint)
	assert.Equal(t, []string{"M001", "M020"}, snapshot.ActiveMandates)
	assert.True(t, snapshot.Validated)
}

func TestSnapshot_RejectsMalformedMetadata(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "metadata.json"), []byte("not-json"), 0o644))
	_, err := New().Snapshot(root)
	assert.ErrorContains(t, err, "parse")
}

func TestSnapshot_RejectsMissingCore(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "metadata.json"), []byte(`{"fingerprints":{"combined":"fp"}}`), 0o644))
	_, err := New().Snapshot(root)
	assert.ErrorContains(t, err, "governance-core.json")
}

func TestSnapshot_RejectsMalformedCore(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "source"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "metadata.json"), []byte(`{"fingerprints":{"combined":"fp"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "source", "governance-core.json"), []byte("not-json"), 0o644))
	_, err := New().Snapshot(root)
	assert.ErrorContains(t, err, "parse")
}

func TestSnapshot_RequiresCombinedFingerprint(t *testing.T) {
	root := t.TempDir()
	writeProvidenceFixture(t, root,
		map[string]any{"governance_fingerprint": "legacy"},
		map[string]any{"items": []any{}})

	_, err := New().Snapshot(root)
	assert.ErrorContains(t, err, "fingerprints.combined")
}

func TestSnapshot_RejectsEmptyRequiredMandateID(t *testing.T) {
	root := t.TempDir()
	writeProvidenceFixture(t, root,
		map[string]any{"fingerprints": map[string]any{"combined": "fp"}},
		map[string]any{"items": []map[string]any{{"type": "MANDATE", "status": "required"}}})

	_, err := New().Snapshot(root)
	assert.ErrorContains(t, err, "without an id")
}

func TestSnapshot_ReportsExplicitPathErrors(t *testing.T) {
	_, err := New().Snapshot(filepath.Join(t.TempDir(), "missing"))
	require.ErrorContains(t, err, "metadata.json")
	_, err = New().Snapshot("")
	assert.ErrorContains(t, err, "governance directory is required")
}

var _ governance.Source = New()
