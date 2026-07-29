package telemetry

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateOutcomeLine_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"mission_id":"m-1","status":"completed","timestamp":"2026-06-02T12:00:00Z"}`,
		`{"mission_id":"m-2","status":"analysis_delivered","timestamp":"2026-06-02T13:00:00Z","gates":[{"type":"approval_gate","approved_at":"2026-06-02T13:01:00Z","response":"sim"}]}`,
		`{"mission_id":"m-3","status":"completed","timestamp":"2026-06-02T14:00:00Z","jewel_ids":["jewel-a","jewel-b"]}`,
	}
	for _, line := range cases {
		if err := ValidateOutcomeLine(line); err != nil {
			t.Fatalf("unexpected error for valid line %q: %v", line, err)
		}
	}
}

func TestValidateOutcomeLine_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		line    string
		missing string
	}{
		{`{"status":"completed","timestamp":"2026-06-02T12:00:00Z"}`, "mission_id"},
		{`{"mission_id":"m-1","timestamp":"2026-06-02T12:00:00Z"}`, "status"},
		{`{"mission_id":"m-1","status":"completed"}`, "timestamp"},
		{`{}`, "mission_id"},
	}
	for _, tc := range cases {
		err := ValidateOutcomeLine(tc.line)
		if err == nil {
			t.Fatalf("expected error for missing %q, got nil", tc.missing)
		}
		if !strings.Contains(err.Error(), tc.missing) {
			t.Fatalf("error should mention %q: %v", tc.missing, err)
		}
	}
}

func TestValidateOutcomeLine_InvalidJSON(t *testing.T) {
	t.Parallel()
	if err := ValidateOutcomeLine("not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestOutcomeEntry_OmitsEmptyJewelIDs(t *testing.T) {
	t.Parallel()
	entry := OutcomeEntry{
		MissionID: "m-no-jewels",
		Status:    "completed",
		Timestamp: "2026-07-25T00:00:00Z",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal outcome entry: %v", err)
	}
	if strings.Contains(string(data), "jewel_ids") {
		t.Fatalf("expected empty jewel_ids to be omitted, got: %s", data)
	}
}
