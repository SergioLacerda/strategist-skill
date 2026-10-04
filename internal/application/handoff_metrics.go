package application

// HandoffMetricsInput is the adapter-neutral input for one Archivist
// handoff-metrics line. Nil pointers mean the measurement was not supplied;
// zero is retained when the caller explicitly measured zero.
type HandoffMetricsInput struct {
	MissionID string

	DiscoveryTokens       *int64
	BriefTokens           *int64
	BriefCompressionRatio *float64
	RefinementReopens     *int
	Revision              *int
	EvidenceCoverageRatio *float64

	Model       string
	Effort      string
	LevelSource string
}

// HandoffMetricsLine is the application DTO consumed by the telemetry
// adapter. It deliberately has no persistence or telemetry dependency.
type HandoffMetricsLine = HandoffMetricsInput
