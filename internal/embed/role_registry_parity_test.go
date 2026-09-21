package embed_test

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The built-in registry (used when no workspace roles are loaded) must match the
// embedded roles/*.yaml definitions exactly.
func TestBuiltInRoleRegistryMatchesEmbeddedRoleFiles(t *testing.T) {
	for _, want := range domain.DefaultRoleRegistry().Roles() {
		raw, err := embed.Extractor{}.ReadFile("roles/" + want.ID + ".yaml")
		require.NoError(t, err, "roles/%s.yaml must exist", want.ID)
		var cfg domain.RoleConfig
		require.NoError(t, yaml.Unmarshal(raw, &cfg), want.ID)
		require.NoError(t, cfg.Validate(), want.ID)
		assert.Equal(t, want, domain.RoleFromConfig(cfg), "built-in registry drifted from roles/%s.yaml", want.ID)
	}
}

func TestEveryEmbeddedRoleFileIsRegistered(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	for _, name := range []string{"ranger", "archivist", "sniper", "scout"} {
		assert.True(t, reg.Has(name), name)
	}
	raw, err := embed.Extractor{}.ReadFile("roles/default.yaml")
	require.NoError(t, err)
	var slots domain.RoleSlotMap
	require.NoError(t, yaml.Unmarshal(raw, &slots))
	for slot, id := range slots {
		role, ok := reg.RoleForSlot(slot)
		require.True(t, ok, "slot %s", slot)
		assert.Equal(t, strings.ToLower(id), role.ID, "roles/default.yaml maps %s to %s", slot, id)
	}
}
