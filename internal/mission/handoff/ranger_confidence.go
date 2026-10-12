package handoff

import (
	"fmt"
	"path/filepath"
)

// RangerSource is one source the Ranger declares it consulted.
type RangerSource struct {
	Path     string
	Coverage string
}

// RangerConfidence is what a normalized Ranger artifact declares about its own
// confidence: the score, what it concluded and what it read. A nil Score means
// the artifact declared none; it is never read as zero.
type RangerConfidence struct {
	Score   *float64
	Verdict string
	Subtype string
	Sources []RangerSource
}

// ReadRangerConfidence reads the confidence declaration from the artifact
// frontmatter. A non-numeric score is an error, not a guess.
func ReadRangerConfidence(path string) (RangerConfidence, error) {
	content, err := readArtifact(filepath.Clean(path))
	if err != nil {
		return RangerConfidence{}, fmt.Errorf("handoff_artifact_invalid: read Ranger artifact: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(content)
	if err != nil {
		return RangerConfidence{}, fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	score, err := frontmatterScore(frontmatter)
	if err != nil {
		return RangerConfidence{}, err
	}
	return RangerConfidence{
		Score: score, Verdict: stringField(frontmatter, "evaluation_verdict"), Subtype: stringField(frontmatter, "discovery_subtype"),
		Sources: frontmatterSources(frontmatter),
	}, nil
}

func frontmatterScore(frontmatter map[string]any) (*float64, error) {
	raw, present := frontmatter["confidence_score"]
	if !present || raw == nil {
		return nil, nil
	}
	switch value := raw.(type) {
	case float64:
		return &value, nil
	case int:
		converted := float64(value)
		return &converted, nil
	default:
		return nil, fmt.Errorf("handoff_artifact_invalid: confidence_score must be a number, got %T", raw)
	}
}

func stringField(values map[string]any, key string) string {
	if text, ok := values[key].(string); ok {
		return text
	}
	return ""
}

func frontmatterSources(frontmatter map[string]any) []RangerSource {
	items, ok := frontmatter["sources_consulted"].([]any)
	if !ok {
		return nil
	}
	var sources []RangerSource
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if path := stringField(entry, "source_path"); path != "" {
			sources = append(sources, RangerSource{Path: path, Coverage: stringField(entry, "coverage_status")})
		}
	}
	return sources
}
