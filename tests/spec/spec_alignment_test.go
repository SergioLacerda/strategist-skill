//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
			path:      filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "09-response.md"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
		{
			path:      filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "09-response.md"),
			forbidden: []string{"📁 discovery:  .analysis/", "📁 refined:    .analysis/", "📁 report:     .analysis/"},
		},
		{
			path:      filepath.Join(repoRoot(t), "strategist", "contracts", "scope-locking.md"),
			forbidden: []string{"em `.analysis/todo/`"},
		},
		{
			path:      filepath.Join(repoRoot(t), "strategist", "personas", "epic.yaml"),
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
	// not strategist/schemas/ (source tree).
	pairs := [][2]string{
		{
			filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "02-intake.md"),
			filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "02-intake.md"),
		},
		{
			filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "03-discovery.md"),
			filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"),
		},
		{
			filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "04-refinement.md"),
			filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md"),
		},
		{
			filepath.Join(repoRoot(t), "strategist", "contracts", "machine", "learning-buffer.yaml"),
			filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "learning-buffer.yaml"),
		},
	}
	for _, pair := range pairs {
		for _, path := range pair {
			content := readFile(t, path)
			if strings.Contains(content, "`strategist/schemas/") || strings.Contains(content, " strategist/schemas/") {
				t.Fatalf("%s references source-tree schema path strategist/schemas/; use .strategist/schemas/ instead", path)
			}
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

	// Source authoring dir must be internal_skills/, not skills/, so it maps directly to runtime.
	sourceDir := filepath.Join(repoRoot(t), "strategist", "internal_skills")
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		t.Fatalf("strategist/internal_skills/ must exist as the source-authoring directory for internal skills")
	}

	// Legacy source directory must not exist.
	legacyDir := filepath.Join(repoRoot(t), "strategist", "skills")
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Fatalf("strategist/skills/ still exists — internal skills must be authored under strategist/internal_skills/")
	}

	// Makefile sync must target the renamed source directory.
	makefile := readFile(t, filepath.Join(repoRoot(t), "Makefile"))
	if !strings.Contains(makefile, "strategist/internal_skills/") {
		t.Fatalf("Makefile does not sync strategist/internal_skills/ — embed sync is broken")
	}
}

func TestRemedationLabelPointsToCanonicalSkillsDir(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(repoRoot(t), "strategist", "schemas", "progress-contract.yaml"),
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
		filepath.Join(repoRoot(t), "strategist", "SKILL.md"),
		filepath.Join(repoRoot(t), "strategist", "skill.yaml"),
		filepath.Join(repoRoot(t), "strategist", "protocol.md"),
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

	path := filepath.Join(repoRoot(t), "strategist", "protocol.md")
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

	path := filepath.Join(repoRoot(t), "strategist", "SKILL.md")
	content := readFile(t, path)

	for _, needle := range []string{
		"`strategist/` — source-only",
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
	if !strings.Contains(feature, "status=plan_only") {
		t.Fatalf("%s missing plan_only assertions", featurePath)
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
		!strings.Contains(fixture.ExpectedEvent, "transition_group=execution") {
		t.Fatalf("%s expected_event must include blocked policy_eval for execution, got: %q", fixturePath, fixture.ExpectedEvent)
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
		"execution: execution",
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

func TestStrategistBootstrapContractKeepsContractValidation(t *testing.T) {
	t.Parallel()
	bootstrapPath := filepath.Join(repoRoot(t), "strategist", "contracts", "bootstrap.md")
	content := readFile(t, bootstrapPath)

	if !strings.Contains(content, "## 2f. Contract validation") {
		t.Fatalf("%s missing contract validation section", bootstrapPath)
	}
	if !strings.Contains(content, "required: true") {
		t.Fatalf("%s missing required field validation rule", bootstrapPath)
	}
	if !strings.Contains(content, "contract_input_missing") {
		t.Fatalf("%s missing contract_input_missing stop condition", bootstrapPath)
	}
}

func TestStrategistResponseContractIsExternalized(t *testing.T) {
	t.Parallel()
	protocolPath := filepath.Join(repoRoot(t), "strategist", "protocol.md")
	skillPath := filepath.Join(repoRoot(t), "strategist", "SKILL.md")

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
		filepath.Join(repoRoot(t), "strategist", "provider-discovery.md"),
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
		filepath.Join(repoRoot(t), "strategist", "provider-discovery.md"),
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
		filepath.Join(repoRoot(t), "strategist", "schemas", "progress-contract.yaml"),
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
			"Ranger runs opportunity attack",
			"Archivist runs opportunity attack",
			"approval gate",
		},
		filepath.Join(testDir(t), "specs", "e2e-approval-gate.feature"): []string{
			"mission result is completed",
			"plan_only",
			"Sniper is not invoked",
		},
		filepath.Join(testDir(t), "specs", "e2e-treasure-chests.feature"): []string{
			"treasure chests",
			"treasure_chests=none",
		},
		filepath.Join(testDir(t), "specs", "e2e-opportunity-attack.feature"): []string{
			"opportunity attack",
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
		filepath.Join(repoRoot(t), "strategist", "internal_skills"),
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
		filepath.Join(repoRoot(t), ".strategist", "contracts", "tests", "preflight.test.yaml"),
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

func TestSniperWriteScopeIsWorkspaceAndDocs(t *testing.T) {
	t.Parallel()

	// The execution contract must declare that Sniper write scope is workspace files
	// and documentation files only — code mutation is always forbidden.
	files := []string{
		filepath.Join(repoRoot(t), "strategist", "contracts", "narrative", "06-execution.md"),
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
		filepath.Join(repoRoot(t), "strategist", "templates"),
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
