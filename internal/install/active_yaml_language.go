package install

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// normalizeStandaloneLanguage upgrades legacy scalar templates while retaining
// the surrounding template comments and formatting. Wizard-generated configs
// already use this canonical per-role shape.
func normalizeStandaloneLanguage(data []byte) ([]byte, error) {
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse active.yaml: %w", err)
	}
	if err := validateStructuredLanguage(document["language"]); err != nil {
		return nil, err
	}
	if language, ok := document["language"]; ok {
		if _, structured := language.(map[string]any); structured {
			return data, nil
		}
	}
	return replaceStandaloneLanguage(data), nil
}

func validateStructuredLanguage(language any) error {
	if language == nil {
		return nil
	}
	if _, ok := language.(string); ok {
		// Legacy scalar templates are upgraded by the caller.
		return nil
	}
	structured, ok := language.(map[string]any)
	if !ok {
		return fmt.Errorf("language must be a scalar legacy value or a complete role map")
	}
	return validateStandaloneLanguage(structured)
}

func replaceStandaloneLanguage(data []byte) []byte {
	const replacement = "language:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en"
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "language:") {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		replacementLines := strings.Split(replacement, "\n")
		for j := range replacementLines {
			replacementLines[j] = indent + replacementLines[j]
		}
		lines = append(lines[:i], append(replacementLines, lines[i+1:]...)...)
		return []byte(strings.Join(lines, "\n"))
	}
	return []byte(strings.TrimRight(string(data), "\n") + "\n\n" + replacement + "\n")
}

func validateStandaloneLanguage(language map[string]any) error {
	const supportedRoles = "ui, docs, chat, code"
	for _, role := range []string{"ui", "docs", "chat", "code"} {
		value, ok := language[role].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("language must define non-empty %s fields (%s)", role, supportedRoles)
		}
		if value != "en" && value != "pt-BR" {
			return fmt.Errorf("language.%s has unsupported value %q", role, value)
		}
	}
	return nil
}
