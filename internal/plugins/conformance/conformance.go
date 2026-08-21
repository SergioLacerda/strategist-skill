// Package conformance evaluates plugin certification and compatibility evidence.
package conformance

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Level is the evidence-backed conformance ladder.
type Level string

const (
	// LevelC0Descriptor covers descriptor-only conformance.
	LevelC0Descriptor Level = "C0"
	// LevelC1Contract covers validated host contracts.
	LevelC1Contract Level = "C1"
	// LevelC2Runtime covers runtime probe evidence.
	LevelC2Runtime Level = "C2"
	// LevelC3Trusted covers trust and provenance evidence.
	LevelC3Trusted Level = "C3"
)

// CertificationRecord binds conformance to exact input digests.
type CertificationRecord struct {
	SchemaVersion   string
	Level           Level
	PackageDigest   string
	AdapterDigest   string
	HostAPIDigest   string
	ConnectorDigest string
	TestSuiteDigest string
	CertifiedAt     string
	ExpiresAt       string
}

// CertificationInputDigests is the current material being evaluated.
type CertificationInputDigests struct {
	PackageDigest   string
	AdapterDigest   string
	HostAPIDigest   string
	ConnectorDigest string
	TestSuiteDigest string
}

// CertificationResult reports whether a record satisfies current policy.
type CertificationResult struct {
	Accepted    bool
	ReasonCodes []string
}

// Validate checks required certification fields.
func (r CertificationRecord) Validate() error {
	if r.SchemaVersion == "" {
		return fmt.Errorf("certification_record_invalid: schema_version is required")
	}
	if levelRank(r.Level) < 0 {
		return fmt.Errorf("certification_record_invalid: unknown level %q", r.Level)
	}
	for field, value := range map[string]string{
		"package_digest":    r.PackageDigest,
		"adapter_digest":    r.AdapterDigest,
		"host_api_digest":   r.HostAPIDigest,
		"connector_digest":  r.ConnectorDigest,
		"test_suite_digest": r.TestSuiteDigest,
		"certified_at":      r.CertifiedAt,
	} {
		if value == "" {
			return fmt.Errorf("certification_record_invalid: %s is required", field)
		}
	}
	return nil
}

// Stale reports whether any bound digest changed or the record expired.
func (r CertificationRecord) Stale(input CertificationInputDigests, now time.Time) bool {
	if r.PackageDigest != input.PackageDigest ||
		r.AdapterDigest != input.AdapterDigest ||
		r.HostAPIDigest != input.HostAPIDigest ||
		r.ConnectorDigest != input.ConnectorDigest ||
		r.TestSuiteDigest != input.TestSuiteDigest {
		return true
	}
	if r.ExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, r.ExpiresAt)
	if err != nil {
		return true
	}
	return !now.Before(expiresAt)
}

// EvaluateCertification checks freshness and minimum conformance.
func EvaluateCertification(record CertificationRecord, minimum Level, input CertificationInputDigests, now time.Time) CertificationResult {
	var reasons []string
	if err := record.Validate(); err != nil {
		reasons = append(reasons, "certification_record_invalid")
	}
	if record.Stale(input, now) {
		reasons = append(reasons, "certification_stale")
	}
	if levelRank(record.Level) < levelRank(minimum) {
		reasons = append(reasons, "certification_level_too_low")
	}
	return CertificationResult{Accepted: len(reasons) == 0, ReasonCodes: reasons}
}

// CompatibilityMatrix records tested host API x adapter x connector support.
type CompatibilityMatrix struct {
	Rows []CompatibilityRow
}

// CompatibilityRow is one matrix verdict.
type CompatibilityRow struct {
	HostAPI      string
	Adapter      string
	ConnectorAPI string
	Supported    bool
	ReasonCode   string
}

// Check returns the matching compatibility verdict or a stable miss reason.
func (m CompatibilityMatrix) Check(hostAPI, adapter, connectorAPI string) CompatibilityRow {
	for _, row := range m.Rows {
		if row.HostAPI == hostAPI && row.Adapter == adapter && row.ConnectorAPI == connectorAPI {
			return row
		}
	}
	return CompatibilityRow{HostAPI: hostAPI, Adapter: adapter, ConnectorAPI: connectorAPI, Supported: false, ReasonCode: "compatibility_matrix_miss"}
}

// TelemetryInput is the stable plugin event payload.
type TelemetryInput struct {
	EventName         string
	PackageDigest     string
	AdapterDigest     string
	ConnectorID       string
	BindingGeneration int64
	ReasonCode        string
}

// PluginTelemetryEvent returns the canonical Strategist event envelope.
func PluginTelemetryEvent(input TelemetryInput) telemetry.Event {
	return telemetry.Event{
		Name:           "strategist.plugin." + input.EventName,
		Timestamp:      time.Now().UTC(),
		SeverityNumber: telemetry.SeverityInfo,
		Body:           input.ReasonCode,
		Attributes: map[string]any{
			"strategist.plugin.package_digest":     input.PackageDigest,
			"strategist.plugin.adapter_digest":     input.AdapterDigest,
			"strategist.plugin.connector_id":       input.ConnectorID,
			"strategist.plugin.binding_generation": input.BindingGeneration,
			"strategist.plugin.reason_code":        input.ReasonCode,
		},
	}
}

func levelRank(level Level) int {
	switch level {
	case LevelC0Descriptor:
		return 0
	case LevelC1Contract:
		return 1
	case LevelC2Runtime:
		return 2
	case LevelC3Trusted:
		return 3
	default:
		return -1
	}
}
