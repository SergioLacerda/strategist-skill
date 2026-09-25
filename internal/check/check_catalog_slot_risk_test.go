package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestEmbeddedCatalogRiskMatchesSlotContract keeps the shipped catalog honest
// with the contract `strategist check` enforces: a catalog Weapon that offers a
// slot must declare the risk_score that slot requires, otherwise the operator
// could bind it and only learn at preflight that it can never pass. This is the
// bind-time face of the Sniper decision — a custom Sniper is never lower than
// `controlled`.
func TestEmbeddedCatalogRiskMatchesSlotContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "plugins", "catalog.yaml"))
	require.NoError(t, err)
	var catalog struct {
		Providers []struct {
			ID             string   `yaml:"id"`
			RiskScore      string   `yaml:"risk_score"`
			SupportedSlots []string `yaml:"supported_slots"`
		} `yaml:"providers"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &catalog))
	require.NotEmpty(t, catalog.Providers)

	offered := 0
	for _, provider := range catalog.Providers {
		for _, slot := range provider.SupportedSlots {
			required, known := slotContract[slot]
			if !known {
				continue
			}
			offered++
			assert.Equalf(t, required, provider.RiskScore, "catalog provider %q offers slot %q but declares risk_score=%q", provider.ID, slot, provider.RiskScore)
		}
	}
	assert.Positive(t, offered, "at least one catalog Weapon must offer a contracted slot")
}

func TestExecutionSlotRejectsCustomSkillProviderBelowControlled(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "custom-sniper"), 0o755))
	manifest := func(risk string) []byte {
		return []byte("id: custom-sniper\nrisk_score: " + risk + "\ncanonical_role: sniper\nroles: [sniper]\n")
	}
	skillPath := filepath.Join(root, "skills", "custom-sniper", "skill.yaml")

	require.NoError(t, os.WriteFile(skillPath, manifest("write_analysis"), 0o644))
	_, errMsg := resolveSkillProviderSlot(root, "execution", "custom-sniper", skillPath, manifest("write_analysis"))
	assert.Contains(t, errMsg, `requires "controlled"`)

	_, errMsg = resolveSkillProviderSlot(root, "execution", "custom-sniper", skillPath, manifest("controlled"))
	assert.NotContains(t, errMsg, "requires", "a controlled custom Sniper satisfies the slot contract")
}

// The catalog is the authority for a Weapon's risk_score: a stale compat view
// can neither weaken nor strengthen what the catalog declares.
func TestSlotRiskComesFromTheCatalogNotTheCompatView(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: cat-sniper\n    risk_score: controlled\n    canonical_role: sniper\n    roles: [sniper]\n  - id: cat-low\n    risk_score: write_analysis\n    canonical_role: sniper\n    roles: [sniper]\n"), 0o644))
	stale := func(risk string) []byte {
		return []byte("risk_score: " + risk + "\ncanonical_role: sniper\nroles: [sniper]\n")
	}
	path := filepath.Join(root, "skills", "x", "skill.yaml")

	_, errMsg := resolveSkillProviderSlot(root, "execution", "cat-sniper", path, stale("write_analysis"))
	assert.NotContains(t, errMsg, "requires", "the catalog says controlled; the stale view is ignored")

	_, errMsg = resolveSkillProviderSlot(root, "execution", "cat-low", path, stale("controlled"))
	assert.Contains(t, errMsg, `requires "controlled"`, "the catalog says write_analysis; the stale view cannot raise it")
}
