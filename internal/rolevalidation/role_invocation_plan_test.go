package rolevalidation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// completeCustomLockYAML renders a plugins.lock whose discovery slot carries a
// complete Custom binding and its lock evidence, as `provider add` produces.
func completeCustomLockYAML(t *testing.T, mutate func(*domain.PluginLockFile)) string {
	t.Helper()
	evidence, err := domain.NewCustomBindingEvidence(domain.CustomPackageFacts{
		PackageID: "brainstorming", PackageVersion: "1.0.0", Role: "ranger", Slot: "discovery",
		PackageDigest: "sha256:pkg", AdapterDigest: "sha256:adapter",
		RuntimeKind: domain.RankedRuntimeHost, ConnectorID: "local_path", Entrypoint: "discover",
	}, 1, "active")
	require.NoError(t, err)
	lock := domain.PluginLockFile{
		SchemaVersion: domain.PluginLockFileSchemaVersion,
		Bindings:      []domain.SlotBinding{evidence.Binding},
		Lock:          domain.PluginLock{Nodes: evidence.Nodes},
	}
	if mutate != nil {
		mutate(&lock)
	}
	raw, err := yaml.Marshal(lock)
	require.NoError(t, err)
	return string(raw)
}

func writeCustomValidationRoot(t *testing.T, mutate func(*domain.PluginLockFile)) string {
	t.Helper()
	root := writeValidationRoot(t, "")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(completeCustomLockYAML(t, mutate)), 0o644))
	return root
}

func TestBuildRoleInvocationPlan_ResolvesRoleFromSlotMapAndLock(t *testing.T) {
	root := writeCustomValidationRoot(t, nil)

	plan, err := BuildRoleInvocationPlan(root, "discovery")
	require.NoError(t, err)
	require.Equal(t, "ranger", plan.Role)
	require.Equal(t, "discovery", plan.Slot)
	require.Equal(t, "brainstorming@1.0.0", plan.WeaponID)
	require.Equal(t, "sha256:adapter", plan.WeaponDigest)
	require.NotEmpty(t, plan.BindingDigest)
	require.Equal(t, "discover", plan.Entrypoint)
}

func TestBuildRoleInvocationPlan_RejectsIncompleteOrTamperedCustomBindings(t *testing.T) {
	cases := map[string]func(*domain.PluginLockFile){
		"missing weapon digest":       func(l *domain.PluginLockFile) { l.Bindings[0].WeaponDigest = "" },
		"missing role-binding digest": func(l *domain.PluginLockFile) { l.Bindings[0].BindingDigest = "" },
		"missing role":                func(l *domain.PluginLockFile) { l.Bindings[0].Role = "" },
		"bare package id":             func(l *domain.PluginLockFile) { l.Bindings[0].InstalledInstanceID = "brainstorming" },
		"tampered entrypoint":         func(l *domain.PluginLockFile) { l.Bindings[0].Entrypoint = "other" },
		"tampered lock node":          func(l *domain.PluginLockFile) { l.Lock.Nodes[1].Digest = "sha256:other" },
		"missing lock node":           func(l *domain.PluginLockFile) { l.Lock.Nodes = l.Lock.Nodes[:2] },
	}
	for name, mutate := range cases {
		root := writeCustomValidationRoot(t, mutate)
		before, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
		require.NoError(t, err)

		_, err = BuildRoleInvocationPlan(root, "discovery")

		require.ErrorContains(t, err, "custom_binding_invalid", name)
		require.ErrorContains(t, err, "re-add the package", name)
		after, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
		require.NoError(t, readErr)
		require.Equal(t, before, after, "%s: the lock is never repaired", name)
	}
}

func TestBuildRoleInvocationPlan_NoBindingForSlot(t *testing.T) {
	root := writeValidationRoot(t, "")

	if _, err := BuildRoleInvocationPlan(root, "discovery"); err == nil {
		t.Fatal("expected an error when no binding is persisted for the slot")
	}
}

func TestBuildRoleInvocationPlan_ResolvesRankedBindingFromCatalog(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
    generation: 1
    status: active
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
`)

	plan, err := BuildRoleInvocationPlan(root, "discovery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Mode != "ranked" {
		t.Fatalf("plan.Mode = %q, want %q", plan.Mode, "ranked")
	}
	if plan.WeaponDigest != "sha256:1111111111111111111111111111111111111111111111111111111111111111" {
		t.Fatalf("plan.WeaponDigest = %q, want the catalog certification digest", plan.WeaponDigest)
	}
}

func TestBuildRoleInvocationPlan_EmbeddedRankedBindingNeedsNoRuntimeState(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
    generation: 1
    status: active
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
    ranked: true
    certification_digest: sha256:embedded
    runtime:
      kind: embedded
`)

	plan, err := BuildRoleInvocationPlan(root, "discovery")
	require.NoError(t, err)
	require.Equal(t, domain.RankedRuntimeEmbedded, plan.Runtime.Kind)
}

func TestBuildRoleInvocationPlan_RankedBindingNotInCatalogErrors(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)

	if _, err := BuildRoleInvocationPlan(root, "discovery"); err == nil {
		t.Fatal("expected an error when plugins/catalog.yaml is missing for a ranked binding")
	}
}

func TestBuildRoleInvocationPlan_ArchivistRequiresPreparedRuntime(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: refinement
    installed_instance_id: openspec-propose
    mode: ranked
    generation: 1
    status: active
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v2
providers:
  - id: openspec-propose
    canonical_role: archivist
    roles: [archivist]
    ranked: true
    certification_digest: sha256:runtime
    runtime:
      kind: openspec_root
      root: .strategist/openspec
      bootstrap: openspec init --profile core --tools codex
      healthcheck: openspec context --json
`)

	_, err := BuildRoleInvocationPlan(root, "refinement")
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime state is unavailable")

	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{
  "schema_version": "strategist-ranked-runtime/v1",
  "entries": [{"role":"archivist","slot":"refinement","provider":"openspec-propose","contract_digest":"sha256:runtime"}]
}`), 0o644))
	_, err = BuildRoleInvocationPlan(root, "refinement")
	require.ErrorContains(t, err, "strategist upgrade", "a legacy runtime record is never used")

	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{
  "schema_version": "strategist-ranked-runtime/v2",
  "entries": [{"role":"ranger","slot":"refinement","provider":"openspec-propose","contract_digest":"sha256:runtime"}]
}`), 0o644))
	_, err = BuildRoleInvocationPlan(root, "refinement")
	require.ErrorContains(t, err, "recorded for role", "a runtime recorded for another role does not match this binding")

	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{
  "schema_version": "strategist-ranked-runtime/v2",
  "entries": [{"role":"archivist","slot":"refinement","provider":"openspec-propose","contract_digest":"sha256:runtime"}]
}`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))

	plan, err := BuildRoleInvocationPlan(root, "refinement")
	require.NoError(t, err)
	require.Equal(t, ".strategist/openspec", plan.Runtime.Root)
}
