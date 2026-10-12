package handoff

import (
	"fmt"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// VerificationMetadata is the optional handoff_verification block a package may
// carry (in analysis.md frontmatter or a fenced YAML block of tasks.md). It is
// a mirror of the policy decision, never proof of it: the persisted outcome and
// the package-derived policy stay authoritative, and a disagreement fails closed.
type VerificationMetadata struct {
	Required *bool `yaml:"required"`
	Enabled  *bool `yaml:"enabled"`
}

var fencedYAML = regexp.MustCompile("(?s)```ya?ml\\n(.*?)```")

// ReadVerificationMetadata returns the package's handoff_verification block, or
// nil when it carries none. The same block declared twice with different
// values, or with a non-boolean field, is an error.
func ReadVerificationMetadata(refined string) (*VerificationMetadata, error) {
	var found []VerificationMetadata
	analysis, err := readArtifact(filepath.Join(refined, "analysis.md"))
	if err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: read Archivist analysis: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(analysis)
	if err != nil {
		return nil, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	if raw, ok := frontmatter["handoff_verification"]; ok {
		meta, err := decodeVerificationMetadata(raw)
		if err != nil {
			return nil, err
		}
		found = append(found, meta)
	}
	tasks, err := readArchivistTasks(refined)
	if err != nil {
		return nil, err
	}
	fenced, err := fencedVerificationMetadata(tasks)
	if err != nil {
		return nil, err
	}
	found = append(found, fenced...)
	return reconcileMetadata(found)
}

func fencedVerificationMetadata(tasks []byte) ([]VerificationMetadata, error) {
	var found []VerificationMetadata
	for _, block := range fencedYAML.FindAllSubmatch(tasks, -1) {
		var document map[string]any
		if err := yaml.Unmarshal(block[1], &document); err != nil {
			continue // not every fenced block is a mapping; only handoff_verification matters here
		}
		raw, ok := document["handoff_verification"]
		if !ok {
			continue
		}
		meta, err := decodeVerificationMetadata(raw)
		if err != nil {
			return nil, err
		}
		found = append(found, meta)
	}
	return found, nil
}

func decodeVerificationMetadata(raw any) (VerificationMetadata, error) {
	encoded, err := yaml.Marshal(raw)
	if err != nil {
		return VerificationMetadata{}, fmt.Errorf("handoff_metadata_invalid: encode handoff_verification: %w", err)
	}
	var meta struct {
		VerificationMetadata `yaml:",inline"`
	}
	if err := yaml.Unmarshal(encoded, &meta); err != nil {
		return VerificationMetadata{}, fmt.Errorf("handoff_metadata_invalid: handoff_verification: %w", err)
	}
	return meta.VerificationMetadata, nil
}

func reconcileMetadata(found []VerificationMetadata) (*VerificationMetadata, error) {
	if len(found) == 0 {
		return nil, nil
	}
	merged := found[0]
	for _, next := range found[1:] {
		if differs(merged.Required, next.Required) || differs(merged.Enabled, next.Enabled) {
			return nil, fmt.Errorf("handoff_metadata_mismatch: handoff_verification is declared twice with different values")
		}
		merged.Required, merged.Enabled = firstSet(merged.Required, next.Required), firstSet(merged.Enabled, next.Enabled)
	}
	return &merged, nil
}

func firstSet(a, b *bool) *bool {
	if a != nil {
		return a
	}
	return b
}

func differs(a, b *bool) bool { return a != nil && b != nil && *a != *b }

// ConsistentWith checks the metadata mirrors the derived decision. Absent
// metadata (nil) or an absent field is consistent: presence is never required
// and never proof. A declared value that contradicts the outcome fails closed.
func (m *VerificationMetadata) ConsistentWith(required bool) error {
	if m == nil {
		return nil
	}
	if m.Required != nil && *m.Required != required {
		return fmt.Errorf("handoff_metadata_mismatch: handoff_verification.required is %t but the package-derived policy has required=%t", *m.Required, required)
	}
	if m.Enabled != nil && *m.Enabled != required {
		return fmt.Errorf("handoff_metadata_mismatch: handoff_verification.enabled is %t but the package-derived policy has required=%t", *m.Enabled, required)
	}
	return nil
}
