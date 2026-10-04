package handoffconsumer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// document is a markdown artifact split into its YAML frontmatter and its
// "## name" sections.
type document struct {
	front    map[string]any
	body     string
	sections map[string]string
}

func readDocument(path string) (document, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the artifact path is resolved by the mission boundary
	if err != nil {
		return document{}, fmt.Errorf("read handoff artifact: %w", err)
	}
	front, body := splitFrontmatter(string(raw))
	return document{front: front, body: body, sections: splitSections(body)}, nil
}

func splitFrontmatter(content string) (map[string]any, string) {
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return nil, content
	}
	header, body, found := strings.Cut(rest, "\n---")
	if !found {
		return nil, content
	}
	var front map[string]any
	if yaml.Unmarshal([]byte(header), &front) != nil {
		return nil, content
	}
	return front, strings.TrimPrefix(strings.TrimPrefix(body, "\n"), "\n")
}

func splitSections(body string) map[string]string {
	sections := map[string]string{}
	name := ""
	var lines []string
	flush := func() {
		if name != "" {
			sections[name] = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			flush()
			name, lines = strings.TrimSpace(heading), nil
			continue
		}
		lines = append(lines, line)
	}
	flush()
	return sections
}

func (d document) frontField(key string) (string, bool) {
	value, ok := d.front[key]
	if !ok {
		return "", false
	}
	raw, err := yaml.Marshal(value)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(raw)), true
}

// artifactSource serves contract fields from a Ranger analysis artifact.
type artifactSource struct{ doc document }

// NewRangerSource reads a Ranger analysis artifact.
func NewRangerSource(path string) (Source, error) {
	doc, err := readDocument(path)
	if err != nil {
		return nil, err
	}
	return artifactSource{doc: doc}, nil
}

// rangerSections maps a contract field to the artifact section that carries it.
var rangerSections = map[string][]string{
	"objective":                    {"mission_objective", "objective"},
	"known_facts":                  {"known_facts"},
	"uncertainties":                {"uncertainties"},
	"recommended_refinement_focus": {"recommended_refinement_focus"},
}

func (r artifactSource) Field(name string) (string, bool) {
	for _, section := range rangerSections[name] {
		if value, ok := r.doc.sections[section]; ok {
			return value, true
		}
	}
	return "", false
}

// packageSource serves contract fields from a refined package.
type packageSource struct{ tasks, analysis document }

// NewPackageSource reads the tasks and analysis of a refined package.
func NewPackageSource(dir string) (Source, error) {
	tasks, err := readDocument(filepath.Join(dir, "tasks.md"))
	if err != nil {
		return nil, err
	}
	analysis, err := readDocument(filepath.Join(dir, "analysis.md"))
	if err != nil {
		return nil, err
	}
	return packageSource{tasks: tasks, analysis: analysis}, nil
}

func (p packageSource) Field(name string) (string, bool) {
	switch name {
	case "implementation_plan":
		return strings.TrimSpace(p.tasks.body), strings.TrimSpace(p.tasks.body) != ""
	case "approved_scope":
		if value, ok := p.tasks.frontField("approved_scope"); ok {
			return value, true
		}
		value, ok := p.analysis.sections["affected_scope"]
		return value, ok
	case "acceptance_checks":
		return p.tasks.frontField("acceptance_checks")
	default:
		return "", false
	}
}
