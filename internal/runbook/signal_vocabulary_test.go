package runbook

import "testing"

func TestCanonicalSignalsInRecognizesCanonicalTermsAndAliases(t *testing.T) {
	t.Parallel()

	got := CanonicalSignalsIn("CI TEST SUITE IS RED after a dependency bump")
	if !got[SignalCITestFailure] || !got[SignalDependencyUpgrade] {
		t.Fatalf("expected canonical and alias matches, got %v", got)
	}
	if len(CanonicalSignalsIn("no governed signal appears here")) != 0 {
		t.Fatal("expected no signal for unrelated text")
	}
	if got := CanonicalSignalsIn(""); got == nil || len(got) != 0 {
		t.Fatalf("expected a non-nil empty signal set, got %v", got)
	}
}

func TestSharesCanonicalSignalReportsOverlap(t *testing.T) {
	t.Parallel()

	left := CanonicalSignalsIn("flaky test")
	right := CanonicalSignalsIn("ci_test_failure")
	if !SharesCanonicalSignal(left, right) {
		t.Fatal("expected alias and canonical term to overlap")
	}
	if SharesCanonicalSignal(left, CanonicalSignalsIn("dependency bump")) {
		t.Fatal("did not expect unrelated signal sets to overlap")
	}
}
