package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func validObserveReview() ObserveReview {
	return ObserveReview{
		SchemaVersion: "1", ReviewedBy: "reviewer", ReviewedAt: "2026-09-20T12:00:00Z",
		CompatibilityEvidenceRef: "docs/evidence/x.md", AgentDenominatorsReconciled: true,
		SamplesPerAgent: map[string]int{"scout": 4},
	}
}

func TestValidateEnforcementChange(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateEnforcementChange(ConfidenceEnforcementAdvisory, nil), "advisory must always be admissible")
	require.ErrorContains(t, ValidateEnforcementChange("strict", nil), "not allowed")
	require.ErrorContains(t, ValidateEnforcementChange(ConfidenceEnforcementBlocking, nil), "stays advisory")

	review := validObserveReview()
	require.NoError(t, ValidateEnforcementChange(ConfidenceEnforcementBlocking, &review))
	review.AgentDenominatorsReconciled = false
	require.ErrorContains(t, ValidateEnforcementChange(ConfidenceEnforcementBlocking, &review), "not reconciled")
}

func TestValidateObserveReviewReportsEveryGap(t *testing.T) {
	t.Parallel()
	err := ValidateObserveReview(ObserveReview{ReviewedAt: "later"})
	for _, want := range []string{"reviewed_by", "compatibility_evidence_ref", "RFC3339", "not reconciled", "samples_per_agent"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q in %v", want, err)
		}
	}
}

func TestReadObserveReview(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "r.yaml")
	body := "schema_version: \"1\"\nreviewed_by: r\nreviewed_at: 2026-09-20T12:00:00Z\ncompatibility_evidence_ref: e\nagent_denominators_reconciled: true\nsamples_per_agent: {scout: 3}\n"
	if err := os.WriteFile(good, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	review, err := ReadObserveReview(good)
	if err != nil || ValidateObserveReview(review) != nil {
		t.Fatalf("review=%+v err=%v", review, err)
	}
	if _, err := ReadObserveReview(filepath.Join(dir, "absent.yaml")); err == nil {
		t.Fatal("missing file must fail")
	}
	bad := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(bad, []byte("reviewed_by: [unterminated"), 0o600)
	if _, err := ReadObserveReview(bad); err == nil {
		t.Fatal("malformed yaml must fail")
	}
}
