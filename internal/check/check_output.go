package check

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printCheckSuccess(root string, providers map[string]string, resolutions map[string]slotResolution, mode string) error {
	printStatusBanner("check")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, section := range []func(*tabwriter.Writer) error{
		func(w *tabwriter.Writer) error { return writeCheckStatusSection(w, root) },
		func(w *tabwriter.Writer) error { return writeCheckSlotsSection(w, providers, resolutions) },
		func(w *tabwriter.Writer) error { return writeCheckPersonaSection(w, mode) },
	} {
		if err := section(w); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("check: flush output: %w", err)
	}
	return nil
}

func writeCheckStatusSection(w *tabwriter.Writer, root string) error {
	if _, err := fmt.Fprintln(w, "STATUS\t"); err != nil {
		return fmt.Errorf("check: write status header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  ok\troot=%s\n", root); err != nil {
		return fmt.Errorf("check: write status row: %w", err)
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeCheckSlotsSection(w *tabwriter.Writer, providers map[string]string, resolutions map[string]slotResolution) error {
	if _, err := fmt.Fprintln(w, "SLOTS\t"); err != nil {
		return fmt.Errorf("check: write slots header: %w", err)
	}
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		kind := resolutions[slot].kind
		if _, err := fmt.Fprintf(w, "  %-12s\t%s\tkind=%s\n", slot, providers[slot], kind); err != nil {
			return fmt.Errorf("check: write slot row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeCheckPersonaSection(w *tabwriter.Writer, mode string) error {
	if _, err := fmt.Fprintln(w, "PERSONA\t"); err != nil {
		return fmt.Errorf("check: write persona header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  mode\t%s\n", mode); err != nil {
		return fmt.Errorf("check: write persona row: %w", err)
	}
	return nil
}
