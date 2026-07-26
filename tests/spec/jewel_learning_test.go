//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

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
		runtimePath := filepath.Join(isolatedStrategistDir(t), rel)

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

// TestLearningCuratorInternalSkillDefinesJewelOutcomeProduction locks the
// provider-owned production boundary for mission_result.jewels_consulted ->
// outcome_entry.jewel_ids across source, embedded defaults, and runtime mirror.
func TestLearningCuratorInternalSkillDefinesJewelOutcomeProduction(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	sourcePath := filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "learning-curator", "skill.yaml")
	embedPath := filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "learning-curator", "skill.yaml")
	runtimePath := filepath.Join(isolatedStrategistDir(t), "internal_skills", "learning-curator", "skill.yaml")

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
