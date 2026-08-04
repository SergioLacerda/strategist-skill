//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrimaryContractsDoNotHardcodeAnalysisAsArtifactRoot(t *testing.T) {
	t.Parallel()

	// These normative runtime contracts and persona templates must use <base_path> or {base_path}
	// for artifact paths, not a hardcoded .analysis/ root.
	type fileCheck struct {
		path      string
		forbidden []string
	}
	checks := []fileCheck{
		{
			path:      filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "09-response.md"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
		{
			path:      filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "09-response.md"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
		{
			path:      filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
		{
			path:      filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
	}
	for _, c := range checks {
		content := readFile(t, c.path)
		for _, bad := range c.forbidden {
			if strings.Contains(content, bad) {
				t.Fatalf("%s hardcodes .analysis/ as artifact root (found %q); use <base_path>/{base_path} instead", c.path, bad)
			}
		}
	}
}

func TestRuntimeContractsDoNotReferenceSourceTreeSchemas(t *testing.T) {
	t.Parallel()

	// Runtime-facing contracts must reference .strategist/schemas/ (runtime tree),
	// never a dot-less strategist/schemas/ path.
	paths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "02-intake.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "learning-buffer.yaml"),
	}
	for _, path := range paths {
		content := readFile(t, path)
		if strings.Contains(content, "`strategist/schemas/") || strings.Contains(content, " strategist/schemas/") {
			t.Fatalf("%s references dot-less schema path strategist/schemas/; use .strategist/schemas/ instead", path)
		}
	}
}

func TestCanonicalProviderPathIsSkillsSubdirectory(t *testing.T) {
	t.Parallel()

	// Guard canonical runtime path in normative surfaces — no root-level .strategist/<provider>/ lookup.
	mustContainCanonical := []string{
		filepath.Join(repoRoot(t), "docs", "strategist-concepts.md"),
		filepath.Join(repoRoot(t), "internal", "domain", "types.go"),
	}
	for _, path := range mustContainCanonical {
		content := readFile(t, path)
		if !strings.Contains(content, "skills/<provider>/skill.yaml") {
			t.Fatalf("%s missing canonical provider path skills/<provider>/skill.yaml", path)
		}
	}

	// Guard that no normative doc instructs users to inspect the legacy root-level path.
	mustNotContainLegacy := []string{
		filepath.Join(repoRoot(t), "README.md"),
		filepath.Join(repoRoot(t), "docs", "strategist-concepts.md"),
		filepath.Join(repoRoot(t), "docs", "onboarding", "readme-en.md"),
		filepath.Join(repoRoot(t), "internal", "domain", "types.go"),
	}
	for _, path := range mustNotContainLegacy {
		content := readFile(t, path)
		if strings.Contains(content, ".strategist/<provider>/skill.yaml") {
			t.Fatalf("%s still references legacy root-level provider path .strategist/<provider>/skill.yaml", path)
		}
	}
}

func TestSourceInternalSkillsDirMirrorsRuntimeLayout(t *testing.T) {
	t.Parallel()

	// Two-tree world (W7a, Option B): internal/embed/defaults/ is the single
	// authoring+generation source; internal skills are authored under internal_skills/
	// (skills/ holds external provider capability mirrors, a different namespace).
	sourceDir := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills")
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		t.Fatalf("internal/embed/defaults/internal_skills/ must exist as the authoring directory for internal skills")
	}

	// The retired authoring mirror must stay deleted. Only a directory at this
	// path is the actual retired mirror reappearing — an unrelated file (e.g. a
	// stray local build of the strategist binary) must not trip this check.
	retiredTree := filepath.Join(repoRoot(t), "strategist")
	if info, err := os.Stat(retiredTree); err == nil && info.IsDir() {
		t.Fatalf("strategist/ still exists — the authoring mirror was retired (W7a); author in internal/embed/defaults/")
	}

	// The manual sync step must stay deleted with it (a prose mention in comments is
	// fine; a target definition is not).
	makefile := readMakefileSystem(t, repoRoot(t))
	if strings.Contains(makefile, "\nsync-embed:") {
		t.Fatalf("Makefile still defines the sync-embed target — the two-tree world has no manual sync step")
	}
}

func TestRemedationLabelPointsToCanonicalSkillsDir(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "progress-contract.yaml"),
	}
	for _, path := range files {
		content := readFile(t, path)
		if strings.Contains(content, "action=check_skill_root") {
			t.Fatalf("%s still uses stale action=check_skill_root remediation label", path)
		}
	}
}

func TestPrimaryRuntimeContractsDoNotHardcodeAnalysisAsInvariant(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md"),
	}

	for _, path := range files {
		content := readFile(t, path)
		if strings.Contains(content, "the invariant Strategist workspace root is .analysis/") {
			t.Fatalf("%s hardcodes .analysis/ as invariant runtime root", path)
		}
	}
}

func TestProtocolReferencesUseRuntimeTreeNotSourceTree(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md")
	content := readFile(t, path)

	for _, needle := range []string{
		".strategist/",
		"base_path",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing runtime path marker %q", path, needle)
		}
	}
}

func TestStrategistSkillDeclaresRuntimeAndWorkspacePathContracts(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md")
	content := readFile(t, path)

	for _, needle := range []string{
		"`internal/embed/defaults/` — the single authoring and generation source",
		"`.strategist/` — runtime instance",
		"only operational read target",
		"`base_path`",
		"not a hardcoded `.analysis/`",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing %q", path, needle)
		}
	}
}

// TestNormativeRuntimeFilesMirrorEmbeddedDefaults was removed in W7a (Option B):
// with strategist/ retired, internal/embed/defaults/ IS the canonical source, so
// source↔embed parity is true by construction. Runtime parity is covered below.

// TestNormativeRuntimeFilesMirrorEmbeddedDefaults was removed in W7a (Option B):
// with strategist/ retired, internal/embed/defaults/ IS the canonical source, so
// source↔embed parity is true by construction. Runtime parity is covered below.

func TestLocalRuntimeMirrorsCanonicalNormativeFilesWhenPresent(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	runtimeRoot := filepath.Join(root, ".strategist")
	if _, err := os.Stat(runtimeRoot); os.IsNotExist(err) {
		t.Skip(".strategist runtime not installed in this workspace")
	}

	for _, rel := range normativeRuntimeFiles() {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			sourcePath := filepath.Join(root, "internal", "embed", "defaults", filepath.FromSlash(rel))
			runtimePath := filepath.Join(runtimeRoot, filepath.FromSlash(rel))

			source := readFile(t, sourcePath)
			runtime := readFile(t, runtimePath)
			if source != runtime {
				t.Fatalf("%s drifted from canonical source %s; reinstall/recompile runtime from internal/embed/defaults", runtimePath, sourcePath)
			}
		})
	}
}

// TestNoRootLevelProviderLookupInCode ensures resolver-facing code never references
// a root-level .strategist/<provider>/skill.yaml without the skills/ subdirectory.
// This guards the canonical runtime layout contract: all external provider manifests
// must resolve from .strategist/skills/<provider>/skill.yaml.
func TestNoRootLevelProviderLookupInCode(t *testing.T) {
	t.Parallel()

	// These files contain the resolver logic; they must use the skills/ subdirectory.
	files := []string{
		filepath.Join(repoRoot(t), "cmd", "strategist", "check.go"),
		filepath.Join(repoRoot(t), "internal", "dojo", "checker.go"),
	}

	// Forbidden: join(root, provider, "skill.yaml") without the "skills" segment.
	// Canonical: join(root, "skills", provider, "skill.yaml").
	forbidden := []string{
		`filepath.Join(root, provider,`,
		`filepath.Join(strategistDir, provider,`,
	}

	for _, path := range files {
		content := readFile(t, path)
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s contains root-level provider lookup %q — must use skills/<provider>/skill.yaml", path, pattern)
			}
		}
	}
}

// TestInternalSkillsSourceBoundarySymmetric ensures the source-authoring tree and
// embed directory both contain an internal_skills/ folder, confirming the direct
// mapping without a semantic remap in the build pipeline.

// TestInternalSkillsSourceBoundarySymmetric ensures the source-authoring tree and
// embed directory both contain an internal_skills/ folder, confirming the direct
// mapping without a semantic remap in the build pipeline.
func TestInternalSkillsSourceBoundarySymmetric(t *testing.T) {
	t.Parallel()

	dirs := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Fatalf("internal_skills directory missing: %s — source/embed boundary broken", dir)
		}
	}
}

// TestPreflightContractNoFallbackChain verifies that preflight test contracts
// describe the actual single-path resolution rule with no .claude/skills/ fallback.

// TestPreflightContractNoFallbackChain verifies that preflight test contracts
// describe the actual single-path resolution rule with no .claude/skills/ fallback.
func TestPreflightContractNoFallbackChain(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(isolatedStrategistDir(t), "contracts", "tests", "preflight.test.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "tests", "preflight.test.yaml"),
	}

	forbidden := ".claude/skills/"

	for _, path := range files {
		content := readFile(t, path)
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still references stale fallback %q in slot_resolution_order invariant", path, forbidden)
		}
	}
}

func TestPreflightProviderManifestIsSlotAuthority(t *testing.T) {
	t.Parallel()

	contractFiles := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml"),
	}

	for _, path := range contractFiles {
		content := readFile(t, path)
		for _, needle := range []string{
			"skill_root/skills/<provider>/skill.yaml",
			"Standalone SKILL.md style",
			"creative-first instructions are not provider-invalid conditions",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider authority preflight term %q", path, needle)
			}
		}
	}

	testFiles := []string{
		filepath.Join(isolatedStrategistDir(t), "contracts", "tests", "preflight.test.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "tests", "preflight.test.yaml"),
	}

	for _, path := range testFiles {
		content := readFile(t, path)
		for _, needle := range []string{
			"provider_manifest_is_slot_authority",
			"brainstorming_creative_not_blocked_by_standalone_creative_first",
			"subtype=creative",
			"diagnostic_subtype_bypasses_external_weapon_for_native_ranger",
			"weapon=internal_skills/ranger",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider authority preflight test term %q", path, needle)
			}
		}
	}
}

// TestOutputProfilesSourceMirrorsEmbeddedDefaults verifies strategist/output-profiles/ (the
// authoring source restored by mission 2026-07-22-output-profiles-undocumented-embed-exception)
// stays byte-identical to its internal/embed/defaults/output-profiles/ mirror. Mirrors the
// TestNormativeRuntimeFilesMirrorEmbeddedDefaults pattern for this content class, which has no
// fixed file list (domain.NormativeRuntimeDefaultPaths() does not cover it) so both trees are
// walked and their relative file sets compared before comparing content.
func TestOutputProfilesSourceMirrorsEmbeddedDefaults(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	sourceDir := filepath.Join(root, "internal", "embed", "defaults", "output-profiles")
	embedDir := filepath.Join(root, "internal", "embed", "defaults", "output-profiles")

	sourceFiles := relativeFileSet(t, sourceDir)
	embedFiles := relativeFileSet(t, embedDir)

	for rel := range sourceFiles {
		if !embedFiles[rel] {
			t.Fatalf("strategist/output-profiles/%s has no embedded counterpart at internal/embed/defaults/output-profiles/%s; run make sync-embed", rel, rel)
		}
	}
	for rel := range embedFiles {
		if !sourceFiles[rel] {
			t.Fatalf("internal/embed/defaults/output-profiles/%s has no source counterpart at strategist/output-profiles/%s; content must be authored under strategist/ and synced, not hand-written directly into the embed mirror", rel, rel)
		}
	}

	for rel := range sourceFiles {
		sourcePath := filepath.Join(sourceDir, filepath.FromSlash(rel))
		embedPath := filepath.Join(embedDir, filepath.FromSlash(rel))
		source := readFile(t, sourcePath)
		embedded := readFile(t, embedPath)
		if source != embedded {
			t.Fatalf("%s drifted from embedded default %s; run make sync-embed after changing strategist/output-profiles/", sourcePath, embedPath)
		}
	}
}

// TestSyncEmbedSchemaExclusionsStayIntentional guards duplicate schema files that used
// to be excluded from sync-embed. Duplicate source/embed schemas must mirror exactly;
// embed-only schemas must remain explicitly absent from the strategist/ authoring tree.
// TestSyncEmbedSchemaExclusionsStayIntentional was removed in W7a (Option B):
// sync-embed no longer exists, so its schema/role exclusion policy is moot —
// embed-only artifacts live directly in internal/embed/defaults/ like everything else.

// TestDiscoveryAndRefinementContractsRequireDocsLanguage guards mission
// 2026-07-24-language-config-not-reflected's root cause 1 fix: Ranger and Archivist must author
// documentation artifacts in active.language.docs, independent of the conversation's language.
// Without this instruction, refined packages drift to whatever language the surrounding chat
// happens to use (observed directly: two missions refined mid-session in Portuguese despite
// docs: en).
