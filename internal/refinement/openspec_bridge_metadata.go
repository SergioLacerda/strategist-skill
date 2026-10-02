package refinement

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"gopkg.in/yaml.v3"
)

func addMetadata(content []byte, input OpenSpecInput) ([]byte, error) {
	metadata := fmt.Sprintf("provider: openspec-propose\nprovider_change_id: %s\nprovider_runtime: %s\n", input.ChangeID, portableRuntimeRef(input))
	facts, err := handoffFactsYAML(input.HandoffFacts)
	if err != nil {
		return nil, err
	}
	metadata += facts
	text := string(content)
	containsFacts, err := frontmatterContainsKey(text, handoff.PolicyFactsKey)
	if err != nil {
		return nil, err
	}
	if containsFacts {
		return nil, fmt.Errorf("openspec bridge: pending analysis already carries %s; Archivist declares it at publication", handoff.PolicyFactsKey)
	}
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---"); end >= 0 {
			pos := 4 + end
			header := replaceMetadata(text[:pos], "mission_status", "archivist_done")
			return []byte(header + "\n" + metadata + text[pos:]), nil
		}
	}
	return []byte("---\nmission_id: " + input.MissionID + "\nmission_status: archivist_done\n" + metadata + "---\n\n" + text), nil
}

func frontmatterContainsKey(text, key string) (bool, error) {
	if !strings.HasPrefix(text, "---\n") {
		return false, nil
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return false, nil
	}
	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(text[4:4+end]), &frontmatter); err != nil {
		return false, fmt.Errorf("openspec bridge: parse pending frontmatter: %w", err)
	}
	_, ok := frontmatter[key]
	return ok, nil
}

// validateHandoffFacts rejects a facts mapping that handoff.ParsePolicyFacts
// would reject, before anything is published. A nil mapping is allowed.
func validateHandoffFacts(facts map[string]any) error {
	if facts == nil {
		return nil
	}
	if _, err := handoff.ParsePolicyFacts(map[string]any{handoff.PolicyFactsKey: facts}); err != nil {
		return fmt.Errorf("openspec bridge: %w", err)
	}
	return nil
}

// handoffFactsYAML renders the facts as a frontmatter block with stable key
// order, or "" when none were declared.
func handoffFactsYAML(facts map[string]any) (string, error) {
	if facts == nil {
		return "", nil
	}
	encoded, err := yaml.Marshal(map[string]any{handoff.PolicyFactsKey: facts})
	if err != nil {
		return "", fmt.Errorf("openspec bridge: encode %s: %w", handoff.PolicyFactsKey, err)
	}
	return string(encoded), nil
}

// portableRuntimeRef keeps durable package metadata independent of the
// absolute workspace checkout. The runtime remains an operational dependency
// resolved from the active Strategist root; this field is provenance, not a
// filesystem authority.
func portableRuntimeRef(input OpenSpecInput) string {
	relative, err := filepath.Rel(filepath.Dir(input.BasePath), input.RuntimeRoot)
	if err == nil && relative != "" && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".." {
		return filepath.ToSlash(relative)
	}
	return ".strategist/openspec"
}

func hasMissionIdentity(content []byte, missionID string) bool {
	needle := "mission_id: " + missionID
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == needle {
			return true
		}
	}
	return false
}

func replaceMetadata(header, key, value string) string {
	lines := strings.Split(header, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n")
		}
	}
	return header + "\n" + key + ": " + value
}
