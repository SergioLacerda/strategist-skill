package telemetry

import (
	"context"
	"fmt"
	"time"
)

// SeverityNumber is the OTel-standard severity enum for log records and
// events (https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber).
// Strategist does not define its own severity taxonomy — it reuses this
// closed, industry-standard scale so every sink (especially sink/otel) can
// forward an Event without a custom translation table.
type SeverityNumber int

// The 24 OTel severity levels, grouped in four steps per standard level
// (matching the OTel spec's own numbering).
const (
	SeverityUnspecified SeverityNumber = 0
	SeverityTrace       SeverityNumber = 1
	SeverityTrace2      SeverityNumber = 2
	SeverityTrace3      SeverityNumber = 3
	SeverityTrace4      SeverityNumber = 4
	SeverityDebug       SeverityNumber = 5
	SeverityDebug2      SeverityNumber = 6
	SeverityDebug3      SeverityNumber = 7
	SeverityDebug4      SeverityNumber = 8
	SeverityInfo        SeverityNumber = 9
	SeverityInfo2       SeverityNumber = 10
	SeverityInfo3       SeverityNumber = 11
	SeverityInfo4       SeverityNumber = 12
	SeverityWarn        SeverityNumber = 13
	SeverityWarn2       SeverityNumber = 14
	SeverityWarn3       SeverityNumber = 15
	SeverityWarn4       SeverityNumber = 16
	SeverityError       SeverityNumber = 17
	SeverityError2      SeverityNumber = 18
	SeverityError3      SeverityNumber = 19
	SeverityError4      SeverityNumber = 20
	SeverityFatal       SeverityNumber = 21
	SeverityFatal2      SeverityNumber = 22
	SeverityFatal3      SeverityNumber = 23
	SeverityFatal4      SeverityNumber = 24
)

// String returns the OTel short severity_text for s (e.g. "INFO", "ERROR").
func (s SeverityNumber) String() string {
	switch {
	case s >= SeverityFatal:
		return "FATAL"
	case s >= SeverityError:
		return "ERROR"
	case s >= SeverityWarn:
		return "WARN"
	case s >= SeverityInfo:
		return "INFO"
	case s >= SeverityDebug:
		return "DEBUG"
	case s >= SeverityTrace:
		return "TRACE"
	default:
		return "UNSPECIFIED"
	}
}

// Authority values — Strategist-specific closed taxonomy (UNC-04; OTel has
// no equivalent concept). AuthorityStrategistLocal marks a decision made by
// Strategist itself with no external governance configured. Use
// AuthorityExternal for a decision returned by a governancebridge.GovernanceBridge
// provider.
const (
	AuthorityStrategistLocal = "strategist-local"
	authorityExternalPrefix  = "external:"
)

// AuthorityExternal formats the Event.Attributes[AttrEventAuthority] value
// for a decision made by an external governance provider identified by providerID
// (e.g. AuthorityExternal("sdd") -> "external:sdd").
func AuthorityExternal(providerID string) string {
	return authorityExternalPrefix + providerID
}

// Event attribute key constants referenced by name elsewhere (acceptance
// checks 6.3 contract_id, 6.4 authority, 6.5 correlation_id via AttrCorrelationID
// in schema.go). Namespaced consistently with schema.go's Attr* constants.
const (
	AttrEventContractID = "strategist.contract_id"
	AttrEventExpected   = "strategist.expected"
	AttrEventObserved   = "strategist.observed"
	AttrEventDecision   = "strategist.decision"
	AttrEventAuthority  = "strategist.authority"
)

// Event is Strategist's canonical telemetry event envelope (item 1 of the
// governança-plugável document). It is additive — PolicyEvent, RouteDecision,
// HandoffMetrics, SniperConflict, and MissionMetrics are unchanged and keep
// emitting exactly as before (UNC-01). Event's shape mirrors OTel's
// LogRecord-based Events data model
// (https://opentelemetry.io/docs/specs/semconv/general/events/) rather than
// an ad hoc struct, so sink/otel can forward it without a custom translation
// layer. Name is namespaced ("strategist.<phase>.<event>") instead of
// carrying a separate domain field, matching the OTel Event WG's own
// decision to fold event.domain into event.name.
type Event struct {
	Name           string
	Timestamp      time.Time
	SeverityNumber SeverityNumber
	Body           string
	// Attributes carries event-specific data as namespaced "strategist.*"
	// keys — contract_id, phase, expected, observed, decision, authority,
	// mission_id, correlation_id, and any other payload the emitter needs.
	Attributes map[string]any
	// TraceID/SpanID are the W3C trace context identifiers — OTel's native
	// correlation mechanism (acceptance check 6.5: correlation_id crosses
	// phases). A Strategist-domain correlation_id may additionally be carried
	// as an attribute (schema.go's AttrCorrelationID) for consumers that key
	// on it directly rather than on trace context.
	TraceID string
	SpanID  string

	// RunID identifies one mission-runtime execution — one process
	// invocation of the pipeline (see MissionRun in mission_run.go). It is
	// deliberately reused from mission_id rather than freshly generated:
	// today's FSM retries (StateRetryingRefinement, StateRetryingExecution,
	// StateRetryingDirectExec in internal/domain/state_machine.go) are
	// intra-run retries of a single slot within one process invocation, not
	// a new run of the whole mission, so mission_id and run_id currently
	// coincide 1:1. If Strategist later gains a genuine "resume this
	// mission as a new process invocation" path, that is the point at which
	// RunID should diverge from mission_id (e.g. by suffixing an attempt
	// number) — see NewEvent and NextSequence.
	RunID string
	// Sequence is a monotonically increasing counter assigned per RunID at
	// emission time (via NextSequence/NewEvent), starting at 1. Zero means
	// "no sequence assigned" (e.g. an event built before this field existed
	// via a bare Event{...} literal). ValidateSequence uses gaps in this
	// numbering to detect an event that was emitted but never reached a
	// given sink — distinct from a phase that simply never ran, which
	// leaves no event and therefore no gap.
	Sequence uint64
	// SchemaVersion tags the shape of this envelope itself (currently
	// CurrentEventSchemaVersion), not any domain/mission schema. Constant
	// for now; exists so a future incompatible envelope change can be
	// detected by readers of persisted event logs.
	SchemaVersion string
	// Complete marks a terminal event — e.g. a phase's done/blocked
	// outcome — as opposed to an in-flight/running event for the same unit
	// of work. Downstream consumers use it to tell "ran to completion"
	// apart from "started but never finished," which a missing event alone
	// cannot distinguish from "never started."
	Complete bool
}

// CurrentEventSchemaVersion is the schema_version stamped onto every Event
// built via NewEvent. Bump it if Event's envelope shape changes in a way
// that matters to a persisted-log reader.
const CurrentEventSchemaVersion = "1.0"

// Validate returns an error if e is missing a required field. Critical
// events (SeverityNumber >= SeverityError) MUST carry a contract_id
// attribute (acceptance check 6.3) — a critical event with no traceable
// contract is exactly the kind of silent gap the envelope exists to prevent.
func (e Event) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("event: name is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("event: timestamp is required")
	}
	if e.SeverityNumber >= SeverityError {
		if _, ok := e.Attributes[AttrEventContractID]; !ok {
			return fmt.Errorf("event: critical event (severity>=ERROR) requires attribute %q", AttrEventContractID)
		}
	}
	return nil
}

// EventSink is the pluggable telemetry emission interface. Every concrete
// sink under internal/telemetry/sink/ implements this — noop, slog, jsonl,
// otel, external — selected by internal/telemetry/sink.Select per the
// mission's configuration policy (item 1 §7).
type EventSink interface {
	Emit(ctx context.Context, event Event) error
}
