package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func formatCount(n int, unit string) string {
	if n < 0 {
		return "—"
	}
	plural := unit + "s"
	if unit == "missão" {
		plural = "missões"
	}
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// countEntries returns the number of entries in dir, or -1 if dir is absent.
func countEntries(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return -1
	}
	return len(entries)
}

// readLastMissionID reads the last line of outcomes.jsonl and returns the mission_id field.
// Returns "—" if the file is absent or the field cannot be parsed.
func readLastMissionID(path string) string {
	f, err := os.Open(path) //nolint:gosec // G304: path derived from install root
	if err != nil {
		return "—"
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read-only file; close error is not actionable

	lastLine := lastNonEmptyLine(f)
	if lastLine == "" {
		return "—"
	}
	return missionIDFromJSONLine(lastLine)
}

func lastNonEmptyLine(f *os.File) string {
	var lastLine string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lastLine = line
		}
	}
	return lastLine
}

func missionIDFromJSONLine(line string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "—"
	}
	if id, ok := obj["mission_id"].(string); ok && id != "" {
		return id
	}
	return "—"
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
