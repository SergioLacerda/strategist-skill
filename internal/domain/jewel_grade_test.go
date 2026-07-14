package domain

import "testing"

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
