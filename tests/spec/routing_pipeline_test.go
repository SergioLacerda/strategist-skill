//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype verifies
// 00-routing.md normatively states that evaluation/diagnostic/closure_evidence
// discovery subtypes always resolve to internal_skills/ranger, bypassing the
// configured external weapon.

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
