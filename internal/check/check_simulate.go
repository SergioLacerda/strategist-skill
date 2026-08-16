package check

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// printSimulateReport prints the --simulate readiness report. It performs no
// provider invocation and no workspace mutation — it only reads the already
// materialized errs computed by the caller's checks and reports them as a
// per-slot/persona readiness table instead of the terse pass/fail banner.
func printSimulateReport(root string, providers map[string]string, resolutions map[string]slotResolution, mode, decisionReason string, errs []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := writeSimulateReport(w, root, providers, resolutions, mode, decisionReason, errs); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("check --simulate: flush output: %w", err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("[Strategist] check=failed errors=%d root=%s (simulate)", len(errs), root)
	}
	return nil
}

// simReportWriter accumulates tabwriter output, short-circuiting after the
// first write error so callers can emit a flat sequence of lines without a
// branch per line.
type simReportWriter struct {
	w   *tabwriter.Writer
	err error
}

func (s *simReportWriter) line(format string, args ...any) {
	if s.err != nil {
		return
	}
	if _, err := fmt.Fprintf(s.w, format, args...); err != nil {
		s.err = fmt.Errorf("check --simulate: write output: %w", err)
	}
}

// writeSimulateReport writes the --simulate readiness table to w, returning the
// first write error encountered (if any) wrapped with context.
func writeSimulateReport(w *tabwriter.Writer, root string, providers map[string]string, resolutions map[string]slotResolution, mode, decisionReason string, errs []string) error {
	sw := &simReportWriter{w: w}

	sw.line("READINESS\t\n")
	sw.line("  root\t%s\n", root)
	sw.line("  pipeline_route\tmain\n")
	sw.line("  decision_reason\t%s\n", decisionReason)
	sw.line("\t\n")
	sw.line("SLOTS\t\n")
	writeSimulateSlots(sw, providers, resolutions)
	sw.line("\t\n")
	sw.line("PERSONA\t\n")
	sw.line("  mode\t%s\n", mode)
	writeSimulateBlockers(sw, errs)

	return sw.err
}

func writeSimulateSlots(sw *simReportWriter, providers map[string]string, resolutions map[string]slotResolution) {
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		status := "ready"
		if providers[slot] == "" {
			status = "missing_provider"
		}
		kind := resolutions[slot].kind
		sw.line("  %-12s\tprovider=%s\tkind=%s\tstatus=%s\n", slot, providers[slot], kind, status)
	}
}

func writeSimulateBlockers(sw *simReportWriter, errs []string) {
	if len(errs) == 0 {
		return
	}
	sw.line("\t\n")
	sw.line("BLOCKERS\t\n")
	for _, e := range errs {
		sw.line("  ✗\t%s\n", e)
	}
}
