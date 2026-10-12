package install

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/activeyaml"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/embeddedweapon"
	"github.com/SergioLacerda/strategist-skill/internal/install/onboarding"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/rankedbinding"
	"github.com/SergioLacerda/strategist-skill/internal/install/sidecar"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
	"github.com/SergioLacerda/strategist-skill/internal/install/wizard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSidecarScaffoldError_Formatting(t *testing.T) {
	t.Parallel()

	err := &sidecar.ScaffoldError{Code: "test_code", Detail: "test detail"}
	assert.Equal(t, "test_code: test detail", err.Error())
}

func TestEmbeddedWeaponSourceDigest_ReadError(t *testing.T) {
	t.Parallel()

	_, err := embeddedweapon.SourceDigestAt(t.TempDir())
	assert.Error(t, err)
}

func TestCustomProviderProjectionSkipsNonCustomAndEmptyModes(t *testing.T) {
	t.Parallel()

	catalog := plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{{ID: "known"}}}
	got, resolved, err := customprovider.CatalogWithCustomProviders(catalog, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, catalog, got)
	assert.Nil(t, resolved)
	assert.False(t, customprovider.NeedsCustomProjection(catalog, "", domain.SlotBindingModeCustom))
	assert.False(t, customprovider.NeedsCustomProjection(catalog, "known", domain.SlotBindingModeCustom))
	assert.False(t, customprovider.NeedsCustomProjection(catalog, "unknown", domain.SlotBindingModeRanked))
	assert.True(t, customprovider.NeedsCustomProjection(catalog, "unknown", domain.SlotBindingModeCustom))
	assert.Len(t, plugincatalog.CatalogResolverCandidates(catalog), 1)
}

func TestRankedRuntimeAndGovernanceNoOpBranches(t *testing.T) {
	// These are pure install-boundary branches; keep the executable lookup
	// override local so this test never depends on the host PATH.
	executable := rankedbinding.RankedExecutable{Name: "node", Prefix: []string{"script.js"}}
	assert.Equal(t, []string{"script.js", "--version"}, executable.Args([]string{"--version"}))

	originalFindHostNode := rankedbinding.FindHostNode
	originalHostNodeVersion := rankedbinding.HostNodeVersion
	_, err := originalHostNodeVersion(t.Context(), t.TempDir(), "node")
	require.Error(t, err)
	rankedbinding.FindHostNode = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() {
		rankedbinding.FindHostNode = originalFindHostNode
		rankedbinding.HostNodeVersion = originalHostNodeVersion
	})
	_, err = rankedbinding.ResolveHostNode(t.Context(), t.TempDir())
	require.Error(t, err)

	rankedbinding.FindHostNode = func(string) (string, error) { return "node", nil }
	rankedbinding.HostNodeVersion = func(context.Context, string, string) ([]byte, error) { return nil, os.ErrNotExist }
	_, err = rankedbinding.ResolveHostNode(t.Context(), t.TempDir())
	require.Error(t, err)
	policy := &domain.TrustPolicy{}
	require.Error(t, persistGovernanceState(t.TempDir(), policy, nil))
	grants := &domain.PermissionGrantFile{Grants: []domain.PermissionGrant{{ID: ""}}}
	assert.Error(t, persistGovernanceState(t.TempDir(), nil, grants))
}

func TestReplaceSidecar_ErrorPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "strategist.yaml")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o644))

	// Check=true returns sidecar_drift
	opts := sidecar.ScaffoldOptions{Check: true}
	_, err := sidecar.ReplaceSidecar(path, []byte("new"), opts)
	require.ErrorContains(t, err, "sidecar_drift")

	// Force=false returns sidecar_exists_differs
	opts = sidecar.ScaffoldOptions{Force: false}
	_, err = sidecar.ReplaceSidecar(path, []byte("new"), opts)
	require.ErrorContains(t, err, "sidecar_exists_differs")

	// Force=true overwrites sidecar
	opts = sidecar.ScaffoldOptions{Force: true}
	res, err := sidecar.ReplaceSidecar(path, []byte("new"), opts)
	require.NoError(t, err)
	assert.Equal(t, sidecar.SidecarOverwritten, res.Status)
}

func TestAtomicWriteFile_ErrorPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "a_file")
	require.NoError(t, os.WriteFile(filePath, []byte("block"), 0o644))

	// Trying to write into a parent that is a file (MkdirAll fails)
	badPath := filepath.Join(filePath, "sub", "test.txt")
	err := support.AtomicWriteFile(badPath, []byte("data"), 0o644)
	require.ErrorContains(t, err, "mkdir parent")
}

func TestSetRankedSlotMode_AllSlots(t *testing.T) {
	t.Parallel()

	wc := &domain.WizardConfig{}
	rankedbinding.SetRankedSlotMode(wc, "discovery")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.DiscoveryMode)

	rankedbinding.SetRankedSlotMode(wc, "refinement")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.RefinementMode)

	rankedbinding.SetRankedSlotMode(wc, "execution")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.ExecutionMode)

	rankedbinding.SetRankedSlotMode(wc, "unknown_slot")
}

func TestProviderNeedsRankedRuntime_Gaps(t *testing.T) {
	t.Parallel()

	catalog := plugincatalog.PluginCatalog{
		Providers: []plugincatalog.Provider{
			{
				ID:                  "prov-unranked",
				Ranked:              false,
				CertificationDigest: "digest",
				Runtime:             domain.WeaponRuntime{Kind: domain.RankedRuntimeExecutable},
			},
			{
				ID:                  "prov-nocert",
				Ranked:              true,
				CertificationDigest: "",
				Runtime:             domain.WeaponRuntime{Kind: domain.RankedRuntimeExecutable},
			},
			{
				ID:                  "prov-noruntime",
				Ranked:              true,
				CertificationDigest: "digest",
				Runtime:             domain.WeaponRuntime{Kind: domain.RankedRuntimeNone},
			},
		},
	}

	assert.False(t, rankedbinding.ProviderNeedsRankedRuntime(catalog, "prov-not-found"))
	assert.False(t, rankedbinding.ProviderNeedsRankedRuntime(catalog, "prov-unranked"))
	assert.False(t, rankedbinding.ProviderNeedsRankedRuntime(catalog, "prov-nocert"))
	assert.False(t, rankedbinding.ProviderNeedsRankedRuntime(catalog, "prov-noruntime"))
}

func TestWeaponPinCompare_Gaps(t *testing.T) {
	t.Parallel()

	// HashFileSHA256 file not found
	_, err := plugincatalog.HashFileSHA256("/nonexistent/file")
	require.ErrorContains(t, err, "hash file")

	// CompareResolvedDigest with invalid catalog path
	_, err = plugincatalog.CompareResolvedDigest("/nonexistent/catalog.yaml", "prov", "sha256:123")
	require.Error(t, err)
}

func TestNormalizeRoles(t *testing.T) {
	t.Parallel()

	input := []string{"ranger", "", "archivist", "ranger", "sniper"}
	out := plugincatalog.NormalizeRoles(input)
	assert.Equal(t, []string{"archivist", "ranger", "sniper"}, out)
}

func TestRankedRuntimeLegacyCleanup_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// removeLegacyNestedOpenSpecRoot with no config.yaml -> no-op
	err := rankedbinding.RemoveLegacyNestedOpenSpecRoot(dir)
	require.NoError(t, err)

	// removeLegacyNestedOpenSpecRoot with unexpected user content
	legacyDir := filepath.Join(dir, "openspec")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("config"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "user_custom.txt"), []byte("data"), 0o644))

	err = rankedbinding.RemoveLegacyNestedOpenSpecRoot(dir)
	require.ErrorContains(t, err, "legacy OpenSpec root contains user content")

	// legacyPathIfUnexpected
	allowed := rankedbinding.LegacyOpenSpecAllowedPaths()
	cand, err := rankedbinding.LegacyPathIfUnexpected(legacyDir, filepath.Join(legacyDir, "config.yaml"), allowed)
	require.NoError(t, err)
	assert.Empty(t, cand)

	cand, err = rankedbinding.LegacyPathIfUnexpected(legacyDir, filepath.Join(legacyDir, "user_custom.txt"), allowed)
	require.NoError(t, err)
	assert.Equal(t, "user_custom.txt", cand)
}

func TestRuntimeDefaultBlocksInstall(t *testing.T) {
	t.Parallel()

	assert.True(t, runtimeDefaultBlocksInstall(domain.RuntimeDecisionConflict))
	assert.True(t, runtimeDefaultBlocksInstall(domain.RuntimeDecisionUnknownManifest))
	assert.True(t, runtimeDefaultBlocksInstall(domain.RuntimeDecisionDowngrade))
	assert.False(t, runtimeDefaultBlocksInstall(domain.RuntimeDecisionKeepCurrent))
}

func TestWizardRoleValidation_Gaps(t *testing.T) {
	t.Parallel()

	// Resolution error in preview
	previewErr := onboarding.RoleProviderMigrationPreview{
		Entries: []onboarding.RoleProviderPreviewEntry{
			{Slot: "discovery", RoleName: "ranger", ResolutionError: "no matching provider"},
		},
	}
	err := wizard.ValidateWizardRoleBindings(previewErr)
	require.ErrorContains(t, err, "no matching provider")

	// Unresolved discovery entry
	previewUnresolved := onboarding.RoleProviderMigrationPreview{
		Entries: []onboarding.RoleProviderPreviewEntry{
			{Slot: "discovery", RoleName: "ranger"},
		},
	}
	err = wizard.ValidateWizardRoleBindings(previewUnresolved)
	require.ErrorContains(t, err, "no compatible weapon")

	// Persisted role binding mismatch
	lockFile := domain.PluginLockFile{
		Bindings: []domain.SlotBinding{
			{Slot: "discovery", InstalledInstanceID: "prov-a"},
		},
	}
	previewMismatch := onboarding.RoleProviderMigrationPreview{
		Entries: []onboarding.RoleProviderPreviewEntry{
			{
				Slot:     "discovery",
				RoleName: "ranger",
				Resolved: domain.ProviderBinding{
					Provider:      domain.ProviderContract{ID: "prov-b"},
					Compatibility: domain.CompatibilityResult{Compatible: true},
				},
			},
		},
	}
	err = wizard.ValidatePersistedRoleBindings(lockFile, previewMismatch)
	require.ErrorContains(t, err, "expected \"prov-b\"")

	// Count != 1 (count == 0)
	previewMissing := onboarding.RoleProviderMigrationPreview{
		Entries: []onboarding.RoleProviderPreviewEntry{
			{
				Slot:     "refinement",
				RoleName: "archivist",
				Resolved: domain.ProviderBinding{
					Provider:      domain.ProviderContract{ID: "prov-b"},
					Compatibility: domain.CompatibilityResult{Compatible: true},
				},
			},
		},
	}
	err = wizard.ValidatePersistedRoleBindings(lockFile, previewMissing)
	require.ErrorContains(t, err, "persisted binding count=0, expected 1")
}

func TestValidateGeneratedSidecar_Gaps(t *testing.T) {
	t.Parallel()

	// Invalid YAML
	err := sidecar.ValidateGeneratedSidecar([]byte("invalid: : : yaml"))
	require.ErrorContains(t, err, "sidecar does not parse")

	// Un-ingestible sidecar (missing mandatory fields like roles or supported_slots)
	err = sidecar.ValidateGeneratedSidecar([]byte("version: 1.0\nkind: atomic\n"))
	require.ErrorContains(t, err, "sidecar is not ingestible")
}

func TestCheckOpenSpecPin_Mismatch(t *testing.T) {
	t.Parallel()

	contract := domain.WeaponRuntime{Version: "2.0.0"}
	state := &domain.RankedRuntimeStateRuntime{
		Components: []domain.RankedRuntimeStateComponent{
			{Name: "openspec", Version: "1.0.0"},
		},
	}
	err := rankedbinding.CheckOpenSpecPin(contract, state)
	require.ErrorContains(t, err, "ranked_runtime_pin_mismatch")

	contractMatch := domain.WeaponRuntime{Version: "1.0.0"}
	require.NoError(t, rankedbinding.CheckOpenSpecPin(contractMatch, state))
}

func TestSidecarScaffoldInputs_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Missing SKILL.md
	_, _, err := sidecar.ReadScaffoldPackage(dir, "")
	require.ErrorContains(t, err, "skill_md_missing")

	// Missing version (SKILL.md exists without version and declared is empty)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---\nbody"), 0o644))
	_, _, err = sidecar.ReadScaffoldPackage(dir, "")
	require.ErrorContains(t, err, "version_missing")

	// Invalid provenance path
	_, _, err = sidecar.ReadScaffoldProvenance(filepath.Join(dir, "nonexistent.yaml"))
	require.ErrorContains(t, err, "provenance_invalid")

	// Invalid provenance YAML
	provPath := filepath.Join(dir, "bad_prov.yaml")
	require.NoError(t, os.WriteFile(provPath, []byte("invalid: : : yaml"), 0o644))
	_, _, err = sidecar.ReadScaffoldProvenance(provPath)
	require.ErrorContains(t, err, "provenance_invalid")

	// Unsupported runtime option
	opts := sidecar.ScaffoldOptions{Runtime: "unsupported_runtime"}
	_, _, err = sidecar.ResolveScaffoldRuntime(opts, sidecar.Provenance{})
	require.ErrorContains(t, err, "runtime_unsupported")

	// OpenSpec scaffold runtime missing runtime/ bundle
	opts.Runtime = domain.RankedRuntimeOpenSpecRoot
	opts.PackageDir = dir
	_, _, err = sidecar.ResolveScaffoldRuntime(opts, sidecar.Provenance{})
	require.ErrorContains(t, err, "runtime_bundle_missing")

	// OpenSpec scaffold runtime missing provenance runtime declaration
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "runtime"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "runtime.lock.yaml"), []byte("lock"), 0o644))
	_, _, err = sidecar.ResolveScaffoldRuntime(opts, sidecar.Provenance{})
	require.ErrorContains(t, err, "runtime_declaration_missing")

	// OpenSpec scaffold runtime with wrong kind
	wrongRuntime := &domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}
	_, _, err = sidecar.ResolveScaffoldRuntime(opts, sidecar.Provenance{Runtime: wrongRuntime})
	require.ErrorContains(t, err, "provenance_invalid")
}

func TestRollbackUpgrade_NonExistentBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := RollbackUpgrade(dir, "20261001-000000")
	require.ErrorContains(t, err, "no backup found")
}

func TestCustomPackageVersion_Gaps(t *testing.T) {
	t.Parallel()

	// Disagreeing versions
	_, err := customprovider.CustomPackageVersion("p-1", "1.0.0", "2.0.0")
	require.ErrorContains(t, err, "declares version \"1.0.0\" in SKILL.md and \"2.0.0\" in its sidecar")

	// Missing both versions
	_, err = customprovider.CustomPackageVersion("p-1", "", "")
	require.ErrorContains(t, err, "declares no version")

	// SKILL version present
	v, err := customprovider.CustomPackageVersion("p-1", "1.0.0", "")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", v)

	// Sidecar version present
	v, err = customprovider.CustomPackageVersion("p-1", "", "2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", v)
}

func TestContainsExact_Gaps(t *testing.T) {
	t.Parallel()

	assert.True(t, customprovider.ContainsExact([]string{"a", "b"}, "a"))
	assert.False(t, customprovider.ContainsExact([]string{"a", "b"}, "c"))
}

func TestFirstNonEmpty_Gaps(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "val", customprovider.FirstNonEmpty("", "  ", "val", "other"))
	assert.Empty(t, customprovider.FirstNonEmpty("", "  "))
}

func TestCommitStagedDirectory_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "installed_provider")
	files := map[string][]byte{"file1.txt": []byte("content1")}

	err := customprovider.CommitStagedDirectory(target, files)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(target, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content))
}

func TestActiveYAMLLanguage_Gaps(t *testing.T) {
	t.Parallel()

	// Invalid YAML
	_, err := activeyaml.NormalizeStandaloneLanguage([]byte("invalid: : : yaml"))
	require.ErrorContains(t, err, "parse active.yaml")

	// Invalid language type (number)
	err = activeyaml.ValidateStructuredLanguage(123)
	require.ErrorContains(t, err, "language must be a scalar legacy value or a complete role map")

	// Missing field in structured map
	err = activeyaml.ValidateStandaloneLanguage(map[string]any{"ui": "en", "docs": "en", "chat": "en"})
	require.ErrorContains(t, err, "language must define non-empty code fields")

	// Unsupported value in structured map
	err = activeyaml.ValidateStandaloneLanguage(map[string]any{"ui": "fr", "docs": "en", "chat": "en", "code": "en"})
	require.ErrorContains(t, err, "language.ui has unsupported value \"fr\"")

	// Scalar language replacement (line exists)
	dataWithLine := []byte("version: 1\nlanguage: pt-BR\n")
	norm, err := activeyaml.NormalizeStandaloneLanguage(dataWithLine)
	require.NoError(t, err)
	assert.Contains(t, string(norm), "ui: pt-BR")

	// Scalar language replacement (no language line)
	dataWithoutLine := []byte("version: 1\n")
	norm, err = activeyaml.NormalizeStandaloneLanguage(dataWithoutLine)
	require.NoError(t, err)
	assert.Contains(t, string(norm), "language:\n  ui: pt-BR")
}

func TestBindingRuntimeIdentity_Gaps(t *testing.T) {
	t.Parallel()

	// native_role
	pNative := plugincatalog.Provider{CompatibilitySource: "native_role"}
	rt, conn := plugincatalog.BindingRuntimeIdentity(pNative)
	assert.Equal(t, domain.RankedRuntimeEmbedded, rt.Kind)
	assert.Equal(t, "strategist-native-role", conn)

	// Embedded with empty execution mode -> defaults to PromptBridge
	pEmb := plugincatalog.Provider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}}
	rt, conn = plugincatalog.BindingRuntimeIdentity(pEmb)
	assert.Equal(t, domain.WeaponExecutionModePromptBridge, rt.ExecutionMode)
	assert.Equal(t, "strategist-embedded", conn)

	// OpenSpecRoot
	pOpenSpec := plugincatalog.Provider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeOpenSpecRoot}}
	_, conn = plugincatalog.BindingRuntimeIdentity(pOpenSpec)
	assert.Equal(t, "strategist-ranked-runtime", conn)

	// Host
	pHost := plugincatalog.Provider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost}}
	_, conn = plugincatalog.BindingRuntimeIdentity(pHost)
	assert.Equal(t, "strategist-host", conn)

	// Default/other
	pOther := plugincatalog.Provider{Runtime: domain.WeaponRuntime{Kind: "other"}}
	_, conn = plugincatalog.BindingRuntimeIdentity(pOther)
	assert.Equal(t, "current-runtime", conn)

	// bindingOrigin
	assert.Equal(t, string(domain.WeaponOriginCustom), plugincatalog.BindingOrigin(plugincatalog.Provider{CompatibilitySource: "external"}))
	assert.Equal(t, string(domain.WeaponOriginEmbedded), plugincatalog.BindingOrigin(plugincatalog.Provider{CompatibilitySource: "internal"}))
}

func TestUpgradeWriteSet_Gaps(t *testing.T) {
	t.Parallel()

	plan := UpgradePlan{
		Entries: []UpgradePlanEntry{
			{Path: "legacy.txt", State: domain.UpgradeLegacyLayout},
			{Path: "missing.txt", State: domain.UpgradeMissing},
			{Path: "auto.txt", State: domain.UpgradeAutoUpgrade},
			{Path: "custom.txt", State: domain.UpgradeCustomized},
			{Path: "managed.txt", State: domain.UpgradeManaged},
			{Path: "orphaned.txt", State: domain.UpgradeOrphaned},
		},
	}

	toWrite, toBackup, toRemove := upgradeWriteSet(plan, false)
	assert.Contains(t, toWrite, "missing.txt")
	assert.Contains(t, toWrite, "auto.txt")
	assert.NotContains(t, toWrite, "custom.txt")
	assert.Contains(t, toBackup, "auto.txt")
	assert.Contains(t, toRemove, "legacy.txt")

	toWriteForce, _, _ := upgradeWriteSet(plan, true)
	assert.Contains(t, toWriteForce, "custom.txt")
}

func TestPrintExcludedCandidatesTo_Gaps(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	excluded := []plugincatalog.ExcludedProviderOption{
		{
			ID: "prov-a",
			Reasons: []domain.CompatibilityReason{
				{Code: "role_mismatch", Detail: "does not support role"},
			},
		},
	}
	err := wizard.PrintExcludedCandidatesTo(&buf, excluded)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "prov-a: excluded — role_mismatch: does not support role")
}

func TestPreselectedRef_Gaps(t *testing.T) {
	t.Parallel()

	// Empty refs
	assert.Empty(t, plugincatalog.PreselectedRef(nil, nil, nil, false))

	// Candidate default = true
	candidates := []domain.ProviderContract{
		{ID: "p1", Version: "1.0", Default: false},
		{ID: "p2", Version: "1.0", Default: true},
	}
	refs := []string{"p1", "p2"}
	perID := map[string]int{"p1": 1, "p2": 1}
	assert.Equal(t, "p2", plugincatalog.PreselectedRef(candidates, refs, perID, false))

	// Prefer last when no candidate default
	candidatesNoDefault := []domain.ProviderContract{
		{ID: "p1", Version: "1.0"},
		{ID: "p2", Version: "2.0"},
	}
	assert.Equal(t, "p2", plugincatalog.PreselectedRef(candidatesNoDefault, refs, perID, true))
	assert.Equal(t, "p1", plugincatalog.PreselectedRef(candidatesNoDefault, refs, perID, false))
}
