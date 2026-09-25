package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DEC-013 (generation N-1): resolving a slot through a hand-made compat view for a
// Weapon the catalog does not list is reported, so it is not a surprise when the
// fallback is removed in generation N.
func TestUncatalogedViewResolutionIsMarkedTransitional(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, "skills", "hand-made", "skill.yaml")
	raw := []byte("id: hand-made\nrisk_score: write_analysis\ncanonical_role: archivist\n")
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0o755))
	require.NoError(t, os.WriteFile(skillPath, raw, 0o644))

	res, errMsg := resolveSkillProviderSlot(root, "refinement", "hand-made", skillPath, raw)

	require.Empty(t, errMsg)
	assert.True(t, res.transitionalView)
}

func TestCatalogResolutionIsNotTransitional(t *testing.T) {
	root := catalogRoot(t)
	writePayload(t, root, "host-weapon")

	res, errMsg := resolveSlotProvider(root, "refinement", "host-weapon")

	require.Empty(t, errMsg)
	assert.False(t, res.transitionalView)
}

func TestTransitionalViewAdvisoryNamesTheProviderAndTheSlot(t *testing.T) {
	resolutions := map[string]slotResolution{
		"refinement": {kind: slotResolutionSkillProvider, transitionalView: true},
		"discovery":  {kind: slotResolutionSkillProvider},
	}
	providers := map[string]string{"refinement": "hand-made", "discovery": "brainstorming"}

	advisories := transitionalViewAdvisories(providers, resolutions)

	require.Len(t, advisories, 1)
	assert.Contains(t, advisories[0], "reason=compat_view_uncataloged")
	assert.Contains(t, advisories[0], "provider=hand-made")
	assert.Contains(t, advisories[0], "slot=refinement")
	assert.Contains(t, advisories[0], "plugins/catalog.yaml", "it tells the operator how to stop depending on the view")
}

// The advisory never blocks: a hand-made view still resolves and status stays ready.
func TestCheckCmd_JSON_UncatalogedViewIsAdvisoryOnly(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	// replace the catalog's refinement Weapon with a hand-made view the catalog does not list
	testutil.WriteWeaponCatalog(t, dir,
		testutil.CatalogProvider{ID: "brainstorming", Risk: "write_analysis", CanonicalRole: "ranger"},
		testutil.CatalogProvider{ID: "openspec-propose", Risk: "write_analysis", CanonicalRole: "archivist"},
		testutil.CatalogProvider{ID: "sdd-ask", Risk: "controlled", Source: "external"},
	)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "skills", "openspec-explore"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skills", "openspec-explore", "skill.yaml"), []byte("id: openspec-explore\nrisk_score: write_analysis\ncanonical_role: archivist\nroles:\n  - archivist\n"), 0o644))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		require.NoError(t, checkCmd.RunE(checkCmd, nil))
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	joined := ""
	for _, w := range result.Warnings {
		joined += w + "\n"
	}
	assert.Contains(t, joined, "reason=compat_view_uncataloged slot=refinement provider=openspec-explore")
}
