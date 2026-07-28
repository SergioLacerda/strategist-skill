//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype verifies
// 00-routing.md normatively states that evaluation/diagnostic/closure_evidence
// discovery subtypes always resolve to internal_skills/ranger, bypassing the
// configured external weapon.
func TestRoutingContractDefinesDiscoveryWeaponResolutionBySubtype(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Weapon Resolution by Subtype",
		"internal_skills/ranger",
		"kind=native_role",
		"regardless of `active.slots.discovery`",
		"never a live behavior guarantee",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing discovery weapon resolution term %q", path, needle)
		}
	}
}

// TestDiscoveryContractDefinesSubtypeVocabulary verifies 03-discovery.md defines
// the four discovery_subtype values and the evaluation_verdict vocabulary.

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

// TestAgentProtocolTemplateRoutesDiscoveryBySubtype verifies the compiled
// agent-protocol template resolves discovery invocation conditionally on
// discovery_subtype instead of unconditionally naming {{.Slots.Discovery}}.

// TestAgentProtocolTemplateRoutesDiscoveryBySubtype verifies the compiled
// agent-protocol template resolves discovery invocation conditionally on
// discovery_subtype instead of unconditionally naming {{.Slots.Discovery}}.
func TestAgentProtocolTemplateRoutesDiscoveryBySubtype(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Routing",
		"discovery_subtype",
		"internal_skills/ranger",
		"native_role",
		"regardless of what `active.slots.discovery` is configured to",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing discovery-by-subtype routing term %q", path, needle)
		}
	}
}

// TestDiscoveryContractCrossReferencesSubtypeResolution verifies 03-discovery.md
// points to 00-routing.md for which concrete invocation target (external weapon
// vs. native Ranger) handles each subtype.

// TestDiscoveryContractCrossReferencesSubtypeResolution verifies 03-discovery.md
// points to 00-routing.md for which concrete invocation target (external weapon
// vs. native Ranger) handles each subtype.
func TestDiscoveryContractCrossReferencesSubtypeResolution(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery Weapon Resolution by Subtype",
		"internal_skills/ranger",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing subtype resolution cross-reference term %q", path, needle)
		}
	}
}

// TestRangerHandoffSchemaSupportsEvaluationVerdict verifies the Ranger→Archivist
// handoff schema carries discovery_subtype and evaluation_verdict fields.

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

// TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport verifies the
// brainstorming provider manifest declares creative-subtype support only.
// It must NOT claim evaluation/diagnostic/closure_evidence adapter support:
// those subtypes bypass this weapon entirely and resolve to
// internal_skills/ranger (see 00-routing.md § Discovery Weapon Resolution by
// Subtype). A prior version of this test required the adapter claims —
// that assumption was falsified by a live invocation showing brainstorming's
// own SKILL.md has no adaptive behavior for those subtypes at all.
// .strategist/ is a generated runtime artifact, so this only asserts the
// embedded-defaults copy — the canonical source for what strategist install stamps
// into a workspace.
func TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skills", "brainstorming", "skill.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"canonical_role: ranger",
		"provider_class: rankeado",
		"risk_score: write_analysis",
		"discovery_subtype_support:",
		"creative: native",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing brainstorming weapon term %q", path, needle)
		}
	}
	for _, forbidden := range []string{
		"evaluation: adapter",
		"diagnostic: adapter",
		"closure_evidence: adapter",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still declares false subtype support %q — these subtypes now bypass this weapon entirely", path, forbidden)
		}
	}
}

// TestEvaluationDiscoveryDoesNotRequireCreativeObligations verifies
// 03-discovery.md explicitly states evaluation discovery is exempt from
// creative-only obligations.

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
