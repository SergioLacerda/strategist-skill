package runbook

import "testing"

func TestValidateCheck_Valid(t *testing.T) {
	t.Parallel()
	err := ValidateCheck(Check{ID: "reproduced-locally", Level: LevelMandatory})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCheck_ConditionalWithWhen(t *testing.T) {
	t.Parallel()
	err := ValidateCheck(Check{ID: "cache-cleared", Level: LevelConditional, When: "go.sum changed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCheck_WhenOnNonConditionalIsRejected(t *testing.T) {
	t.Parallel()
	err := ValidateCheck(Check{ID: "reproduced-locally", Level: LevelMandatory, When: "go.sum changed"})
	if err == nil {
		t.Fatal("expected error: when is only meaningful for level=conditional")
	}
}

func TestValidateCheck_MissingIDOrLevel(t *testing.T) {
	t.Parallel()
	cases := []Check{
		{Level: LevelMandatory},
		{ID: "reproduced-locally"},
		{ID: "reproduced-locally", Level: "urgent"},
	}
	for _, c := range cases {
		if err := ValidateCheck(c); err == nil {
			t.Errorf("expected error for %+v", c)
		}
	}
}
