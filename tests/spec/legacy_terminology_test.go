//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
//   - strategist/contracts/machine/adr.yaml — pt-BR section name list (data).
//   - strategist/contracts/narrative/07-adr.md — pt-BR language mapping (data).
//   - strategist/contracts/adr.md — docs: pt-BR language mapping (data).
//   - strategist/contracts/machine/critical-hit.yaml — reserved input tokens with inline doc.

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
