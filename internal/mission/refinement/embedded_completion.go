package refinement

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"gopkg.in/yaml.v3"
)

// EmbeddedCompletion is one embedded Archivist completion split into the three
// refinement documents and the typed handoff facts the Archivist declares at
// publication (ADR-0067). The Weapon's result is untrusted until this parse
// and the package validation both succeed.
type EmbeddedCompletion struct {
	HandoffFacts map[string]any
	Documents    map[string][]byte
}

// embeddedDocuments are the level-2 section names, in the order the output
// contract lists them. Headings inside a document start at level 3.
var embeddedDocuments = []string{"proposal.md", "design.md", "tasks.md"}

var (
	embeddedSection    = regexp.MustCompile(`^## (.+?)\s*$`)
	embeddedPlansWrite = regexp.MustCompile(`(?i)\b(save[sd]?|writ(?:e|es|ten|ing)|creat(?:e|es|ed|ing)|stor(?:e|es|ed|ing)|materializ(?:e|es|ed|ing)|output)\b`)
	embeddedCommitStep = regexp.MustCompile(`(?i)\bgit\s+(commit|add|push|reset|merge|rebase)\b`)
)

// ParseEmbeddedCompletion validates the shape of a completion before anything
// is published: a frontmatter holding only the typed facts, exactly the three
// documents, no text outside them, and no destination or commit step Strategist
// forbids for an Archivist (a plan under docs/plans/, a git commit task).
func ParseEmbeddedCompletion(result string) (EmbeddedCompletion, error) {
	facts, body, err := splitEmbeddedFrontmatter(result)
	if err != nil {
		return EmbeddedCompletion{}, err
	}
	if err := validateHandoffFacts(facts); err != nil {
		return EmbeddedCompletion{}, fmt.Errorf("embedded completion: %w", err)
	}
	documents, err := splitEmbeddedDocuments(body)
	if err != nil {
		return EmbeddedCompletion{}, err
	}
	if err := requireEmbeddedContainment(documents); err != nil {
		return EmbeddedCompletion{}, err
	}
	return EmbeddedCompletion{HandoffFacts: facts, Documents: documents}, nil
}

func splitEmbeddedFrontmatter(result string) (map[string]any, string, error) {
	text := strings.ReplaceAll(result, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", fmt.Errorf("embedded completion: the result must start with a YAML frontmatter holding %s", handoff.PolicyFactsKey)
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("embedded completion: the frontmatter is not closed")
	}
	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(text[4:4+end]), &frontmatter); err != nil {
		return nil, "", fmt.Errorf("embedded completion: parse frontmatter: %w", err)
	}
	for key := range frontmatter {
		if key != handoff.PolicyFactsKey {
			return nil, "", fmt.Errorf("embedded completion: unexpected frontmatter key %q", key)
		}
	}
	facts, ok := frontmatter[handoff.PolicyFactsKey].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("embedded completion: %s must be a mapping", handoff.PolicyFactsKey)
	}
	return facts, text[4+end+len("\n---"):], nil
}

func splitEmbeddedDocuments(body string) (map[string][]byte, error) {
	splitter := embeddedSplitter{sections: map[string][]string{}}
	for _, line := range strings.Split(body, "\n") {
		if err := splitter.consume(line); err != nil {
			return nil, err
		}
	}
	return finishEmbeddedDocuments(splitter.sections)
}

// embeddedSplitter walks the body line by line, tracking the open section and
// whether the walk is inside a code fence (a fenced `## x` is not a heading).
type embeddedSplitter struct {
	sections map[string][]string
	current  string
	inFence  bool
}

func (s *embeddedSplitter) consume(line string) error {
	if strings.HasPrefix(strings.TrimSpace(line), "```") {
		s.inFence = !s.inFence
	}
	if name, isHeading := embeddedSectionName(line, s.inFence); isHeading {
		return s.open(name)
	}
	return s.add(line)
}

func (s *embeddedSplitter) open(name string) error {
	if err := openEmbeddedSection(s.sections, name); err != nil {
		return err
	}
	s.current = name
	return nil
}

func (s *embeddedSplitter) add(line string) error {
	if s.current != "" {
		s.sections[s.current] = append(s.sections[s.current], line)
		return nil
	}
	if strings.TrimSpace(line) != "" {
		return fmt.Errorf("embedded completion: text before the first document section")
	}
	return nil
}

// embeddedSectionName reports a level-2 heading outside a code fence.
func embeddedSectionName(line string, inFence bool) (string, bool) {
	if inFence {
		return "", false
	}
	match := embeddedSection.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	return match[1], true
}

func openEmbeddedSection(sections map[string][]string, name string) error {
	if !containsString(embeddedDocuments, name) {
		return fmt.Errorf("embedded completion: unexpected level-2 section %q; headings inside a document start at level 3", name)
	}
	if _, seen := sections[name]; seen {
		return fmt.Errorf("embedded completion: document %s appears more than once", name)
	}
	sections[name] = []string{}
	return nil
}

func finishEmbeddedDocuments(sections map[string][]string) (map[string][]byte, error) {
	documents := make(map[string][]byte, len(embeddedDocuments))
	for _, name := range embeddedDocuments {
		lines, ok := sections[name]
		if !ok {
			return nil, fmt.Errorf("embedded completion: document %s is missing", name)
		}
		text := strings.TrimSpace(strings.Join(lines, "\n"))
		if text == "" {
			return nil, fmt.Errorf("embedded completion: document %s is empty", name)
		}
		documents[name] = []byte(text + "\n")
	}
	return documents, nil
}

// requireEmbeddedContainment keeps the plan inside the mission's canonical
// package: a Weapon cannot direct it to a repository planning directory, and
// the Archivist never produces a commit step (M010).
func requireEmbeddedContainment(documents map[string][]byte) error {
	for _, name := range embeddedDocuments {
		for number, line := range strings.Split(string(documents[name]), "\n") {
			if reason := embeddedLineViolation(line); reason != "" {
				return fmt.Errorf("embedded_containment_violation: %s line %d: %s", name, number+1, reason)
			}
		}
	}
	return nil
}

func embeddedLineViolation(line string) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "docs/plans/") && embeddedPlansWrite.MatchString(line) {
		return "directs a plan to docs/plans/, which is outside the mission package"
	}
	if strings.HasPrefix(strings.TrimSpace(line), "- [") && embeddedCommitStep.MatchString(line) {
		return "a task runs a git mutation command"
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
