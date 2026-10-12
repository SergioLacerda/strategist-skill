package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// levelingExtractor serves a fixed LEVELING policy payload or a fixed error.
type levelingExtractor struct {
	raw []byte
	err error
}

func (e levelingExtractor) Extract(string, bool) error { return nil }

func (e levelingExtractor) ReadFile(string) ([]byte, error) { return e.raw, e.err }

func TestWriteAuditLineReportsEachFailingStep(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "plain-file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	err := writeAuditLine(filepath.Join(file, "nested", "ledger.jsonl"), AuditRecord{})
	require.ErrorContains(t, err, "create install audit dir")

	asDir := filepath.Join(dir, "ledger-is-a-directory")
	require.NoError(t, os.MkdirAll(asDir, 0o755))
	err = writeAuditLine(asDir, AuditRecord{})
	require.ErrorContains(t, err, "open install audit ledger")
}

func TestAppendAuditRecordNeverFailsTheCaller(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))
	assert.NotPanics(t, func() { appendAuditRecord(file, AuditRecord{Command: "install"}) })
}

func TestSaveInstallManifestReportsWriteFailure(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))
	err := saveInstallManifest(file, domain.InstallManifest{})
	require.ErrorContains(t, err, "install: write manifest")
}

func TestRemoveLegacyLayoutFilesFailureModes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	err := removeLegacyLayoutFiles(root, []string{"../escape.yaml"})
	require.ErrorContains(t, err, "upgrade: resolve")

	nonEmpty := filepath.Join(root, "legacy")
	require.NoError(t, os.MkdirAll(nonEmpty, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nonEmpty, "keep"), []byte("x"), 0o600))
	err = removeLegacyLayoutFiles(root, []string{"legacy"})
	require.ErrorContains(t, err, "upgrade: remove legacy")

	require.NoError(t, removeLegacyLayoutFiles(root, []string{"already-gone.yaml"}))
}

func TestRestoreBackupFileFailureModes(t *testing.T) {
	t.Parallel()
	backup := t.TempDir()
	target := t.TempDir()
	walkErr := errors.New("walk failed")

	restored, err := restoreBackupFile(backup, target, filepath.Join(backup, "a"), nil, walkErr)
	require.ErrorIs(t, err, walkErr)
	assert.False(t, restored)

	gone := filepath.Join(backup, "gone.yaml")
	require.NoError(t, os.WriteFile(gone, []byte("x"), 0o600))
	entries, readErr := os.ReadDir(backup)
	require.NoError(t, readErr)
	require.Len(t, entries, 1)
	require.NoError(t, os.Remove(gone))
	_, err = restoreBackupFile(backup, target, gone, entries[0], nil)
	require.ErrorContains(t, err, "read backup")
}

func TestEmbeddedLevelingPolicyFailureModes(t *testing.T) {
	t.Parallel()
	readFailure := Service{Extractor: levelingExtractor{err: os.ErrPermission}}
	_, err := readFailure.embeddedLevelingPolicy()
	require.ErrorContains(t, err, "read embedded LEVELING policy")

	empty := Service{Extractor: levelingExtractor{raw: []byte{}}}
	_, err = empty.embeddedLevelingPolicy()
	require.ErrorContains(t, err, "is empty")

	invalid := Service{Extractor: levelingExtractor{raw: []byte("version: [unclosed")}}
	_, err = invalid.embeddedLevelingPolicy()
	require.ErrorContains(t, err, "parse embedded LEVELING policy")
}

func TestLevelingAuthoritySkipsOnlyMissingOptionalPolicies(t *testing.T) {
	t.Parallel()
	missing := Service{Extractor: levelingExtractor{err: os.ErrNotExist}}
	require.NoError(t, missing.populateLevelingAuthority(&runtimeDefaultPlan{}))
	require.NoError(t, missing.applyLevelingAuthority(&domain.InstallManifest{}))

	denied := Service{Extractor: levelingExtractor{err: os.ErrPermission}}
	require.Error(t, denied.populateLevelingAuthority(&runtimeDefaultPlan{}))
	require.Error(t, denied.applyLevelingAuthority(&domain.InstallManifest{}))
}
