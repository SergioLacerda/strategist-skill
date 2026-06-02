package install

import (
	"fmt"
	"os"
	"path/filepath"
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
	return frontmatter + skillContent
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

// installShimTo writes the shim under homeDir with the given skill content.
// skillContent is the raw content of ~/.strategist/SKILL.md. An empty string
// produces a shim with frontmatter only (used in error-path tests).
func installShimTo(homeDir, skillContent, skillRoot string) error {
	shimDir := filepath.Join(homeDir, ".claude", "skills", "strategist")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		return fmt.Errorf("mkdir shim dir: %w", err)
	}
	shimPath := filepath.Join(shimDir, "SKILL.md")
	content := []byte(generateShimContent(skillContent, skillRoot))
	if err := os.WriteFile(shimPath, content, 0o644); err != nil {
		return fmt.Errorf("write shim: %w", err)
	}
	return nil
}
