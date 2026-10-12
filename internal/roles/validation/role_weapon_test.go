package rolevalidation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/SergioLacerda/strategist-skill/internal/testutil/customws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateRuntimeBindingsAcceptsValidExternalBindings(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

func TestValidateRuntimeBindingsRejectsMissingBinding(t *testing.T) {
	root := writeValidationRoot(t, "")
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.Len(t, failures, 2)
	require.Contains(t, failures[0].Error(), "slot=discovery")
	require.Contains(t, failures[0].Error(), "no persisted weapon binding")
}

func TestValidateRuntimeBindingsRejectsRoleMismatch(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: openspec-propose
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "openspec-propose", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "role affinity")
}

func TestValidateSkillManifestRejectsAuxiliaryToolAsMissionProvider(t *testing.T) {
	failures := validateSkillManifest("refinement", "archivist", "writing-plans", []byte("risk_score: write_analysis\ncanonical_role: auxiliary\nroles: [auxiliary]\n"))
	require.Len(t, failures, 1)
	require.Contains(t, failures[0].Error(), "role affinity")
}

func TestValidateRuntimeBindingsRejectsUncertifiedRankedMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "not a certified ranked candidate")
}

func TestValidateRuntimeBindingsAcceptsCertifiedRankedMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
  - id: openspec-propose
    risk_score: write_analysis
    canonical_role: archivist
    roles: [archivist]
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

// TestValidateRuntimeBindingsAcceptsCertifiedRankedModeOnRefinementSlot is
// the 20260916-ranked-skills-end-to-end-evaluation Layer 3 coverage-gap
// fix: the refinement-slot counterpart to
// TestValidateRuntimeBindingsAcceptsCertifiedRankedMode, which only ever
// exercised mode: ranked on the discovery binding.
func TestValidateRuntimeBindingsAcceptsCertifiedRankedModeOnRefinementSlot(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
    mode: ranked
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    risk_score: write_analysis
    canonical_role: ranger
    roles: [ranger]
  - id: openspec-propose
    canonical_role: archivist
    roles: [archivist]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

func TestValidateRuntimeBindingsRejectsRankedModeMissingFromCatalog(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: openspec-propose
    canonical_role: archivist
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "not found in catalog")
}

func writeRankedCatalogFile(t *testing.T, root, catalogYAML string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(catalogYAML), 0o644))
}

func TestValidateRuntimeBindingsRejectsInvalidMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: typo
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "invalid mode")
}

func writeValidationRoot(t *testing.T, bindings string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "brainstorming"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "openspec-propose"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "default.yaml"), []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: brainstorming\n    risk_score: write_analysis\n    canonical_role: ranger\n    roles: [ranger]\n    compatibility_source: embedded\n  - id: openspec-propose\n    risk_score: write_analysis\n    canonical_role: archivist\n    roles: [archivist]\n    compatibility_source: embedded\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "brainstorming", "skill.yaml"), []byte("risk_score: write_analysis\nroles:\n  - ranger\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "openspec-propose", "skill.yaml"), []byte("risk_score: write_analysis\nroles:\n  - archivist\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n"+bindings), 0o644))
	return root
}

func TestValidateProviderManifestReadsTheCatalogWithoutAnyCompatView(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: cat-archivist\n    risk_score: write_analysis\n    canonical_role: archivist\n    roles: [archivist]\n  - id: cat-wrong-risk\n    risk_score: controlled\n    roles: [archivist]\n"), 0o644))

	assert.Empty(t, validateProviderManifest(root, "refinement", "archivist", "cat-archivist"), "no skills/ directory exists")

	failures := validateProviderManifest(root, "refinement", "archivist", "cat-wrong-risk")
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Reason, `risk_score="controlled"`)
}

// A native role listed in the catalog (compatibility_source native_role) is validated
// as a native binding, not as a Weapon that lacks role affinity.
func TestValidateProviderManifestSendsCatalogNativeRolesToTheNativeBranch(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: archivist\n    risk_score: write_analysis\n    compatibility_source: native_role\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "archivist.yaml"), []byte("role: archivist\nslot: refinement\nextensibility: pluggable\n"), 0o644))

	assert.Empty(t, validateProviderManifest(root, "refinement", "archivist", "archivist"))

	require.NoError(t, os.Remove(filepath.Join(root, "roles", "archivist.yaml")))
	failures := validateProviderManifest(root, "refinement", "archivist", "archivist")
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Reason, "native role")
}

// A clean install ships no skills/<id>/skill.yaml: bindings of cataloged Weapons
// validate end to end from the catalog entry and the plugins.lock alone.
func TestValidateRuntimeBindingsAcceptsCatalogedWeaponsWithNoCompatView(t *testing.T) {
	root := t.TempDir()
	testutil.WriteWeaponCatalog(t, root,
		testutil.CatalogProvider{ID: "brainstorming", Risk: "write_analysis", CanonicalRole: "ranger"},
		testutil.CatalogProvider{ID: "openspec-propose", Risk: "write_analysis", CanonicalRole: "archivist"},
	)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "default.yaml"), []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: discovery\n    installed_instance_id: brainstorming\n  - slot: refinement\n    installed_instance_id: openspec-propose\n"), 0o644))
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming", "refinement": "openspec-propose"}}

	require.Empty(t, ValidateRuntimeBindings(root, active))
	for _, id := range []string{"brainstorming", "openspec-propose"} {
		_, statErr := os.Stat(filepath.Join(root, "skills", id, "skill.yaml"))
		require.ErrorIs(t, statErr, os.ErrNotExist, "the fixture writes no compat view for %s", id)
	}
}

// After a real `provider add`, the slot names the package by its instance id and
// the persisted binding is custom: the role validation accepts that spelling and
// rejects the bare package id, the same rule `strategist check` applies.
func TestValidateRuntimeBindingsAcceptsTheInstanceIdOfAnAddedPackage(t *testing.T) {
	root := customws.Workspace(t)
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming", "refinement": customws.Instance}}

	failures := ValidateRuntimeBindings(root, active)

	for _, failure := range failures {
		require.NotEqual(t, "refinement", failure.Slot, "instance-id spelling must validate: %v", failure)
	}
}

func TestValidateRuntimeBindingsRejectsThePackageIdOfAnAddedPackage(t *testing.T) {
	root := customws.Workspace(t)
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming", "refinement": "fixture-provider"}}

	failures := ValidateRuntimeBindings(root, active)

	var refinement []Failure
	for _, failure := range failures {
		if failure.Slot == "refinement" {
			refinement = append(refinement, failure)
		}
	}
	require.Len(t, refinement, 1)
	require.Contains(t, refinement[0].Reason, `persisted binding points to "`+customws.Instance+`"`)
}

func TestBuildRoleInvocationPlanResolvesAnAddedPackageByItsInstanceId(t *testing.T) {
	root := customws.Workspace(t)

	plan, err := BuildRoleInvocationPlan(root, "refinement")

	require.NoError(t, err)
	require.Equal(t, customws.Instance, plan.WeaponID)
	require.Equal(t, domain.SlotBindingModeCustom, plan.Mode)
	require.Equal(t, "archivist", plan.Role)
	require.Equal(t, int64(1), plan.BindingGeneration)
	require.Equal(t, "active", plan.BindingStatus)
}

func TestBuildRoleInvocationPlanOfAnAddedPackageCarriesTheCompleteBinding(t *testing.T) {
	root := customws.Workspace(t)

	plan, err := BuildRoleInvocationPlan(root, "refinement")

	require.NoError(t, err)
	require.NotEmpty(t, plan.WeaponDigest)
	require.NotEmpty(t, plan.SourceDigest)
	require.NotEmpty(t, plan.BindingDigest)
	require.Equal(t, "fixture-provider", strings.Split(plan.WeaponID, "@")[0])
	require.Equal(t, "1.0.0", plan.WeaponVersion)
	require.Equal(t, domain.RankedRuntimeHost, plan.Runtime.Kind)
	require.Equal(t, "local_path", plan.ConnectorID)
	require.Equal(t, "host.prompt", plan.Entrypoint)
}

// Every tampered or missing piece of evidence a real `provider add` left in
// plugins.lock fails closed with reinstall guidance and never rewrites the lock.
func TestBuildRoleInvocationPlanOfAnAddedPackageFailsClosedOnTamperedEvidence(t *testing.T) {
	cases := map[string]func(*domain.PluginLockFile){
		"incomplete adapter digest": func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].WeaponDigest = "" },
		"incomplete binding digest": func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].BindingDigest = "" },
		"identity mismatch": func(l *domain.PluginLockFile) {
			l.Bindings[bindingIndex(l, "refinement")].InstalledInstanceID = "fixture-provider"
		},
		"version mismatch":    func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].WeaponVersion = "9.9.9" },
		"role mismatch":       func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].Role = "ranger" },
		"tampered entrypoint": func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].Entrypoint = "elsewhere" },
		"tampered runtime":    func(l *domain.PluginLockFile) { l.Bindings[bindingIndex(l, "refinement")].RuntimeKind = "executable" },
		"tampered package node": func(l *domain.PluginLockFile) {
			l.Lock.Nodes[nodeIndex(l, "fixture-provider", "package")].Digest = "sha256:x"
		},
		"tampered adapter node": func(l *domain.PluginLockFile) {
			l.Lock.Nodes[nodeIndex(l, "fixture-provider", "adapter_contract")].Digest = "sha256:x"
		},
		"missing role-binding node": func(l *domain.PluginLockFile) { dropNode(l, domain.CustomRoleBindingNodeKind) },
	}
	for name, mutate := range cases {
		root := customws.Workspace(t)
		path := filepath.Join(root, "plugins.lock")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var lock domain.PluginLockFile
		require.NoError(t, yaml.Unmarshal(raw, &lock))
		mutate(&lock)
		tampered, err := yaml.Marshal(lock)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, tampered, 0o644))

		_, err = BuildRoleInvocationPlan(root, "refinement")

		if name == "role mismatch" {
			require.ErrorContains(t, err, "does not match requested Role", name) // rejected earlier by binding resolution
		} else {
			require.ErrorContains(t, err, "custom_binding_invalid", name)
			require.ErrorContains(t, err, "re-add the package", name)
		}
		after, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, tampered, after, "%s: the lock is never repaired", name)
	}
}

func bindingIndex(lock *domain.PluginLockFile, slot string) int {
	for i, binding := range lock.Bindings {
		if binding.Slot == slot {
			return i
		}
	}
	panic("no binding for slot " + slot)
}

func nodeIndex(lock *domain.PluginLockFile, id, kind string) int {
	for i, node := range lock.Lock.Nodes {
		if node.ID == id && node.Kind == kind {
			return i
		}
	}
	panic("no node " + id + "/" + kind)
}

func dropNode(lock *domain.PluginLockFile, kind string) {
	kept := lock.Lock.Nodes[:0]
	for _, node := range lock.Lock.Nodes {
		if node.Kind != kind {
			kept = append(kept, node)
		}
	}
	lock.Lock.Nodes = kept
}
