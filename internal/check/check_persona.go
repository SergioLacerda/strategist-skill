package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// validateActivePersona validates that active.yaml's mode references an
// existing, runtime-valid persona definition under personas/<mode>.yaml.
func validateActivePersona(root, mode string) []string {
	if mode == "" {
		return []string{"active.yaml: mode is empty — must be epic or pragmatic"}
	}
	personaPath := filepath.Join(root, "personas", mode+".yaml")
	personaRaw, err := os.ReadFile(personaPath) //nolint:gosec // G304: persona path is derived from active runtime mode
	if err != nil {
		return []string{fmt.Sprintf("persona: mode=%q file missing (%s)", mode, personaPath)}
	}
	var persona domain.PersonaConfig
	if err := yaml.Unmarshal(personaRaw, &persona); err != nil {
		return []string{fmt.Sprintf("persona: mode=%q invalid yaml: %v", mode, err)}
	}
	if err := persona.ValidateForRuntime(); err != nil {
		return []string{fmt.Sprintf("persona: mode=%q %v", mode, err)}
	}
	return nil
}
