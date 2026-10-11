package missionview

import "github.com/SergioLacerda/strategist-skill/internal/telemetry"

func n(v int64) *int64 { return &v }

type ur = telemetry.MissionUsageRecord

func inc(id, phase, inv string, attempt int, in, out *int64, at string) ur {
	return ur{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: id, MissionID: "m-1",
		Phase: phase, Role: "ranger", InvocationID: inv, Attempt: attempt, UsageScope: "invocation",
		RecordKind: "incremental", TokensInput: in, TokensOutput: out,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: at,
	}
}

func snap(id, session, phase string, in, out *int64, at string) ur {
	r := inc(id, phase, "", 1, in, out, at)
	r.RecordKind, r.SessionID, r.UsageScope = "cumulative_snapshot", session, "session"
	return r
}
