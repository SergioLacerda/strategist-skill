package hostusage

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// CodexSnapshot is one cumulative thread counter reading.
type CodexSnapshot struct {
	At                               time.Time
	Input, Cached, Output, Reasoning int64
}

type codexLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Payload   struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Info *struct {
			Total *codexTotal `json:"total_token_usage"`
		} `json:"info"`
	} `json:"payload"`
}

type codexTotal struct {
	Input     *int64 `json:"input_tokens"`
	Cached    *int64 `json:"cached_input_tokens"`
	Output    *int64 `json:"output_tokens"`
	Reasoning *int64 `json:"reasoning_output_tokens"`
}

// ParseCodexSession reads a Codex rollout and returns the cumulative
// `total_token_usage` snapshots in file order plus the session id. Per-event
// `last_token_usage` and per-response records are ignored on purpose: they
// repeat and drift from the cumulative counter. Everything else is discarded.
func ParseCodexSession(r io.Reader) (snaps []CodexSnapshot, sessionID string, unrecognized int, err error) {
	var p codexParser
	if err = forEachLine(r, p.feed); err != nil {
		return nil, "", 0, fmt.Errorf("hostusage: %w", err)
	}
	if len(p.snaps) == 0 && p.unrecognized > 0 {
		return nil, p.sessionID, p.unrecognized, ErrUnrecognizedFormat
	}
	return p.snaps, p.sessionID, p.unrecognized, nil
}

type codexParser struct {
	snaps        []CodexSnapshot
	sessionID    string
	unrecognized int
}

func (p *codexParser) feed(line []byte) {
	var rec codexLine
	if json.Unmarshal(line, &rec) != nil {
		return
	}
	if rec.Type == "session_meta" && p.sessionID == "" {
		p.sessionID = rec.Payload.ID
		return
	}
	snap, isCount, ok := codexSnapshotOf(rec)
	if !isCount {
		return
	}
	if !ok {
		p.unrecognized++
		return
	}
	p.snaps = append(p.snaps, snap)
}

// codexSnapshotOf reports whether the record is a token_count event and, when
// it is, whether it carries a recognized cumulative counter.
func codexSnapshotOf(rec codexLine) (snap CodexSnapshot, isCount, ok bool) {
	if rec.Type != "event_msg" || rec.Payload.Type != "token_count" || rec.Payload.Info == nil {
		return CodexSnapshot{}, false, false
	}
	t := rec.Payload.Info.Total
	at, okTime := parseTime(rec.Timestamp)
	if t == nil || t.Input == nil || t.Output == nil || !okTime {
		return CodexSnapshot{}, true, false
	}
	snap = CodexSnapshot{At: at, Input: *t.Input, Output: *t.Output}
	if t.Cached != nil {
		snap.Cached = *t.Cached
	}
	if t.Reasoning != nil {
		snap.Reasoning = *t.Reasoning
	}
	return snap, true, true
}

// ObserveCodex returns the usage inside the window as the difference between
// the last cumulative snapshot before the window end and the last one before
// its start (zero when the thread began inside the window). A counter that
// goes backwards makes the interval unavailable instead of negative.
func ObserveCodex(snaps []CodexSnapshot, sessionID string, w Window) (Observation, error) {
	if err := w.Validate(); err != nil {
		return Observation{}, err
	}
	obs := Observation{Host: HostCodex, Provider: "openai", SessionID: sessionID, WindowStart: w.Start, WindowEnd: w.End}
	var before, within []CodexSnapshot
	for _, s := range snaps {
		switch {
		case !s.At.Before(w.End):
		case s.At.Before(w.Start):
			before = append(before, s)
		default:
			within = append(within, s)
		}
	}
	if len(within) == 0 {
		return obs, nil
	}
	if hasDecrease(append(append([]CodexSnapshot(nil), before...), within...)) {
		obs.CounterReset = true
		return obs, nil
	}
	var base CodexSnapshot
	if len(before) > 0 {
		base = before[len(before)-1]
	}
	last := within[len(within)-1]
	obs.Calls = countChanges(base, within)
	in, out, cache, reasoning := last.Input-base.Input, last.Output-base.Output, last.Cached-base.Cached, last.Reasoning-base.Reasoning
	obs.Input, obs.Output, obs.Cache, obs.Reasoning = &in, &out, &cache, &reasoning
	return obs, nil
}

func hasDecrease(seq []CodexSnapshot) bool {
	for i := 1; i < len(seq); i++ {
		if seq[i].Input < seq[i-1].Input || seq[i].Output < seq[i-1].Output {
			return true
		}
	}
	return false
}

// countChanges counts the snapshots that moved the counter; repeated
// snapshots are not calls.
func countChanges(base CodexSnapshot, within []CodexSnapshot) int {
	n, prev := 0, base
	for _, s := range within {
		if s.Input != prev.Input || s.Output != prev.Output || s.Cached != prev.Cached {
			n++
		}
		prev = s
	}
	return n
}
