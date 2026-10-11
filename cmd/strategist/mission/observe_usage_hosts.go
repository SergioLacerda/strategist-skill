package mission

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/hostusage"
)

func observeClaudeFiles(opts ObserveUsageOptions, w hostusage.Window) (hostusage.Observation, []string, error) {
	paths := opts.Transcripts
	if opts.IncludeSubagents {
		paths = withSubagentFiles(opts.Transcripts)
	}
	calls, unrecognized, err := readClaudeFiles(paths)
	if err != nil {
		return hostusage.Observation{}, nil, err
	}
	obs, err := hostusage.ObserveClaude(calls, w)
	if err != nil {
		return hostusage.Observation{}, nil, fmt.Errorf("observe claude window: %w", err)
	}
	notes := []string{fmt.Sprintf("files read: %d (sub-agent files included: %t)", len(paths), opts.IncludeSubagents),
		"helper-model calls are not in transcripts and are not counted"}
	if unrecognized > 0 {
		notes = append(notes, fmt.Sprintf("unrecognized assistant records skipped: %d", unrecognized))
	}
	return obs, notes, nil
}

// withSubagentFiles adds the sub-agent transcripts that Claude Code keeps in a
// directory next to each session file.
func withSubagentFiles(transcripts []string) []string {
	paths := append([]string(nil), transcripts...)
	for _, p := range transcripts {
		pattern := filepath.Join(strings.TrimSuffix(p, ".jsonl"), "subagents", "*.jsonl")
		subs, _ := filepath.Glob(pattern) //nolint:errcheck // only ErrBadPattern, which a fixed pattern cannot produce
		paths = append(paths, subs...)
	}
	return paths
}

func readClaudeFiles(paths []string) ([]hostusage.ClaudeCall, int, error) {
	var calls []hostusage.ClaudeCall
	unrecognized := 0
	for _, p := range paths {
		got, bad, err := withFile(p, hostusage.ParseClaudeSession)
		if err != nil {
			return nil, 0, err
		}
		calls, unrecognized = append(calls, got...), unrecognized+bad
	}
	return calls, unrecognized, nil
}

func observeCodexFile(opts ObserveUsageOptions, w hostusage.Window) (hostusage.Observation, []string, error) {
	if len(opts.Transcripts) != 1 {
		return hostusage.Observation{}, nil, errors.New("codex observes exactly one --transcript (one thread's cumulative counter)")
	}
	f, err := os.Open(opts.Transcripts[0]) //nolint:gosec // G304: the operator names this file explicitly
	if err != nil {
		return hostusage.Observation{}, nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file: a close error carries no information
	snaps, session, bad, err := hostusage.ParseCodexSession(f)
	if err != nil {
		return hostusage.Observation{}, nil, fmt.Errorf("read session file: %w", err)
	}
	obs, err := hostusage.ObserveCodex(snaps, session, w)
	if err != nil {
		return hostusage.Observation{}, nil, fmt.Errorf("observe codex window: %w", err)
	}
	notes := []string{"input includes cached input; counts are deltas of the cumulative thread counter"}
	if obs.CounterReset {
		notes = append(notes, "the counter went backwards inside the window: the interval is unavailable")
	}
	if bad > 0 {
		notes = append(notes, fmt.Sprintf("unrecognized counter events skipped: %d", bad))
	}
	return obs, notes, nil
}

func withFile[T any](path string, parse func(io.Reader) (T, int, error)) (T, int, error) {
	var zero T
	f, err := os.Open(path) //nolint:gosec // G304: the operator names this file explicitly
	if err != nil {
		return zero, 0, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file: a close error carries no information
	out, bad, err := parse(f)
	if err != nil {
		return zero, 0, fmt.Errorf("read session file %s: %w", filepath.Base(path), err)
	}
	return out, bad, nil
}
