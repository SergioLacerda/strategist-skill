package domain

import "testing"

func TestValidatePotionStatus_ValidStatuses(t *testing.T) {
	t.Parallel()
	for _, status := range []string{PotionStatusProposed, PotionStatusAccepted, PotionStatusVerified, PotionStatusDeprecated} {
		if err := ValidatePotionStatus("potion-1", status); err != nil {
			t.Errorf("status %q: expected no error, got %v", status, err)
		}
	}
}

func TestValidatePotionStatus_UnknownStatus(t *testing.T) {
	t.Parallel()
	err := ValidatePotionStatus("potion-1", "bogus")
	if err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func TestValidatePotionTrust_EqualToChestTier(t *testing.T) {
	t.Parallel()
	if err := ValidatePotionTrust("potion-1", "T2", "T2"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidatePotionTrust_LowerThanChestTier(t *testing.T) {
	t.Parallel()
	if err := ValidatePotionTrust("potion-1", "T3", "T1"); err != nil {
		t.Fatalf("expected no error for potion less trusted than chest, got %v", err)
	}
}

func TestValidatePotionTrust_ExceedsChestTier(t *testing.T) {
	t.Parallel()
	err := ValidatePotionTrust("potion-1", "T0", "T2")
	if err == nil {
		t.Fatal("expected error when potion trust exceeds parent chest tier")
	}
}

func TestValidatePotionTrust_InvalidPotionTier(t *testing.T) {
	t.Parallel()
	err := ValidatePotionTrust("potion-1", "T9", "T2")
	if err == nil {
		t.Fatal("expected error for invalid potion trust value")
	}
}

func TestValidatePotionTrust_InvalidChestTier(t *testing.T) {
	t.Parallel()
	err := ValidatePotionTrust("potion-1", "T2", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid chest trust tier value")
	}
}
