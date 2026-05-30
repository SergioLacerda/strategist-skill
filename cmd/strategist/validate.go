package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateRoot string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the .strategist/ configuration tree",
	Long: `Validate all configuration files inside a .strategist/ directory.

Checks performed:
  - active.yaml: exists, valid YAML, required fields (mode, roles_config)
  - personas/*.yaml: each has tone_directive and phase_labels
  - roles/*.yaml: each has discovery, refinement, execution slots
  - knowledge.index.yaml: if present, valid YAML`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if validateRoot == "" {
			validateRoot = ".strategist"
		}

		var errs []string
		checks := 0

		// 1. active.yaml
		activeErr := validateActiveYAML(filepath.Join(validateRoot, "active.yaml"))
		checks++
		if activeErr != nil {
			errs = append(errs, fmt.Sprintf("active.yaml: %v", activeErr))
		}

		// 2. personas/*.yaml
		personaErrs, personaChecks := validatePersonasDir(filepath.Join(validateRoot, "personas"))
		checks += personaChecks
		errs = append(errs, personaErrs...)

		// 3. roles/*.yaml
		roleErrs, roleChecks := validateRolesDir(filepath.Join(validateRoot, "roles"))
		checks += roleChecks
		errs = append(errs, roleErrs...)

		// 4. knowledge.index.yaml (optional)
		kiPath := filepath.Join(validateRoot, "knowledge.index.yaml")
		if _, err := os.Stat(kiPath); err == nil {
			checks++
			if kiErr := validateYAMLFile(kiPath); kiErr != nil {
				errs = append(errs, fmt.Sprintf("knowledge.index.yaml: %v", kiErr))
			}
		}

		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
			}
			return fmt.Errorf("validate: %d error(s) in %s", len(errs), validateRoot)
		}

		fmt.Printf("[Strategist] validate OK — %d check(s) passed (%s)\n", checks, validateRoot)
		return nil
	},
}

func validateActiveYAML(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found")
		}
		return fmt.Errorf("read: %w", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	for _, field := range []string{"mode", "roles_config"} {
		if _, ok := cfg[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	if mode, ok := cfg["mode"].(string); ok {
		if mode != "pragmatic" && mode != "epic" {
			return fmt.Errorf("invalid mode %q (must be pragmatic or epic)", mode)
		}
	}

	return nil
}

func validatePersonasDir(dir string) (errs []string, checks int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("personas/: %v", err)}, 0
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		checks++
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("personas/%s: read: %v", e.Name(), err))
			continue
		}
		var p map[string]any
		if err := yaml.Unmarshal(raw, &p); err != nil {
			errs = append(errs, fmt.Sprintf("personas/%s: invalid YAML: %v", e.Name(), err))
			continue
		}
		for _, field := range []string{"tone_directive", "phase_labels"} {
			if _, ok := p[field]; !ok {
				errs = append(errs, fmt.Sprintf("personas/%s: missing field: %s", e.Name(), field))
			}
		}
	}
	return errs, checks
}

func validateRolesDir(dir string) (errs []string, checks int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("roles/: %v", err)}, 0
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		checks++
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("roles/%s: read: %v", e.Name(), err))
			continue
		}
		var r map[string]any
		if err := yaml.Unmarshal(raw, &r); err != nil {
			errs = append(errs, fmt.Sprintf("roles/%s: invalid YAML: %v", e.Name(), err))
			continue
		}
		for _, slot := range []string{"discovery", "refinement", "execution"} {
			if _, ok := r[slot]; !ok {
				errs = append(errs, fmt.Sprintf("roles/%s: missing slot: %s", e.Name(), slot))
			}
		}
	}
	return errs, checks
}

func validateYAMLFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	return nil
}

func init() {
	validateCmd.Flags().StringVar(&validateRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
