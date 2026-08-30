package conformance_test

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pkgDigest     = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	adapterDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hostDigest    = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	connDigest    = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	suiteDigest   = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

func TestCertificationRecordBindsToExactDigestsAndBecomesStale(t *testing.T) {
	t.Parallel()

	record := conformance.CertificationRecord{
		SchemaVersion:   "strategist-plugin-certification/v1",
		Level:           conformance.LevelC2Runtime,
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
		CertifiedAt:     "2026-08-20T00:00:00Z",
		ExpiresAt:       "2026-09-20T00:00:00Z",
	}
	require.NoError(t, record.Validate())
	assert.False(t, record.Stale(conformance.CertificationInputDigests{
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
	}, mustTime(t, "2026-08-21T00:00:00Z")))
	assert.True(t, record.Stale(conformance.CertificationInputDigests{
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}, mustTime(t, "2026-08-21T00:00:00Z")))
	assert.True(t, record.Stale(conformance.CertificationInputDigests{
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
	}, mustTime(t, "2026-10-01T00:00:00Z")))
}

func TestEvaluateCertificationRequiresMinimumLevel(t *testing.T) {
	t.Parallel()

	record := conformance.CertificationRecord{
		SchemaVersion:   "strategist-plugin-certification/v1",
		Level:           conformance.LevelC1Contract,
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
		CertifiedAt:     "2026-08-20T00:00:00Z",
	}
	result := conformance.EvaluateCertification(record, conformance.LevelC2Runtime, conformance.CertificationInputDigests{
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
	}, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Accepted)
	assert.Contains(t, result.ReasonCodes, "certification_level_too_low")
}

func TestCompatibilityMatrixReportsHostAdapterConnectorSupport(t *testing.T) {
	t.Parallel()

	matrix := conformance.CompatibilityMatrix{Rows: []conformance.CompatibilityRow{
		{HostAPI: "strategist-plugin-api/1", Adapter: "brainstorming", ConnectorAPI: "strategist-connector-api/1", Supported: true},
		{HostAPI: "strategist-plugin-api/2", Adapter: "brainstorming", ConnectorAPI: "strategist-connector-api/1", Supported: false, ReasonCode: "host_api_unsupported"},
	}}

	ok := matrix.Check("strategist-plugin-api/1", "brainstorming", "strategist-connector-api/1")
	assert.True(t, ok.Supported)

	bad := matrix.Check("strategist-plugin-api/2", "brainstorming", "strategist-connector-api/1")
	assert.False(t, bad.Supported)
	assert.Equal(t, "host_api_unsupported", bad.ReasonCode)

	missing := matrix.Check("strategist-plugin-api/9", "brainstorming", "strategist-connector-api/1")
	assert.False(t, missing.Supported)
	assert.Equal(t, "compatibility_matrix_miss", missing.ReasonCode)
}

func TestPluginTelemetryEventCarriesStableReasonAndDigests(t *testing.T) {
	t.Parallel()

	event := conformance.PluginTelemetryEvent(conformance.TelemetryInput{
		EventName:         "activation",
		PackageDigest:     pkgDigest,
		AdapterDigest:     adapterDigest,
		ConnectorID:       "native",
		BindingGeneration: 7,
		ReasonCode:        "probe_passed",
	})

	require.NoError(t, event.Validate())
	assert.Equal(t, "strategist.plugin.activation", event.Name)
	assert.Equal(t, telemetry.SeverityInfo, event.SeverityNumber)
	assert.Equal(t, pkgDigest, event.Attributes["strategist.plugin.package_digest"])
	assert.Equal(t, int64(7), event.Attributes["strategist.plugin.binding_generation"])
	assert.NotContains(t, event.Attributes, "prompt")
	assert.NotContains(t, event.Attributes, "secret")
}

func TestPluginTelemetryEventSupportsDeprecationReason(t *testing.T) {
	t.Parallel()

	event := conformance.PluginTelemetryEvent(conformance.TelemetryInput{
		EventName:         "deprecation",
		PackageDigest:     pkgDigest,
		AdapterDigest:     adapterDigest,
		ConnectorID:       "native",
		BindingGeneration: 8,
		ReasonCode:        "upstream_eol",
	})

	require.NoError(t, event.Validate())
	assert.Equal(t, "strategist.plugin.deprecation", event.Name)
	assert.Equal(t, "upstream_eol", event.Attributes["strategist.plugin.reason_code"])
}

func validCertificationRecord() conformance.CertificationRecord {
	return conformance.CertificationRecord{
		SchemaVersion:   "strategist-plugin-certification/v1",
		Level:           conformance.LevelC2Runtime,
		PackageDigest:   pkgDigest,
		AdapterDigest:   adapterDigest,
		HostAPIDigest:   hostDigest,
		ConnectorDigest: connDigest,
		TestSuiteDigest: suiteDigest,
		CertifiedAt:     "2026-08-20T00:00:00Z",
	}
}

func TestCertificationRecordValidate_RequiresSchemaVersion(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.SchemaVersion = ""
	err := record.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version is required")
}

func TestCertificationRecordValidate_RejectsUnknownLevel(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.Level = "C99"
	err := record.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown level "C99"`)
}

func TestCertificationRecordValidate_RequiresEachDigestField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*conformance.CertificationRecord)
		field  string
	}{
		{"package_digest", func(r *conformance.CertificationRecord) { r.PackageDigest = "" }, "package_digest"},
		{"adapter_digest", func(r *conformance.CertificationRecord) { r.AdapterDigest = "" }, "adapter_digest"},
		{"host_api_digest", func(r *conformance.CertificationRecord) { r.HostAPIDigest = "" }, "host_api_digest"},
		{"connector_digest", func(r *conformance.CertificationRecord) { r.ConnectorDigest = "" }, "connector_digest"},
		{"test_suite_digest", func(r *conformance.CertificationRecord) { r.TestSuiteDigest = "" }, "test_suite_digest"},
		{"certified_at", func(r *conformance.CertificationRecord) { r.CertifiedAt = "" }, "certified_at"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := validCertificationRecord()
			tt.mutate(&record)
			err := record.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field+" is required")
		})
	}
}

func TestCertificationRecordValidate_AcceptsCompleteRecord(t *testing.T) {
	t.Parallel()

	require.NoError(t, validCertificationRecord().Validate())
}

func TestCertificationRecordStale_NoExpiryNeverGoesStaleOnItsOwn(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	// ExpiresAt left empty.
	input := conformance.CertificationInputDigests{
		PackageDigest: pkgDigest, AdapterDigest: adapterDigest, HostAPIDigest: hostDigest,
		ConnectorDigest: connDigest, TestSuiteDigest: suiteDigest,
	}
	assert.False(t, record.Stale(input, mustTime(t, "2099-01-01T00:00:00Z")))
}

func TestCertificationRecordStale_InvalidExpiryTimestampIsTreatedAsStale(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.ExpiresAt = "not-a-timestamp"
	input := conformance.CertificationInputDigests{
		PackageDigest: pkgDigest, AdapterDigest: adapterDigest, HostAPIDigest: hostDigest,
		ConnectorDigest: connDigest, TestSuiteDigest: suiteDigest,
	}
	assert.True(t, record.Stale(input, mustTime(t, "2026-08-21T00:00:00Z")))
}

func TestEvaluateCertificationRejectsInvalidRecord(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.SchemaVersion = ""
	input := conformance.CertificationInputDigests{
		PackageDigest: pkgDigest, AdapterDigest: adapterDigest, HostAPIDigest: hostDigest,
		ConnectorDigest: connDigest, TestSuiteDigest: suiteDigest,
	}
	result := conformance.EvaluateCertification(record, conformance.LevelC1Contract, input, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Accepted)
	assert.Contains(t, result.ReasonCodes, "certification_record_invalid")
}

func TestEvaluateCertificationTreatsUnknownLevelAsBelowAnyMinimum(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.Level = "C99"
	input := conformance.CertificationInputDigests{
		PackageDigest: pkgDigest, AdapterDigest: adapterDigest, HostAPIDigest: hostDigest,
		ConnectorDigest: connDigest, TestSuiteDigest: suiteDigest,
	}
	result := conformance.EvaluateCertification(record, conformance.LevelC0Descriptor, input, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Accepted)
	assert.Contains(t, result.ReasonCodes, "certification_level_too_low")
}

func TestEvaluateCertificationAcceptsFreshRecordAtOrAboveMinimum(t *testing.T) {
	t.Parallel()

	record := validCertificationRecord()
	record.Level = conformance.LevelC3Trusted
	input := conformance.CertificationInputDigests{
		PackageDigest: pkgDigest, AdapterDigest: adapterDigest, HostAPIDigest: hostDigest,
		ConnectorDigest: connDigest, TestSuiteDigest: suiteDigest,
	}
	result := conformance.EvaluateCertification(record, conformance.LevelC0Descriptor, input, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.True(t, result.Accepted)
	assert.Empty(t, result.ReasonCodes)
}

func mustTime(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, raw)
	require.NoError(t, err)
	return parsed
}
