package mechanisms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const validRegistry = `schema_version: "1"
mechanisms:
  - id: gate
    family: mechanism
    enforcement_kind: code
    enforcement_tier: machine_enforced
    summary: the approval gate
    when_to_use: before execution
    invoked_by: [archivist, sniper]
    how_to_invoke: mission submit
  - id: search
    family: ability
    enforcement_kind: contract
    summary: filters candidates
    invoked_by: [ranger]
    how_to_invoke: ranger.yaml#canonical.search
  - id: everywhere
    family: mechanism
    enforcement_kind: prose
    summary: applies to every role
    invoked_by: [all]
    how_to_invoke: none
`

func TestParseAcceptsAValidRegistry(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)
	assert.Len(t, reg.Rows, 3)
}

func TestParseRejectsInvalidRows(t *testing.T) {
	cases := map[string]string{
		"duplicate id":       strings.Replace(validRegistry, "id: search", "id: gate", 1),
		"unknown family":     strings.Replace(validRegistry, "family: ability", "family: route", 1),
		"unknown kind":       strings.Replace(validRegistry, "enforcement_kind: contract", "enforcement_kind: magic", 1),
		"unknown tier":       strings.Replace(validRegistry, "machine_enforced", "sort_of", 1),
		"missing summary":    strings.Replace(validRegistry, "summary: filters candidates\n", "", 1),
		"missing invoked_by": strings.Replace(validRegistry, "invoked_by: [ranger]\n", "", 1),
		"unknown role":       strings.Replace(validRegistry, "invoked_by: [ranger]", "invoked_by: [wizard]", 1),
		"no rows":            "schema_version: \"1\"\nmechanisms: []\n",
	}
	for name, raw := range cases {
		_, err := Parse([]byte(raw))
		assert.Error(t, err, name)
	}
}

func TestForRoleIncludesAllRowsAndRoleRows(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)

	assert.Equal(t, []string{"gate", "everywhere"}, rowIDs(reg.ForRole("sniper")))
	assert.Equal(t, []string{"search", "everywhere"}, rowIDs(reg.ForRole("ranger")))
	assert.Equal(t, []string{"everywhere"}, rowIDs(reg.ForRole("scout")))
}

func TestBriefIsCompactAndNamesHowToInvoke(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)

	brief := reg.Brief("sniper")

	assert.Contains(t, brief, "gate")
	assert.Contains(t, brief, "mission submit")
	assert.NotContains(t, brief, "machine_enforced", "enforcement detail is only in the full brief")
	assert.NotContains(t, brief, "search", "rows of other roles are not shown")

	full := reg.BriefFull("sniper")
	assert.Contains(t, full, "code/machine_enforced")
	assert.Contains(t, full, "use: before execution")
}

func rowIDs(rows []Row) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// --- the shipped registry ---

func shippedRegistry(t *testing.T) (Registry, string) {
	t.Helper()
	root := filepath.Join("..", "embed", "defaults")
	raw, err := os.ReadFile(filepath.Join(root, "contracts", "machine", "mechanisms.yaml"))
	require.NoError(t, err)
	reg, err := Parse(raw)
	require.NoError(t, err)
	return reg, root
}

// A brief is paid for by every role at every phase start (M005), so it is capped.
const briefByteCap = 3200

func TestShippedRegistryBriefsStayUnderTheTokenCap(t *testing.T) {
	reg, _ := shippedRegistry(t)
	for _, role := range domain.DefaultRoleRegistry().IDs() {
		brief := reg.Brief(role)
		assert.NotEmpty(t, reg.ForRole(role), role)
		assert.LessOrEqualf(t, len(brief), briefByteCap, "%s brief is %d bytes", role, len(brief))
	}
}

func TestShippedRegistryContractFilesExist(t *testing.T) {
	reg, root := shippedRegistry(t)
	for _, row := range reg.Rows {
		if row.ContractFile == "" {
			continue
		}
		for _, ref := range strings.Split(row.ContractFile, ";") {
			path := strings.TrimSpace(strings.SplitN(ref, "#", 2)[0])
			assert.FileExists(t, filepath.Join(root, filepath.FromSlash(path)), "row %s contract_file %q", row.ID, ref)
		}
	}
}

func TestShippedRegistryNamesTheDecidedItems(t *testing.T) {
	reg, _ := shippedRegistry(t)
	byID := map[string]Row{}
	for _, row := range reg.Rows {
		byID[row.ID] = row
	}
	mechanisms := []string{"critical_hit", "opportunity_attack", "initiative", "leveling", "precise_shot", "approval_gate", "handoff_challenge", "confidence_governance", "pipeline_bypass", "weapon_binding"}
	for _, id := range mechanisms {
		assert.Equal(t, FamilyMechanism, byID[id].Family, id)
	}
	for _, id := range []string{"search", "select_runbook", "riposte"} {
		assert.Equal(t, FamilyAbility, byID[id].Family, id)
	}
	assert.NotEmpty(t, byID["precise_shot"].JudgmentFacet, "PRECISE-SHOT is a hybrid: its judgment facet is named")
	assert.Equal(t, "strategist runbook select --format json --signal <signal>", byID["select_runbook"].HowToInvoke)
}

// skill.yaml names the items the prose talks about; every one must be a registry row
// of the same family, so the two vocabularies cannot drift apart again.
func TestSkillTaxonomyListsMatchTheRegistryFamilies(t *testing.T) {
	reg, root := shippedRegistry(t)
	raw, err := os.ReadFile(filepath.Join(root, "skill.yaml"))
	require.NoError(t, err)
	type item struct {
		ID string `yaml:"id"`
	}
	var skill struct {
		Taxonomy struct {
			Mechanisms []item `yaml:"mechanisms"`
			Abilities  []item `yaml:"abilities"`
		} `yaml:"taxonomy"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &skill))
	families := map[string]string{}
	for _, row := range reg.Rows {
		families[row.ID] = row.Family
	}
	require.NotEmpty(t, skill.Taxonomy.Mechanisms)
	require.NotEmpty(t, skill.Taxonomy.Abilities)
	for _, m := range skill.Taxonomy.Mechanisms {
		assert.Equalf(t, FamilyMechanism, families[m.ID], "skill.yaml lists %q as a Mechanism", m.ID)
	}
	for _, a := range skill.Taxonomy.Abilities {
		assert.Equalf(t, FamilyAbility, families[a.ID], "skill.yaml lists %q as an Ability", a.ID)
	}
}

func TestLoadReadsTheRegistryUnderARoot(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "contracts", "machine")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mechanisms.yaml"), []byte(validRegistry), 0o644))

	reg, err := Load(root)

	require.NoError(t, err)
	assert.Len(t, reg.Rows, 3)
}

func TestLoadReportsAMissingOrMalformedRegistry(t *testing.T) {
	_, err := Load(t.TempDir())
	require.Error(t, err)

	root := t.TempDir()
	dir := filepath.Join(root, "contracts", "machine")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mechanisms.yaml"), []byte("mechanisms: [unclosed"), 0o644))
	_, err = Load(root)
	require.Error(t, err)
}

func TestParseRejectsMissingRequiredFields(t *testing.T) {
	for name, raw := range map[string]string{
		"no id":            strings.Replace(validRegistry, "id: gate", "id: \"\"", 1),
		"no how_to_invoke": strings.Replace(validRegistry, "how_to_invoke: mission submit\n", "", 1),
	} {
		_, err := Parse([]byte(raw))
		assert.Error(t, err, name)
	}
}

func TestOrchestratorRowsAppearInNoRoleBrief(t *testing.T) {
	raw := strings.Replace(validRegistry, "invoked_by: [all]", "invoked_by: [orchestrator]", 1)
	reg, err := Parse([]byte(raw))
	require.NoError(t, err)

	assert.Empty(t, reg.ForRole("scout"))
	assert.Equal(t, "Tools available to scout:\n", reg.Brief("scout"))
}

// SQ-004: the tools agents reported missing from the brief.
func TestShippedRegistryCoversTheToolsAgentsFoundMissing(t *testing.T) {
	reg, _ := shippedRegistry(t)
	byID := map[string]Row{}
	for _, row := range reg.Rows {
		byID[row.ID] = row
	}

	assert.Equal(t, FamilyMechanism, byID["normalize_openspec"].Family)
	assert.Contains(t, byID["normalize_openspec"].HowToInvoke, "strategist mission normalize-openspec")
	assert.Equal(t, FamilyAbility, byID["resolve_weapon_scratch_root"].Family)
	assert.Contains(t, byID["handoff_challenge"].HowToInvoke, "--transition", "the flag a Sniper needed and did not find")
	assert.Contains(t, rowIDs(reg.ForRole("archivist")), "normalize_openspec")
	assert.Contains(t, rowIDs(reg.ForRole("ranger")), "resolve_weapon_scratch_root")
}
