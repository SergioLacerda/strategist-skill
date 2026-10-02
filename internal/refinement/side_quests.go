package refinement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SideQuest is one typed entry of a package's side_quests_approved list.
type SideQuest struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Strategy    string `yaml:"strategy"`
	Status      string `yaml:"status"`
}

type sideQuestDocument struct {
	SideQuests []SideQuest `yaml:"side_quests_approved"`
}

var fencedYAMLBlock = regexp.MustCompile("(?s)```ya?ml\\s*\n(.*?)\n```")

// ReadSideQuests returns the side_quests_approved entries declared by the
// refined package <basePath>/refined/<missionID>: a fenced YAML block in
// tasks.md, or the analysis.md frontmatter. A YAML document that does not
// parse is skipped (a tolerant reader, as in the treasure scan); a missing
// package declares none.
func ReadSideQuests(basePath, missionID string) ([]SideQuest, error) {
	dir := filepath.Join(basePath, "refined", missionID)
	var out []SideQuest
	for _, name := range []string{"tasks.md", "analysis.md"} {
		found, err := readSideQuestFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	return out, nil
}

func readSideQuestFile(path string) ([]SideQuest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is <base_path>/refined/<mission_id>/<file>
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("refinement: read side quests: %w", err)
	}
	var out []SideQuest
	for _, doc := range yamlDocuments(string(raw)) {
		var parsed sideQuestDocument
		if yaml.Unmarshal([]byte(doc), &parsed) == nil {
			out = append(out, parsed.SideQuests...)
		}
	}
	return out, nil
}

// FindSideQuest returns the entry with the given id.
func FindSideQuest(quests []SideQuest, id string) (SideQuest, bool) {
	for _, q := range quests {
		if q.ID == id {
			return q, true
		}
	}
	return SideQuest{}, false
}

func yamlDocuments(text string) []string {
	var docs []string
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		if front, _, found := strings.Cut(rest, "\n---"); found {
			docs = append(docs, front)
		}
	}
	for _, m := range fencedYAMLBlock.FindAllStringSubmatch(text, -1) {
		docs = append(docs, m[1])
	}
	return docs
}
