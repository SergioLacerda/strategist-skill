package treasurecli

import (
	"fmt"
	"path"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
)

func renderExternalRunbookProjection(projection jeweler.Projection, format string) error {
	if len(projection.Items) == 0 {
		fmt.Println("[Strategist] runbook select: no runbook matched the given signals")
		return nil
	}
	rows, err := externalRunbookRows(projection.Items)
	if err != nil {
		return err
	}
	sortRunbookSelectionRows(rows)
	return renderExternalRunbookRows(rows, format)
}

func externalRunbookRows(items []jeweler.Item) ([]runbookSelectionRow, error) {
	rows := make([]runbookSelectionRow, 0, len(items))
	for i, item := range items {
		if item.Source.Path == "" {
			return nil, fmt.Errorf("runbook select: JEWELER item %q has no source path", item.ID)
		}
		role := "supporting"
		if i == 0 {
			role = "primary"
		}
		rows = append(rows, runbookSelectionRow{
			RunbookID: externalRunbookID(item),
			Role:      role,
			ChestID:   runbookChestID,
			Ref:       item.Source.Path,
			Reason:    item.Reason,
		})
	}
	return rows, nil
}

func renderExternalRunbookRows(rows []runbookSelectionRow, format string) error {
	switch format {
	case "", outputFormatTable:
		return renderRunbookSelectionTable(rows)
	case outputFormatJSON:
		return renderRunbookSelectionJSON(rows)
	default:
		return fmt.Errorf("runbook select: unknown --format %q (want table or json)", format)
	}
}

func externalRunbookID(item jeweler.Item) string {
	base := path.Base(item.Source.Path)
	if strings.HasSuffix(base, ".md") {
		return strings.TrimSuffix(base, ".md")
	}
	return item.ID
}
