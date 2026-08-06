package eval

import "testing"

func TestRunScenario_StateMachine_Pass(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "sm-pass",
		Input: Input{
			Target: TargetStateMachine,
			Params: map[string]any{
				"start":  "APPROVAL_GATE",
				"events": []any{"gate_approved"},
			},
		},
		Expected:   Expected{State: "EXECUTION"},
		Assertions: []Assertion{{Type: AssertRequiredEvent, Value: "gate_approved"}},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_StateMachine_Fail(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "sm-fail",
		Input: Input{
			Target: TargetStateMachine,
			Params: map[string]any{
				"start":  "APPROVAL_GATE",
				"events": []any{"gate_denied"},
			},
		},
		Expected: Expected{State: "EXECUTION"},
	})
	if res.Passed {
		t.Fatalf("expected failure (final state is DONE_ANALYSIS, not EXECUTION)")
	}
}

func TestRunScenario_RouteDecision(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "rd-blocked",
		Input: Input{
			Target: TargetRouteDecision,
			Params: map[string]any{
				"route":               "direct_execute",
				"has_context":         true,
				"touches_source_code": true,
			},
		},
		Expected: Expected{Status: "blocked", Reason: "cannot mutate source"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_SlotWriteScope(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "sws-blocked",
		Input: Input{
			Target: TargetSlotWriteScope,
			Params: map[string]any{
				"slot_name":      "execution",
				"allowed_prefix": ".analysis/archived",
				"allowed_ext":    ".md",
				"attempted_path": "internal/eval/scenario.go",
			},
		},
		Expected: Expected{Status: "blocked", Reason: "slot_write_scope_violation"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_ChestGrade_Allowed(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "chest-grade-allowed",
		Input: Input{
			Target: TargetChestGrade,
			Params: map[string]any{
				"chest_id":              "runbooks",
				"source_grade":          "A",
				"reuse_value":           "high",
				"implementation_status": "implemented",
			},
		},
		Expected: Expected{Status: "allowed"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_ChestGrade_Blocked(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "chest-grade-blocked",
		Input: Input{
			Target: TargetChestGrade,
			Params: map[string]any{
				"chest_id":     "runbooks",
				"source_grade": "Z",
			},
		},
		Expected: Expected{Status: "blocked", Reason: "must be one of A, B, C"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_JewelTrust_Allowed(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "jewel-trust-allowed",
		Input: Input{
			Target: TargetJewelTrust,
			Params: map[string]any{
				"jewel_id":         "jewel-001",
				"jewel_trust":      "T2",
				"chest_trust_tier": "T2",
			},
		},
		Expected: Expected{Status: "allowed"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_JewelTrust_Blocked(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "jewel-trust-blocked",
		Input: Input{
			Target: TargetJewelTrust,
			Params: map[string]any{
				"jewel_id":         "jewel-002",
				"jewel_trust":      "T0",
				"chest_trust_tier": "T2",
			},
		},
		Expected: Expected{Status: "blocked", Reason: "exceeds parent chest's trust tier"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_ScopeFilter(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "sf-pass",
		Input: Input{
			Target: TargetScopeFilter,
			Params: map[string]any{
				"rows": []any{
					map[string]any{"id": "a", "scope": []any{"discovery"}},
					map[string]any{"id": "b", "scope": []any{"execution"}},
					map[string]any{"id": "c", "scope": []any{"all"}},
				},
				"value": "discovery",
			},
		},
		Expected: Expected{IDs: []string{"a", "c"}},
		Assertions: []Assertion{
			{Type: AssertScopeIncludes, Value: "a"},
			{Type: AssertScopeExcludes, Value: "b"},
		},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_ArtifactCheck(t *testing.T) {
	res := RunScenario(Scenario{
		ID: "ac-pass",
		Input: Input{
			Target: TargetArtifactCheck,
			Params: map[string]any{
				"path":             "../embed/defaults/schemas/progress-contract.yaml",
				"must_parse_yaml":  true,
				"must_contain_key": "event_format",
			},
		},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got violations: %+v", res.Violations)
	}
}

func TestRunScenario_UnknownTarget(t *testing.T) {
	res := RunScenario(Scenario{ID: "unknown", Input: Input{Target: "nonsense"}})
	if res.Passed {
		t.Fatalf("expected failure for unknown target")
	}
}
