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
    family: feat
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
		"duplicate identity": strings.Replace(validRegistry, "id: everywhere", "id: gate", 1),
		"unknown family":     strings.Replace(validRegistry, "family: feat", "family: route", 1),
		"unknown kind":       strings.Replace(validRegistry, "enforcement_kind: contract", "enforcement_kind: magic", 1),
		"unknown tier":       strings.Replace(validRegistry, "machine_enforced", "sort_of", 1),
		"missing summary":    strings.Replace(validRegistry, "summary: filters candidates\n", "", 1),
		"missing invoked_by": strings.Replace(validRegistry, "invoked_by: [ranger]\n", "", 1),
		"unknown role":       strings.Replace(validRegistry, "invoked_by: [ranger]", "invoked_by: [wizard]", 1),
		"no rows":            "schema_version: \"1\"\nmechanisms: []\n",
		"unsupported schema": strings.Replace(validRegistry, `schema_version: "1"`, `schema_version: "2"`, 1),
	}
	for name, raw := range cases {
		_, err := Parse([]byte(raw))
		assert.Error(t, err, name)
	}
}

func TestParseUsesFamilyAwareIdentities(t *testing.T) {
	raw := strings.Replace(validRegistry, "id: search", "id: gate", 1)
	reg, err := Parse([]byte(raw))
	require.NoError(t, err)

	mechanism, err := reg.Rows[0].CanonicalIdentity()
	require.NoError(t, err)
	feat, err := reg.Rows[1].CanonicalIdentity()
	require.NoError(t, err)
	assert.NotEqual(t, mechanism, feat)
	assert.Equal(t, domain.TaxonomyMechanism, mechanism.Family)
	assert.Equal(t, domain.TaxonomyFeat, feat.Family)
}

func TestParseAcceptsAllCanonicalFamilies(t *testing.T) {
	raw := `schema_version: "1"
mechanisms:
  - id: ranger
    family: role
    enforcement_kind: contract
    summary: role identity
    invoked_by: [all]
    how_to_invoke: role.yaml
  - id: native-search
    version: 1.0.0
    family: weapon
    enforcement_kind: code
    summary: pinned weapon identity
    invoked_by: [ranger]
    how_to_invoke: embedded payload
  - id: full
    family: stage
    enforcement_kind: code
    summary: governed stage
    invoked_by: [orchestrator]
    how_to_invoke: stage resolver
  - id: roster-plan
    family: artifact
    enforcement_kind: contract
    summary: reproducible plan
    invoked_by: [orchestrator]
    how_to_invoke: install plan
`
	reg, err := Parse([]byte(raw))
	require.NoError(t, err)
	require.Len(t, reg.Rows, 4)

	identities := make([]domain.CanonicalIdentity, 0, len(reg.Rows))
	for _, row := range reg.Rows {
		identity, identityErr := row.CanonicalIdentity()
		require.NoError(t, identityErr)
		identities = append(identities, identity)
	}
	assert.Equal(t, domain.TaxonomyRole, identities[0].Family)
	assert.Equal(t, domain.TaxonomyWeapon, identities[1].Family)
	assert.Equal(t, "1.0.0", identities[1].Version)
	assert.Equal(t, domain.TaxonomyStage, identities[2].Family)
	assert.Equal(t, domain.TaxonomyArtifact, identities[3].Family)
}

func TestParseRejectsUnversionedWeaponIdentity(t *testing.T) {
	raw := strings.Replace(validRegistry, "family: feat", "family: weapon", 1)
	_, err := Parse([]byte(raw))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires version")
}

func TestForRoleIncludesAllRowsAndRoleRows(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)

	assert.Equal(t, []string{"gate", "everywhere"}, rowIDs(reg.ForRole("sniper")))
	assert.Equal(t, []string{"search", "everywhere"}, rowIDs(reg.ForRole("ranger")))
	assert.Equal(t, []string{"everywhere"}, rowIDs(reg.ForRole("scout")))
}

func TestRoleLoadoutCapabilitiesSeparatesFeatsAndTools(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)
	feats, tools := reg.RoleLoadoutCapabilities("ranger", "discovery")
	assert.Equal(t, []string{"search"}, capabilityIDs(feats))
	assert.Empty(t, tools)
}

func TestRoleLoadoutCapabilitiesFiltersSlotsAndSeparatesTools(t *testing.T) {
	raw := validRegistry + `
  - id: discovery-tool
    family: tool
    enforcement_kind: code
    summary: discovery operation
    invoked_by: [ranger]
    how_to_invoke: discovery tool
    phase_scope: [discovery]
  - id: all-tool
    family: tool
    enforcement_kind: code
    summary: universal operation
    invoked_by: [ranger]
    how_to_invoke: universal tool
    phase_scope: [all]
  - id: roster-tool
    family: tool
    enforcement_kind: code
    summary: roster operation
    invoked_by: [ranger]
    how_to_invoke: roster tool
    phase_scope: [roster]
`
	reg, err := Parse([]byte(raw))
	require.NoError(t, err)

	feats, tools := reg.RoleLoadoutCapabilities("ranger", "discovery")
	assert.Equal(t, []string{"search"}, capabilityIDs(feats))
	assert.Equal(t, []string{"discovery-tool", "all-tool"}, capabilityIDs(tools))
}

func TestBuildRoleLoadoutCombinesStageAndPinnedWeapon(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)
	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: "full_pipeline", Role: "ranger"})
	require.NoError(t, err)
	loadout, err := reg.BuildRoleLoadout(resolution, domain.RoleInvocationPlan{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0",
		WeaponDigest: "sha256:weapon", BindingDigest: "sha256:binding",
		Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded},
	})
	require.NoError(t, err)
	assert.Equal(t, domain.StageFull, loadout.Resolution.Stage)
	assert.Equal(t, []string{"search"}, capabilityIDs(loadout.Feats))
}

func TestBuildRoleLoadoutReportsDomainValidationError(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)

	_, err = reg.BuildRoleLoadout(domain.StageResolution{}, domain.RoleInvocationPlan{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build role loadout")
}

func capabilityIDs(capabilities []domain.LoadoutCapability) []string {
	ids := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		ids = append(ids, capability.ID)
	}
	return ids
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

func TestBriefForStageScopesAwarenessToRoleAndStage(t *testing.T) {
	raw := validRegistry + `
  - id: discovery-only
    family: tool
    enforcement_kind: code
    summary: discovery-only operation
    invoked_by: [ranger]
    how_to_invoke: discovery tool
    phase_scope: [discovery]
`
	reg, err := Parse([]byte(raw))
	require.NoError(t, err)

	short, err := reg.BriefForStage("ranger", domain.StageShort)
	require.NoError(t, err)
	assert.Contains(t, short, "search")
	assert.NotContains(t, short, "discovery-only")
	assert.Contains(t, short, "SHORT")

	full, err := reg.BriefFullForStage("ranger", domain.StageFull)
	require.NoError(t, err)
	assert.Contains(t, full, "discovery-only")
	assert.Contains(t, full, "code")
}

func TestBriefForStageRejectsUnknownStage(t *testing.T) {
	reg, err := Parse([]byte(validRegistry))
	require.NoError(t, err)
	_, err = reg.BriefForStage("ranger", domain.Stage("UNKNOWN"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not canonical")
}

func TestStagePhaseScopesCoversRosterAndUnknownStages(t *testing.T) {
	assert.Equal(t, map[string]bool{"bootstrap": true, "roster": true}, stagePhaseScopes(domain.StageRoster))
	assert.Nil(t, stagePhaseScopes(domain.Stage("UNKNOWN")))
}

func TestValidationHelpersReportMissingFields(t *testing.T) {
	base := Row{
		ID: "row", Family: FamilyMechanism, EnforcementKind: "code",
		Summary: "summary", HowToInvoke: "invoke", InvokedBy: []string{"ranger"},
	}
	for name, row := range map[string]Row{
		"missing summary":    func() Row { row := base; row.Summary = ""; return row }(),
		"missing invocation": func() Row { row := base; row.HowToInvoke = ""; return row }(),
		"missing roles":      func() Row { row := base; row.InvokedBy = nil; return row }(),
	} {
		require.Error(t, validateRow(row), name)
	}

	_, err := (Row{ID: "row", Family: "unknown"}).CanonicalIdentity()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown family")
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
	mechanisms := []string{"approval_gate", "handoff_challenge", "confidence_governance", "pipeline_bypass", "weapon_binding"}
	for _, id := range mechanisms {
		assert.Equal(t, FamilyMechanism, byID[id].Family, id)
	}
	for _, id := range []string{"leveling", "normalize_openspec", "resolve_weapon_scratch_root"} {
		assert.Equal(t, FamilyTool, byID[id].Family, id)
	}
	for _, id := range []string{"initiative", "critical_hit", "opportunity_attack", "side_quest", "search", "select_runbook", "riposte", "keen_senses"} {
		assert.Equal(t, FamilyFeat, byID[id].Family, id)
	}
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
			Feats      []item `yaml:"feats"`
			Tools      []item `yaml:"tools"`
		} `yaml:"taxonomy"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &skill))
	families := map[string]string{}
	for _, row := range reg.Rows {
		families[row.ID] = row.Family
	}
	require.NotEmpty(t, skill.Taxonomy.Mechanisms)
	require.NotEmpty(t, skill.Taxonomy.Feats)
	require.NotEmpty(t, skill.Taxonomy.Tools)
	for _, m := range skill.Taxonomy.Mechanisms {
		assert.Equalf(t, FamilyMechanism, families[m.ID], "skill.yaml lists %q as a Mechanism", m.ID)
	}
	for _, f := range skill.Taxonomy.Feats {
		assert.Equalf(t, FamilyFeat, families[f.ID], "skill.yaml lists %q as a Feat", f.ID)
	}
	for _, tool := range skill.Taxonomy.Tools {
		assert.Equalf(t, FamilyTool, families[tool.ID], "skill.yaml lists %q as a Tool", tool.ID)
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
	assert.Equal(t, "Tools, Mechanisms, and Feats available to scout:\n", reg.Brief("scout"))
}

// SQ-004: the tools agents reported missing from the brief.
func TestShippedRegistryCoversTheToolsAgentsFoundMissing(t *testing.T) {
	reg, _ := shippedRegistry(t)
	byID := map[string]Row{}
	for _, row := range reg.Rows {
		byID[row.ID] = row
	}

	assert.Equal(t, FamilyTool, byID["normalize_openspec"].Family)
	assert.Contains(t, byID["normalize_openspec"].HowToInvoke, "strategist mission normalize-openspec")
	assert.Equal(t, FamilyTool, byID["resolve_weapon_scratch_root"].Family)
	assert.Contains(t, byID["handoff_challenge"].HowToInvoke, "--transition", "the flag a Sniper needed and did not find")
	assert.Contains(t, rowIDs(reg.ForRole("archivist")), "normalize_openspec")
	assert.Contains(t, rowIDs(reg.ForRole("ranger")), "resolve_weapon_scratch_root")
}
