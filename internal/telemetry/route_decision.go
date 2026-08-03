package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// RouteDecision is the canonical JSON structure written to route-decisions.jsonl
// per mission, mirroring the fields in
// .strategist/schemas/scout-route-decision.schema.yaml. It is the durable
// counterpart to Scout's route_decision record, which is otherwise only
// emitted as a human-readable log line
// (contracts/machine/scout-routing.yaml#emit.on_route_selected).
type RouteDecision struct {
	MissionID        string  `json:"mission_id"`
	RequestCategory  string  `json:"request_category"`
	SelectedRoute    string  `json:"selected_route"`
	RouteReason      string  `json:"route_reason"`
	RouteConfidence  float64 `json:"route_confidence"`
	EvidenceState    string  `json:"evidence_state"`
	DiscoverySubtype string  `json:"discovery_subtype,omitempty"`
	FallbackRoute    string  `json:"fallback_route"`
	Provider         string  `json:"provider,omitempty"`
	Timestamp        string  `json:"timestamp"`
}

// allowedSelectedRoutes mirrors scout-route-decision.schema.yaml#selected_route.allowed_values.
var allowedSelectedRoutes = map[string]bool{
	"critical_hit":               true,
	"implementation_short_route": true,
	"full_pipeline":              true,
}

// allowedEvidenceStates mirrors scout-route-decision.schema.yaml#evidence_state.allowed_values.
var allowedEvidenceStates = map[string]bool{
	"explicit":           true,
	"insufficient":       true,
	"requires_discovery": true,
}

// ValidateRouteDecisionLine parses a single JSON line and checks required
// fields and allowed values per scout-route-decision.schema.yaml.
func ValidateRouteDecisionLine(line string) error {
	var d RouteDecision
	if err := json.Unmarshal([]byte(line), &d); err != nil {
		return fmt.Errorf("route decision line is not valid JSON: %w", err)
	}
	var errs []error
	errs = append(errs, requiredRouteField("mission_id", d.MissionID)...)
	errs = append(errs, requiredRouteField("request_category", d.RequestCategory)...)
	errs = append(errs, allowedRouteValue("selected_route", d.SelectedRoute, allowedSelectedRoutes)...)
	errs = append(errs, requiredRouteField("route_reason", d.RouteReason)...)
	errs = append(errs, routeConfidenceRange(d.RouteConfidence)...)
	errs = append(errs, allowedRouteValue("evidence_state", d.EvidenceState, allowedEvidenceStates)...)
	errs = append(errs, fallbackRouteValue(d.FallbackRoute)...)
	errs = append(errs, requiredRouteField("timestamp", d.Timestamp)...)
	return errors.Join(errs...)
}

func requiredRouteField(name, value string) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	return nil
}

func allowedRouteValue(name, value string, allowed map[string]bool) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	if !allowed[value] {
		return []error{fmt.Errorf("%s %q is not an allowed value", name, value)}
	}
	return nil
}

func routeConfidenceRange(confidence float64) []error {
	if confidence < 0.0 || confidence > 1.0 {
		return []error{fmt.Errorf("route_confidence %v is out of range [0.0, 1.0]", confidence)}
	}
	return nil
}

func fallbackRouteValue(fallbackRoute string) []error {
	if fallbackRoute != "" && fallbackRoute != "full_pipeline" {
		return []error{fmt.Errorf("fallback_route %q must be full_pipeline", fallbackRoute)}
	}
	return nil
}

// AppendRouteDecisionLine validates line and appends it with a newline to
// path, unless an entry with the same mission_id already exists in path —
// the same idempotency-by-mission_id discipline as AppendOutcomeLine
// (ADR-0004), reusing the same shared-flock read-then-write pattern so
// concurrent appenders and a future flush routine remain compatible.
// If validation fails the line is not written and the error is returned.
// appended reports whether the line was written (false means a duplicate
// mission_id was skipped). The file is created if absent.
func AppendRouteDecisionLine(path, line string) (appended bool, err error) { //nolint:dupl // mirrors outcome append semantics for a separate schema
	if err = ValidateRouteDecisionLine(line); err != nil {
		return false, fmt.Errorf("route decision validation failed: %w", err)
	}
	var entry RouteDecision
	if err = json.Unmarshal([]byte(line), &entry); err != nil {
		return false, fmt.Errorf("route decision line is not valid JSON: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // G304: route-decision path is owned by runtime memory
	if err != nil {
		return false, fmt.Errorf("open route decisions file: %w", err)
	}
	defer closeFileWithContext(f, &err, "close route decisions file")
	return appendRouteDecisionLineLocked(f, entry.MissionID, line)
}

func appendRouteDecisionLineLocked(f *os.File, missionID, line string) (appended bool, err error) {
	if err = lockFile(f); err != nil {
		return false, fmt.Errorf("lock route decisions file: %w", err)
	}
	defer unlockRouteDecisionFile(f, &err)

	exists, err := routeDecisionMissionIDExists(f, missionID)
	if err != nil {
		return false, fmt.Errorf("scan route decisions file: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return false, fmt.Errorf("seek route decisions file: %w", err)
	}
	if _, err = fmt.Fprintln(f, line); err != nil {
		return false, fmt.Errorf("write route decision line: %w", err)
	}
	return true, nil
}

// routeDecisionMissionIDExists scans f for an existing route decision entry
// whose mission_id matches. Lines that fail to parse are tolerated and
// skipped rather than treated as an error, consistent with
// missionIDExists's treatment of outcomes.jsonl as append-only historical
// data that may include entries from schema revisions.
func routeDecisionMissionIDExists(f *os.File, missionID string) (bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek route decisions file start: %w", err)
	}
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		if routeDecisionLineHasMissionID(scanner.Bytes(), missionID) {
			return true, nil
		}
	}
	return false, jsonlScannerErr(scanner, "scan route decisions file")
}

func routeDecisionLineHasMissionID(line []byte, missionID string) bool {
	if len(line) == 0 {
		return false
	}
	var entry RouteDecision
	if err := json.Unmarshal(line, &entry); err != nil {
		return false
	}
	return entry.MissionID == missionID
}

func unlockRouteDecisionFile(f *os.File, err *error) {
	if unlockErr := unlockFile(f); unlockErr != nil && *err == nil {
		*err = fmt.Errorf("unlock route decisions file: %w", unlockErr)
	}
}
