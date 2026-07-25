package domain

import (
	"strings"
	"testing"
)

func TestValidateJewelTrust_EqualToChestTier(t *testing.T) {
	t.Parallel()
	if err := ValidateJewelTrust("jewel-1", "T2", "T2"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateJewelTrust_LowerThanChestTier(t *testing.T) {
	t.Parallel()
	if err := ValidateJewelTrust("jewel-1", "T3", "T1"); err != nil {
		t.Fatalf("expected no error for jewel less trusted than chest, got %v", err)
	}
}

func TestValidateJewelTrust_ExceedsChestTier(t *testing.T) {
	t.Parallel()
	err := ValidateJewelTrust("jewel-1", "T0", "T2")
	if err == nil {
		t.Fatal("expected error when jewel trust exceeds parent chest tier")
	}
}

func TestValidateJewelTrust_InvalidJewelTier(t *testing.T) {
	t.Parallel()
	err := ValidateJewelTrust("jewel-1", "T9", "T2")
	if err == nil {
		t.Fatal("expected error for invalid jewel trust value")
	}
}

func TestValidateJewelTrust_InvalidChestTier(t *testing.T) {
	t.Parallel()
	err := ValidateJewelTrust("jewel-1", "T2", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid chest trust tier value")
	}
}

func TestValidateJewelStatus_ValidStatuses(t *testing.T) {
	t.Parallel()
	for _, status := range []string{JewelStatusProposed, JewelStatusAccepted, JewelStatusVerified, JewelStatusDeprecated} {
		if err := ValidateJewelStatus("jewel-1", status); err != nil {
			t.Errorf("status %q: expected no error, got %v", status, err)
		}
	}
}

func TestValidateJewelStatus_LegacyActiveRejected(t *testing.T) {
	t.Parallel()
	err := ValidateJewelStatus("jewel-1", "active")
	if err == nil {
		t.Fatal("expected error for legacy active status")
	}
	if got := err.Error(); !strings.Contains(got, "migrate-status") {
		t.Errorf("expected migration hint in error, got %q", got)
	}
}

func TestValidateJewelStatus_UnknownStatus(t *testing.T) {
	t.Parallel()
	err := ValidateJewelStatus("jewel-1", "bogus")
	if err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func TestValidateJewelKind_ValidKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{
		"decision", "pattern", "anti_pattern", "gap", "risk",
		"constraint", "example", "heuristic", "template", "question",
	} {
		if err := ValidateJewelKind("jewel-1", kind); err != nil {
			t.Errorf("kind %q: expected no error, got %v", kind, err)
		}
	}
}

func TestValidateJewelKind_InvalidKind(t *testing.T) {
	t.Parallel()
	err := ValidateJewelKind("jewel-1", "bogus")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if got := err.Error(); !strings.Contains(got, "jewel-1") || !strings.Contains(got, "bogus") {
		t.Errorf("expected error to mention jewel id and kind, got %q", got)
	}
}

func TestValidateJewelScore_ValidRange(t *testing.T) {
	t.Parallel()
	for _, score := range []int{0, 50, 100} {
		if err := ValidateJewelScore("jewel-1", score); err != nil {
			t.Errorf("score %d: expected no error, got %v", score, err)
		}
	}
}

func TestValidateJewelScore_BelowRange(t *testing.T) {
	t.Parallel()
	if err := ValidateJewelScore("jewel-1", -1); err == nil {
		t.Fatal("expected error for negative score")
	}
}

func TestValidateJewelScore_AboveRange(t *testing.T) {
	t.Parallel()
	if err := ValidateJewelScore("jewel-1", 101); err == nil {
		t.Fatal("expected error for score above 100")
	}
}
