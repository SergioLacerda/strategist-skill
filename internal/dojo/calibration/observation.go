package calibration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// Observation is one judge call about one case. It never carries content: only
// identifiers, the case's label metadata and the numbers the judge returned, so
// rates and thresholds are recomputed from observations alone.
type Observation struct {
	CaseID   string  `json:"case_id"`
	ParentID string  `json:"parent_id,omitempty"`
	Kind     Kind    `json:"kind"`
	Operator string  `json:"operator,omitempty"`
	Stage    int     `json:"stage,omitempty"`
	Band     Band    `json:"band,omitempty"`
	Level    float64 `json:"level,omitempty"`
	Expect   Expect  `json:"expect"`
	Golden   bool    `json:"golden,omitempty"`
	HeldOut  bool    `json:"held_out,omitempty"`
	Session  string  `json:"session,omitempty"`
	// Local is true when the local checks were evaluated for this case; LocalFlags
	// then names the ones that flagged it (none is an empty list).
	Local         bool         `json:"local,omitempty"`
	LocalFlags    []string     `json:"local_flags,omitempty"`
	Run           int          `json:"run"`
	Decision      string       `json:"decision,omitempty"`
	Probabilities Distribution `json:"probabilities,omitempty"`
	PAccept       float64      `json:"p_accept"`
	// Scores are the per-criterion answers of an atomic profile (numbers only).
	Scores         map[string]float64 `json:"scores,omitempty"`
	Confidence     float64            `json:"confidence,omitempty"`
	Model          string             `json:"model,omitempty"`
	Profile        string             `json:"profile,omitempty"`
	ProfileVersion string             `json:"profile_version,omitempty"`
	LatencyMS      int64              `json:"latency_ms,omitempty"`
	InputTokens    int                `json:"input_tokens,omitempty"`
	OutputTokens   int                `json:"output_tokens,omitempty"`
	State          string             `json:"state,omitempty"`
	At             string             `json:"at,omitempty"`
}

// SessionOf keeps the observations of one session; an empty session keeps all.
func SessionOf(observations []Observation, session string) []Observation {
	if session == "" {
		return observations
	}
	var out []Observation
	for _, obs := range observations {
		if obs.Session == session {
			out = append(out, obs)
		}
	}
	return out
}

// LatestSession is the session of the last observation that has one.
func LatestSession(observations []Observation) string {
	for i := len(observations) - 1; i >= 0; i-- {
		if observations[i].Session != "" {
			return observations[i].Session
		}
	}
	return ""
}

// DigestOf identifies exactly a set of observations.
func DigestOf(observations []Observation) string {
	var text strings.Builder
	for _, obs := range observations {
		if raw, err := obs.Encode(); err == nil {
			text.Write(raw)
			text.WriteByte('\n')
		}
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(text.String())))
}

// Distribution is the probability the judge gave each option of its choice.
type Distribution map[string]float64

// Answered reports whether the call produced a usable answer.
func (o Observation) Answered() bool { return o.State == "" && o.Decision != "" }

// Approved applies the approval rule at a threshold: the aggregate choice is
// accept and its probability of accept is strictly above the threshold.
func (o Observation) Approved(threshold float64) bool {
	return o.Answered() && o.Decision == "accept" && o.PAccept > threshold
}

// Encode renders one observation as a JSON line.
func (o Observation) Encode() ([]byte, error) {
	raw, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("encode observation: %w", err)
	}
	return raw, nil
}

// DecodeObservations parses a JSONL ledger. Malformed lines are returned as a
// count so the caller can report them; valid lines are kept.
func DecodeObservations(raw []byte) ([]Observation, int) {
	var out []Observation
	bad := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var obs Observation
		if json.Unmarshal([]byte(line), &obs) != nil || obs.CaseID == "" {
			bad++
			continue
		}
		out = append(out, obs)
	}
	return out, bad
}
