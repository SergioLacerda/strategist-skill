package rolevalidation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRoleInvocationPlan_ResolvesRoleFromSlotMapAndLock(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
`)

	plan, err := BuildRoleInvocationPlan(root, "discovery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Role != "ranger" {
		t.Fatalf("plan.Role = %q, want %q", plan.Role, "ranger")
	}
	if plan.Slot != "discovery" {
		t.Fatalf("plan.Slot = %q, want %q", plan.Slot, "discovery")
	}
	if plan.WeaponID != "brainstorming" {
		t.Fatalf("plan.WeaponID = %q, want %q", plan.WeaponID, "brainstorming")
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
schema_version: strategist-plugin-catalog/v1
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
schema_version: strategist-plugin-catalog/v1
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
