package runbook

import "testing"

func TestValidateDecisionGate_Valid(t *testing.T) {
	t.Parallel()
	err := ValidateDecisionGate(DecisionGate{ID: "root-cause-confirmed", Statement: "The failing dependency is identified."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDecisionGate_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []DecisionGate{
		{Statement: "missing id"},
		{ID: "missing-statement"},
		{},
	}
	for _, g := range cases {
		if err := ValidateDecisionGate(g); err == nil {
			t.Errorf("expected error for %+v", g)
		}
	}
}
