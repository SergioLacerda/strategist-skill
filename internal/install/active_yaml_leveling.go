package install

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// levelingActiveYAML renders the optional `leveling:` block. An unanswered
// wizard step emits nothing, which readers treat as automatic.
func levelingActiveYAML(cfg domain.LevelingConfig) (string, error) {
	if strings.TrimSpace(cfg.Mode) == "" {
		return "", nil
	}
	if err := cfg.Validate(); err != nil {
		return "", fmt.Errorf("install: %w", err)
	}
	return renderLevelingActiveYAML(cfg), nil
}

func renderLevelingActiveYAML(cfg domain.LevelingConfig) string {
	var b strings.Builder
	b.WriteString("\n# Model x effort per role: automatic applies the LEVELING policy; manual uses\n")
	b.WriteString("# the values below (manual > host-reported > policy).\nleveling:\n")
	fmt.Fprintf(&b, "  mode: %s\n", cfg.EffectiveMode())
	if cfg.EffectiveMode() == domain.LevelingModeManual {
		b.WriteString(renderManualLevelingRoles(cfg))
	}
	return b.String()
}

func renderManualLevelingRoles(cfg domain.LevelingConfig) string {
	var b strings.Builder
	b.WriteString("  roles:\n")
	for _, role := range domain.LevelingRoleIDs() {
		choice, ok := cfg.Choice(role)
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "    %s:\n", role)
		writeLevelingChoice(&b, choice)
	}
	return b.String()
}

func writeLevelingChoice(b *strings.Builder, choice domain.LevelingRoleChoice) {
	if model := strings.TrimSpace(choice.Model); model != "" {
		fmt.Fprintf(b, "      model: %s\n", yamlScalar(model))
	}
	if effort := strings.ToLower(strings.TrimSpace(choice.Effort)); effort != "" {
		fmt.Fprintf(b, "      effort: %s\n", effort)
	}
}

func yamlScalar(value string) string {
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return strconv.Quote(value)
	}
	return strings.TrimSpace(string(encoded))
}
