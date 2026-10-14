package embeddedweapon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCompositionViolationsOnlyMarksCompositeSkills(t *testing.T) {
	violations := map[string][]string{}
	accepted := []plugincatalog.IngestedSkill{
		{ID: "composite", Adapter: plugincatalog.ExternalSkillAdapter{Kind: "composite"}},
		{ID: "plain", Adapter: plugincatalog.ExternalSkillAdapter{Kind: "skill"}},
	}

	AddCompositionViolations(violations, accepted, errors.New("invalid composition"))

	assert.Equal(t, []string{"invalid composition"}, violations["composite"])
	assert.NotContains(t, violations, "plain")
}

func TestWriteSkillMirrorsRemovesLegacyDirectory(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "skills", "demo")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "old.txt"), []byte("old"), 0o600))

	skill := plugincatalog.IngestedSkill{
		ID:      "demo",
		Package: domain.PluginPackage{Version: "1.0.0"},
	}
	require.NoError(t, WriteSkillMirrors(plugincatalog.PluginCatalog{}, []plugincatalog.IngestedSkill{skill}, root))

	assert.NoDirExists(t, legacy)
	assert.DirExists(t, filepath.Join(root, "skills", "demo@1.0.0"))
}
