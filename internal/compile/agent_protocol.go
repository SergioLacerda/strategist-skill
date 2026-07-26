package compile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

type agentProtocolData struct {
	Version     string
	GeneratedAt string
	Slots       agentProtocolSlots
}

type agentProtocolSlots struct {
	Discovery  string
	Refinement string
	Execution  string
}

// agentProtocol generates or upserts <root>/agent-protocol.md from templateBytes,
// stamping runtime values from <root>/active.yaml.
//
// If the file exists and has valid frontmatter: replaces the body section, updates
// version and generated_at in the existing frontmatter, and preserves any extra
// frontmatter fields the user may have added.
// If the file is absent or frontmatter is malformed: writes the rendered output entirely.
func agentProtocol(root string, templateBytes []byte, version string) error {
	var active domain.ActiveConfig
	if err := loadYAMLInto(filepath.Join(root, "active.yaml"), &active); err != nil {
		return fmt.Errorf("agent protocol: %w", err)
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)

	data := agentProtocolData{
		Version:     version,
		GeneratedAt: generatedAt,
		Slots: agentProtocolSlots{
			Discovery:  active.Slots["discovery"],
			Refinement: active.Slots["refinement"],
			Execution:  active.Slots["execution"],
		},
	}

	tpl, err := template.New("agent-protocol").Parse(string(templateBytes))
	if err != nil {
		return fmt.Errorf("agent protocol: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("agent protocol: render template: %w", err)
	}

	outPath := filepath.Join(root, "agent-protocol.md")
	return upsertAgentProtocol(outPath, buf.String(), version, generatedAt)
}

// upsertAgentProtocol writes rendered to path.
// If path exists with valid frontmatter: replaces the body after the closing "---",
// updates version and generated_at in the existing frontmatter, preserves extra fields.
// Otherwise: writes rendered entirely.
func upsertAgentProtocol(path, rendered, version, generatedAt string) error {
	existing, err := os.ReadFile(path) //nolint:gosec // G304: path derived from root, not user input
	if os.IsNotExist(err) {
		return writeFile(path, []byte(rendered), "agent protocol")
	}
	if err != nil {
		return fmt.Errorf("agent protocol: read %s: %w", path, err)
	}

	// Frontmatter format: "---\n<fields>\n---\n<body>"
	const openMarker = "---\n"
	const closeMarker = "\n---\n"
	content := string(existing)
	if !strings.HasPrefix(content, openMarker) {
		return writeFile(path, []byte(rendered), "agent protocol")
	}
	idx := strings.Index(content[len(openMarker):], closeMarker)
	if idx == -1 {
		return writeFile(path, []byte(rendered), "agent protocol")
	}
	fmEnd := len(openMarker) + idx + len(closeMarker) // first char after closing "---\n"
	existingFM := content[:fmEnd]

	// Update version and generated_at in the preserved frontmatter.
	existingFM = replaceYAMLLine(existingFM, "version:", version)
	existingFM = replaceYAMLLine(existingFM, "generated_at:", generatedAt)

	return writeFile(path, []byte(existingFM+renderedBodyOnly(rendered)), "agent protocol")
}

// writeFile wraps os.WriteFile with a context-aware error for wrapcheck compliance.
// errPrefix is prepended to the error message so each call site can keep its own
// wrapped-error text (e.g. "agent protocol", "upsert section", "codex seed").
func writeFile(path string, content []byte, errPrefix string) error {
	if err := runtimefs.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("%s: write %s: %w", errPrefix, path, err)
	}
	return nil
}

// replaceYAMLLine replaces the value of the first top-level YAML key matching
// "key " prefix. Returns s unchanged if the key is not found.
func replaceYAMLLine(s, key, value string) string {
	lines := strings.Split(s, "\n")
	prefix := key + " "
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = key + " " + value
			return strings.Join(lines, "\n")
		}
	}
	return s
}

// renderedBodyOnly returns the body section (content after the closing "---\n")
// of a rendered agent-protocol document. Returns rendered unchanged if the
// boundary is not found.
func renderedBodyOnly(rendered string) string {
	const openMarker = "---\n"
	const closeMarker = "\n---\n"
	if !strings.HasPrefix(rendered, openMarker) {
		return rendered
	}
	idx := strings.Index(rendered[len(openMarker):], closeMarker)
	if idx == -1 {
		return rendered
	}
	start := len(openMarker) + idx + len(closeMarker)
	return rendered[start:]
}
