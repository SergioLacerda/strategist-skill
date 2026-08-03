package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// RuntimeProfile holds the resolved persona information for the startup banner.
type RuntimeProfile struct {
	ProfileMode     string
	ProfilePath     string
	ActiveYAMLPath  string
	PersonaResolved string
	Reason          string
	OutputProfile   string
}

// resolveRuntimeProfile attempts to load active.yaml and the referenced persona
// from strategistDir. All failures are non-blocking — a fallback profile is returned.
func resolveRuntimeProfile(strategistDir string) RuntimeProfile {
	base := RuntimeProfile{
		ProfileMode:     "local",
		ProfilePath:     strategistDir,
		ActiveYAMLPath:  filepath.Join(strategistDir, "active.yaml"),
		PersonaResolved: "unknown",
		Reason:          "active_yaml_missing",
		OutputProfile:   "default",
	}

	raw, err := os.ReadFile(base.ActiveYAMLPath) //nolint:gosec // G304: active.yaml path is derived from the discovered .strategist root
	if err != nil {
		return base
	}

	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		base.Reason = "active_yaml_invalid_yaml"
		return base
	}
	if cfg.Mode == "" {
		base.Reason = "mode_missing"
		return base
	}

	personaPath := filepath.Join(strategistDir, "personas", cfg.Mode+".yaml")
	personaRaw, err := os.ReadFile(personaPath) //nolint:gosec // G304: persona path is derived from the active runtime mode
	if err != nil {
		base.Reason = "persona_file_missing"
		return base
	}

	var persona domain.PersonaConfig
	if err := yaml.Unmarshal(personaRaw, &persona); err != nil {
		base.Reason = "persona_invalid_yaml"
		return base
	}
	if persona.Diagnostics.PipelineHeader == "" || persona.Diagnostics.BootstrapOrigin == "" {
		base.Reason = "persona_diagnostics_missing"
		return base
	}

	return RuntimeProfile{
		ProfileMode:     "local",
		ProfilePath:     strategistDir,
		ActiveYAMLPath:  base.ActiveYAMLPath,
		PersonaResolved: cfg.Mode,
		Reason:          "active_yaml",
		OutputProfile:   "default",
	}
}

// renderPersonaHeader returns the pipeline_header template rendered with banner values,
// trimming trailing whitespace/newlines for inline emission.
func renderPersonaHeader(tpl, missionID, mode, output string) string {
	r := strings.NewReplacer(
		"{id}", missionID,
		"{mode}", mode,
		"{persona}", mode,
		"{output}", output,
	)
	return strings.TrimRight(r.Replace(tpl), "\n ")
}
