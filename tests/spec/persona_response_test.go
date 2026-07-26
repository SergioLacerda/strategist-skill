//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

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
