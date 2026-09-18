package integrity_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/hardening"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
)

func TestVerifyRuntimeReportsMissingManifestAsUnavailable(t *testing.T) {
	result := integrity.VerifyRuntime(t.TempDir())
	if result.Status != hardening.StatusUnavailable || result.Success() {
		t.Fatalf("unexpected result: %+v", result)
	}
}
