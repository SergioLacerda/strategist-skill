package telemetry

// MissionLifecycleRecordFromEvent rebuilds a lifecycle record from an event;
// ok is false for any other event.
func MissionLifecycleRecordFromEvent(e Event) (MissionLifecycleRecord, bool) {
	kind, ok := lifecycleKindsByEvent[e.Name]
	if !ok || attrString(e.Attributes, AttrEventContractID) != MissionLifecycleContractID {
		return MissionLifecycleRecord{}, false
	}
	return MissionLifecycleRecord{
		SchemaVersion: attrString(e.Attributes, AttrLifecycleSchema),
		MarkerID:      attrString(e.Attributes, AttrLifecycleMarkerID), MarkerKind: kind,
		MissionID: attrString(e.Attributes, AttrMissionID), Phase: attrString(e.Attributes, AttrPhase),
		PhaseRunID: attrString(e.Attributes, AttrLifecyclePhaseRunID),
		EndReason:  attrString(e.Attributes, AttrLifecycleEndReason),
		Stage:      attrString(e.Attributes, AttrLifecycleStage),
		ObservedAt: attrString(e.Attributes, AttrLifecycleObservedAt),
	}, true
}

// MissionUsageRecordFromEvent rebuilds a usage record from an event; ok is
// false for any other event.
func MissionUsageRecordFromEvent(e Event) (MissionUsageRecord, bool) {
	if e.Name != EventUsageRecorded || attrString(e.Attributes, AttrEventContractID) != MissionUsageContractID {
		return MissionUsageRecord{}, false
	}
	a := e.Attributes
	return MissionUsageRecord{
		SchemaVersion: attrString(a, AttrUsageSchema), RecordID: attrString(a, AttrUsageRecordID),
		EventID: attrString(a, AttrUsageEventID), MissionID: attrString(a, AttrMissionID),
		Phase: attrString(a, AttrPhase), PhaseRunID: attrString(a, AttrUsagePhaseRunID),
		Role: attrString(a, AttrUsageRole), InvocationID: attrString(a, AttrUsageInvocationID),
		Attempt: int(attrCount(a, AttrUsageAttempt)), UsageScope: attrString(a, AttrUsageScope),
		SessionID: attrString(a, AttrUsageSessionID), RecordKind: attrString(a, AttrUsageRecordKind),
		TokensInput: attrOptionalCount(a, AttrUsageTokensInput), TokensOutput: attrOptionalCount(a, AttrUsageTokensOutput),
		TokensCache: attrOptionalCount(a, AttrUsageTokensCache), TokensReasoning: attrOptionalCount(a, AttrUsageTokensReasoning),
		Provider: attrString(a, AttrUsageProvider), Model: attrString(a, AttrUsageModel),
		MeasurementSource: attrString(a, AttrUsageSource), MeasurementQuality: attrString(a, AttrUsageQuality),
		DedupUnavailable: attrBool(a, AttrUsageDedupMissing), Calls: int(attrCount(a, AttrUsageCalls)),
		ObservedAt: attrString(a, AttrUsageObservedAt),
	}, true
}

func attrString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func attrBool(m map[string]any, key string) bool {
	if b, ok := m[key].(bool); ok {
		return b
	}
	return false
}

// attrCount reads an integer attribute that may have been through JSON (float64).
func attrCount(m map[string]any, key string) int64 {
	if v := attrOptionalCount(m, key); v != nil {
		return *v
	}
	return 0
}

func attrOptionalCount(m map[string]any, key string) *int64 {
	switch v := m[key].(type) {
	case int64:
		return &v
	case int:
		n := int64(v)
		return &n
	case float64:
		if v == float64(int64(v)) {
			n := int64(v)
			return &n
		}
	}
	return nil
}
