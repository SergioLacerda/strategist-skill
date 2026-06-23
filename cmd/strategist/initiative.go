package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initiativeRoot string

var initiativeCmd = &cobra.Command{
	Use:   "initiative",
	Short: "Show the skill provider occupying each mission slot",
	Long: `Show the skill configured for each Strategist mission slot (discovery, refinement, execution).

For each slot the command reports:
  - provider id
  - canonical role and specialization class (rankeado or base)
  - whether the local provider manifest exists at .strategist/skills/<provider>/skill.yaml`,
	RunE: runInitiative,
}

func runInitiative(_ *cobra.Command, _ []string) error {
	if initiativeRoot == "" {
		initiativeRoot = ".strategist"
	}

	raw, err := os.ReadFile(filepath.Join(initiativeRoot, "active.yaml"))
	if err != nil {
		return fmt.Errorf("initiative: read active.yaml: %w", err)
	}

	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("initiative: parse active.yaml: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		providerID := cfg.Slots[slot]
		role, class, status := providerRow(initiativeRoot, slot, providerID)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\n", slot, providerID, role, class, status); err != nil {
			return fmt.Errorf("initiative: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("initiative: flush output: %w", err)
	}
	return nil
}

func slotDefaultRole(slot string) string {
	switch slot {
	case "discovery":
		return "Ranger"
	case "refinement":
		return "Archivist"
	case "execution":
		return "Sniper"
	default:
		return strings.ToUpper(slot[:1]) + slot[1:]
	}
}

func canonicalRoleLabel(raw string) string {
	switch strings.ToLower(raw) {
	case "ranger":
		return "Ranger"
	case "archivist":
		return "Archivist"
	case "sniper":
		return "Sniper"
	default:
		if len(raw) == 0 {
			return raw
		}
		return strings.ToUpper(raw[:1]) + raw[1:]
	}
}

// providerRow derives the display columns for one slot row.
func providerRow(strategistDir, slot, providerID string) (role, class, manifestStatus string) {
	role = slotDefaultRole(slot)
	class = "(base)"
	manifestStatus = "⚠ manifest ausente"

	if providerID == "" {
		return
	}

	data, err := os.ReadFile(filepath.Join(strategistDir, "skills", providerID, "skill.yaml"))
	if err != nil {
		return
	}

	var m domain.ProviderManifest
	if yaml.Unmarshal(data, &m) != nil {
		return
	}

	if cr := m.SpecializationTaxonomy.CanonicalRole; cr != "" {
		role = canonicalRoleLabel(cr)
	}
	if m.ProviderClass == "rankeado" || m.SpecializationTaxonomy.ProviderClass == "rankeado" {
		class = "rankeado"
	}
	manifestStatus = "✓ manifest OK"
	return
}

func init() {
	initiativeCmd.Flags().StringVar(&initiativeRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	rootCmd.AddCommand(initiativeCmd)
}
