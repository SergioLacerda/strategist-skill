package hostusage

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ClaudeCall is one deduplicated API response of a Claude Code session.
type ClaudeCall struct {
	MessageID string
	At        time.Time
	Model     string
	SessionID string
	Sidechain bool
	Input     int64 // input + cache creation + cache read
	Output    int64
	Cache     int64 // cache read
	Reasoning *int64
}

type claudeLine struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	IsSidechain bool   `json:"isSidechain"`
	Message     *struct {
		ID    string       `json:"id"`
		Model string       `json:"model"`
		Usage *claudeUsage `json:"usage"`
	} `json:"message"`
}

type claudeUsage struct {
	Input         *int64 `json:"input_tokens"`
	Output        *int64 `json:"output_tokens"`
	CacheCreation *int64 `json:"cache_creation_input_tokens"`
	CacheRead     *int64 `json:"cache_read_input_tokens"`
	Details       *struct {
		Thinking *int64 `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
}

// ParseClaudeSession reads Claude Code transcript lines and returns one call
// per message id. A response is written as one record per content block, all
// repeating the same usage, so lines are deduplicated by message id (the last
// line wins) before anything is summed. Content, tool input and every other
// field are discarded. unrecognized counts assistant records whose usage
// lacks the expected counters.
func ParseClaudeSession(r io.Reader) (calls []ClaudeCall, unrecognized int, err error) {
	byID := map[string]*ClaudeCall{}
	var order []string
	if err = forEachLine(r, func(line []byte) { noteClaudeLine(line, byID, &order, &unrecognized) }); err != nil {
		return nil, 0, fmt.Errorf("hostusage: %w", err)
	}
	for _, id := range order {
		calls = append(calls, *byID[id])
	}
	if len(calls) == 0 && unrecognized > 0 {
		return nil, unrecognized, ErrUnrecognizedFormat
	}
	return calls, unrecognized, nil
}

func noteClaudeLine(line []byte, byID map[string]*ClaudeCall, order *[]string, unrecognized *int) {
	var rec claudeLine
	if json.Unmarshal(line, &rec) != nil || rec.Type != "assistant" || rec.Message == nil {
		return
	}
	call, ok := claudeCallOf(rec)
	if !ok {
		*unrecognized++
		return
	}
	if prior, seen := byID[call.MessageID]; seen {
		call.At = prior.At // a message starts at its first line
		*prior = call
		return
	}
	byID[call.MessageID] = &call
	*order = append(*order, call.MessageID)
}

func claudeCallOf(rec claudeLine) (ClaudeCall, bool) {
	u := rec.Message.Usage
	at, okTime := parseTime(rec.Timestamp)
	if u == nil || u.Input == nil || u.Output == nil || rec.Message.ID == "" || !okTime {
		return ClaudeCall{}, false
	}
	call := ClaudeCall{
		MessageID: rec.Message.ID, At: at, Model: rec.Message.Model, SessionID: rec.SessionID,
		Sidechain: rec.IsSidechain, Input: *u.Input, Output: *u.Output,
	}
	if u.CacheCreation != nil {
		call.Input += *u.CacheCreation
	}
	if u.CacheRead != nil {
		call.Input += *u.CacheRead
		call.Cache = *u.CacheRead
	}
	if u.Details != nil {
		call.Reasoning = u.Details.Thinking
	}
	return call, true
}

// ObserveClaude sums the calls inside the window. Calls seen in several
// files (a main transcript and a sub-agent transcript) are counted once.
func ObserveClaude(calls []ClaudeCall, w Window) (Observation, error) {
	if err := w.Validate(); err != nil {
		return Observation{}, err
	}
	obs := Observation{Host: HostClaude, Provider: "anthropic", WindowStart: w.Start, WindowEnd: w.End, HelperModelsMayBeUnseen: true}
	seen := map[string]bool{}
	var models []string
	for _, c := range calls {
		if !w.contains(c.At) || seen[c.MessageID] {
			continue
		}
		seen[c.MessageID] = true
		obs.Calls++
		in, out, cache := c.Input, c.Output, c.Cache
		addCount(&obs.Input, &in)
		addCount(&obs.Output, &out)
		addCount(&obs.Cache, &cache)
		addCount(&obs.Reasoning, c.Reasoning)
		models = append(models, c.Model)
		if obs.SessionID == "" {
			obs.SessionID = c.SessionID
		}
	}
	obs.Models = sortedUnique(models)
	return obs, nil
}
