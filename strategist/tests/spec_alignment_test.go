package tests

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

func TestEmitTaxonomyMandatoryVisibilityLevels(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "output-profiles", "emit-taxonomy.yaml")
	content := readFile(t, path)

	assertions := []string{
		"opportunity_attack_done:     INFO",
		"treasure_chest_found:        INFO",
		"compliance_summary:          INFO",
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
	bootstrapPath := filepath.Join(repoRoot(t), ".strategist", "contracts", "bootstrap.yaml")
	content := readFile(t, bootstrapPath)
	if !strings.Contains(content, "invalid_local_profile") {
		t.Fatalf("%s missing error_condition code \"invalid_local_profile\"", bootstrapPath)
	}
}

func TestSkillDefinesMissingProfileDiagnosticsBlock(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), ".strategist", "skill.yaml"),
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
		filepath.Join(repoRoot(t), ".strategist", "skill.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "persona_render_mismatch") {
			t.Fatalf("%s missing forbidden_behavior \"persona_render_mismatch\"", p)
		}
	}
}

func TestMissionMetricsSignalPresent(t *testing.T) {
	t.Parallel()

	files := []string{
		filepath.Join(testDir(t), "..", "schemas", "progress-contract.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "schemas", "progress-contract.yaml"),
		filepath.Join(repoRoot(t), ".strategist", "contracts", "intake.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "intake.yaml"),
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
