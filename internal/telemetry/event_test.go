package telemetry_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // G304: test-controlled temp path
	require.NoError(t, err)
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestEvent_Validate(t *testing.T) {
	t.Parallel()
	valid := telemetry.Event{Name: "strategist.governance.decision", Timestamp: time.Now()}
	require.NoError(t, valid.Validate())

	require.ErrorContains(t, (telemetry.Event{Timestamp: time.Now()}).Validate(), "name is required")
	require.ErrorContains(t, (telemetry.Event{Name: "x"}).Validate(), "timestamp is required")
}

func TestSeverityNumber_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sev  telemetry.SeverityNumber
		want string
	}{
		{telemetry.SeverityUnspecified, "UNSPECIFIED"},
		{telemetry.SeverityTrace, "TRACE"},
		{telemetry.SeverityDebug, "DEBUG"},
		{telemetry.SeverityInfo, "INFO"},
		{telemetry.SeverityWarn, "WARN"},
		{telemetry.SeverityError, "ERROR"},
		{telemetry.SeverityFatal4, "FATAL"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.sev.String())
	}
}

func TestAuthorityExternal(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "external:sdd", telemetry.AuthorityExternal("sdd"))
	assert.Equal(t, "strategist-local", telemetry.AuthorityStrategistLocal)
}

func TestMarshalEventLine_RequiredFieldsAndAttributes(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	event := telemetry.Event{
		Name:           "strategist.governance.decision",
		Timestamp:      ts,
		SeverityNumber: telemetry.SeverityInfo,
		Body:           "governance decision evaluated",
		Attributes: map[string]any{
			telemetry.AttrEventContractID: "compile-domain",
			telemetry.AttrEventAuthority:  telemetry.AuthorityStrategistLocal,
		},
		TraceID: "trace-abc",
	}
	line, err := telemetry.MarshalEventLine(event)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &decoded))
	assert.Equal(t, "strategist.governance.decision", decoded["name"])
	assert.Equal(t, "INFO", decoded["severity_text"])
	assert.Equal(t, "trace-abc", decoded["trace_id"])
	attrs, ok := decoded["attributes"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "compile-domain", attrs[telemetry.AttrEventContractID])
}

// TestEvent_Validate_CriticalRequiresContractID is acceptance check 6.3:
// critical events (severity >= ERROR) must carry a contract_id attribute.
func TestEvent_Validate_CriticalRequiresContractID(t *testing.T) {
	t.Parallel()
	base := telemetry.Event{Name: "strategist.governance.decision", Timestamp: time.Now(), SeverityNumber: telemetry.SeverityError}
	err := base.Validate()
	require.ErrorContains(t, err, "strategist.contract_id")

	base.Attributes = map[string]any{telemetry.AttrEventContractID: "compile-domain"}
	require.NoError(t, base.Validate())

	// Below ERROR severity, contract_id is not required.
	warn := telemetry.Event{Name: "strategist.route.decision", Timestamp: time.Now(), SeverityNumber: telemetry.SeverityWarn}
	require.NoError(t, warn.Validate())
}

// TestMarshalEventLine_IncludesRunEnvelopeFields covers Task 11 (telemetry
// envelope completeness, G23): run_id, sequence, schema_version, and
// complete must survive the jsonl round trip so a reader of the persisted
// event log can run ValidateSequence against it.
func TestMarshalEventLine_IncludesRunEnvelopeFields(t *testing.T) {
	t.Parallel()
	event := telemetry.NewEvent("strategist.discovery.done", telemetry.SeverityInfo, "run-123", true)

	line, err := telemetry.MarshalEventLine(event)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &decoded))
	assert.Equal(t, "run-123", decoded["run_id"])
	assert.InDelta(t, float64(1), decoded["sequence"], 0)
	assert.Equal(t, telemetry.CurrentEventSchemaVersion, decoded["schema_version"])
	assert.Equal(t, true, decoded["complete"])
}

func TestMarshalEventLine_InvalidEvent(t *testing.T) {
	t.Parallel()
	_, err := telemetry.MarshalEventLine(telemetry.Event{})
	require.ErrorContains(t, err, "name is required")
}

// TestAppendEventLine_CorrelationIDCrossesPhases exercises acceptance check
// 6.5: two events sharing the same correlation_id/trace_id (as if emitted
// from two different mission phases) both land in the log with the id
// intact and unmodified.
func TestAppendEventLine_CorrelationIDCrossesPhases(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	discoveryEvent := telemetry.Event{
		Name:      "strategist.discovery.done",
		Timestamp: time.Now(),
		TraceID:   "trace-xyz",
		Attributes: map[string]any{
			telemetry.AttrCorrelationID: "corr-1",
		},
	}
	refinementEvent := telemetry.Event{
		Name:      "strategist.refinement.done",
		Timestamp: time.Now(),
		TraceID:   "trace-xyz",
		Attributes: map[string]any{
			telemetry.AttrCorrelationID: "corr-1",
		},
	}
	require.NoError(t, telemetry.AppendEventLine(path, discoveryEvent))
	require.NoError(t, telemetry.AppendEventLine(path, refinementEvent))

	lines := readLines(t, path)
	require.Len(t, lines, 2)
	for _, line := range lines {
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &decoded))
		assert.Equal(t, "trace-xyz", decoded["trace_id"])
		attrs := decoded["attributes"].(map[string]any)
		assert.Equal(t, "corr-1", attrs[telemetry.AttrCorrelationID])
	}
}

func TestAppendEventLine_InvalidEventNotWritten(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	err := telemetry.AppendEventLine(path, telemetry.Event{})
	require.Error(t, err)
}
