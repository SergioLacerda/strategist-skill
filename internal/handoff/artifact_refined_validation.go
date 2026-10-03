package handoff

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func validateRefinedAnalysis(files map[string][]byte, missionID string) (map[string]any, error) {
	analysis, ok := files["analysis.md"]
	if !ok {
		return nil, fmt.Errorf("handoff_artifact_invalid: read Archivist analysis.md: missing")
	}
	frontmatter, _, err := parseFrontmatter(analysis)
	if err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	if err := validateIdentity(frontmatter, missionID, []string{"archivist_done", "gate_pending", "gate_revision_requested", "gate_analysis_accepted", "sniper_running", "documentation_applied"}); err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	if _, declared := frontmatter[PolicyFactsKey]; declared {
		if _, err := ParsePolicyFacts(frontmatter); err != nil {
			return nil, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
		}
	}
	if err := validateSideQuestStructure(frontmatter, files["tasks.md"]); err != nil {
		return nil, err
	}
	return frontmatter, nil
}

func validateRefinedSupportFiles(files map[string][]byte) error {
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		if len(bytes.TrimSpace(files[name])) == 0 {
			return fmt.Errorf("handoff_artifact_invalid: Archivist %s is empty or missing", name)
		}
	}
	return nil
}

func validateRefinedTasks(tasks []byte) ([]string, error) {
	if err := validateTaskClassifications(tasks); err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: Archivist tasks: %w", err)
	}
	targets, err := ValidateDocumentationTargetContent(tasks)
	if err != nil {
		return nil, err
	}
	return targets, nil
}

func requireRefinedTargetFacts(frontmatter map[string]any, targets []string) error {
	if len(targets) == 0 {
		return nil
	}
	if _, err := ParsePolicyFacts(frontmatter); err != nil {
		return err
	}
	return nil
}

var sideQuestYAMLBlock = regexp.MustCompile("(?s)```ya?ml\\s*\\n(.*?)\\n```")

func validateSideQuestStructure(frontmatter map[string]any, tasks []byte) error {
	seen := map[string]struct{}{}
	if raw, ok := frontmatter["side_quests_approved"]; ok {
		if err := validateSideQuestList(raw, "analysis frontmatter", seen); err != nil {
			return err
		}
	}
	return validateTaskSideQuestBlocks(tasks, seen)
}

func validateTaskSideQuestBlocks(tasks []byte, seen map[string]struct{}) error {
	for _, block := range sideQuestYAMLBlock.FindAllSubmatch(tasks, -1) {
		if err := validateTaskSideQuestBlock(block[1], seen); err != nil {
			return err
		}
	}
	return nil
}

func validateTaskSideQuestBlock(raw []byte, seen map[string]struct{}) error {
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("handoff_artifact_invalid: side_quests_approved YAML in tasks.md: %w", err)
	}
	if approved, ok := document["side_quests_approved"]; ok {
		return validateSideQuestList(approved, "tasks.md", seen)
	}
	return nil
}

func validateSideQuestList(raw any, source string, seen map[string]struct{}) error {
	entries, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("handoff_artifact_invalid: side_quests_approved in %s must be a list", source)
	}
	for _, rawEntry := range entries {
		id, err := validateSideQuestEntry(rawEntry, source)
		if err != nil {
			return err
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("handoff_artifact_invalid: side_quests_approved duplicates %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateSideQuestEntry(rawEntry any, source string) (string, error) {
	entry, ok := rawEntry.(map[string]any)
	if !ok {
		return "", fmt.Errorf("handoff_artifact_invalid: side_quests_approved in %s contains a non-mapping entry", source)
	}
	for _, field := range []string{"id", "description", "strategy", "status"} {
		value, ok := entry[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("handoff_artifact_invalid: side_quests_approved in %s has no non-empty %q", source, field)
		}
	}
	id, ok := entry["id"].(string)
	if !ok {
		return "", fmt.Errorf("handoff_artifact_invalid: side_quests_approved in %s has no non-empty %q", source, "id")
	}
	return id, nil
}
