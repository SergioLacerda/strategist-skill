package compile

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	agentsRuntimeStartMarker = "<!-- strategist-runtime-start -->"
	agentsRuntimeEndMarker   = "<!-- strategist-runtime-end -->"
	strategistEntryPath      = ".strategist"
)

// ideAwareness registers the .strategist runtime with IDEs that discover skills
// via a workspace customizations root, when .strategist/ exists at projectRoot.
// Currently only Antigravity needs this — VSCode already discovers the Claude
// seed natively via its extension, so it needs no additional registration.
// Non-blocking: all failures are logged and skipped.
func ideAwareness(projectRoot string) error {
	if _, err := os.Stat(filepath.Join(projectRoot, ".strategist")); err != nil {
		return nil
	}

	agentsDir := filepath.Join(projectRoot, ".agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("ide awareness: mkdir %s: %w", agentsDir, err)
	}

	if err := upsertAgentsSkillsJSON(filepath.Join(agentsDir, "skills.json")); err != nil {
		slog.Warn("[Strategist] ide awareness: skills.json update failed", "error", err)
	}
	if err := upsertAgentsMarkdown(filepath.Join(agentsDir, "AGENTS.md")); err != nil {
		slog.Warn("[Strategist] ide awareness: AGENTS.md update failed", "error", err)
	}
	return nil
}

// upsertAgentsSkillsJSON ensures {"path": ".strategist"} is present in the
// entries array of the .agents/skills.json registry, creating the file if
// absent and preserving any other entries and top-level keys already present.
func upsertAgentsSkillsJSON(path string) error {
	doc, err := readAgentsSkillsDoc(path)
	if err != nil {
		return err
	}

	entries, ok := doc["entries"].([]any)
	if !ok {
		entries = nil
	}
	for _, e := range entries {
		if m, ok := e.(map[string]any); ok && m["path"] == strategistEntryPath {
			return nil // already registered, nothing to write
		}
	}
	doc["entries"] = append(entries, map[string]any{"path": strategistEntryPath})

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("agents skills.json: marshal %s: %w", path, err)
	}
	return writeFile(path, append(out, '\n'), "agents skills.json")
}

func readAgentsSkillsDoc(path string) (map[string]any, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path derived from projectRoot
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("agents skills.json: read %s: %w", path, err)
	}

	doc := map[string]any{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("agents skills.json: parse %s: %w", path, err)
	}
	return doc, nil
}

// upsertAgentsMarkdown upserts the delimited Strategist Runtime Discovery block
// into .agents/AGENTS.md. Creates the file with a minimal header if absent.
// If the delimiters are absent from existing content, appends the block.
func upsertAgentsMarkdown(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path derived from projectRoot
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("agents markdown: read %s: %w", path, err)
		}
		data = []byte("# Agent Rules\n")
	}

	content := upsertDelimitedSection(string(data), agentsRuntimeStartMarker, agentsRuntimeEndMarker, strategistRuntimeDiscoverySection)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return writeFile(path, []byte(content), "agents markdown")
}

// upsertDelimitedSection replaces the content between startMarker and endMarker
// with newBlock, wrapped in the markers. If the markers are absent (or malformed —
// end before start), a new delimited block is appended to the end of content.
func upsertDelimitedSection(content, startMarker, endMarker, newBlock string) string {
	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)
	block := startMarker + "\n" + newBlock + "\n" + endMarker

	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + block + "\n"
	}
	return content[:startIdx] + block + content[endIdx+len(endMarker):]
}
