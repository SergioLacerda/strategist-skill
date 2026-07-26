//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	Scenario      string `yaml:"scenario"`
	ExpectedEvent string `yaml:"expected_event"`
}

func testDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(testDir(t), "..", ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func normativeRuntimeFiles() []string {
	return domain.NormativeRuntimeDefaultPaths()
}

func readFixture(t *testing.T, path string) fixture {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var f fixture
	if err := yaml.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return f
}

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
		filepath.Join(repoRoot(t), "cmd", "strategist", "initiative.go"),
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

	// The retired authoring mirror must stay deleted.
	retiredTree := filepath.Join(repoRoot(t), "strategist")
	if _, err := os.Stat(retiredTree); !os.IsNotExist(err) {
		t.Fatalf("strategist/ still exists — the authoring mirror was retired (W7a); author in internal/embed/defaults/")
	}

	// The manual sync step must stay deleted with it (a prose mention in comments is
	// fine; a target definition is not).
	makefile := readFile(t, filepath.Join(repoRoot(t), "Makefile"))
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

func TestStrategistPipelineHasNoImplementationShortRoute(t *testing.T) {
	t.Parallel()

	for _, rel := range []string{
		filepath.Join("internal", "embed", "defaults", "skill.yaml"),
	} {
		path := filepath.Join(repoRoot(t), rel)
		content := readFile(t, path)
		for _, forbidden := range []string{
			"implementation_context_validation",
			"skip full discovery/refinement expansion",
			"Implementation Short Route",
			"implementation_intent",
		} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s still contains pipeline bypass marker %q", path, forbidden)
			}
		}
	}
}

func TestOpportunityBypassFixtureAlignedWithForbiddenBehaviorsSpec(t *testing.T) {
	t.Parallel()
	featurePath := filepath.Join(testDir(t), "specs", "forbidden-behaviors.feature")
	fixturePath := filepath.Join(testDir(t), "fixtures", "side-quest-bypass.yaml")

	feature := readFile(t, featurePath)
	fixture := readFixture(t, fixturePath)

	if !strings.Contains(feature, "skip_opportunity_attack_routine") {
		t.Fatalf("%s missing scenario skip_opportunity_attack_routine", featurePath)
	}
	if !strings.Contains(feature, "opportunity_attack") {
		t.Fatalf("%s missing opportunity_attack routine reference", featurePath)
	}
	if fixture.Scenario != "skip_opportunity_attack_routine" {
		t.Fatalf("%s scenario must be skip_opportunity_attack_routine, got: %q", fixturePath, fixture.Scenario)
	}
	if !strings.Contains(fixture.ExpectedEvent, "drift") {
		t.Fatalf("%s expected_event must contain drift, got: %q", fixturePath, fixture.ExpectedEvent)
	}
}

func TestTriageGateFixtureAlignedWithTokenEconomySpec(t *testing.T) {
	t.Parallel()
	featurePath := filepath.Join(testDir(t), "specs", "token-economy.feature")
	fixturePath := filepath.Join(testDir(t), "fixtures", "triage-gate-blocked.yaml")

	feature := readFile(t, featurePath)
	fixture := readFixture(t, fixturePath)

	if !strings.Contains(feature, "triage_gate_blocked") {
		t.Fatalf("%s missing scenario triage_gate_blocked", featurePath)
	}
	if fixture.Scenario != "triage_gate_blocked" {
		t.Fatalf("%s scenario must be triage_gate_blocked, got: %q", fixturePath, fixture.Scenario)
	}
	if !strings.Contains(fixture.ExpectedEvent, "blocked") {
		t.Fatalf("%s expected_event must contain blocked, got: %q", fixturePath, fixture.ExpectedEvent)
	}
}

func TestApprovalBypassFixtureAlignedWithApprovalGateSpec(t *testing.T) {
	t.Parallel()
	featurePath := filepath.Join(testDir(t), "specs", "approval-gate.feature")
	fixturePath := filepath.Join(testDir(t), "fixtures", "approval-bypass.yaml")

	feature := readFile(t, featurePath)
	fixture := readFixture(t, fixturePath)

	if !strings.Contains(feature, "phase=approval_gate status=pending") {
		t.Fatalf("%s missing approval_gate pending assertion", featurePath)
	}
	for _, term := range []string{"analysis_accepted", "revision_requested", "rejected"} {
		if !strings.Contains(feature, term) {
			t.Fatalf("%s missing review gate response %q", featurePath, term)
		}
	}
	if fixture.Scenario != "approval_bypass" {
		t.Fatalf("%s scenario must be approval_bypass, got: %q", fixturePath, fixture.Scenario)
	}
	if !strings.Contains(fixture.ExpectedEvent, "approval_bypass") || !strings.Contains(fixture.ExpectedEvent, "blocked") {
		t.Fatalf("%s expected_event must block approval_bypass, got: %q", fixturePath, fixture.ExpectedEvent)
	}
}

func TestSingleTargetSweepBypassFixtureAlignedWithForbiddenBehaviorsSpec(t *testing.T) {
	t.Parallel()
	featurePath := filepath.Join(testDir(t), "specs", "forbidden-behaviors.feature")
	fixturePath := filepath.Join(testDir(t), "fixtures", "single-target-sweep-bypass.yaml")

	feature := readFile(t, featurePath)
	fixture := readFixture(t, fixturePath)

	if !strings.Contains(feature, "single_target_sweep_bypass") {
		t.Fatalf("%s missing scenario single_target_sweep_bypass", featurePath)
	}
	if !strings.Contains(feature, "opportunity_sweep_failed") {
		t.Fatalf("%s missing opportunity_sweep_failed assertion", featurePath)
	}
	if fixture.Scenario != "single_target_sweep_bypass" {
		t.Fatalf("%s scenario must be single_target_sweep_bypass, got: %q", fixturePath, fixture.Scenario)
	}
	if !strings.Contains(fixture.ExpectedEvent, "opportunity_sweep_failed") || !strings.Contains(fixture.ExpectedEvent, "blocked") {
		t.Fatalf("%s expected_event must block on opportunity_sweep_failed, got: %q", fixturePath, fixture.ExpectedEvent)
	}
}

func TestPolicyGuardrailsSpecAlignedWithFixture(t *testing.T) {
	t.Parallel()
	featurePath := filepath.Join(testDir(t), "specs", "policy-guardrails.feature")
	fixturePath := filepath.Join(testDir(t), "fixtures", "policy-guardrails-e2e.yaml")

	feature := readFile(t, featurePath)
	fixture := readFixture(t, fixturePath)

	if !strings.Contains(feature, "quick_draw append is blocked") {
		t.Fatalf("%s missing quick_draw guardrail scenario", featurePath)
	}
	if !strings.Contains(feature, "opportunity execution is skipped") {
		t.Fatalf("%s missing opportunity execution policy scenario", featurePath)
	}
	if !strings.Contains(feature, "phase=policy_eval status=blocked") {
		t.Fatalf("%s missing canonical policy_eval blocked event", featurePath)
	}
	if fixture.Scenario != "policy_guardrails_e2e" {
		t.Fatalf("%s scenario must be policy_guardrails_e2e, got: %q", fixturePath, fixture.Scenario)
	}
	if !strings.Contains(fixture.ExpectedEvent, "phase=policy_eval status=blocked") ||
		!strings.Contains(fixture.ExpectedEvent, "transition_group=documentation_materialization") {
		t.Fatalf("%s expected_event must include blocked policy_eval for documentation_materialization, got: %q", fixturePath, fixture.ExpectedEvent)
	}
}

func TestSkillProfileResolutionContractPresent(t *testing.T) {
	t.Parallel()
	skillPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml")
	skill := readFile(t, skillPath)

	mustContain := []string{
		"name: profile",
		"default: local",
		"allowed: [local]",
		"name: mode",
		"name: speed",
		"default: balanced",
		"values: [fast, balanced, full]",
		"speed_mode:",
	}
	for _, needle := range mustContain {
		if !strings.Contains(skill, needle) {
			t.Fatalf("%s must contain %q", skillPath, needle)
		}
	}
}

func TestPersonasExposeMandatoryRuntimeEvidenceKeys(t *testing.T) {
	t.Parallel()
	pragmaticPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "pragmatic.yaml")
	epicPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml")

	for _, p := range []string{pragmaticPath, epicPath} {
		content := readFile(t, p)
		if !strings.Contains(content, "pipeline_starting:") {
			t.Fatalf("%s missing pipeline_starting key", p)
		}
		if !strings.Contains(content, "compliance_summary:") {
			t.Fatalf("%s missing compliance_summary key", p)
		}
		if !strings.Contains(content, "mission_metrics:") {
			t.Fatalf("%s missing mission_metrics key", p)
		}
		for _, key := range []string{
			"profile_mode",
			"profile_source_path",
			"active_yaml_path",
			"persona_resolved",
			"reason",
		} {
			if !strings.Contains(content, key) {
				t.Fatalf("%s missing runtime diagnostic key %q", p, key)
			}
		}
	}
}

func TestPersonasExposeVisibleComplianceAndNextAction(t *testing.T) {
	t.Parallel()
	pragmaticPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "pragmatic.yaml")
	epicPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml")

	for _, p := range []string{pragmaticPath, epicPath} {
		content := readFile(t, p)
		if !strings.Contains(content, "content_by_lang:") {
			t.Fatalf("%s missing content_by_lang", p)
		}
		if !strings.Contains(content, "compliance_summary: >") {
			t.Fatalf("%s missing visible compliance_summary template", p)
		}
		if !strings.Contains(content, "next_action={next_action}") &&
			!strings.Contains(content, "Next action: {next_action}") &&
			!strings.Contains(content, "Próxima ação: {next_action}") {
			t.Fatalf("%s missing next_action in mission_complete template", p)
		}
	}
}

func TestPragmaticPersonaUsesDistinctPhaseLabels(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "pragmatic.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"discovery: analysis",
		"refinement: refinement",
		"execution: materialization",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing %q", path, needle)
		}
	}
}

func TestEmitTaxonomyMandatoryVisibilityLevels(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "output-profiles", "emit-taxonomy.yaml")
	content := readFile(t, path)

	assertions := []string{
		"opportunity_attack_done:     INFO",
		"treasure_chest_found:        INFO",
		"compliance_summary:          DEBUG",
		"pipeline_starting:           DEBUG",
		"mission_metrics:             DEBUG",
		"Speed policy bridge:",
		"- balanced -> default mission flow, default profile threshold",
	}
	for _, line := range assertions {
		if !strings.Contains(content, line) {
			t.Fatalf("%s missing expected line %q", path, line)
		}
	}
}

func TestBootstrapContractDefinesInvalidLocalProfileErrorCode(t *testing.T) {
	t.Parallel()
	bootstrapPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "bootstrap.yaml")
	content := readFile(t, bootstrapPath)
	if !strings.Contains(content, "invalid_local_profile") {
		t.Fatalf("%s missing error_condition code \"invalid_local_profile\"", bootstrapPath)
	}
}

func TestStrategistResponseContractIsExternalized(t *testing.T) {
	t.Parallel()
	protocolPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md")
	skillPath := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md")

	protocol := readFile(t, protocolPath)
	skill := readFile(t, skillPath)

	if !strings.Contains(protocol, "## Response Contract") {
		t.Fatalf("%s missing Response Contract section", protocolPath)
	}
	if !strings.Contains(protocol, "## Compliance Summary") {
		t.Fatalf("%s missing Compliance Summary section", protocolPath)
	}
	if !strings.Contains(protocol, "## Mission Result") {
		t.Fatalf("%s missing Mission Result section", protocolPath)
	}
	if !strings.Contains(skill, "See `.strategist/protocol.md#response-contract`") {
		t.Fatalf("%s must reference .strategist/protocol.md#response-contract", skillPath)
	}
	if strings.Contains(skill, "## 11. Mission Result") {
		t.Fatalf("%s still embeds Mission Result inline", skillPath)
	}
}

func TestSkillDefinesMissingProfileDiagnosticsBlock(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "missing_profile_diagnostics") {
			t.Fatalf("%s missing compliance enforcement for \"missing_profile_diagnostics\"", p)
		}
	}
}

func TestSkillDefinesPersonaRenderMismatchForbiddenBehavior(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "persona_render_mismatch") {
			t.Fatalf("%s missing forbidden_behavior \"persona_render_mismatch\"", p)
		}
	}
}

func TestApprovalGateContractDefinesEmitOnShow(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "emit_on_show") {
			t.Fatalf("%s missing \"emit_on_show\" — approval gate must log when shown to user", p)
		}
	}
}

func TestContextEnrichmentDefinesTreasureChestLoadedNone(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "treasure_chest_loaded none") {
			t.Fatalf("%s missing \"treasure_chest_loaded none\" emit for empty chest list", p)
		}
	}
}

func TestComplianceSummaryDefinesPhaseCounters(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "compliance-summary.yaml"),
	} {
		content := readFile(t, p)
		for _, needle := range []string{"expected_phases", "executed_phases"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q counter field in compliance emit_format", p, needle)
			}
		}
	}
}

// --- Provider discovery conformance tests ---

// providerBootstrapFiles lists all provider bootstrap surfaces that must declare
// Strategist runtime discovery semantics.
func providerBootstrapFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	return []string{
		filepath.Join(root, ".codex", "commands.md"),
		filepath.Join(root, ".claude", "claude-instructions.md"),
		filepath.Join(root, ".antigravity", "antigravity-instructions.md"),
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "GEMINI.md"),
	}
}

func TestProviderBootstrapsRequireStrategistRuntimeFiles(t *testing.T) {
	t.Parallel()

	for _, path := range providerBootstrapFiles(t) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			// Provider bootstrap files are machine-local, generated by external
			// governance tooling (sdd governance generate) — never git-tracked.
			// Conformance is asserted when present; absence is an environment
			// state, not a repo defect (deep analysis 2026-07-26 follow-up).
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not generated in this workspace — run sdd governance generate", path)
			}
			content := readFile(t, path)
			for _, needle := range []string{
				".strategist/SKILL.md",
				".strategist/skill.yaml",
			} {
				if !strings.Contains(content, needle) {
					t.Fatalf("%s missing required Strategist runtime file reference %q", path, needle)
				}
			}
		})
	}
}

func TestProviderBootstrapsDoNotTreatSddAsStrategistRuntime(t *testing.T) {
	t.Parallel()

	// These phrases indicate a provider is treating .sdd/ as the Strategist
	// runtime source rather than keeping governance and runtime separate.
	forbidden := []string{
		"load .sdd/ as Strategist runtime",
		"resolve Strategist from .sdd/",
		"sdd/ provides the Strategist pipeline",
	}

	for _, path := range providerBootstrapFiles(t) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			// Provider bootstrap files are machine-local, generated by external
			// governance tooling (sdd governance generate) — never git-tracked.
			// Conformance is asserted when present; absence is an environment
			// state, not a repo defect (deep analysis 2026-07-26 follow-up).
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not generated in this workspace — run sdd governance generate", path)
			}
			content := readFile(t, path)
			for _, bad := range forbidden {
				if strings.Contains(content, bad) {
					t.Fatalf("%s contains forbidden phrase %q — .sdd/ must not be treated as Strategist runtime", path, bad)
				}
			}
		})
	}
}

func TestProviderBootstrapsDoNotLoadFromSourceTree(t *testing.T) {
	t.Parallel()

	// Provider bootstrap files must not instruct loading from the source tree
	// path "strategist/" (without the leading dot).
	for _, path := range providerBootstrapFiles(t) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			// Provider bootstrap files are machine-local, generated by external
			// governance tooling (sdd governance generate) — never git-tracked.
			// Conformance is asserted when present; absence is an environment
			// state, not a repo defect (deep analysis 2026-07-26 follow-up).
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not generated in this workspace — run sdd governance generate", path)
			}
			content := readFile(t, path)
			if strings.Contains(content, "strategist/SKILL.md") &&
				!strings.Contains(content, ".strategist/SKILL.md") {
				t.Fatalf("%s references source-tree strategist/SKILL.md without leading dot", path)
			}
		})
	}
}

func TestCommonDiscoveryContractExists(t *testing.T) {
	t.Parallel()

	// The common discovery contract must exist in both the source tree and the
	// embedded defaults so it is available after install.
	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "provider-discovery.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("common discovery contract missing at %s: %v", path, err)
		}
	}
}

func TestCommonDiscoveryContractDefinesMandatoryFields(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "provider-discovery.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			".strategist/SKILL.md",
			".strategist/skill.yaml",
			"error=not_installed",
			"Forbidden Behaviors",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing required discovery contract field %q", path, needle)
			}
		}
	}
}

func TestMissionMetricsSignalPresent(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "progress-contract.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "intake.yaml"),
	}
	for _, path := range files {
		content := readFile(t, path)
		needles := []string{
			"mission_metrics:",
			"t_start_to_intake_ms",
			"t_intake_to_ranger_ms",
			"total_wall_time_ms",
			"tokens_in",
			"tokens_out",
			"lines_emitted",
		}
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q", path, needle)
			}
		}
	}
}

func TestE2EFeatureFilesCoverHappyPathContracts(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		filepath.Join(testDir(t), "specs", "e2e-happy-path.feature"): []string{
			"Ranger consults treasure chests",
			"Ranger records scope observations",
			"Archivist runs opportunity attack",
			"approval gate",
		},
		filepath.Join(testDir(t), "specs", "e2e-approval-gate.feature"): []string{
			"mission result is documentation_applied",
			"analysis_delivered",
			"Sniper is not invoked",
		},
		filepath.Join(testDir(t), "specs", "e2e-treasure-chests.feature"): []string{
			"treasure chests",
			"treasure_chests=none",
		},
		filepath.Join(testDir(t), "specs", "e2e-opportunity-attack.feature"): []string{
			"only Archivist performs Opportunity Attack",
			"Critical Hit remains responsible",
			"approval gate",
		},
		filepath.Join(testDir(t), "specs", "e2e-install-compile.feature"): []string{
			"customized active.yaml",
			"--force",
			"check-stale reports the compiled config as fresh",
		},
	}

	for path, needles := range files {
		content := readFile(t, path)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q", path, needle)
			}
		}
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
		filepath.Join(repoRoot(t), "cmd", "strategist", "initiative.go"),
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
			"brainstorming_diagnostic_not_blocked_by_standalone_creative_first",
			"subtype=diagnostic",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider authority preflight test term %q", path, needle)
			}
		}
	}
}

func TestSniperWriteScopeIsWorkspaceAndDocs(t *testing.T) {
	t.Parallel()

	// The execution contract must declare that Sniper write scope is workspace files
	// and documentation files only — code mutation is always forbidden.
	files := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"),
	}

	requiredPhrases := []string{
		"workspace",
		"documentation",
		"code mutation",
	}
	forbidden := []string{
		"execution_mode",
		"git_persistence_mode",
		"plan_only",
		"apply_workspace",
	}

	for _, path := range files {
		content := readFile(t, path)
		for _, phrase := range requiredPhrases {
			if !strings.Contains(content, phrase) {
				t.Errorf("%s missing required write-scope phrase %q", path, phrase)
			}
		}
		for _, term := range forbidden {
			if strings.Contains(content, term) {
				t.Errorf("%s still references removed policy term %q", path, term)
			}
		}
	}
}

func TestActiveYAMLTemplatesDoNotContainLegacyPolicyFields(t *testing.T) {
	t.Parallel()

	// All active.yaml templates in the embed tree must not contain legacy execution_mode
	// or git_persistence_mode fields — they were removed in the scope simplification.
	templateDirs := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates"),
	}

	forbidden := []string{"execution_mode", "git_persistence_mode"}

	for _, dir := range templateDirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			content := readFile(t, path)
			for _, term := range forbidden {
				if strings.Contains(content, term) {
					t.Errorf("template %s contains legacy policy field %q", path, term)
				}
			}
		}
	}
}

func TestStandaloneTemplatesDefaultExecutionToSniper(t *testing.T) {
	t.Parallel()

	// Standalone active.yaml templates must ship with sniper as the default execution
	// provider. This ensures both silent install and wizard defaults align.
	standaloneTemplates := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "epic-standalone.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "pragmatic-standalone.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "epic-standalone.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "pragmatic-standalone.yaml"),
	}

	for _, path := range standaloneTemplates {
		content := readFile(t, path)
		if !strings.Contains(content, "execution: sniper") {
			t.Errorf("template %s must default to execution: sniper", path)
		}
		if strings.Contains(content, "execution: openspec-apply-change") {
			t.Errorf("template %s must not default to execution: openspec-apply-change — use sniper instead", path)
		}
	}
}

// assertNoToken fails if the file at path contains token.
func assertNoToken(t *testing.T, path, token string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(data), token) {
		t.Fatalf("%s must not contain %q", path, token)
	}
}

func TestSchemaFilesDoNotContainLegacyPlanOnly(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "mission-result.schema.yaml"), "plan_only")
	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "outcome-entry.schema.yaml"), "plan_only")
	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "progress-contract.yaml"), "plan_only")
	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "mission-result.schema.yaml"), "plan_only")
	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "outcome-entry.schema.yaml"), "plan_only")
	assertNoToken(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "progress-contract.yaml"), "plan_only")
}

func TestApprovalGateContractUsesReviewGateSemantics(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	paths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"),
	}

	for _, path := range paths {
		content := readFile(t, path)
		assertNoToken(t, path, "plan_only")
		for _, needle := range []string{"analysis_accepted", "revision_requested", "rejected"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing review gate response %q", path, needle)
			}
		}
	}
}

func TestSniperIsDocumentationMaterializerNotExecutionSkill(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	paths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "sniper", "SKILL.md"),
	}
	forbidden := []string{
		"Execution Skill",
		"execute the approved refined package",
		"execution_done",
	}
	required := []string{
		"documentation materialization",
		"documentation_applied",
		"documentation_targets",
		"Git mutating commands are forbidden",
	}

	for _, path := range paths {
		content := readFile(t, path)
		for _, bad := range forbidden {
			if strings.Contains(content, bad) {
				t.Fatalf("%s must not contain %q", path, bad)
			}
		}
		for _, needle := range required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing required documentation phrase %q", path, needle)
			}
		}
	}
}

func TestDocumentationPipelineDoesNotContainLegacyExecutionTerms(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	forbidden := []string{"apply_workspace", "execution_mode", "git_persistence_mode", "CanExecute"}
	paths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"),
		filepath.Join(root, "internal", "embed", "defaults", "skill.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "protocol.md"),
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"),
		filepath.Join(root, "internal", "embed", "defaults", "protocol.md"),
	}

	for _, path := range paths {
		for _, term := range forbidden {
			assertNoToken(t, path, term)
		}
	}
}

// TestStrategistSourceTreeEnglishOnly scans strategist/ for Portuguese prose markers.
//
// Allowlisted paths and fields that legitimately contain non-English data:
//   - strategist/schemas/intake.schema.yaml — user input aliases (não pode quebrar, etc.)
//     are intent-matching tokens, not prose (design non-goal: preserve Portuguese input tokens).
//   - strategist/contracts/machine/quick-draw.yaml — pt-BR bucket name list (data).
//   - strategist/contracts/machine/adr.yaml — pt-BR section name list (data).
//   - strategist/contracts/narrative/07-adr.md — pt-BR language mapping (data).
//   - strategist/contracts/adr.md — docs: pt-BR language mapping (data).
//   - strategist/contracts/machine/critical-hit.yaml — reserved input tokens with inline doc.
func TestStrategistSourceTreeEnglishOnly(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	strategistDir := filepath.Join(root, "internal", "embed", "defaults")

	// Portuguese prose markers that must NOT appear in canonical strategist/ files.
	// These are not language-code data — they are prose fragments in Portuguese.
	forbiddenProse := []string{
		"Não processe",
		"execute antes de qualquer coisa",
		"execute exatamente nessa ordem",
		"fluxo completo",
		"fluxo direto",
		"orquestração em lote",
		"análises capturadas",
		"Caminho para arquivo",
		"processar todas",
		"pacotes de negócio",
		"camada pura",
		"ponto de wiring",
		"Nunca executar",
		"Nunca ler de",
		"Nunca pular",
		"Nunca invocar",
		"Missão: {mission_id}",
		"Perfil: {profile}",
		"Arquivista →",
		"aprovação concedida",
		"reconhecimento concluído",
		"implementação concluída",
		"Autorizar Sniper",
		"Aguardando confirmação",
		"Commitar?",
	}

	err := filepath.WalkDir(strategistDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".yaml" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(data)
		for _, marker := range forbiddenProse {
			if strings.Contains(content, marker) {
				t.Errorf("strategist/ prose violation: %s contains Portuguese prose marker %q", path, marker)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk strategist/: %v", err)
	}
}

func TestRoleInvocationFailureContractPresent(t *testing.T) {
	t.Parallel()

	// agent-protocol.md templates must declare role_invocation_failed as a named internal
	// error state and forbid direct simulation of role work.
	templatePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
	}
	for _, path := range templatePaths {
		content := readFile(t, path)
		if !strings.Contains(content, "role_invocation_failed") {
			t.Fatalf("%s missing error state \"role_invocation_failed\"", path)
		}
		if !strings.Contains(content, "simulate role work") {
			t.Fatalf("%s missing NEVER DO rule about simulating role work", path)
		}
	}

	// drift-patterns.yaml files must include a role_invocation_failed pattern.
	driftPaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	}
	for _, path := range driftPaths {
		content := readFile(t, path)
		if !strings.Contains(content, "id: role_invocation_failed") {
			t.Fatalf("%s missing drift pattern \"role_invocation_failed\"", path)
		}
		if !strings.Contains(content, "role_invocation_failed") {
			t.Fatalf("%s direct_execution correction must reference role_invocation_failed", path)
		}
	}
}

// TestStrategistNoLegacyExecutionTerminology scans strategist/ for forbidden legacy terms
// that were replaced by documentation-materialization semantics.
func TestStrategistNoLegacyExecutionTerminology(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	strategistDir := filepath.Join(root, "internal", "embed", "defaults")

	// Forbidden legacy terms. See design doc: 2026-06-25-strategist-english-canonical-i18n-design.md.
	forbidden := []string{
		"plan_only",
		"apply_workspace",
		"execution_mode",
		"git_persistence_mode",
		"CanExecute",
		"Commit?",
		"Implement?",
		"Authorize Sniper?",
	}

	err := filepath.WalkDir(strategistDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".yaml" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(data)
		for _, term := range forbidden {
			if strings.Contains(content, term) {
				t.Errorf("legacy terminology violation: %s contains forbidden term %q", path, term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk strategist/: %v", err)
	}
}

// TestPersonaFilesHaveNoPtBRContentBlocks verifies that persona YAML files
// contain no pt-BR localized content_by_lang blocks. Localized strings live
// in internal/i18n/strategist_messages.go.
func TestPersonaFilesHaveNoPtBRContentBlocks(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	personaPaths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "personas", "epic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "epic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
	}

	forbidden := []string{
		"phase_announcements:\n  pt-BR:",
		"content_by_lang:\n  pt-BR:",
		"  pt-BR:\n    intake_summary",
		"  pt-BR:\n    ranger_start",
	}

	for _, path := range personaPaths {
		content := readFile(t, path)
		for _, marker := range forbidden {
			if strings.Contains(content, marker) {
				t.Errorf("%s must not contain pt-BR localized content block: found %q", path, marker)
			}
		}
		// Must have English canonical content
		if !strings.Contains(content, "content_by_lang:\n  en:") {
			t.Errorf("%s must contain content_by_lang.en canonical block", path)
		}
	}
}

// TestPersonasUseApprovalGateSemantics verifies personas use approval_gate_prompt
// and do not contain the legacy approval_prompt key (renamed during i18n cleanup).
func TestPersonasUseApprovalGateSemantics(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	paths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "personas", "epic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "epic.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
	}

	for _, path := range paths {
		content := readFile(t, path)
		if strings.Contains(content, "approval_prompt:") {
			t.Errorf("%s must not contain legacy key 'approval_prompt:' — use 'approval_gate_prompt:' instead", path)
		}
		if !strings.Contains(content, "approval_gate_prompt:") {
			t.Errorf("%s must define 'approval_gate_prompt:' key", path)
		}
	}
}

// TestRoleInvocationFailuresInSKILLMD verifies that SKILL.md declares role
// invocation failures as internal skill errors, not caller delegation gates.
func TestRoleInvocationFailuresInSKILLMD(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md")
	content := readFile(t, path)

	required := []string{
		"Role Invocation Failures",
		"role_invocation_failed",
		"strategist check",
		"slot_provider_not_found",
		"slot_risk_mismatch",
		"role_provider_invalid",
	}
	for _, needle := range required {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing required role invocation term %q", path, needle)
		}
	}
}

// TestProtocolDeclaresRoleSimulationForbidden verifies that protocol.md explicitly
// forbids simulating role work (performing slot work in the Strategist shell).
func TestProtocolDeclaresRoleSimulationForbidden(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md")
	content := readFile(t, path)

	for _, needle := range []string{
		"role_invocation_failed",
		"simulate role work",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing required role invocation rule %q", path, needle)
		}
	}
}

// TestDriftPatternsIncludeApprovalAndRuntimePatterns verifies both source and
// embedded drift-patterns files declare the two new patterns added in the
// direct-execution drift correction.
func TestDriftPatternsIncludeApprovalAndRuntimePatterns(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	}
	required := []string{
		"id: approval_design_confused_with_approval_gate",
		"id: runtime_source_confusion",
	}
	for _, path := range paths {
		content := readFile(t, path)
		for _, needle := range required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing drift pattern %q", path, needle)
			}
		}
	}
}

// TestPreflightContractDeclaresRoleInvocationFailure verifies the preflight
// machine contract declares role_invocation_failed as a named error condition.
func TestPreflightContractDeclaresRoleInvocationFailure(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml")
	content := readFile(t, path)

	for _, needle := range []string{
		"code: role_invocation_failed",
		"slot=<slot_name>",
		"provider=<provider_id>",
		"strategist check",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing role invocation failure term %q", path, needle)
		}
	}
}

// TestSkillDeclaresRoleLockForParentAgent verifies SKILL.md contains the
// hard role-lock sentence that forbids the parent agent from solving the
// user's task directly instead of invoking configured slot providers.
func TestSkillDeclaresRoleLockForParentAgent(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"When this skill is invoked, the parent agent MUST NOT solve the user's task directly.",
			"role_invocation_failed",
			"replace a missing provider with its own built-in capabilities",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing role-lock term %q", path, needle)
			}
		}
	}
}

// TestAgentProtocolDeclaresParentAgentBoundary verifies agent-protocol.md
// states that direct phase work without invoking the configured provider is
// direct_execution drift, even when the parent agent's own answer is correct.
func TestAgentProtocolDeclaresParentAgentBoundary(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"PARENT AGENT BOUNDARY",
			"`direct_execution` drift, even if the output is correct",
			"Correctness of the parent agent's independent answer does not repair the drift",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing parent-agent boundary term %q", path, needle)
			}
		}
	}
}

// TestRoutingContractExcludesCodeMutationFromCriticalHit verifies the routing
// contract explicitly classifies code/test mutation requests as
// implementation/materialization, never Critical Hit.
func TestRoutingContractExcludesCodeMutationFromCriticalHit(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Requests to remove, edit, merge, or refactor source files or tests are not Critical Hit",
			"The default Sniper contract does not",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing code-mutation routing term %q", path, needle)
			}
		}
	}
}

// TestExecutionContractForbidsParentAgentMutationBypass verifies the execution
// contract states that Sniper's code/test mutation prohibition cannot be
// bypassed by the parent agent performing the mutation directly.
func TestExecutionContractForbidsParentAgentMutationBypass(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"be bypassed by the parent agent performing the mutation directly",
			"produces analysis/handoff only",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing mutation-bypass term %q", path, needle)
			}
		}
	}
}

// TestCriticalHitSupportsEvidenceGatedClosureIntoDone verifies Critical Hit's
// narrative and machine contracts describe a closure move into done/ that
// requires an explicit completion claim and a supplied evidence summary, and
// never infers completion from code alone. Close Card was folded into
// Critical Hit as a second mode rather than kept as a separate route.
func TestCriticalHitSupportsEvidenceGatedClosureIntoDone(t *testing.T) {
	t.Parallel()

	narrativePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	}
	for _, path := range narrativePaths {
		content := readFile(t, path)
		for _, needle := range []string{
			"Closure move",
			"`<base_path>/done/<id>`",
			"evidence summary is available",
			"never infers completion",
			"Stale Card Detection",
			"Discovery (Ranger)",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing closure-move term %q", path, needle)
			}
		}
		if strings.Contains(content, "close-card.md") || strings.Contains(content, "12-close-card") {
			t.Fatalf("%s still references a separate close-card file after consolidation", path)
		}
	}

	machinePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	}
	for _, path := range machinePaths {
		content := readFile(t, path)
		for _, needle := range []string{
			"closure_move",
			"evidence_summary_present: true",
			"completion_inferred_from_code_only",
			"stale_card_detection",
			"on: discovery",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing closure-move term %q", path, needle)
			}
		}
	}
}

// TestCloseCardIsNotASeparateRoute verifies close-card.yaml and
// 12-close-card.md were removed as standalone files (source and embedded
// defaults) after Close Card was consolidated into Critical Hit, and that no
// other contract or skill.yaml still refers to close_card as an independent
// resolution path.
func TestCloseCardIsNotASeparateRoute(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	for _, rel := range []string{
		filepath.Join("internal", "embed", "defaults", "contracts", "narrative", "12-close-card.md"),
		filepath.Join("internal", "embed", "defaults", "contracts", "machine", "close-card.yaml"),
		filepath.Join("internal", "embed", "defaults", "contracts", "narrative", "12-close-card.md"),
		filepath.Join("internal", "embed", "defaults", "contracts", "machine", "close-card.yaml"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Fatalf("%s should not exist — close-card was folded into critical-hit.yaml/11-critical-hit.md", rel)
		}
	}

	for _, path := range []string{
		filepath.Join(root, "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "close_card:") {
			t.Fatalf("%s still declares close_card as a separate resolution path", path)
		}
	}
}

// TestDocumentationAppliedDoesNotTriggerClosure verifies the corrected
// lifecycle model: reaching documentation_applied at the end of a
// main_mission is documentation completion only, does not trigger a Critical
// Hit closure candidacy check, and does not imply the package should move to
// done/. A completed main_mission ending with its package in refined/ is the
// normal, expected terminal state — not a gap the pipeline auto-corrects.
// This supersedes the earlier (incorrect) auto-closure-check design.
func TestDocumentationAppliedDoesNotTriggerClosure(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"is documentation completion, not implementation or validation evidence",
			"does not by itself trigger Critical Hit closure",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing corrected documentation_applied term %q", path, needle)
			}
		}
		if strings.Contains(content, "triggers Critical Hit's stale-card detection") {
			t.Fatalf("%s still contains the superseded auto-closure-check wiring", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "closure_check") {
			t.Fatalf("%s still contains the superseded closure_check step in the Main Mission Sequence", path)
		}
		if !strings.Contains(content, "Main mission completion does not imply implementation completion") {
			t.Fatalf("%s missing the corrected main-mission-completion statement", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "main_mission_execution_complete") {
			t.Fatalf("%s still contains the superseded main_mission_execution_complete trigger", path)
		}
		if !strings.Contains(content, "insufficient_evidence") {
			t.Fatalf("%s missing insufficient_evidence list", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Insufficient Evidence",
			"does NOT trigger a closure candidacy check",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing corrected Critical Hit term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "Fires\n        automatically at end of main_mission execution") ||
			strings.Contains(content, "Fires automatically at end of main_mission execution") {
			t.Fatalf("%s still claims critical_hit fires automatically at end of main_mission execution", path)
		}
	}
}

// TestApprovalGateAcceptanceDoesNotAuthorizeCodeMutation verifies the approval
// gate contract explicitly states that gate acceptance approves the refined
// analysis and documentation_target items only, never code/hook/config/test
// mutation — closing the drift where a prior mission treated gate acceptance
// as permission to edit source files directly.
func TestApprovalGateAcceptanceDoesNotAuthorizeCodeMutation(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Gate Acceptance Is Not Code Mutation Approval",
			"implementation_handoff",
			"not as a separate mission status",
			"requires a separate coding task outside",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing gate/code-mutation term %q", path, needle)
			}
		}
		assertNoToken(t, path, "implementation_handoff_ready")
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "Never treat Strategist Approval Gate acceptance") {
			t.Fatalf("%s missing NEVER DO bullet for gate-acceptance/code-mutation confusion", path)
		}
	}
}

func TestNoLegacyImplementationHandoffReadyStatus(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	for _, path := range []string{
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"),
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"),
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	} {
		assertNoToken(t, path, "implementation_handoff_ready")
	}
}

// TestSniperBlocksImplementationHandoffInTasks verifies the execution
// contract and the Sniper internal skill both require a pre-materialization
// scan of tasks.md/implementation_plan for code-changing items, and block
// with documentation_scope_violation instead of executing them.
func TestSniperBlocksImplementationHandoffInTasks(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Pre-Materialization Scan",
			"blocked reason=documentation_scope_violation",
			"details=tasks.md contains implementation handoff items",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing pre-materialization scan term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "sniper", "SKILL.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Pre-Materialization Scan",
			"documentation_scope_violation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing pre-materialization scan term %q", path, needle)
			}
		}
	}
}

// TestArchivistClassifiesTaskTypeForSniperScope verifies the refinement
// contract and handoff schema require Archivist to classify every task by
// task_type, and that only documentation_target items are Sniper-executable —
// this is what lets Sniper's pre-materialization scan and the approval gate
// distinguish documentation work from implementation handoff.
func TestArchivistClassifiesTaskTypeForSniperScope(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"task_type",
			"documentation_target",
			"implementation_handoff",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing task_type classification term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "handoff-archivist-to-sniper.schema.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"task_type",
			"[documentation_target, analysis_artifact, implementation_handoff, out_of_scope]",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing task_type enum term %q", path, needle)
			}
		}
	}
}

// TestDriftPatternsIncludeApprovalGateCodeExecutionConfusion verifies the
// drift-patterns template declares the approval_gate_code_execution_confusion
// pattern so the agent self-corrects instead of re-reading full governance.
func TestDriftPatternsIncludeApprovalGateCodeExecutionConfusion(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"id: approval_gate_code_execution_confusion",
			"never authorizes code mutation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing drift pattern term %q", path, needle)
			}
		}
	}
}

// TestEvidencePackContractDefinesNonBlockingEmptyState verifies the Evidence Pack
// contract (Track T-A) declares its fields and the empty-state non-blocking behavior,
// and never turns evidence packs into a raw-chest-load or new retrieval unit.
func TestEvidencePackContractDefinesNonBlockingEmptyState(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"evidence_pack:",
			"never loads raw chest contents",
			"never introduces a new retrieval unit",
			"Non-blocking. evidence_pack_path is null.",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing evidence_pack contract term %q", path, needle)
			}
		}
	}
}

// TestDossierBuilderGeneratesEvidencePackFromSourceCards verifies dossier-builder
// declares evidence_pack_path in its output and the empty_source_cards failure mode
// leaves it null without writing a pack file.
func TestDossierBuilderGeneratesEvidencePackFromSourceCards(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "dossier-builder", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"evidence_pack_path: string | null",
			"write a mission Evidence Pack artifact",
			"evidence_pack_path is null; no Evidence Pack file is written",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing evidence pack generation term %q", path, needle)
			}
		}
	}
}

// TestRangerAndArchivistThreadEvidencePackPath verifies Ranger cites evidence_pack_path
// in the analysis artifact when present, and Archivist preserves it through promotion
// without adding a fifth file to the refined package.
func TestRangerAndArchivistThreadEvidencePackPath(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "ranger", "skill.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "evidence_pack_path: string | null") {
			t.Fatalf("%s missing evidence_pack_path output field", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "archivist", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"evidence_pack_path: string | null",
			"Never add a fifth file to the refined package for it",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing evidence_pack_path propagation term %q", path, needle)
			}
		}
	}
}

// TestTelemetryContractDocumentsChestEventDistinction verifies the telemetry contract
// (Track T-D1) documents treasure_chest_loaded vs treasure_chest_found as intentionally
// distinct events rather than leaving them as unexplained naming drift.
func TestTelemetryContractDocumentsChestEventDistinction(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Chest Event Naming",
			"intentionally distinct events",
			"No rename or consolidation without an explicit approval-gate extension",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing chest event naming classification term %q", path, needle)
			}
		}
	}
}

// TestScoutRouteDecisionSchemaExists verifies the Scout route-decision schema
// file exists in both the canonical source tree and the embedded defaults tree.
func TestScoutRouteDecisionSchemaExists(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "scout-route-decision.schema.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scout route decision schema at %s: %v", path, err)
		}
	}
}

// TestScoutRouteDecisionSchemaDefinesRequiredFields verifies the schema declares
// the full route_decision field contract, including the evidence_state enum.
func TestScoutRouteDecisionSchemaDefinesRequiredFields(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "scout-route-decision.schema.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"role:",
			"component:",
			"request_category:",
			"selected_route:",
			"route_reason:",
			"route_confidence:",
			"evidence_state:",
			"discovery_subtype:",
			"fallback_route:",
			"gate_required:",
			"explicit",
			"insufficient",
			"requires_discovery",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout route decision field %q", path, needle)
			}
		}
	}
}

// TestRoutingContractNamesScoutAsRouteOwner verifies 00-routing.md names Scout
// as the owner of route selection and cross-references the Intake Router label.
func TestRoutingContractNamesScoutAsRouteOwner(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Scout",
			"Intake Router",
			"scout-route-decision.schema.yaml",
			"scout-routing.yaml",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout route owner term %q", path, needle)
			}
		}
	}
}

// TestScoutRoutingMachineContractFallsBackToFullPipeline verifies the Scout
// machine contract defines the conservative full_pipeline fallback and never
// substitutes for the Strategist Approval Gate.
func TestScoutRoutingMachineContractFallsBackToFullPipeline(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"full_pipeline_default",
			"conservatism is the safe default",
			"route_confidence_threshold",
			"Scout NEVER bypasses the Strategist Approval Gate",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout routing machine contract term %q", path, needle)
			}
		}
	}
}

// TestScoutSkillFilesExist verifies the Scout internal skill (SKILL.md + skill.yaml)
// exists in both the canonical source tree and the embedded defaults tree.
func TestScoutSkillFilesExist(t *testing.T) {
	t.Parallel()

	for _, rel := range []string{
		filepath.Join("internal_skills", "scout", "SKILL.md"),
		filepath.Join("internal_skills", "scout", "skill.yaml"),
	} {
		for _, root := range []string{
			filepath.Join(repoRoot(t), "internal", "embed", "defaults"),
		} {
			path := filepath.Join(root, rel)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected scout skill file at %s: %v", path, err)
			}
		}
	}
}

// TestScoutSkillDeclaresForbiddenBehaviors verifies Scout's skill.yaml declares
// the boundaries that keep it from becoming a mini-Ranger.
func TestScoutSkillDeclaresForbiddenBehaviors(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "scout", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"perform_deep_discovery",
			"invoke_sniper_directly",
			"bypass_approval_gate",
			"replace_ranger",
			"set_gate_required_false",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout forbidden behavior %q", path, needle)
			}
		}
	}
}

// TestSkillYamlPipelineIncludesScoutRoutingStage verifies the master pipeline
// wires in the scout_routing stage before discovery, and that Scout is
// documented as never being a configurable slot.
func TestSkillYamlPipelineIncludesScoutRoutingStage(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"stage: scout_routing",
			"skill: scout",
			"scout-route-decision.schema.yaml",
			"never_a_slot: true",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout pipeline wiring term %q", path, needle)
			}
		}
	}
}

// TestRoleLockForbidsSkippingScout verifies the parent-agent Role Lock contract
// forbids performing Scout's classification directly or skipping Scout entirely.
func TestRoleLockForbidsSkippingScout(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Scout's route classification",
			"skip Scout",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing role lock scout term %q", path, needle)
			}
		}
	}
}

// TestDiscoveryContractDefinesSubtypeVocabulary verifies 03-discovery.md defines
// the four discovery_subtype values and the evaluation_verdict vocabulary.
func TestDiscoveryContractDefinesSubtypeVocabulary(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"creative",
			"evaluation",
			"diagnostic",
			"closure_evidence",
			"evaluation_verdict",
			"implemented",
			"partially_implemented",
			"not_implemented",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing discovery subtype vocabulary term %q", path, needle)
			}
		}
	}
}

// TestRangerHandoffSchemaSupportsEvaluationVerdict verifies the Ranger→Archivist
// handoff schema carries discovery_subtype and evaluation_verdict fields.
func TestRangerHandoffSchemaSupportsEvaluationVerdict(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "handoff-ranger-to-archivist.schema.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"discovery_subtype:",
			"evaluation_verdict:",
			"partially_implemented",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing evaluation verdict field %q", path, needle)
			}
		}
	}
}

// TestRangerRoleFileReferencesEvaluationVerdict verifies roles/ranger.yaml
// instructs Ranger to record evaluation_verdict for evaluation-subtype discovery.
func TestRangerRoleFileReferencesEvaluationVerdict(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "roles", "ranger.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "evaluation_verdict") {
			t.Fatalf("%s missing evaluation_verdict reference", path)
		}
	}
}

// TestShortRouteAnnotationRequiresExplicitEvidence verifies 00-routing.md
// narrows Implementation Short Route's ability to annotate implementation status
// to cases with explicit, narrow evidence, falling back to full_pipeline otherwise.
func TestShortRouteAnnotationRequiresExplicitEvidence(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Annotation Limits",
			"evidence_state: explicit",
			"discovery_subtype: evaluation",
			"Short Route must not infer it",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing short route annotation limit term %q", path, needle)
			}
		}
	}
}

// TestCriticalHitClosureCrossReferencesEvidenceState verifies the critical-hit
// narrative and machine contracts tie closure evidence to Scout's evidence_state
// vocabulary without weakening the existing Insufficient Evidence invariants.
func TestCriticalHitClosureCrossReferencesEvidenceState(t *testing.T) {
	t.Parallel()

	narrativePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	}
	for _, path := range narrativePaths {
		content := readFile(t, path)
		if !strings.Contains(content, "evidence_state: explicit") {
			t.Fatalf("%s missing evidence_state cross-reference", path)
		}
	}

	machinePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	}
	for _, path := range machinePaths {
		content := readFile(t, path)
		if !strings.Contains(content, "evidence_state: explicit") {
			t.Fatalf("%s missing evidence_state cross-reference", path)
		}
	}
}

// TestBrainstormingProviderDeclaresSubtypeSupport verifies the brainstorming
// provider manifest declares the subtype support Scout checks after route_decision.
// .strategist/ is a generated runtime artifact, so this only asserts the
// embedded-defaults copy — the canonical source for what strategist install stamps
// into a workspace.
func TestBrainstormingProviderDeclaresSubtypeSupport(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skills", "brainstorming", "skill.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"canonical_role: ranger",
		"provider_class: rankeado",
		"risk_score: write_analysis",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing brainstorming weapon term %q", path, needle)
		}
	}
	for _, needle := range []string{
		"discovery_subtype_support:",
		"creative: native",
		"evaluation: adapter",
		"diagnostic: adapter",
		"closure_evidence: adapter",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing brainstorming subtype support term %q", path, needle)
		}
	}
}

// TestEvaluationDiscoveryDoesNotRequireCreativeObligations verifies
// 03-discovery.md explicitly states evaluation discovery is exempt from
// creative-only obligations.
func TestEvaluationDiscoveryDoesNotRequireCreativeObligations(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"does not require design-option exploration",
			"writing-plans` handoff",
			"design-doc commit",
			"creative`-subtype obligations only",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing evaluation-exemption term %q", path, needle)
			}
		}
	}
}

// TestRoleLockRequiresSubtypeCapabilityCheck verifies the parent-agent Role Lock
// blocks unsupported discovery subtype/weapon pairings before invoking the weapon.
func TestRoleLockRequiresSubtypeCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Discovery subtypes are selected by Scout and executed through Ranger",
			"discovery_subtype_support",
			"provider_capability_mismatch",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing subtype capability-check term %q", path, needle)
			}
		}
	}
}

// TestPreflightContractDefinesProviderCapabilityMismatch verifies preflight.yaml
// documents the post-route provider/subtype mismatch block and honest remediation hint.
func TestPreflightContractDefinesProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"code: provider_capability_mismatch",
			"reason=provider_capability_mismatch",
			"scout-routing.yaml",
			"editing skill.yaml metadata alone does not change",
			"Verify with a live invocation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing preflight provider_capability_mismatch term %q", path, needle)
			}
		}
	}
}

// TestDriftPatternsCoverProviderCapabilityMismatch verifies the normative
// drift-patterns.yaml teaches subtype-capability blocking on weapons.
func TestDriftPatternsCoverProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"id: provider_capability_mismatch",
			"editing skill.yaml metadata alone does not change",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider_capability_mismatch drift pattern term %q", path, needle)
			}
		}
	}
}

// TestSkillYamlStopConditionsIncludeProviderCapabilityMismatch verifies the
// master pipeline declares provider_capability_mismatch as a stop condition.
func TestSkillYamlStopConditionsIncludeProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "provider_capability_mismatch") {
			t.Fatalf("%s stop_conditions must include provider_capability_mismatch", path)
		}
	}
}

// TestTelemetryContractDefinesScoutFields verifies 10-telemetry.md's canonical
// event payload lists Scout's route-decision fields and distinguishes them from
// Ranger's discovery-result events.
func TestTelemetryContractDefinesScoutFields(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"`role`",
			"`route`",
			"`route_reason`",
			"`route_confidence`",
			"`evidence_state`",
			"`discovery_subtype`",
			"`provider`",
			"component: scout",
			"component: ranger",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout telemetry field %q", path, needle)
			}
		}
	}
}

// TestRoutingContractDefinesPostRouteCapabilityCheck verifies 00-routing.md
// describes the weapon-capability check running immediately after Scout emits
// route_decision, before the discovery weapon is invoked.
func TestRoutingContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Post-Route Capability Check",
			"post_route_capability_check",
			"before the discovery weapon is invoked",
			"discovery_subtype_support",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post-route capability check term %q", path, needle)
			}
		}
	}
}

// TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck verifies
// scout-routing.yaml defines the post_route_capability_check block that runs
// before weapon invocation, checking discovery_subtype_support.
func TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"post_route_capability_check:",
			"discovery_subtype_support",
			"before_weapon_invocation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post_route_capability_check term %q", path, needle)
			}
		}
	}
}

// TestJewelGenerationContractDefinesTrustCeiling verifies the jewel_generation contract
// (Track T-J / SQ-009) caps generation per consultation and enforces the trust-tier ceiling
// that replaced the human pre-approval gate for agent-generated jewels.
func TestJewelGenerationContractDefinesTrustCeiling(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"jewel_generation:",
			"cap_per_consultation: 1",
			"write_target: .strategist/jewels.yaml",
			"generating a jewel with trust exceeding the parent chest's trust.tier",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing jewel_generation contract term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "ranger", "skill.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "archivist", "skill.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "jewel_generation contract") {
			t.Fatalf("%s missing reference to the jewel_generation contract", path)
		}
	}
}

// TestJewelRetrievalContractDefinesMandatoryFallback verifies the jewel_retrieval contract
// (Track T-J2, jewels-retrieval-precedence) wires jewel consultation into the retrieval
// fallback order as a ranking hint and enforces the mandatory fallback to full source_cards
// assembly when a jewel is stale, disputed, missing, or insufficient.
func TestJewelRetrievalContractDefinesMandatoryFallback(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"jewel_consultation_as_ranking_hint",
			"jewel_retrieval:",
			"condition: jewel is stale | disputed | missing | insufficient",
			"action: proceed with full source_cards assembly unchanged",
			"jewels_consulted: [id, chest_id, trust, status, statement]",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing jewel_retrieval contract term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "dossier-builder", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"jewel_retrieval contract",
			"jewels_consulted",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing reference to jewel_retrieval / jewels_consulted", path)
			}
		}
	}
}

// TestOutcomeEntryCoordinatedShapeMirrorsRuntime locks the coordinated outcome-entry
// shape from 2026-07-25-outcome-entry-schema-coordination across source, embedded
// defaults, and the installed .strategist runtime mirror.
func TestOutcomeEntryCoordinatedShapeMirrorsRuntime(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	coordinatedFiles := map[string][]string{
		filepath.Join("schemas", "outcome-entry.schema.yaml"): {
			"mission_id",
			"jewel_ids",
			"idempotency:",
			"key: mission_id",
			"mission_id_is_unique",
		},
		filepath.Join("schemas", "mission-result.schema.yaml"): {
			"mission_id",
			"jewels_consulted",
			"jewel_ids",
			"no jewels were consulted",
			"not an error",
		},
		filepath.Join("contracts", "machine", "context-enrichment.yaml"): {
			"jewel_retrieval:",
			"jewels_consulted",
			"outcome_forwarding:",
			"mission_result.jewels_consulted",
			"outcome-entry.schema.yaml's jewel_ids",
		},
		filepath.Join("contracts", "machine", "learning-curator.yaml"): {
			"idempotency:",
			"mission_id is the idempotency key",
			"jewel_outcome_mapping:",
			"mission_result.jewels_consulted",
			"jewel_ids",
		},
		filepath.Join("contracts", "machine", "learning-buffer.yaml"): {
			"idempotency_key: mission_id",
			"Reference implementation: internal/telemetry.FlushOutcomeBuffer",
			"dedup-on-mission_id",
			"idempotency_invariant:",
		},
	}

	for rel, needles := range coordinatedFiles {
		sourcePath := filepath.Join(root, "internal", "embed", "defaults", rel)
		embedPath := filepath.Join(root, "internal", "embed", "defaults", rel)
		runtimePath := filepath.Join(root, ".strategist", rel)

		source := readFile(t, sourcePath)
		if embed := readFile(t, embedPath); source != embed {
			t.Fatalf("%s drifted from embedded default %s; run make sync-embed", sourcePath, embedPath)
		}
		if runtime := readFile(t, runtimePath); source != runtime {
			t.Fatalf("%s drifted from runtime mirror %s; refresh .strategist from the canonical source", sourcePath, runtimePath)
		}

		for _, needle := range needles {
			if !strings.Contains(source, needle) {
				t.Fatalf("%s missing coordinated outcome-entry term %q", sourcePath, needle)
			}
		}
	}
}

// TestLearningCuratorInternalSkillDefinesJewelOutcomeProduction locks the
// provider-owned production boundary for mission_result.jewels_consulted ->
// outcome_entry.jewel_ids across source, embedded defaults, and runtime mirror.
func TestLearningCuratorInternalSkillDefinesJewelOutcomeProduction(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	sourcePath := filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "learning-curator", "skill.yaml")
	embedPath := filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "learning-curator", "skill.yaml")
	runtimePath := filepath.Join(root, ".strategist", "internal_skills", "learning-curator", "skill.yaml")

	source := readFile(t, sourcePath)
	if embed := readFile(t, embedPath); source != embed {
		t.Fatalf("%s drifted from embedded default %s; run make sync-embed", sourcePath, embedPath)
	}
	if runtime := readFile(t, runtimePath); source != runtime {
		t.Fatalf("%s drifted from runtime mirror %s; refresh .strategist from the canonical source", sourcePath, runtimePath)
	}

	for _, needle := range []string{
		"mission_result.jewels_consulted",
		"verbatim into the outcome entry's jewel_ids field",
		"absent or empty, write the outcome without jewel_ids",
		"non-blocking",
		"mission_id as the only duplicate/idempotency key",
		"jewel_ids never participate in duplicate detection",
	} {
		if !strings.Contains(source, needle) {
			t.Fatalf("%s missing provider-owned jewel outcome instruction %q", sourcePath, needle)
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
func TestDiscoveryAndRefinementContractsRequireDocsLanguage(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "active.language.docs") {
			t.Fatalf("%s missing instruction to write documentation artifacts in active.language.docs", path)
		}
	}
}

// TestQuickDrawRunbookOpportunityIsExplicitGateOnly verifies the runbook_opportunity
// routine added to Quick Draw (source and embedded mirror) is advisory-only: it must
// declare that it never writes a runbook file directly, that the runbook gate option
// is only offered when warranted, and that candidate creation requires its own
// explicit confirmation independent of the ordinary idea-append gate response.
func TestQuickDrawRunbookOpportunityIsExplicitGateOnly(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "quick-draw.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"phase: runbook_opportunity",
			"MUST NOT write a runbook file directly",
			"MUST NOT perform discovery beyond the already-normalized Quick Draw idea",
			"runbook: create_runbook_candidate    # only shown when runbook_opportunity.warranted=true",
			"Each action requires its\n      own explicit confirmation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing explicit-gate-only term %q", path, needle)
			}
		}
	}
}

// TestQuickDrawRunbookOpportunityDoesNotClaimADROrClosureRole verifies the new
// routine explicitly defers ADR-worthiness to Opportunity Attack and card
// closure/movement to Critical Hit, rather than growing into either role.
func TestQuickDrawRunbookOpportunityDoesNotClaimADROrClosureRole(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "quick-draw.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"MUST NOT evaluate ADR-worthiness — that remains Opportunity Attack's responsibility",
			"MUST NOT evaluate card closure or movement — that remains Critical Hit's responsibility",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing role-deferral term %q", path, needle)
			}
		}
	}
}

// TestRunbookCandidateNeverWritesCanonicalDirectly verifies sniper_quick_draw's
// runbook candidate action can only produce a reviewable candidate — never a
// direct write to the canonical docs/runbooks/ tree — and that promotion to
// canonical requires human acceptance.
func TestRunbookCandidateNeverWritesCanonicalDirectly(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "quick-draw.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"writing directly to docs/runbooks/<slug>.md from this phase",
			"promoting the candidate to canonical without human acceptance",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing candidate-forbidden term %q", path, needle)
			}
		}
	}
}

// TestOpportunityAttackRemainsADROnlyAfterQuickDrawRunbookAddition is a regression
// guard: adding a Quick Draw-scoped runbook routine must not expand Opportunity
// Attack's ADR-only remit or make it aware of runbooks at all.
func TestOpportunityAttackRemainsADROnlyAfterQuickDrawRunbookAddition(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "opportunity-attack.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "Opportunity Attack is ADR-only") {
			t.Fatalf("%s lost its ADR-only declaration", path)
		}
		if strings.Contains(strings.ToLower(content), "runbook") {
			t.Fatalf("%s should not reference runbooks — that is Quick Draw's runbook_opportunity routine", path)
		}
	}
}

// TestCriticalHitRemainsClosureOnlyAfterQuickDrawRunbookAddition is a regression
// guard: Critical Hit must not gain runbook-worthiness evaluation as a side
// effect of the Quick Draw runbook_opportunity routine.
func TestCriticalHitRemainsClosureOnlyAfterQuickDrawRunbookAddition(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(strings.ToLower(content), "runbook") {
			t.Fatalf("%s should not reference runbooks — closure/move only", path)
		}
	}
}

// TestPersonasExposeRunbookOpportunityGateKey verifies both personas define the
// runbook_opportunity template key used to surface the runbook candidate option,
// and that it is framed as an independent confirmation rather than a default action.
func TestPersonasExposeRunbookOpportunityGateKey(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "runbook_opportunity: >") {
			t.Fatalf("%s missing runbook_opportunity template key", path)
		}
		if !strings.Contains(content, "candidate_id") || !strings.Contains(content, "candidate_title") {
			t.Fatalf("%s runbook_opportunity template missing candidate_id/candidate_title placeholders", path)
		}
	}
}

// TestDocsRunbooksPolicyDeclaresSourceFirstProvenance verifies the runbook
// ownership policy documents docs/runbooks/ as canonical, requires provenance
// metadata on any runtime-optimized artifact, and defines a deterministic
// source-vs-runtime mismatch outcome.
func TestDocsRunbooksPolicyDeclaresSourceFirstProvenance(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "docs", "runbooks", "README.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"canonical",
		"derived cache",
		"source_hash",
		"freshness: fresh|stale|unknown",
		"prefer canonical source",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing runbook policy term %q", path, needle)
		}
	}
}

// relativeFileSet walks dir and returns the set of file paths relative to dir, using forward
// slashes regardless of OS.
func relativeFileSet(t *testing.T, dir string) map[string]bool {
	t.Helper()
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return files
}
