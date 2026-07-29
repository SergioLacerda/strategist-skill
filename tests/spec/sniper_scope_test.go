//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

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
