package opportunity_attack

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

func analysisFrontmatter(raw []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if !bytes.HasPrefix(trimmed, []byte("---\n")) {
		return nil, fmt.Errorf("opportunity attack: analysis frontmatter is missing")
	}
	end := bytes.Index(trimmed[4:], []byte("\n---"))
	if end < 0 {
		return nil, fmt.Errorf("opportunity attack: analysis frontmatter is unclosed")
	}
	frontmatter := map[string]any{}
	if err := yaml.Unmarshal(trimmed[4:end+4], &frontmatter); err != nil {
		return nil, fmt.Errorf("opportunity attack: parse analysis frontmatter: %w", err)
	}
	return frontmatter, nil
}

func updateAnalysisFrontmatter(raw []byte, manifest Manifest) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if !bytes.HasPrefix(trimmed, []byte("---\n")) {
		return nil, fmt.Errorf("opportunity attack: analysis frontmatter is missing")
	}
	end := bytes.Index(trimmed[4:], []byte("\n---"))
	if end < 0 {
		return nil, fmt.Errorf("opportunity attack: analysis frontmatter is unclosed")
	}
	frontmatter := map[string]any{}
	if err := yaml.Unmarshal(trimmed[4:end+4], &frontmatter); err != nil {
		return nil, fmt.Errorf("opportunity attack: parse analysis frontmatter: %w", err)
	}
	frontmatter["opportunity_attack"] = manifest
	// The existing side-quest reader is intentionally typed. Only warranted
	// candidates become actionable side quests; the manifest still exposes both
	// candidate families, including false records.
	approved := make([]Candidate, 0, len(manifest.Candidates))
	for _, candidate := range manifest.Candidates {
		if candidate.Warranted {
			approved = append(approved, candidate)
		}
	}
	frontmatter["side_quests_approved"] = approved
	header, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("opportunity attack: encode analysis frontmatter: %w", err)
	}
	body := trimmed[end+8:]
	// Keep the body byte-for-byte stable: handoff.PackageDigest includes it and
	// lifecycle status changes must not turn OA metadata into a package edit.
	// `body` begins at the same delimiter-adjacent newline returned by the
	// handoff parser, so do not add a second newline before appending it.
	return append(append([]byte("---\n"), append(header, []byte("---")...)...), body...), nil
}
