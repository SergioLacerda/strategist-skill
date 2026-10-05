package missionview

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

const unavailableCell = "—"

// RenderPhaseTimeline writes the per-phase table. With details it adds one
// line per run. Unobserved values print as an unavailable marker, never 0.
func RenderPhaseTimeline(w io.Writer, tl PhaseTimeline, details bool) error {
	if err := writef(w, "Phases (lifecycle)\n  availability: %s\n", tl.Availability); err != nil {
		return err
	}
	if tl.Note != "" && tl.UsageAvailability != Available {
		return writef(w, "  %s\n", tl.Note)
	}
	if tl.Availability == Available {
		if err := writef(w, "  mission: status=%s route=%s start=%s end=%s duration=%s last_marker=%s\n",
			tl.Status, orCell(tl.Route), cell(tl.StartedAt), cell(tl.EndedAt), duration(tl.DurationSeconds), cell(tl.LastMarkerAt)); err != nil {
			return err
		}
	}
	if err := renderTimelineRows(w, tl.Phases, details); err != nil {
		return err
	}
	return renderUsageTotals(w, tl)
}

// renderUsageTotals prints the unattributed consumption, the mission total and
// the coverage counters. The total sums observations only: legacy records and
// handoff-local estimates are never part of it.
func renderUsageTotals(w io.Writer, tl PhaseTimeline) error {
	if tl.UsageAvailability != Available {
		return writef(w, "MISSION TOTAL   tokens: %s   quality: %s\n", unavailableCell, QualityUnavailable)
	}
	if err := writef(w, "  UNATTRIBUTED   %s\n", bucketCells(tl.Unattributed)); err != nil {
		return err
	}
	if err := writef(w, "MISSION TOTAL   %s\n", bucketCells(tl.Total)); err != nil {
		return err
	}
	c := tl.Coverage
	return writef(w, "  coverage: invocations known=%d observed=%d missing=%d; unkeyed=%d; baseline_snapshots=%d; resets=%d\n",
		c.InvocationsKnown, c.InvocationsObserved, c.InvocationsMissing, c.UnkeyedObservations, c.BaselineSnapshots, c.ResetIntervals)
}

func bucketCells(b UsageBucket) string {
	return fmt.Sprintf("in=%s out=%s total=%s   quality: %s   observations: %d",
		count(b.Tokens.Input), count(b.Tokens.Output), count(b.Tokens.Total), b.Quality, b.Observations)
}

func renderTimelineRows(w io.Writer, rows []PhaseRow, details bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writef(tw, "  PHASE\tSTART\tEND\tDURATION\tRUNS\tSTATUS\tINPUT\tOUTPUT\tTOTAL\tQUALITY\n"); err != nil {
		return err
	}
	for _, r := range rows {
		if err := writef(tw, "  %s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n", r.Label, cell(r.StartedAt), cell(r.EndedAt),
			duration(r.DurationSeconds), r.Runs, r.Status, count(r.Tokens.Input), count(r.Tokens.Output), count(r.Tokens.Total), r.Quality); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("render phase timeline: %w", err)
	}
	if !details {
		return nil
	}
	return renderRunDetails(w, rows)
}

func renderRunDetails(w io.Writer, rows []PhaseRow) error {
	for _, r := range rows {
		for _, run := range r.RunDetails {
			if err := writef(w, "  run %s %s start=%s end=%s reason=%s\n", run.Phase, run.RunID, run.StartedAt, cell(run.EndedAt), orCell(run.EndReason)); err != nil {
				return err
			}
		}
	}
	return nil
}

func cell(p *string) string {
	if p == nil {
		return unavailableCell
	}
	return *p
}

func orCell(s string) string {
	if s == "" {
		return unavailableCell
	}
	return s
}

func count(p *int64) string {
	if p == nil {
		return unavailableCell
	}
	return fmt.Sprintf("%d", *p)
}

func duration(secs *int64) string {
	if secs == nil {
		return unavailableCell
	}
	return (time.Duration(*secs) * time.Second).String()
}
