package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RunbookDestinationRule is resolved when a Runbook opportunity is accepted.
// Empty CanonicalPath preserves the legacy docs/runbooks fallback for reads
// and future materialization; it does not trigger a second write location.
type RunbookDestinationRule struct {
	CanonicalPath string
	Fallback      string
}

// ResolveRunbookDestinationRule validates and resolves the accepted Runbook destination.
func ResolveRunbookDestinationRule(missionID, canonicalPath string) (RunbookDestinationRule, error) {
	rule := RunbookDestinationRule{Fallback: filepath.ToSlash(filepath.Join("docs", "runbooks", missionID+"-runbook.md"))}
	if err := domain.ValidateProjectRelativeDirectory(canonicalPath, "runbooks.canonical_path"); err != nil {
		return RunbookDestinationRule{}, fmt.Errorf("runbook_destination_invalid: %w", err)
	}
	if canonicalPath != "" {
		rule.CanonicalPath = filepath.ToSlash(filepath.Clean(canonicalPath))
	}
	return rule, nil
}
