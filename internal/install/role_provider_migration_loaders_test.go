package install

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedFileExtractor serves fixed content (or a fixed error) per relative
// path, for exercising loadRoleSlotMap/loadRoleConfig's error branches
// directly without needing a full install fixture.
type fixedFileExtractor map[string]fixedFileEntry

type fixedFileEntry struct {
	data []byte
	err  error
}

func (f fixedFileExtractor) Extract(string, bool) error { return nil }

func (f fixedFileExtractor) ReadFile(relPath string) ([]byte, error) {
	entry, ok := f[relPath]
	if !ok {
		return nil, errors.New("fixedFileExtractor: no entry for " + relPath)
	}
	return entry.data, entry.err
}

func TestLoadRoleSlotMap_ReadFileError(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{roleSlotMapPath: {err: errors.New("boom")}}
	_, err := loadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, "read "+roleSlotMapPath)
	require.ErrorContains(t, err, "boom")
}

func TestLoadRoleSlotMap_InvalidYAML(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{roleSlotMapPath: {data: []byte("discovery: [unterminated")}}
	_, err := loadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, roleSlotMapPath)
}

func TestLoadRoleSlotMap_ValidationError(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{roleSlotMapPath: {data: []byte("discovery: \"\"\n")}}
	_, err := loadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, "role slot map invalid")
}

func TestLoadRoleSlotMap_Success(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{roleSlotMapPath: {data: []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n")}}
	m, err := loadRoleSlotMap(ext)
	require.NoError(t, err)
	assert.Equal(t, "ranger", m["discovery"])
}

func TestLoadRoleConfig_ReadFileError(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{}
	_, err := loadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "read roles/ranger.yaml")
}

func TestLoadRoleConfig_InvalidYAML(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{"roles/ranger.yaml": {data: []byte("role: [unterminated")}}
	_, err := loadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "roles/ranger.yaml")
}

func TestLoadRoleConfig_ValidationError(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{"roles/ranger.yaml": {data: []byte("role: \"\"\nslot: \"\"\n")}}
	_, err := loadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "roles/ranger.yaml")
}

func TestLoadRoleConfig_Success(t *testing.T) {
	t.Parallel()
	ext := fixedFileExtractor{"roles/ranger.yaml": {data: []byte("role: ranger\nslot: discovery\n")}}
	cfg, err := loadRoleConfig(ext, "ranger")
	require.NoError(t, err)
	assert.Equal(t, "ranger", cfg.Role)
	assert.Equal(t, "discovery", cfg.Slot)
}
