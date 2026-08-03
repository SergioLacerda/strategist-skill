package treasure

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var taskTitleRe = regexp.MustCompile(`^##\s+Task\s+\d+\s*[—-]\s*(.+)$`)

// ParseMissionTasks extracts task titles and the side_quests_approved: block from a
// tasks.md file. The block is YAML embedded in markdown with optional fenced YAML
// and backtick-wrapped scalars (e.g. “ `SQ-001` “) — backticks are stripped before
// unmarshaling.
func ParseMissionTasks(missionID, path string) (ScannedMission, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: mission scan reads task files selected from the analysis root
	if err != nil {
		return ScannedMission{}, fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")

	sqs, err := parseMissionSideQuests(lines, path)
	if err != nil {
		return ScannedMission{}, err
	}
	return ScannedMission{MissionID: missionID, SQs: sqs, TaskTitles: parseMissionTaskTitles(lines)}, nil
}

func parseMissionTaskTitles(lines []string) []string {
	var taskTitles []string
	for _, line := range lines {
		if m := taskTitleRe.FindStringSubmatch(line); m != nil {
			taskTitles = append(taskTitles, strings.TrimSpace(m[1]))
		}
	}
	return taskTitles
}

func parseMissionSideQuests(lines []string, path string) ([]SQEntry, error) {
	startIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "side_quests_approved:") {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return nil, nil
	}
	return parseSideQuestBlock(lines[startIdx:sideQuestBlockEnd(lines, startIdx)], path)
}

func sideQuestBlockEnd(lines []string, startIdx int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			return i
		}
	}
	return len(lines)
}

func parseSideQuestBlock(lines []string, path string) ([]SQEntry, error) {
	block := normalizeSideQuestYAML(lines)
	block = normalizeLegacySideQuestFields(block)
	block = strings.ReplaceAll(block, "`", "")
	var parsed struct {
		SideQuestsApproved []SQEntry `yaml:"side_quests_approved"`
	}
	if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
		return nil, fmt.Errorf("parse side_quests_approved in %s: %w", path, err)
	}
	return parsed.SideQuestsApproved, nil
}

func normalizeSideQuestYAML(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	contentStart := firstSideQuestContentLine(lines)
	if contentStart >= len(lines) || !isFenceLine(lines[contentStart]) {
		return strings.Join(lines, "\n")
	}
	return lines[0] + "\n" + strings.Join(fencedBody(lines, contentStart), "\n")
}

func firstSideQuestContentLine(lines []string) int {
	contentStart := 1
	for contentStart < len(lines) && strings.TrimSpace(lines[contentStart]) == "" {
		contentStart++
	}
	return contentStart
}

func fencedBody(lines []string, contentStart int) []string {
	var body []string
	for i := contentStart + 1; i < len(lines); i++ {
		if isFenceLine(lines[i]) {
			break
		}
		body = append(body, lines[i])
	}
	return body
}

func isFenceLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return false
	}
	lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	return lang == "" || lang == "yaml" || lang == "yml"
}

func normalizeLegacySideQuestFields(block string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 && strings.HasPrefix(trimmed, "- ") && looksLikeMappingField(strings.TrimPrefix(trimmed, "- ")) {
			trimmed = strings.TrimPrefix(trimmed, "- ")
			lines[i] = line[:indent] + trimmed
		}
		if isNoneDependencyField(trimmed) {
			lines[i] = line[:indent] + "dependencies: []"
		}
	}
	return strings.Join(lines, "\n")
}

func looksLikeMappingField(value string) bool {
	key, _, ok := strings.Cut(value, ":")
	if !ok || key == "" {
		return false
	}
	for _, r := range key {
		if !isMappingKeyRune(r) {
			return false
		}
	}
	return true
}

func isMappingKeyRune(r rune) bool {
	return isASCIILetter(r) || (r >= '0' && r <= '9') || r == '_' || r == '-'
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isNoneDependencyField(trimmed string) bool {
	key, value, ok := strings.Cut(trimmed, ":")
	if !ok || strings.TrimSpace(key) != "dependencies" {
		return false
	}
	value = strings.ToLower(strings.Trim(strings.TrimSpace(value), "`\"'"))
	return value == "none" || value == "null" || value == "n/a"
}
