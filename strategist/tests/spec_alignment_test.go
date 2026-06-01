package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fixture struct {
	Scenario      string `yaml:"scenario"`
	ExpectedEvent string `yaml:"expected_event"`
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
	featurePath := filepath.Join("specs", "forbidden-behaviors.feature")
	fixturePath := filepath.Join("fixtures", "side-quest-bypass.yaml")

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
	featurePath := filepath.Join("specs", "token-economy.feature")
	fixturePath := filepath.Join("fixtures", "triage-gate-blocked.yaml")

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
	featurePath := filepath.Join("specs", "approval-gate.feature")
	fixturePath := filepath.Join("fixtures", "approval-bypass.yaml")

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
	featurePath := filepath.Join("specs", "forbidden-behaviors.feature")
	fixturePath := filepath.Join("fixtures", "single-target-sweep-bypass.yaml")

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
	featurePath := filepath.Join("specs", "policy-guardrails.feature")
	fixturePath := filepath.Join("fixtures", "policy-guardrails-e2e.yaml")

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
