// Package hostusage observes token usage that a coding host already recorded
// in its own session files. It is a content-free projection: a reader keeps
// ids, timestamps, model names and numeric counters and drops everything
// else. It never persists, prints or logs conversation content, and it never
// claims more than the host's own counters support.
package hostusage

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// Hosts supported by an observer.
const (
	HostClaude = "claude"
	HostCodex  = "codex"
)

// ErrUnrecognizedFormat means the file looked like a host session file but no
// usage record had the expected shape: the observer fails closed.
var ErrUnrecognizedFormat = errors.New("hostusage: the session file has no usage records in a recognized shape")

// Observation is the usage a host recorded inside one time window. Input is
// normalized to include cached input, with the cache portion kept in Cache, so
// hosts are comparable. A nil member was not observed; it is never zero.
type Observation struct {
	Host                    string
	SessionID               string
	Provider                string
	Models                  []string
	WindowStart, WindowEnd  time.Time
	Calls                   int
	Input, Output           *int64
	Cache, Reasoning        *int64
	CounterReset            bool
	UnrecognizedRecords     int
	HelperModelsMayBeUnseen bool
}

// Window is the half-open interval [Start, End) an observation covers.
type Window struct{ Start, End time.Time }

// Validate rejects an empty or inverted window.
func (w Window) Validate() error {
	if w.Start.IsZero() || w.End.IsZero() {
		return errors.New("hostusage: the window needs both a start and an end")
	}
	if !w.End.After(w.Start) {
		return fmt.Errorf("hostusage: the window end %s must be after its start %s", w.End.Format(time.RFC3339), w.Start.Format(time.RFC3339))
	}
	return nil
}

func (w Window) contains(at time.Time) bool {
	return !at.Before(w.Start) && at.Before(w.End)
}

func addCount(dst **int64, v *int64) {
	if v == nil {
		return
	}
	if *dst == nil {
		zero := int64(0)
		*dst = &zero
	}
	**dst += *v
}

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func parseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

// forEachLine streams r line by line with no line-length limit, so a very long
// record (a conversation block) is read and dropped without being retained.
func forEachLine(r io.Reader, fn func(line []byte)) error {
	br := bufio.NewReaderSize(r, 1<<20)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			fn(line)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read session file: %w", err)
		}
	}
}
