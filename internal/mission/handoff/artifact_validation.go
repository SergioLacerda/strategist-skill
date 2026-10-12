package handoff

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func readArtifact(path string) ([]byte, error) {
	content, err := os.ReadFile(path) //nolint:gosec // callers resolve paths inside the mission workspace
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, fmt.Errorf("artifact is empty")
	}
	return content, nil
}

func parseFrontmatter(content []byte) (map[string]any, []byte, error) {
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---\n")) {
		return nil, nil, fmt.Errorf("frontmatter is missing")
	}
	closing := bytes.Index(trimmed[4:], []byte("\n---"))
	if closing < 0 {
		return nil, nil, fmt.Errorf("frontmatter is unclosed")
	}
	closing += 4
	frontmatter := map[string]any{}
	if err := yaml.Unmarshal(trimmed[4:closing], &frontmatter); err != nil {
		return nil, nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return frontmatter, trimmed[closing+4:], nil
}

func validateIdentity(frontmatter map[string]any, missionID string, statuses []string) error {
	gotMission, ok := frontmatter["mission_id"].(string)
	if !ok {
		gotMission = ""
	}
	if strings.TrimSpace(gotMission) == "" || gotMission != missionID {
		return fmt.Errorf("mission_id %q does not match %q", gotMission, missionID)
	}
	status, ok := frontmatter["mission_status"].(string)
	if !ok {
		status = ""
	}
	for _, allowed := range statuses {
		if status == allowed {
			return nil
		}
	}
	return fmt.Errorf("mission_status %q is not valid for this boundary", status)
}

func hasSection(body []byte, section string) bool {
	want := "## " + section
	for _, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

func validateTaskClassifications(tasks []byte) error {
	const marker = "- ["
	seen := false
	for _, line := range strings.Split(string(tasks), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, marker) {
			continue
		}
		seen = true
		if !hasTaskClassification(line) {
			return fmt.Errorf("task has no explicit classification: %q", line)
		}
	}
	if !seen {
		return fmt.Errorf("no classified tasks found")
	}
	return nil
}

var taskClassifications = []string{"analysis_artifact", "implementation_handoff", "documentation_target", "out_of_scope"}

// hasTaskClassification accepts both the bare `[classification]` token and the
// `[task_type: classification]` form OpenSpec-generated task lists use.
func hasTaskClassification(line string) bool {
	for _, classification := range taskClassifications {
		if strings.Contains(line, "["+classification+"]") || strings.Contains(line, "[task_type: "+classification+"]") {
			return true
		}
	}
	return false
}
