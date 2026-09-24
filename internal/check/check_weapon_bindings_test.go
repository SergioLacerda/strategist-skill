package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeWeaponFixture writes a minimal <root>/skills/<id>/skill.yaml and,
// when roleYAML is non-empty, a matching <root>/roles/<role>.yaml.
func writeWeaponFixture(t *testing.T, root, skillID, canonicalRole, roleYAML string) {
	t.Helper()
	skillDir := filepath.Join(root, "skills", skillID)
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	body := "id: " + skillID + "\nrisk_score: write_analysis\n"
	if canonicalRole != "" {
		body += "specialization_taxonomy:\n  canonical_role: " + canonicalRole + "\n"
		if canonicalRole == "ranger" {
			body += "weapon_contract:\n  role_owner: ranger\n  participation: required\n  invocation_evidence: required\n  unavailable_behavior: role_invocation_failed\n  native_substitution: forbidden\n"
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(body), 0o644))

	if roleYAML != "" {
		rolesDir := filepath.Join(root, "roles")
		require.NoError(t, os.MkdirAll(rolesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(rolesDir, canonicalRole+".yaml"), []byte(roleYAML), 0o644))
	}
}

func writeDefaultRoleSlotMap(t *testing.T, root string) {
	t.Helper()
	rolesDir := filepath.Join(root, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
}

// writeFullRoster writes valid fixtures for both ADR-0035 DEC-001 permanent
// pairings, so a test can focus on one additional skill/scenario without
// verifyEmbeddedWeaponRoster's presence check adding unrelated
// missing-pairing rows to the result.
func writeFullRoster(t *testing.T, root string) {
	t.Helper()
	writeWeaponFixture(t, root, "brainstorming", "ranger", "role: ranger\nslot: discovery\n")
	writeWeaponFixture(t, root, "openspec-propose", "archivist", "role: archivist\nslot: refinement\n")
}

// findBinding returns the binding for skillID, if present.
func findBinding(bindings []weaponBinding, skillID string) (weaponBinding, bool) {
	for _, b := range bindings {
		if b.SkillID == skillID {
			return b, true
		}
	}
	return weaponBinding{}, false
}

func TestVerifyEmbeddedWeaponBindings_NoSkillsDir_ReportsMissingRoster(t *testing.T) {
	t.Parallel()
	bindings, err := verifyEmbeddedWeaponBindings(t.TempDir())
	require.NoError(t, err)
	require.Len(t, bindings, len(embeddedWeaponRoster))
	for _, b := range bindings {
		assert.False(t, b.OK)
		assert.Contains(t, b.Reason, "embedded weapon missing")
	}
}

func TestVerifyEmbeddedWeaponBindings_SkipsSkillsWithoutCanonicalRole(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWeaponFixture(t, root, "sdd-ask", "", "")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	// sdd-ask itself declares no canonical_role and is skipped; the rows
	// present are the roster's missing-pairing rows, not sdd-ask.
	require.Len(t, bindings, len(embeddedWeaponRoster))
	_, found := findBinding(bindings, "sdd-ask")
	assert.False(t, found)
}

func TestVerifyEmbeddedWeaponBindings_SkipsAuxiliaryTools(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWeaponFixture(t, root, "writing-plans", "auxiliary", "")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	_, found := findBinding(bindings, "writing-plans")
	assert.False(t, found, "auxiliary tools must not be treated as mission role bindings")
}

func TestVerifyEmbeddedWeaponBindings_ValidPairing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	writeFullRoster(t, root)

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "brainstorming")
	require.True(t, found)
	assert.Equal(t, "ranger", b.CanonicalRole)
	assert.Equal(t, "discovery", b.Slot)
	assert.True(t, b.OK)
	assert.Empty(t, b.Reason)
	// The full permanent roster is satisfied — no failing rows at all.
	for _, binding := range bindings {
		assert.True(t, binding.OK, "unexpected failing binding: %+v", binding)
	}
}

func TestSelectedDiscoveryWeaponRequiresBoundaryContract(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	writeWeaponFixture(t, root, "brainstorming", "ranger", "role: ranger\nslot: discovery\n")

	// Remove the helper's valid contract so the failure is isolated to the
	// discovery Weapon boundary rather than role or slot resolution.
	skillPath := filepath.Join(root, "skills", "brainstorming", "skill.yaml")
	require.NoError(t, os.WriteFile(skillPath, []byte("id: brainstorming\nrisk_score: write_analysis\nspecialization_taxonomy:\n  canonical_role: ranger\n"), 0o644))

	err := validateSelectedDiscoveryWeaponContract(root, "brainstorming")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discovery weapon contract missing")
}

func TestValidateWeaponBoundaryRejectsOwnerMismatch(t *testing.T) {
	err := validateWeaponBoundary("discovery", "ranger", domain.WeaponContract{
		RoleOwner:           "archivist",
		Participation:       "required",
		InvocationEvidence:  "required",
		UnavailableBehavior: "role_invocation_failed",
		NativeSubstitution:  "forbidden",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match role")
}

func TestVerifyEmbeddedWeaponBindings_MissingRoleFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	writeWeaponFixture(t, root, "openspec-propose", "archivist", "") // no role file written

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "openspec-propose")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "role file missing")
	// brainstorming's directory is absent altogether in this fixture — the
	// roster check must report it as missing, not merge it into the row above.
	missing, found := findBinding(bindings, "brainstorming")
	require.True(t, found)
	assert.False(t, missing.OK)
	assert.Contains(t, missing.Reason, "embedded weapon missing")
}

func TestVerifyEmbeddedWeaponBindings_RoleSlotMapMismatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root) // refinement: archivist
	// A skill that claims canonical_role=ranger for a role file whose own
	// slot is "refinement" — roles/default.yaml maps refinement to
	// archivist, not ranger, so this must fail as a mismatch.
	writeWeaponFixture(t, root, "mismatched-skill", "ranger", "role: ranger\nslot: refinement\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "mismatched-skill")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "roles/default.yaml maps slot")
	// Neither roster pairing's directory exists in this fixture.
	for _, expected := range embeddedWeaponRoster {
		rb, found := findBinding(bindings, expected.SkillID)
		require.True(t, found)
		assert.False(t, rb.OK)
	}
}

func TestVerifyEmbeddedWeaponBindings_InvalidRoleConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	// slot missing entirely from the role file — RoleConfig.Validate fails.
	writeWeaponFixture(t, root, "broken-skill", "archivist", "role: archivist\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "broken-skill")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "role config invalid")
}

func TestVerifyEmbeddedWeaponBindings_MalformedSkillYAML(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "broken")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte("id: [unterminated\n"), 0o644))

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "broken")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "skill.yaml invalid")
}

func TestVerifyEmbeddedWeaponBindings_SkipsNonDirectoryEntry(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	// A stray file directly under skills/ (not a skill directory) must be
	// skipped rather than treated as a skill ID.
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "README.md"), []byte("not a skill"), 0o644))

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	_, found := findBinding(bindings, "README.md")
	assert.False(t, found)
}

func TestVerifyEmbeddedWeaponBindings_RoleFileMalformedYAML(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	writeWeaponFixture(t, root, "broken-role-yaml", "archivist", "role: [unterminated")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "broken-role-yaml")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "role file malformed YAML")
}

func TestVerifyEmbeddedWeaponBindings_MissingRoleSlotMap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// No roles/default.yaml written at all.
	writeWeaponFixture(t, root, "brainstorming", "ranger", "role: ranger\nslot: discovery\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "brainstorming")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "roles/default.yaml unreadable")
}

func TestVerifyEmbeddedWeaponRoster_AllPresent(t *testing.T) {
	t.Parallel()
	bindings := []weaponBinding{
		{SkillID: "brainstorming", OK: true},
		{SkillID: "openspec-propose", OK: true},
	}
	assert.Empty(t, verifyEmbeddedWeaponRoster(bindings))
}

func TestVerifyEmbeddedWeaponRoster_ReportsMissingPairing(t *testing.T) {
	t.Parallel()
	bindings := []weaponBinding{
		{SkillID: "brainstorming", OK: true},
	}
	missing := verifyEmbeddedWeaponRoster(bindings)
	require.Len(t, missing, 1)
	assert.Equal(t, "openspec-propose", missing[0].SkillID)
	assert.Equal(t, "archivist", missing[0].CanonicalRole)
	assert.False(t, missing[0].OK)
	assert.Contains(t, missing[0].Reason, "embedded weapon missing")
	assert.Contains(t, missing[0].Reason, "DEC-001")
}

func TestVerifyEmbeddedWeaponRoster_PresentButFailingIsNotDoubleReported(t *testing.T) {
	t.Parallel()
	// A roster pairing whose skill directory exists but fails some other
	// check (e.g. missing role file) is already reported by that check —
	// presence, not validity, is verifyEmbeddedWeaponRoster's only concern.
	bindings := []weaponBinding{
		{SkillID: "brainstorming", OK: false, Reason: "role file missing"},
		{SkillID: "openspec-propose", OK: true},
	}
	assert.Empty(t, verifyEmbeddedWeaponRoster(bindings))
}

func TestVerifyEmbeddedWeaponRoster_EmptyBindings(t *testing.T) {
	t.Parallel()
	missing := verifyEmbeddedWeaponRoster(nil)
	require.Len(t, missing, len(embeddedWeaponRoster))
}

func TestWeaponBindingErrors(t *testing.T) {
	t.Parallel()
	errs := weaponBindingErrors([]weaponBinding{
		{SkillID: "ok-skill", OK: true},
		{SkillID: "bad-skill", OK: false, Reason: "role file missing"},
	})
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "bad-skill")
	assert.Contains(t, errs[0], "role file missing")
}
