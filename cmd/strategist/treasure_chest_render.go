package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

func renderTreasureChestTable(root string, rows []treasure.StatusRow, govErr, idxErr, compErr error, compiledAt int64) error {
	printTreasureChestBanner()
	if err := renderTableSection("chests", func(w *tabwriter.Writer) error {
		return renderChestsSection(w, rows)
	}); err != nil {
		return err
	}
	if err := renderTableSection("index", func(w *tabwriter.Writer) error {
		return renderIndexSection(w, root, compiledAt, compErr)
	}); err != nil {
		return err
	}
	renderWarningsSection(collectWarnings(rows, govErr, idxErr, compErr, compiledAt))
	fmt.Println()
	return nil
}

func renderTableSection(label string, render func(*tabwriter.Writer) error) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := render(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush %s: %w", label, err)
	}
	fmt.Println()
	return nil
}

func renderChestsSection(w *tabwriter.Writer, rows []treasure.StatusRow) error {
	if _, err := fmt.Fprintln(w, "  CHESTS\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"ID", "PATH", "SCOPE", "TRUST", "FRESHNESS", "DRIFT", "GRADE", "REUSE", "GAPS", "JEWELS"); err != nil {
		return fmt.Errorf("treasure-chest: write column header: %w", err)
	}
	for _, r := range rows {
		if err := renderChestRow(w, r); err != nil {
			return fmt.Errorf("treasure-chest: write row: %w", err)
		}
	}
	return nil
}

func renderChestRow(w *tabwriter.Writer, r treasure.StatusRow) error {
	if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		r.ID,
		r.Path,
		dashIfEmpty(strings.Join(r.Scope, ",")),
		dashIfEmpty(r.TrustTier),
		r.Freshness,
		driftText(r.Drift),
		dashIfEmpty(r.SourceGrade),
		dashIfEmpty(r.ReuseValue),
		countOrDash(len(r.OpenGaps)),
		countOrDash(r.JewelCount),
	); err != nil {
		return fmt.Errorf("treasure-chest status: write chest row: %w", err)
	}
	return nil
}

func driftText(drift []string) string {
	if len(drift) == 0 {
		return "none"
	}
	return strings.Join(drift, " ")
}

func countOrDash(count int) string {
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", count)
}

func dashIfEmpty(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

func renderIndexSection(w *tabwriter.Writer, root string, compiledAt int64, compErr error) error {
	if _, err := fmt.Fprintln(w, "  INDEX\t\t\t\t\t"); err != nil {
		return fmt.Errorf("treasure-chest: write header: %w", err)
	}

	indexPath := filepath.Join(root, ".compiled", ".index.gz")
	var health, ts string

	switch {
	case compErr != nil:
		health = "corrupt"
		ts = "—"
	case compiledAt == 0:
		health = "absent"
		ts = "—"
	default:
		health = "ok"
		ts = time.Unix(compiledAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	rows := []struct{ k, v string }{
		{"  artifact", indexPath},
		{"  health", health},
		{"  compiled_at", ts},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\t\t\t\t\n", row.k, row.v); err != nil {
			return fmt.Errorf("treasure-chest: write index row: %w", err)
		}
	}
	return nil
}

// printTreasureChestBanner prints the ASCII header for the treasure-chest command.
func printTreasureChestBanner() {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  treasure-chest                                     │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}
