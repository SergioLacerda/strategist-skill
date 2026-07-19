package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func generateShimContent(skillContent, skillRoot string) string {
	frontmatter := `---
name: strategist
description: "Multi-phase mission orchestrator. Coordinates discovery, refinement, and execution through three pluggable slots."
`
	if skillRoot != "" {
		frontmatter += fmt.Sprintf("skill_root: %s\n", skillRoot)
	}
	frontmatter += `---

`
	return frontmatter + stripFrontmatter(skillContent)
}

// stripFrontmatter removes a leading YAML frontmatter block (---...---) from s,
// if present. Guards against embedded SKILL.md files that were accidentally
// committed with a frontmatter block from a previous install.
func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---") {
		return s
	}
	// Find the closing --- after the opening one.
	rest := s[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return s
	}
	// Skip past the closing --- and any trailing newline.
	after := rest[idx+4:]
	return strings.TrimLeft(after, "\n")
}

// installShim creates ~/.claude/skills/strategist/SKILL.md containing the full
// pipeline instructions so Claude receives them inline at skill invocation time.
func installShim(target string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	skillRoot := ""
	if target != "" {
		absTarget, absErr := filepath.Abs(target)
		if absErr != nil {
			return fmt.Errorf("resolve target: %w", absErr)
		}
		skillRoot = filepath.Join(absTarget, ".strategist")
	}
	return installShimTo(home, "", skillRoot)
}

// installShimTo writes the Claude shim under homeDir and any optional
// Gemini/Codex shims if their root directories exist.
// skillContent is the raw content of SKILL.md. An empty string produces a
// shim with frontmatter only (used in error-path tests).
func installShimTo(homeDir, skillContent, skillRoot string) error {
	shimPath := filepath.Join(homeDir, ".claude", "skills", "strategist", "SKILL.md")
	if err := writeShimFile(shimPath, skillContent, skillRoot); err != nil {
		return err
	}
	installOptionalShims(homeDir, skillContent, skillRoot)
	return nil
}

// installShimToPath writes the Claude shim to an explicit path, bypassing the
// default ~/.claude/skills/strategist/SKILL.md location. Used for --shim-path.
func installShimToPath(shimPath, skillContent, skillRoot string) error {
	return writeShimFile(shimPath, skillContent, skillRoot)
}

// writeShimFile creates parent directories and writes a SKILL.md shim file.
func writeShimFile(shimPath, skillContent, skillRoot string) error {
	if err := os.MkdirAll(filepath.Dir(shimPath), 0o755); err != nil {
		return fmt.Errorf("mkdir shim dir: %w", err)
	}
	content := []byte(generateShimContent(skillContent, skillRoot))
	if err := os.WriteFile(shimPath, content, 0o644); err != nil {
		return fmt.Errorf("write shim: %w", err)
	}
	return nil
}

// installOptionalShims installs shims for Gemini and Codex if their root
// directories exist under homeDir. Errors are silently swallowed — optional
// shims must never fail the main install flow.
func installOptionalShims(homeDir, skillContent, skillRoot string) {
	installGeminiShims(filepath.Join(homeDir, ".gemini"), skillContent, skillRoot)
	installCodexShim(filepath.Join(homeDir, ".codex"), skillContent, skillRoot)
}

func installGeminiShims(geminiRoot, skillContent, skillRoot string) {
	if !dirExists(geminiRoot) {
		return
	}
	for _, p := range geminiShimPaths(geminiRoot) {
		if err := writeShimFile(p, skillContent, skillRoot); err != nil {
			continue
		}
	}
}

func installCodexShim(codexRoot, skillContent, skillRoot string) {
	if !dirExists(codexRoot) {
		return
	}
	if err := writeShimFile(filepath.Join(codexRoot, "skills", "strategist", "SKILL.md"), skillContent, skillRoot); err != nil {
		return
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func geminiShimPaths(geminiRoot string) []string {
	return []string{
		filepath.Join(geminiRoot, "skills", "strategist", "SKILL.md"),
		filepath.Join(geminiRoot, "antigravity", "skills", "strategist", "SKILL.md"),
	}
}
