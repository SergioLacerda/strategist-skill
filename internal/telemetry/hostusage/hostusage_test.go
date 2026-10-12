package hostusage

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "TOP-SECRET-CONVERSATION-CONTENT"

func claudeAssistant(id, ts, model string, in, out, cacheRead, cacheCreate int, extra string) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"sessionId":"sess-1","isSidechain":false,`+
		`"message":{"id":%q,"model":%q,"content":[{"type":"text","text":%q}],"usage":{"input_tokens":%d,"output_tokens":%d,`+
		`"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d%s}}}`,
		ts, id, model, secret, in, out, cacheRead, cacheCreate, extra)
}

func win(start, end string) Window {
	s, _ := time.Parse(time.RFC3339, start)
	e, _ := time.Parse(time.RFC3339, end)
	return Window{Start: s, End: e}
}

func TestParseClaudeSessionDeduplicatesContentBlockLinesByMessageID(t *testing.T) {
	// three content-block lines of one response repeat the same usage
	lines := []string{
		claudeAssistant("msg_1", "2026-10-10T10:00:00Z", "m-a", 10, 500, 1000, 200, ""),
		claudeAssistant("msg_1", "2026-10-10T10:00:01Z", "m-a", 10, 500, 1000, 200, ""),
		claudeAssistant("msg_1", "2026-10-10T10:00:02Z", "m-a", 10, 500, 1000, 200, ""),
		claudeAssistant("msg_2", "2026-10-10T10:05:00Z", "m-a", 5, 100, 2000, 0, `,"output_tokens_details":{"thinking_tokens":40}`),
	}
	calls, unrecognized, err := ParseClaudeSession(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	require.NoError(t, err)
	assert.Zero(t, unrecognized)
	require.Len(t, calls, 2)
	assert.Equal(t, int64(1210), calls[0].Input, "input includes cache creation and cache read")
	assert.Equal(t, int64(1000), calls[0].Cache)
	assert.Equal(t, int64(500), calls[0].Output)
	require.NotNil(t, calls[1].Reasoning)
	assert.Equal(t, int64(40), *calls[1].Reasoning)

	obs, err := ObserveClaude(calls, win("2026-10-10T09:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, 2, obs.Calls)
	assert.Equal(t, int64(500+100), *obs.Output, "a naive sum over lines would be 1600")
	assert.Equal(t, int64(1210+2005), *obs.Input)
	assert.Equal(t, int64(3000), *obs.Cache)
	assert.Equal(t, "sess-1", obs.SessionID)
	assert.Equal(t, []string{"m-a"}, obs.Models)
	assert.Equal(t, "anthropic", obs.Provider)
	assert.True(t, obs.HelperModelsMayBeUnseen)
}

func TestObservationNeverCarriesConversationContent(t *testing.T) {
	line := claudeAssistant("msg_1", "2026-10-10T10:00:00Z", "m-a", 1, 2, 3, 4, "")
	calls, _, err := ParseClaudeSession(strings.NewReader(line + "\n"))
	require.NoError(t, err)
	obs, err := ObserveClaude(calls, win("2026-10-10T09:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.NotContains(t, fmt.Sprintf("%#v %#v", calls, obs), secret)
	assert.NotContains(t, fmt.Sprintf("%+v %+v", *obs.Input, *obs.Output), secret)
}

func TestObserveClaudeWindowIsHalfOpenAndAcrossFilesCountsOnce(t *testing.T) {
	a := claudeAssistant("msg_1", "2026-10-10T10:00:00Z", "m-a", 1, 100, 0, 0, "")
	b := claudeAssistant("msg_2", "2026-10-10T10:10:00Z", "m-a", 1, 200, 0, 0, "")
	c := claudeAssistant("msg_3", "2026-10-10T10:20:00Z", "m-a", 1, 400, 0, 0, "")
	main, _, err := ParseClaudeSession(strings.NewReader(a + "\n" + b + "\n"))
	require.NoError(t, err)
	sub, _, err := ParseClaudeSession(strings.NewReader(b + "\n" + c + "\n")) // msg_2 appears in both files
	require.NoError(t, err)

	obs, err := ObserveClaude(append(main, sub...), win("2026-10-10T10:00:00Z", "2026-10-10T10:20:00Z"))
	require.NoError(t, err)
	assert.Equal(t, 2, obs.Calls, "start inclusive, end exclusive, msg_2 once")
	assert.Equal(t, int64(300), *obs.Output)
}

func TestObserveClaudeEmptyWindowObservesNothingNotZero(t *testing.T) {
	line := claudeAssistant("msg_1", "2026-10-10T10:00:00Z", "m-a", 1, 2, 3, 4, "")
	calls, _, err := ParseClaudeSession(strings.NewReader(line + "\n"))
	require.NoError(t, err)
	obs, err := ObserveClaude(calls, win("2026-10-10T11:00:00Z", "2026-10-10T12:00:00Z"))
	require.NoError(t, err)
	assert.Zero(t, obs.Calls)
	assert.Nil(t, obs.Input)
	assert.Nil(t, obs.Output)
}

func TestParseClaudeSessionFailsClosedOnAnUnknownUsageShape(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-10-10T10:00:00Z","message":{"id":"m","usage":{"tokens":5}}}`
	_, unrecognized, err := ParseClaudeSession(strings.NewReader(line + "\n"))
	require.ErrorIs(t, err, ErrUnrecognizedFormat)
	assert.Equal(t, 1, unrecognized)

	calls, unrecognized, err := ParseClaudeSession(strings.NewReader("{not json}\n\n"))
	require.NoError(t, err)
	assert.Empty(t, calls)
	assert.Zero(t, unrecognized)
}

func TestParseClaudeSessionReadsVeryLongLines(t *testing.T) {
	long := strings.Repeat("x", 3<<20)
	line := fmt.Sprintf(`{"type":"user","message":{"content":%q}}`, long)
	good := claudeAssistant("msg_1", "2026-10-10T10:00:00Z", "m-a", 1, 2, 0, 0, "")
	calls, _, err := ParseClaudeSession(strings.NewReader(line + "\n" + good + "\n"))
	require.NoError(t, err)
	assert.Len(t, calls, 1)
}

func codexCount(ts string, in, cached, out, reasoning int) string {
	return fmt.Sprintf(`{"type":"event_msg","timestamp":%q,"payload":{"type":"token_count","info":{`+
		`"total_token_usage":{"input_tokens":%d,"cached_input_tokens":%d,"output_tokens":%d,"reasoning_output_tokens":%d,"total_tokens":%d},`+
		`"last_token_usage":{"input_tokens":999999,"output_tokens":999999}}}}`, ts, in, cached, out, reasoning, in+out)
}

func TestObserveCodexUsesDeltasOfTheCumulativeCounter(t *testing.T) {
	file := strings.Join([]string{
		`{"type":"session_meta","timestamp":"2026-10-10T09:00:00Z","payload":{"id":"thread-1"}}`,
		codexCount("2026-10-10T09:30:00Z", 1000, 800, 100, 10),
		codexCount("2026-10-10T10:10:00Z", 3000, 2500, 400, 50),
		codexCount("2026-10-10T10:10:00Z", 3000, 2500, 400, 50), // repeated snapshot
		codexCount("2026-10-10T10:40:00Z", 7000, 6000, 900, 120),
		codexCount("2026-10-10T11:30:00Z", 9000, 8000, 1200, 150),
	}, "\n") + "\n"
	snaps, session, unrecognized, err := ParseCodexSession(strings.NewReader(file))
	require.NoError(t, err)
	assert.Zero(t, unrecognized)
	assert.Equal(t, "thread-1", session)

	obs, err := ObserveCodex(snaps, session, win("2026-10-10T10:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, int64(7000-1000), *obs.Input, "delta of the counter, never the repeated last_token_usage")
	assert.Equal(t, int64(900-100), *obs.Output)
	assert.Equal(t, int64(6000-800), *obs.Cache)
	assert.Equal(t, int64(120-10), *obs.Reasoning)
	assert.Equal(t, 2, obs.Calls, "the repeated snapshot is not a call")
	assert.Equal(t, "openai", obs.Provider)
	assert.False(t, obs.HelperModelsMayBeUnseen)
}

func TestObserveCodexThreadStartingInsideTheWindowUsesAZeroBase(t *testing.T) {
	file := codexCount("2026-10-10T10:10:00Z", 500, 0, 50, 0) + "\n" + codexCount("2026-10-10T10:20:00Z", 800, 100, 90, 5) + "\n"
	snaps, _, _, err := ParseCodexSession(strings.NewReader(file))
	require.NoError(t, err)
	obs, err := ObserveCodex(snaps, "t", win("2026-10-10T10:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, int64(800), *obs.Input)
}

func TestObserveCodexCounterResetMakesTheIntervalUnavailable(t *testing.T) {
	file := codexCount("2026-10-10T10:10:00Z", 5000, 0, 500, 0) + "\n" + codexCount("2026-10-10T10:20:00Z", 100, 0, 10, 0) + "\n"
	snaps, _, _, err := ParseCodexSession(strings.NewReader(file))
	require.NoError(t, err)
	obs, err := ObserveCodex(snaps, "t", win("2026-10-10T10:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.True(t, obs.CounterReset)
	assert.Nil(t, obs.Input)
	assert.Nil(t, obs.Output)
	assert.Zero(t, obs.Calls)
}

func TestObserveCodexNoSnapshotInWindowObservesNothing(t *testing.T) {
	snaps, _, _, err := ParseCodexSession(strings.NewReader(codexCount("2026-10-10T08:00:00Z", 1, 0, 1, 0) + "\n"))
	require.NoError(t, err)
	obs, err := ObserveCodex(snaps, "t", win("2026-10-10T10:00:00Z", "2026-10-10T11:00:00Z"))
	require.NoError(t, err)
	assert.Nil(t, obs.Input)
	assert.Zero(t, obs.Calls)
}

func TestParseCodexSessionFailsClosedOnAnUnknownCounterShape(t *testing.T) {
	line := `{"type":"event_msg","timestamp":"2026-10-10T10:00:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"tokens":9}}}}`
	_, _, unrecognized, err := ParseCodexSession(strings.NewReader(line + "\n"))
	require.ErrorIs(t, err, ErrUnrecognizedFormat)
	assert.Equal(t, 1, unrecognized)
}

func TestWindowValidation(t *testing.T) {
	require.Error(t, Window{}.Validate())
	require.Error(t, win("2026-10-10T11:00:00Z", "2026-10-10T10:00:00Z").Validate())
	require.NoError(t, win("2026-10-10T10:00:00Z", "2026-10-10T11:00:00Z").Validate())
	_, err := ObserveClaude(nil, Window{})
	require.Error(t, err)
	_, err = ObserveCodex(nil, "", Window{})
	require.Error(t, err)
}
