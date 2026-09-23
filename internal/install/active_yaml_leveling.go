package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
	var b strings.Builder
	b.WriteString("\n# Model x effort per role: manual = the host's model and effort are used as-is\n")
	b.WriteString("# (never changed by Strategist); automatic = the LEVELING policy picks them per\n")
	b.WriteString("# role by estimated load.\nleveling:\n")
	fmt.Fprintf(&b, "  mode: %s\n", cfg.EffectiveMode())
	return b.String(), nil
}
