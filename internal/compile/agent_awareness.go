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
	agPath := filepath.Join(projectRoot, ".antigravity", "antigravity-instructions.md")
	if _, err := os.Stat(agPath); err == nil {
		if err := upsertSection(agPath); err != nil {
			slog.Warn("[Strategist] agent awareness: antigravity update failed", "error", err)
		}
	}

	codexPath := filepath.Join(projectRoot, ".sdd", "seedlings", "codex.seed.json")
	if _, err := os.Stat(codexPath); err == nil {
		if err := upsertCodexSeed(codexPath); err != nil {
			slog.Warn("[Strategist] agent awareness: codex update failed", "error", err)
		}
	}

	claudePath := filepath.Join(projectRoot, ".claude", "claude-instructions.md")
	if _, err := os.Stat(claudePath); err == nil {
		if err := upsertSection(claudePath); err != nil {
			slog.Warn("[Strategist] agent awareness: claude-instructions update failed", "error", err)
		}
	}

	codexCmdPath := filepath.Join(projectRoot, ".codex", "commands.md")
	if _, err := os.Stat(codexCmdPath); err == nil {
		if err := upsertSection(codexCmdPath); err != nil {
			slog.Warn("[Strategist] agent awareness: codex commands update failed", "error", err)
		}
	}

	return nil
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
	newSection := strategistRuntimeDiscoverySection

	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		// Section absent — append.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + newSection + "\n"
	} else {
		// Section present — replace to next "## " header or EOF.
		after := content[idx+len(sectionHeader):]
		nextSection := strings.Index(after, "\n## ")
		var tail string
		if nextSection != -1 {
			tail = after[nextSection:]
		}
		content = content[:idx] + newSection + tail
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // G306
		return fmt.Errorf("upsert section: write %s: %w", path, err)
	}
	return nil
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
	var ctx []any
	if existing, ok := seed["required_context"].([]any); ok {
		ctx = existing
	}
	hasProt := false
	for _, v := range ctx {
		if v == protocolPath {
			hasProt = true
			break
		}
	}
	if !hasProt {
		ctx = append([]any{protocolPath}, ctx...)
	}
	seed["required_context"] = ctx

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
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf("marshal key %q: %w", k, err)
		}
		valBytes, err := json.MarshalIndent(m[k], "  ", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal value for key %q: %w", k, err)
		}
		buf.WriteString("\n  ")
		buf.Write(keyBytes)
		buf.WriteString(": ")
		buf.Write(valBytes)
	}
	if len(keys) > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
