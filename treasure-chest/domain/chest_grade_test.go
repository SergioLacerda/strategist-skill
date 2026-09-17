package domain

import "testing"

func TestValidateChestGrade_EmptyIsValid(t *testing.T) {
	t.Parallel()
	if err := ValidateChestGrade("chest-1", ChestGrade{}); err != nil {
		t.Fatalf("expected no error for empty grade, got %v", err)
	}
}

func TestValidateChestGrade_ValidValues(t *testing.T) {
	t.Parallel()
	g := ChestGrade{SourceGrade: "A", ReuseValue: "high", ImplementationStatus: "implemented"}
	if err := ValidateChestGrade("chest-1", g); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateChestGrade_InvalidSourceGrade(t *testing.T) {
	t.Parallel()
	err := ValidateChestGrade("chest-1", ChestGrade{SourceGrade: "Z"})
	if err == nil {
		t.Fatal("expected error for invalid source_grade")
	}
}

func TestValidateChestGrade_InvalidReuseValue(t *testing.T) {
	t.Parallel()
	err := ValidateChestGrade("chest-1", ChestGrade{ReuseValue: "extreme"})
	if err == nil {
		t.Fatal("expected error for invalid reuse_value")
	}
}

func TestValidateChestGrade_InvalidImplementationStatus(t *testing.T) {
	t.Parallel()
	err := ValidateChestGrade("chest-1", ChestGrade{ImplementationStatus: "maybe"})
	if err == nil {
		t.Fatal("expected error for invalid implementation_status")
	}
}
