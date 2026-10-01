package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSidecarScaffoldError_Formatting(t *testing.T) {
	t.Parallel()

	err := &SidecarScaffoldError{Code: "test_code", Detail: "test detail"}
	assert.Equal(t, "test_code: test detail", err.Error())
}

func TestReplaceSidecar_ErrorPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "strategist.yaml")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o644))

	// Check=true returns sidecar_drift
	opts := SidecarScaffoldOptions{Check: true}
	_, err := replaceSidecar(path, []byte("new"), opts)
	require.ErrorContains(t, err, "sidecar_drift")

	// Force=false returns sidecar_exists_differs
	opts = SidecarScaffoldOptions{Force: false}
	_, err = replaceSidecar(path, []byte("new"), opts)
	require.ErrorContains(t, err, "sidecar_exists_differs")

	// Force=true overwrites sidecar
	opts = SidecarScaffoldOptions{Force: true}
	res, err := replaceSidecar(path, []byte("new"), opts)
	require.NoError(t, err)
	assert.Equal(t, SidecarOverwritten, res.Status)
}

func TestAtomicWriteFile_ErrorPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "a_file")
	require.NoError(t, os.WriteFile(filePath, []byte("block"), 0o644))

	// Trying to write into a parent that is a file (MkdirAll fails)
	badPath := filepath.Join(filePath, "sub", "test.txt")
	err := atomicWriteFile(badPath, []byte("data"), 0o644)
	require.ErrorContains(t, err, "mkdir parent")
}

func TestSetRankedSlotMode_AllSlots(t *testing.T) {
	t.Parallel()

	wc := &domain.WizardConfig{}
	setRankedSlotMode(wc, "discovery")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.DiscoveryMode)

	setRankedSlotMode(wc, "refinement")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.RefinementMode)

	setRankedSlotMode(wc, "execution")
	assert.Equal(t, domain.SlotBindingModeRanked, wc.ExecutionMode)

	setRankedSlotMode(wc, "unknown_slot")
}

func TestProviderNeedsRankedRuntime_Gaps(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{
		Providers: []pluginCatalogProvider{
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

	assert.False(t, providerNeedsRankedRuntime(catalog, "prov-not-found"))
	assert.False(t, providerNeedsRankedRuntime(catalog, "prov-unranked"))
	assert.False(t, providerNeedsRankedRuntime(catalog, "prov-nocert"))
	assert.False(t, providerNeedsRankedRuntime(catalog, "prov-noruntime"))
}

func TestWeaponPinCompare_Gaps(t *testing.T) {
	t.Parallel()

	// HashFileSHA256 file not found
	_, err := HashFileSHA256("/nonexistent/file")
	require.ErrorContains(t, err, "hash file")

	// CompareResolvedDigest with invalid catalog path
	_, err = CompareResolvedDigest("/nonexistent/catalog.yaml", "prov", "sha256:123")
	require.Error(t, err)
}

func TestNormalizeRoles(t *testing.T) {
	t.Parallel()

	input := []string{"ranger", "", "archivist", "ranger", "sniper"}
	out := normalizeRoles(input)
	assert.Equal(t, []string{"archivist", "ranger", "sniper"}, out)
}

func TestRankedRuntimeLegacyCleanup_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// removeLegacyNestedOpenSpecRoot with no config.yaml -> no-op
	err := removeLegacyNestedOpenSpecRoot(dir)
	require.NoError(t, err)

	// removeLegacyNestedOpenSpecRoot with unexpected user content
	legacyDir := filepath.Join(dir, "openspec")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("config"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "user_custom.txt"), []byte("data"), 0o644))

	err = removeLegacyNestedOpenSpecRoot(dir)
	require.ErrorContains(t, err, "legacy OpenSpec root contains user content")

	// legacyPathIfUnexpected
	allowed := legacyOpenSpecAllowedPaths()
	cand, err := legacyPathIfUnexpected(legacyDir, filepath.Join(legacyDir, "config.yaml"), allowed)
	require.NoError(t, err)
	assert.Empty(t, cand)

	cand, err = legacyPathIfUnexpected(legacyDir, filepath.Join(legacyDir, "user_custom.txt"), allowed)
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
	previewErr := RoleProviderMigrationPreview{
		Entries: []RoleProviderPreviewEntry{
			{Slot: "discovery", RoleName: "ranger", ResolutionError: "no matching provider"},
		},
	}
	err := validateWizardRoleBindings(previewErr)
	require.ErrorContains(t, err, "no matching provider")

	// Unresolved discovery entry
	previewUnresolved := RoleProviderMigrationPreview{
		Entries: []RoleProviderPreviewEntry{
			{Slot: "discovery", RoleName: "ranger"},
		},
	}
	err = validateWizardRoleBindings(previewUnresolved)
	require.ErrorContains(t, err, "no compatible weapon")

	// Persisted role binding mismatch
	lockFile := domain.PluginLockFile{
		Bindings: []domain.SlotBinding{
			{Slot: "discovery", InstalledInstanceID: "prov-a"},
		},
	}
	previewMismatch := RoleProviderMigrationPreview{
		Entries: []RoleProviderPreviewEntry{
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
	err = validatePersistedRoleBindings(lockFile, previewMismatch)
	require.ErrorContains(t, err, "expected \"prov-b\"")

	// Count != 1 (count == 0)
	previewMissing := RoleProviderMigrationPreview{
		Entries: []RoleProviderPreviewEntry{
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
	err = validatePersistedRoleBindings(lockFile, previewMissing)
	require.ErrorContains(t, err, "persisted binding count=0, expected 1")
}

func TestValidateGeneratedSidecar_Gaps(t *testing.T) {
	t.Parallel()

	// Invalid YAML
	err := validateGeneratedSidecar([]byte("invalid: : : yaml"))
	require.ErrorContains(t, err, "sidecar does not parse")

	// Un-ingestible sidecar (missing mandatory fields like roles or supported_slots)
	err = validateGeneratedSidecar([]byte("version: 1.0\nkind: atomic\n"))
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
	err := checkOpenSpecPin(contract, state)
	require.ErrorContains(t, err, "ranked_runtime_pin_mismatch")

	contractMatch := domain.WeaponRuntime{Version: "1.0.0"}
	require.NoError(t, checkOpenSpecPin(contractMatch, state))
}

func TestSidecarScaffoldInputs_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Missing SKILL.md
	_, _, err := readScaffoldPackage(dir, "")
	require.ErrorContains(t, err, "skill_md_missing")

	// Missing version (SKILL.md exists without version and declared is empty)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---\nbody"), 0o644))
	_, _, err = readScaffoldPackage(dir, "")
	require.ErrorContains(t, err, "version_missing")

	// Invalid provenance path
	_, _, err = readScaffoldProvenance(filepath.Join(dir, "nonexistent.yaml"))
	require.ErrorContains(t, err, "provenance_invalid")

	// Invalid provenance YAML
	provPath := filepath.Join(dir, "bad_prov.yaml")
	require.NoError(t, os.WriteFile(provPath, []byte("invalid: : : yaml"), 0o644))
	_, _, err = readScaffoldProvenance(provPath)
	require.ErrorContains(t, err, "provenance_invalid")

	// Unsupported runtime option
	opts := SidecarScaffoldOptions{Runtime: "unsupported_runtime"}
	_, _, err = resolveScaffoldRuntime(opts, sidecarProvenance{})
	require.ErrorContains(t, err, "runtime_unsupported")

	// OpenSpec scaffold runtime missing runtime/ bundle
	opts.Runtime = domain.RankedRuntimeOpenSpecRoot
	opts.PackageDir = dir
	_, _, err = resolveScaffoldRuntime(opts, sidecarProvenance{})
	require.ErrorContains(t, err, "runtime_bundle_missing")

	// OpenSpec scaffold runtime missing provenance runtime declaration
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "runtime"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "runtime.lock.yaml"), []byte("lock"), 0o644))
	_, _, err = resolveScaffoldRuntime(opts, sidecarProvenance{})
	require.ErrorContains(t, err, "runtime_declaration_missing")

	// OpenSpec scaffold runtime with wrong kind
	wrongRuntime := &domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}
	_, _, err = resolveScaffoldRuntime(opts, sidecarProvenance{Runtime: wrongRuntime})
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
	_, err := customPackageVersion("p-1", "1.0.0", "2.0.0")
	require.ErrorContains(t, err, "declares version \"1.0.0\" in SKILL.md and \"2.0.0\" in its sidecar")

	// Missing both versions
	_, err = customPackageVersion("p-1", "", "")
	require.ErrorContains(t, err, "declares no version")

	// SKILL version present
	v, err := customPackageVersion("p-1", "1.0.0", "")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", v)

	// Sidecar version present
	v, err = customPackageVersion("p-1", "", "2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", v)
}

func TestContainsExact_Gaps(t *testing.T) {
	t.Parallel()

	assert.True(t, containsExact([]string{"a", "b"}, "a"))
	assert.False(t, containsExact([]string{"a", "b"}, "c"))
}

func TestFirstNonEmpty_Gaps(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "val", firstNonEmpty("", "  ", "val", "other"))
	assert.Empty(t, firstNonEmpty("", "  "))
}

func TestCommitStagedDirectory_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "installed_provider")
	files := map[string][]byte{"file1.txt": []byte("content1")}

	err := commitStagedDirectory(target, files)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(target, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content))
}

func TestActiveYAMLLanguage_Gaps(t *testing.T) {
	t.Parallel()

	// Invalid YAML
	_, err := normalizeStandaloneLanguage([]byte("invalid: : : yaml"))
	require.ErrorContains(t, err, "parse active.yaml")

	// Invalid language type (number)
	err = validateStructuredLanguage(123)
	require.ErrorContains(t, err, "language must be a scalar legacy value or a complete role map")

	// Missing field in structured map
	err = validateStandaloneLanguage(map[string]any{"ui": "en", "docs": "en", "chat": "en"})
	require.ErrorContains(t, err, "language must define non-empty code fields")

	// Unsupported value in structured map
	err = validateStandaloneLanguage(map[string]any{"ui": "fr", "docs": "en", "chat": "en", "code": "en"})
	require.ErrorContains(t, err, "language.ui has unsupported value \"fr\"")

	// Scalar language replacement (line exists)
	dataWithLine := []byte("version: 1\nlanguage: pt-BR\n")
	norm, err := normalizeStandaloneLanguage(dataWithLine)
	require.NoError(t, err)
	assert.Contains(t, string(norm), "ui: pt-BR")

	// Scalar language replacement (no language line)
	dataWithoutLine := []byte("version: 1\n")
	norm, err = normalizeStandaloneLanguage(dataWithoutLine)
	require.NoError(t, err)
	assert.Contains(t, string(norm), "language:\n  ui: pt-BR")
}

func TestBindingRuntimeIdentity_Gaps(t *testing.T) {
	t.Parallel()

	// native_role
	pNative := pluginCatalogProvider{CompatibilitySource: "native_role"}
	rt, conn := bindingRuntimeIdentity(pNative)
	assert.Equal(t, domain.RankedRuntimeEmbedded, rt.Kind)
	assert.Equal(t, "strategist-native-role", conn)

	// Embedded with empty execution mode -> defaults to PromptBridge
	pEmb := pluginCatalogProvider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}}
	rt, conn = bindingRuntimeIdentity(pEmb)
	assert.Equal(t, domain.WeaponExecutionModePromptBridge, rt.ExecutionMode)
	assert.Equal(t, "strategist-embedded", conn)

	// OpenSpecRoot
	pOpenSpec := pluginCatalogProvider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeOpenSpecRoot}}
	_, conn = bindingRuntimeIdentity(pOpenSpec)
	assert.Equal(t, "strategist-ranked-runtime", conn)

	// Host
	pHost := pluginCatalogProvider{Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost}}
	_, conn = bindingRuntimeIdentity(pHost)
	assert.Equal(t, "strategist-host", conn)

	// Default/other
	pOther := pluginCatalogProvider{Runtime: domain.WeaponRuntime{Kind: "other"}}
	_, conn = bindingRuntimeIdentity(pOther)
	assert.Equal(t, "current-runtime", conn)

	// bindingOrigin
	assert.Equal(t, string(domain.WeaponOriginCustom), bindingOrigin(pluginCatalogProvider{CompatibilitySource: "external"}))
	assert.Equal(t, string(domain.WeaponOriginEmbedded), bindingOrigin(pluginCatalogProvider{CompatibilitySource: "internal"}))
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
	excluded := []excludedProviderOption{
		{
			id: "prov-a",
			reasons: []domain.CompatibilityReason{
				{Code: "role_mismatch", Detail: "does not support role"},
			},
		},
	}
	err := printExcludedCandidatesTo(&buf, excluded)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "prov-a: excluded — role_mismatch: does not support role")
}

func TestPreselectedRef_Gaps(t *testing.T) {
	t.Parallel()

	// Empty refs
	assert.Empty(t, preselectedRef(nil, nil, nil, false))

	// Candidate default = true
	candidates := []domain.ProviderContract{
		{ID: "p1", Version: "1.0", Default: false},
		{ID: "p2", Version: "1.0", Default: true},
	}
	refs := []string{"p1", "p2"}
	perID := map[string]int{"p1": 1, "p2": 1}
	assert.Equal(t, "p2", preselectedRef(candidates, refs, perID, false))

	// Prefer last when no candidate default
	candidatesNoDefault := []domain.ProviderContract{
		{ID: "p1", Version: "1.0"},
		{ID: "p2", Version: "2.0"},
	}
	assert.Equal(t, "p2", preselectedRef(candidatesNoDefault, refs, perID, true))
	assert.Equal(t, "p1", preselectedRef(candidatesNoDefault, refs, perID, false))
}
