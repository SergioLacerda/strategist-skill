package eval

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScanWarning is an eval-owned warning for an inconsistent mission fixture.
// Harvesting is a filesystem concern and must not import the Treasure Chest
// implementation package.
type ScanWarning struct {
	Path string
	Err  error
}

func (w ScanWarning) Error() string {
	if w.Path == "" {
		return w.Err.Error()
	}
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
}

type scannedMission struct{ MissionID string }

type sideQuestEntry struct {
	ID              string   `yaml:"id"`
	Description     string   `yaml:"description"`
	Strategy        string   `yaml:"strategy"`
	EstimatedImpact string   `yaml:"estimated_impact"`
	Dependencies    []string `yaml:"dependencies"`
	Status          string   `yaml:"status"`
}

func scanMissionsTolerant(basePath string) ([]scannedMission, []ScanWarning, error) {
	var missions []scannedMission
	var warnings []ScanWarning
	for _, name := range []string{"refined", "done"} {
		found, skipped, err := scanMissionDir(filepath.Join(basePath, name))
		if err != nil {
			return nil, nil, err
		}
		missions = append(missions, found...)
		warnings = append(warnings, skipped...)
	}
	sort.Slice(missions, func(i, j int) bool { return missions[i].MissionID < missions[j].MissionID })
	return missions, warnings, nil
}

func scanMissionDir(dir string) ([]scannedMission, []ScanWarning, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var missions []scannedMission
	var warnings []ScanWarning
	for _, entry := range entries {
		mission, warning, ok := scanMissionEntry(dir, entry)
		if warning != nil {
			warnings = append(warnings, *warning)
		}
		if ok {
			missions = append(missions, mission)
		}
	}
	return missions, warnings, nil
}

func scanMissionEntry(dir string, entry os.DirEntry) (scannedMission, *ScanWarning, bool) {
	if !entry.IsDir() {
		return scannedMission{}, nil, false
	}
	path := filepath.Join(dir, entry.Name(), "tasks.md")
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return scannedMission{}, nil, false
	}
	if err != nil {
		return scannedMission{}, &ScanWarning{Path: path, Err: err}, false
	}
	if err := parseHarvestTasks(path); err != nil {
		return scannedMission{}, &ScanWarning{Path: path, Err: err}, false
	}
	return scannedMission{MissionID: entry.Name()}, nil, true
}

func parseHarvestTasks(path string) error {
	raw, err := os.ReadFile(path) //nolint:gosec // path is selected beneath the declared analysis root
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	block, ok := harvestBlock(string(raw))
	if !ok {
		return nil
	}
	block = normalizeHarvestBlock(block)
	var parsed struct {
		SideQuestsApproved []sideQuestEntry `yaml:"side_quests_approved"`
	}
	if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
		return fmt.Errorf("parse side_quests_approved in %s: %w", path, err)
	}
	return nil
}

func harvestBlock(raw string) (string, bool) {
	lines := strings.Split(raw, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "side_quests_approved:") {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

func firstFenceLine(block string) int {
	lines := strings.Split(block, "\n")
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if trimmed == "```" || trimmed == "```yaml" || trimmed == "```yml" {
			return i
		}
		return -1
	}
	return -1
}

func fencedBody(block string, start int) string {
	lines := strings.Split(block, "\n")
	var body []string
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			break
		}
		body = append(body, lines[i])
	}
	return strings.Join(body, "\n")
}
