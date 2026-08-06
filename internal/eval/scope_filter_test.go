package eval

import "testing"

func TestCheckScopeIncludes(t *testing.T) {
	ids := []string{"a", "b"}
	if v := checkScopeIncludes(ids, Assertion{Type: AssertScopeIncludes, Value: "a"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkScopeIncludes(ids, Assertion{Type: AssertScopeIncludes, Value: "z"}); v == nil {
		t.Fatal("expected a violation for a missing id, got nil")
	}
}

func TestCheckScopeExcludes(t *testing.T) {
	ids := []string{"a", "b"}
	if v := checkScopeExcludes(ids, Assertion{Type: AssertScopeExcludes, Value: "z"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkScopeExcludes(ids, Assertion{Type: AssertScopeExcludes, Value: "a"}); v == nil {
		t.Fatal("expected a violation for a present id, got nil")
	}
}

func TestCompareIDSets(t *testing.T) {
	res := &ScenarioResult{}
	compareIDSets(res, []string{"a", "b"}, []string{"a", "b"})
	if len(res.Violations) != 0 {
		t.Fatalf("expected no violations for matching sets, got %+v", res.Violations)
	}

	res2 := &ScenarioResult{}
	compareIDSets(res2, []string{"a", "b"}, []string{"a"})
	if len(res2.Violations) != 1 {
		t.Fatalf("expected 1 length-mismatch violation, got %d: %+v", len(res2.Violations), res2.Violations)
	}

	res3 := &ScenarioResult{}
	compareIDSets(res3, []string{"a", "b"}, []string{"a", "c"})
	if len(res3.Violations) != 1 {
		t.Fatalf("expected 1 missing-id violation, got %d: %+v", len(res3.Violations), res3.Violations)
	}
}

func TestParseScopeFilterParams(t *testing.T) {
	rows, value := parseScopeFilterParams(map[string]any{
		"value": "discovery",
		"rows": []any{
			map[string]any{"id": "r1", "scope": []any{"discovery"}},
			"not-a-map", // tolerated, skipped
		},
	})
	if value != "discovery" {
		t.Fatalf("unexpected value: %q", value)
	}
	if len(rows) != 1 || rows[0].ID != "r1" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestParseScopeFilterParams_NoRows(t *testing.T) {
	rows, value := parseScopeFilterParams(map[string]any{"value": "all"})
	if rows != nil {
		t.Fatalf("expected nil rows when 'rows' param is absent, got %v", rows)
	}
	if value != "all" {
		t.Fatalf("unexpected value: %q", value)
	}
}

func TestRunScopeFilterScenario(t *testing.T) {
	res := &ScenarioResult{}
	s := Scenario{
		Input: Input{Params: map[string]any{
			"value": "discovery",
			"rows": []any{
				map[string]any{"id": "r1", "scope": []any{"discovery"}},
				map[string]any{"id": "r2", "scope": []any{"execution"}},
			},
		}},
		Expected:   Expected{IDs: []string{"r1"}},
		Assertions: []Assertion{{Type: AssertScopeIncludes, Value: "r1"}, {Type: AssertScopeExcludes, Value: "r2"}},
	}
	runScopeFilterScenario(s, res)
	if len(res.Violations) != 0 {
		t.Fatalf("expected no violations, got %+v", res.Violations)
	}
}
