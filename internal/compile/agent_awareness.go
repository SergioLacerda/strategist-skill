package compile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const strategistRuntimeDiscoverySection = `## Strategist Runtime Discovery

When ` + "`" + `.strategist/` + "`" + ` is present in the workspace, Strategist is active. The parent
agent MUST NOT solve the user's task directly — it is the orchestrator shell, not the
implementer:

1. Run ` + "`" + `strategist check` + "`" + ` — if it fails: emit the CLI output and stop
2. Read ` + "`" + `.strategist/agent-protocol.md` + "`" + ` — full role and pipeline protocol
3. Read ` + "`" + `.strategist/SKILL.md` + "`" + ` — contracts and path model
4. Read ` + "`" + `.strategist/skill.yaml` + "`" + ` — slot mapping

Mandatory restrictions after bootstrap:
- Do not skip phases unless the internal routing contract selected Quick Draw or Critical Hit
- Do not perform source-code mutation; Strategist produces analysis, documentation, and handoff artifacts
- Route selection is internal to Strategist; invoke the skill with the request context
- If a configured slot/provider cannot be invoked, emit ` + "`" + `error=role_invocation_failed` + "`" + ` with slot and provider — do not fall back to solving the phase directly

When ` + "`" + `.strategist/` + "`" + ` is absent:
→ emit ` + "`" + `error=not_installed` + "`" + `, instruct the user to run ` + "`" + `strategist install` + "`" + `, stop.

` + "`" + `.sdd/` + "`" + ` is governance — it is not part of the Strategist runtime.
` + "`" + `.strategist/agent-protocol.md` + "`" + ` is the runtime authority for agent behavior.
Discovery contract: ` + "`" + `.strategist/provider-discovery.md` + "`"

// AgentAwareness upserts per-agent entrypoint files at projectRoot if they already exist.
// Does not create files that do not exist. Failures per file are logged and skipped —
// this function always returns nil (non-blocking by contract).
func AgentAwareness(projectRoot string) error {
	upsertIfExists(filepath.Join(projectRoot, ".antigravity", "antigravity-instructions.md"), upsertSection, "antigravity")
	upsertIfExists(filepath.Join(projectRoot, ".sdd", "seedlings", "codex.seed.json"), upsertCodexSeed, "codex")
	upsertIfExists(filepath.Join(projectRoot, ".claude", "claude-instructions.md"), upsertSection, "claude-instructions")
	upsertIfExists(filepath.Join(projectRoot, ".codex", "commands.md"), upsertSection, "codex commands")
	return nil
}

func upsertIfExists(path string, update func(string) error, label string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	if err := update(path); err != nil {
		slog.Warn("[Strategist] agent awareness: "+label+" update failed", "error", err)
	}
}

// RefreshAgentAwareness is the single coordinating entry point for both the
// compile command and the install wizard/silent paths.
// tplBytes is the content of the agent-protocol.md template (read by the caller
// from the embed FS so this package does not depend on internal/embed).
// It generates or upserts <strategistRoot>/agent-protocol.md and then runs
// AgentAwareness for all per-agent entrypoint files.
// All sub-steps are non-blocking: failures are logged as warnings.
// Returns whether agent-protocol.md was successfully written, for caller output.
func RefreshAgentAwareness(strategistRoot, projectRoot, version string, tplBytes []byte) (protocolOK bool) {
	if len(tplBytes) == 0 {
		slog.Warn("[Strategist] agent-protocol template not found or empty")
	} else if err := AgentProtocol(strategistRoot, tplBytes, version); err != nil {
		slog.Warn("[Strategist] agent-protocol generation failed", "error", err)
	} else {
		protocolOK = true
	}

	if err := AgentAwareness(projectRoot); err != nil {
		slog.Warn("[Strategist] agent awareness update skipped", "error", err)
	}

	return protocolOK
}

// upsertSection replaces the "## Strategist Runtime Discovery" section in path.
// If the section is absent, appends it. Other content is preserved.
// Used for all markdown-based agent entrypoint files (antigravity, claude, codex commands).
func upsertSection(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path derived from projectRoot
	if err != nil {
		return fmt.Errorf("upsert section: read %s: %w", path, err)
	}

	content := string(data)
	const sectionHeader = "## Strategist Runtime Discovery"
	content = upsertMarkdownSection(content, sectionHeader, strategistRuntimeDiscoverySection)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("upsert section: write %s: %w", path, err)
	}
	return nil
}

func upsertMarkdownSection(content, sectionHeader, newSection string) string {
	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		return appendMarkdownSection(content, newSection)
	}
	after := content[idx+len(sectionHeader):]
	return content[:idx] + newSection + markdownTail(after)
}

func appendMarkdownSection(content, newSection string) string {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + newSection + "\n"
}

func markdownTail(after string) string {
	nextSection := strings.Index(after, "\n## ")
	if nextSection == -1 {
		return ""
	}
	return after[nextSection:]
}

// upsertCodexSeed updates required_context and on_strategist_invoke in a codex.seed.json.
// Prepends ".strategist/agent-protocol.md" to required_context exactly once (idempotent).
// Adds/updates the on_strategist_invoke field.
func upsertCodexSeed(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return fmt.Errorf("codex seed: read %s: %w", path, err)
	}

	var seed map[string]any
	if err := json.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("codex seed: parse %s: %w", path, err)
	}

	const protocolPath = ".strategist/agent-protocol.md"
	seed["required_context"] = requiredContextWithProtocol(seed["required_context"], protocolPath)

	seed["on_strategist_invoke"] = map[string]any{
		"header":                    "Strategist Active",
		"preflight":                 "strategist check",
		"protocol":                  ".strategist/agent-protocol.md",
		"on_not_installed":          "emit error=not_installed and stop",
		"role_lock":                 "you are the Strategist orchestrator, not a general coding agent for this turn — do not solve the task directly",
		"allowed_actions":           []any{"bootstrap", "route", "invoke_providers", "present_gates", "relay_outputs", "report_blocked_states"},
		"forbidden_actions":         []any{"direct_discovery", "direct_refinement", "direct_execution", "code_or_test_mutation", "git_mutation", "provider_fallback"},
		"on_role_invocation_failed": "emit error=role_invocation_failed with slot and provider, then stop",
	}

	out, err := marshalSortedJSON(seed)
	if err != nil {
		return fmt.Errorf("codex seed: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("codex seed: write %s: %w", path, err)
	}
	return nil
}

func requiredContextWithProtocol(raw any, protocolPath string) []any {
	ctx := requiredContext(raw)
	if containsAnyString(ctx, protocolPath) {
		return ctx
	}
	return append([]any{protocolPath}, ctx...)
}

func requiredContext(raw any) []any {
	ctx, ok := raw.([]any)
	if !ok {
		return nil
	}
	return ctx
}

func containsAnyString(values []any, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}

// marshalSortedJSON marshals a map[string]any to indented JSON with keys sorted
// alphabetically, producing deterministic output across multiple runs.
func marshalSortedJSON(m map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if err := writeSortedJSONEntry(&buf, m, k, i > 0); err != nil {
			return nil, err
		}
	}
	if len(keys) > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func writeSortedJSONEntry(buf *bytes.Buffer, m map[string]any, key string, comma bool) error {
	if comma {
		buf.WriteByte(',')
	}
	keyBytes, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("marshal key %q: %w", key, err)
	}
	valBytes, err := json.MarshalIndent(m[key], "  ", "  ")
	if err != nil {
		return fmt.Errorf("marshal value for key %q: %w", key, err)
	}
	buf.WriteString("\n  ")
	buf.Write(keyBytes)
	buf.WriteString(": ")
	buf.Write(valBytes)
	return nil
}
