package missionview

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/gatecriteria"
)

// CriteriaSection is the quality-criteria report of a mission, built only from
// records earlier stages wrote. Every row is filled: a value, "não avaliado" or
// "não sei". It is omitted from the JSON when no report was supplied, so a view
// without it keeps its previous shape.
type CriteriaSection struct {
	Rows []gatecriteria.Row `json:"rows"`
}

func buildCriteria(report *gatecriteria.Report) *CriteriaSection {
	if report == nil {
		return nil
	}
	return &CriteriaSection{Rows: report.Rows}
}

func renderCriteria(w io.Writer, section *CriteriaSection) error {
	if section == nil {
		return nil
	}
	if err := writef(w, "Quality criteria\n"); err != nil {
		return err
	}
	for _, row := range section.Rows {
		if err := writef(w, "  %s: %s [%s]\n", row.Criterion, row.Value, row.State); err != nil {
			return fmt.Errorf("render criteria row %s: %w", row.Criterion, err)
		}
	}
	return nil
}
