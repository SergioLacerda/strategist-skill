package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

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
