package refinement

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const archivistEmbeddedCorrelation = "archivist-embedded-normalization"

// recordEmbeddedConfidence is the Archivist's own boundary claim for an
// embedded completion: the documents parsed, validated and are about to be
// published as the canonical package. It claims structure, not plan quality.
func recordEmbeddedConfidence(input EmbeddedInput) error {
	percent := 100
	source := func(name string) string {
		return filepath.ToSlash(filepath.Join("invocation", input.RequestID, name))
	}
	evidence := []domain.Evidence{
		openSpecEvidence("archivist-analysis", filepath.ToSlash(filepath.Join("pending", input.MissionID+"-analysis.md")), &percent),
		openSpecEvidence("archivist-proposal", source("proposal.md"), &percent),
		openSpecEvidence("archivist-design", source("design.md"), &percent),
		openSpecEvidence("archivist-tasks", source("tasks.md"), &percent),
	}
	claim := domain.ConfidenceClaim{
		ID:                "archivist-embedded-ready",
		Statement:         fmt.Sprintf("Embedded completion %q of Weapon %q contains the canonical refinement documents required to normalize mission %q.", input.RequestID, input.ProviderID, input.MissionID),
		Agent:             "archivist",
		CorrelationKey:    archivistEmbeddedCorrelation,
		ClaimKind:         domain.ClaimKindAssertion,
		ConfidencePercent: percent,
		ConfidenceLevel:   domain.ConfidenceHigh,
		EvidenceIDs:       []string{"archivist-analysis", "archivist-proposal", "archivist-design", "archivist-tasks"},
		EvidenceClasses:   []string{domain.EvidenceClassExplicit, domain.EvidenceClassExplicit, domain.EvidenceClassExplicit, domain.EvidenceClassExplicit},
		CalibrationStatus: domain.CalibrationNoSample,
	}
	return input.RecordConfidence(claim, evidence)
}
