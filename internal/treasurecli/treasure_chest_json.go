package treasurecli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

type jsonChestRow struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Scope       []string `json:"scope"`
	Trust       string   `json:"trust,omitempty"`
	Freshness   string   `json:"freshness"`
	Drift       []string `json:"drift,omitempty"`
	SourceGrade string   `json:"source_grade,omitempty"` // SQ-002/SQ-001
	ReuseValue  string   `json:"reuse_value,omitempty"`  // SQ-002/SQ-001
	OpenGaps    []string `json:"open_gaps,omitempty"`    // SQ-002/SQ-001
	JewelCount  int      `json:"jewel_count,omitempty"`  // SQ-009
}

type jsonIndex struct {
	Artifact   string `json:"artifact"`
	Health     string `json:"health"`
	CompiledAt string `json:"compiled_at,omitempty"`
}

type jsonTreasureChestOutput struct {
	Chests   []jsonChestRow `json:"chests"`
	Index    jsonIndex      `json:"index"`
	Warnings []string       `json:"warnings,omitempty"`
}

func renderTreasureChestJSON(w *os.File, root string, rows []treasure.StatusRow, compErr, govErr, idxErr error, compiledAt int64) error {
	indexPath := filepath.Join(root, ".compiled", ".index.gz")
	var health, ts string
	switch {
	case compErr != nil:
		health = "corrupt"
	case compiledAt == 0:
		health = "absent"
	default:
		health = "ok"
		ts = time.Unix(compiledAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	out := jsonTreasureChestOutput{
		Chests:   make([]jsonChestRow, 0, len(rows)),
		Index:    jsonIndex{Artifact: indexPath, Health: health, CompiledAt: ts},
		Warnings: collectWarnings(rows, govErr, idxErr, compErr, compiledAt),
	}
	for _, r := range rows {
		out.Chests = append(out.Chests, jsonChestRow{
			ID:          r.ID,
			Path:        r.Path,
			Scope:       r.Scope,
			Trust:       r.TrustTier,
			Freshness:   r.Freshness,
			Drift:       r.Drift,
			SourceGrade: r.SourceGrade,
			ReuseValue:  r.ReuseValue,
			OpenGaps:    r.OpenGaps,
			JewelCount:  r.JewelCount,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest: encode json: %w", err)
	}
	return nil
}
