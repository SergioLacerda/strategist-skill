package rolevalidation

import "testing"

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
