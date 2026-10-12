package refinement

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const archivistOpenSpecCorrelation = "archivist-openspec-normalization"

func recordArchivistConfidence(input OpenSpecInput) error {
	percent := 100
	evidence := []domain.Evidence{
		openSpecEvidence("archivist-analysis", filepath.ToSlash(filepath.Join("pending", input.MissionID+"-analysis.md")), &percent),
		openSpecEvidence("archivist-proposal", openSpecChangeRef(input.ChangeID, "proposal.md"), &percent),
		openSpecEvidence("archivist-design", openSpecChangeRef(input.ChangeID, "design.md"), &percent),
		openSpecEvidence("archivist-tasks", openSpecChangeRef(input.ChangeID, "tasks.md"), &percent),
	}
	claim := domain.ConfidenceClaim{
		ID:                "archivist-openspec-ready",
		Statement:         fmt.Sprintf("OpenSpec change %q contains the canonical refinement artifacts required to normalize mission %q.", input.ChangeID, input.MissionID),
		Agent:             "archivist",
		CorrelationKey:    archivistOpenSpecCorrelation,
		ClaimKind:         domain.ClaimKindAssertion,
		ConfidencePercent: percent,
		ConfidenceLevel:   domain.ConfidenceHigh,
		EvidenceIDs:       []string{"archivist-analysis", "archivist-proposal", "archivist-design", "archivist-tasks"},
		EvidenceClasses:   []string{domain.EvidenceClassExplicit, domain.EvidenceClassExplicit, domain.EvidenceClassExplicit, domain.EvidenceClassExplicit},
		CalibrationStatus: domain.CalibrationNoSample,
	}
	return input.RecordConfidence(claim, evidence)
}

func openSpecEvidence(id, sourceRef string, percent *int) domain.Evidence {
	return domain.Evidence{
		ID: id, SourceRef: sourceRef, Class: domain.EvidenceClassExplicit,
		Confidence: domain.ConfidenceHigh, ConfidencePercent: percent,
	}
}

func openSpecChangeRef(changeID, name string) string {
	return filepath.ToSlash(filepath.Join("openspec", "changes", changeID, name))
}
