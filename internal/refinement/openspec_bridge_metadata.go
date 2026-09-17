package refinement

import (
	"fmt"
	"path/filepath"
	"strings"
)

func addMetadata(content []byte, input OpenSpecInput) []byte {
	metadata := fmt.Sprintf("provider: openspec-propose\nprovider_change_id: %s\nprovider_runtime: %s\n", input.ChangeID, filepath.ToSlash(input.RuntimeRoot))
	text := string(content)
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---"); end >= 0 {
			pos := 4 + end
			header := replaceMetadata(text[:pos], "mission_status", "archivist_done")
			return []byte(header + "\n" + metadata + text[pos:])
		}
	}
	return []byte("---\nmission_id: " + input.MissionID + "\nmission_status: archivist_done\n" + metadata + "---\n\n" + text)
}

func hasMissionIdentity(content []byte, missionID string) bool {
	needle := "mission_id: " + missionID
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == needle {
			return true
		}
	}
	return false
}

func replaceMetadata(header, key, value string) string {
	lines := strings.Split(header, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n")
		}
	}
	return header + "\n" + key + ": " + value
}
