package dojo

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func itemLine(item domain.DojoCheckItem) string {
	if item.Passed {
		return fmt.Sprintf("  %s\t✓\n", item.Label)
	}
	detail := item.Detail
	if detail == "" {
		detail = "FAIL"
	}
	return fmt.Sprintf("  %s\t✗   ← %s\n", item.Label, detail)
}

func summaryLine(result domain.DojoCheckResult) string {
	if result.Passed() {
		return fmt.Sprintf("result\tPASS (%d checks)\n", len(result.Items))
	}
	return fmt.Sprintf("result\tFAIL (%d of %d checks failed)\n", result.FailCount(), len(result.Items))
}

func printResult(out io.Writer, result domain.DojoCheckResult) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	lines := []string{
		fmt.Sprintf("scenario\t%s\n", result.Scenario),
		"────────────────────────────────────────────────────\n",
	}
	for _, item := range result.Items {
		lines = append(lines, itemLine(item))
	}
	lines = append(lines, "\n", summaryLine(result))
	for _, line := range lines {
		if _, err := fmt.Fprint(w, line); err != nil {
			return fmt.Errorf("dojo: write result: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("dojo: flush result: %w", err)
	}
	return nil
}
