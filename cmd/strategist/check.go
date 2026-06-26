package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var checkRoot string

// slotContract maps slot names to their required risk_score contract.
var slotContract = map[string]string{
	"discovery":  "write_analysis",
	"refinement": "write_analysis",
	"execution":  "controlled",
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Pre-mission slot validation",
	Long: `Validate that slot providers declared in active.yaml are installed and
satisfy their risk_score contracts.

Checks performed:
  - active.yaml is present and parseable
  - For each slot (discovery, refinement, execution):
      • skills/<provider>/skill.yaml exists (skill provider), OR
        roles/<provider>.yaml exists with matching slot field (native role)
      • skill providers must declare the correct risk_score for the slot contract:
        discovery/refinement → write_analysis, execution → controlled
      • native roles are accepted by slot field match; no risk_score check`,
	RunE: func(_ *cobra.Command, _ []string) error {
		root := checkRoot
		if root == "" {
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return fmt.Errorf("[Strategist] check=blocked reason=cwd_error: %w", cwdErr)
			}
			discovered, _, discErr := findStrategistRoot(cwd)
			if discErr != nil {
				return fmt.Errorf("[Strategist] check=blocked reason=runtime_not_found\n→ Run: strategist install")
			}
			root = discovered
		}

		activeYAML := filepath.Join(root, "active.yaml")
		raw, err := os.ReadFile(activeYAML)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_not_found\n→ Run: strategist install")
			}
			return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_read_error: %w", err)
		}

		var cfg domain.ActiveConfig
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("[Strategist] check=blocked reason=active_yaml_invalid_yaml: %w", err)
		}

		providers := map[string]string{
			"discovery":  cfg.Slots["discovery"],
			"refinement": cfg.Slots["refinement"],
			"execution":  cfg.Slots["execution"],
		}

		var errs []string
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			provider := providers[slot]
			if provider == "" {
				errs = append(errs, fmt.Sprintf("slot %s: no provider configured in active.yaml", slot))
				continue
			}

			skillPath := filepath.Join(root, "skills", provider, "skill.yaml")
			skillRaw, readErr := os.ReadFile(skillPath)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					// Fallback: accept native roles declared in roles/<provider>.yaml.
					rolePath := filepath.Join(root, "roles", provider+".yaml")
					roleRaw, roleErr := os.ReadFile(rolePath)
					if roleErr == nil {
						var roleDef struct {
							Slot string `yaml:"slot"`
						}
						if yamlErr := yaml.Unmarshal(roleRaw, &roleDef); yamlErr == nil {
							if roleDef.Slot == slot {
								continue // valid native role for this slot
							}
							errs = append(errs, fmt.Sprintf("slot %s: role %q declares slot=%q (mismatch)", slot, provider, roleDef.Slot))
							continue
						}
					}
					errs = append(errs, fmt.Sprintf("slot %s: provider %q not installed (missing %s)", slot, provider, skillPath))
				} else {
					errs = append(errs, fmt.Sprintf("slot %s: read %s: %v", slot, skillPath, readErr))
				}
				continue
			}

			var skillDef struct {
				RiskScore string `yaml:"risk_score"`
			}
			if yamlErr := yaml.Unmarshal(skillRaw, &skillDef); yamlErr != nil {
				errs = append(errs, fmt.Sprintf("slot %s: provider %q skill.yaml invalid: %v", slot, provider, yamlErr))
				continue
			}

			required := slotContract[slot]
			if skillDef.RiskScore != required {
				errs = append(errs, fmt.Sprintf("slot %s: provider %q has risk_score=%q but slot requires %q — preflight will block", slot, provider, skillDef.RiskScore, required))
			}
		}

		// Validate active persona.
		if cfg.Mode == "" {
			errs = append(errs, "active.yaml: mode is empty — must be epic or pragmatic")
		} else {
			personaPath := filepath.Join(root, "personas", cfg.Mode+".yaml")
			personaRaw, personaErr := os.ReadFile(personaPath)
			if personaErr != nil {
				errs = append(errs, fmt.Sprintf("persona: mode=%q file missing (%s)", cfg.Mode, personaPath))
			} else {
				var persona domain.PersonaConfig
				if yamlErr := yaml.Unmarshal(personaRaw, &persona); yamlErr != nil {
					errs = append(errs, fmt.Sprintf("persona: mode=%q invalid yaml: %v", cfg.Mode, yamlErr))
				} else if rtErr := persona.ValidateForRuntime(); rtErr != nil {
					errs = append(errs, fmt.Sprintf("persona: mode=%q %v", cfg.Mode, rtErr))
				}
			}
		}

		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
			}
			return fmt.Errorf("[Strategist] check=failed errors=%d root=%s", len(errs), root)
		}

		fmt.Printf("[Strategist] check=ok slots=[discovery:%s, refinement:%s, execution:%s] persona=%s root=%s\n",
			providers["discovery"], providers["refinement"], providers["execution"], cfg.Mode, root)
		return nil
	},
}

func init() {
	checkCmd.Flags().StringVar(&checkRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	rootCmd.AddCommand(checkCmd)
}
