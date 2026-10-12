package onboarding

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"

	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRoleSlotMap_ReadFileError(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{names.RoleSlotMapPath: {Err: errors.New("boom")}}
	_, err := LoadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, "read "+names.RoleSlotMapPath)
	require.ErrorContains(t, err, "boom")
}

func TestLoadRoleSlotMap_InvalidYAML(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{names.RoleSlotMapPath: {Data: []byte("discovery: [unterminated")}}
	_, err := LoadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, names.RoleSlotMapPath)
}

func TestLoadRoleSlotMap_ValidationError(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{names.RoleSlotMapPath: {Data: []byte("discovery: \"\"\n")}}
	_, err := LoadRoleSlotMap(ext)
	require.Error(t, err)
	require.ErrorContains(t, err, "role slot map invalid")
}

func TestLoadRoleSlotMap_Success(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{names.RoleSlotMapPath: {Data: []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n")}}
	m, err := LoadRoleSlotMap(ext)
	require.NoError(t, err)
	assert.Equal(t, "ranger", m["discovery"])
}

func TestLoadRoleConfig_ReadFileError(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{}
	_, err := LoadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "read roles/ranger.yaml")
}

func TestLoadRoleConfig_InvalidYAML(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{"roles/ranger.yaml": {Data: []byte("role: [unterminated")}}
	_, err := LoadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "roles/ranger.yaml")
}

func TestLoadRoleConfig_ValidationError(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{"roles/ranger.yaml": {Data: []byte("role: \"\"\nslot: \"\"\n")}}
	_, err := LoadRoleConfig(ext, "ranger")
	require.Error(t, err)
	require.ErrorContains(t, err, "roles/ranger.yaml")
}

func TestLoadRoleConfig_Success(t *testing.T) {
	t.Parallel()
	ext := installtest.FixedFileExtractor{"roles/ranger.yaml": {Data: []byte("role: ranger\nslot: discovery\n")}}
	cfg, err := LoadRoleConfig(ext, "ranger")
	require.NoError(t, err)
	assert.Equal(t, "ranger", cfg.Role)
	assert.Equal(t, "discovery", cfg.Slot)
}
