package telemetry

import (
	"bytes"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// ArchivistAdvisoryInput is the immutable package context used to persist the
// bounded advisory observations made at the Archivist-to-Gate boundary.
type ArchivistAdvisoryInput struct {
	MissionID     string
	PackageDigest string
	Analysis      []byte
}

type missionQualityDocument struct {
	Decisions                 []domain.Decision `yaml:"decisions"`
	Evidence                  []domain.Evidence `yaml:"evidence"`
	AcceptanceCriteria        []string          `yaml:"acceptance_criteria"`
	PreviouslyOpenDecisionIDs []string          `yaml:"previously_open_decision_ids"`
	ApprovedScopePrefixes     []string          `yaml:"approved_scope_prefixes"`
}

// RecordArchivistAdvisories records the two optional advisory boundaries. A
// missing rubric or missing decision/evidence model is explicit telemetry, not
// a pass. The function is idempotent through ConfidenceProducerAdapter's
// event-id based append semantics.
func RecordArchivistAdvisories(path string, input ArchivistAdvisoryInput) error {
	if err := validateArchivistAdvisoryInput(input); err != nil {
		return err
	}
	doc, present, err := parseMissionQualityDocument(input.Analysis)
	if err != nil {
		return err
	}
	if err := recordCriticAdvisory(path, input); err != nil {
		return err
	}
	return recordMissionQualityAdvisory(path, input, doc, present)
}

func validateArchivistAdvisoryInput(input ArchivistAdvisoryInput) error {
	if input.MissionID == "" {
		return fmt.Errorf("archivist advisories: mission_id is required")
	}
	if input.PackageDigest == "" {
		return fmt.Errorf("archivist advisories: package digest is required")
	}
	return nil
}

func recordCriticAdvisory(path string, input ArchivistAdvisoryInput) error {
	critic, err := NewConfidenceProducerAdapter(path, ConfidenceAgentCritic, input.MissionID)
	if err != nil {
		return fmt.Errorf("create critic producer: %w", err)
	}
	if err := critic.RecordMissing("critic|"+input.PackageDigest, "no_rubric"); err != nil {
		return fmt.Errorf("persist critic advisory: %w", err)
	}
	return nil
}

func recordMissionQualityAdvisory(path string, input ArchivistAdvisoryInput, doc missionQualityDocument, present bool) error {
	quality, err := NewConfidenceProducerAdapter(path, ConfidenceAgentMissionQuality, input.MissionID)
	if err != nil {
		return fmt.Errorf("create mission-quality producer: %w", err)
	}
	key := "mission_quality|" + input.PackageDigest
	if !present {
		return recordMissingMissionQuality(quality, key)
	}
	return recordMissionQualityClaim(quality, input.PackageDigest, key, doc)
}

func recordMissingMissionQuality(quality ConfidenceProducerAdapter, key string) error {
	if err := quality.RecordMissing(key, "no_decisions_or_evidence"); err != nil {
		return fmt.Errorf("persist mission-quality advisory: %w", err)
	}
	return nil
}

func recordMissionQualityClaim(quality ConfidenceProducerAdapter, digest, key string, doc missionQualityDocument) error {
	result := domain.EvaluateMissionQuality(domain.MissionQualityInput{
		Decisions: doc.Decisions, Evidence: doc.Evidence, AcceptanceCriteria: doc.AcceptanceCriteria,
		PreviouslyOpenDecisionIDs: doc.PreviouslyOpenDecisionIDs, ApprovedScopePrefixes: doc.ApprovedScopePrefixes,
	})
	passed := result.Passed()
	claim := domain.ConfidenceClaim{
		ID:        "mission-quality|" + digest,
		Statement: fmt.Sprintf("mission_quality evaluation for package %s", digest),
		Result:    resultLabel(passed), Agent: ConfidenceAgentMissionQuality, CorrelationKey: key,
		ClaimKind: domain.ClaimKindQuestion, ConfidencePercent: 0, CalibrationStatus: domain.CalibrationNoSample,
	}
	if passed && len(doc.Evidence) > 0 {
		claim.ClaimKind = domain.ClaimKindAssertion
		claim.ConfidencePercent = 100
		for _, evidence := range doc.Evidence {
			claim.EvidenceIDs = append(claim.EvidenceIDs, evidence.ID)
			claim.EvidenceClasses = append(claim.EvidenceClasses, evidence.Class)
		}
	}
	if _, err := quality.RecordClaim(claim, doc.Evidence); err != nil {
		return fmt.Errorf("persist mission-quality advisory: %w", err)
	}
	return nil
}

func resultLabel(passed bool) string {
	if passed {
		return "mission_quality=passed"
	}
	return "mission_quality=failed"
}

func parseMissionQualityDocument(analysis []byte) (missionQualityDocument, bool, error) {
	header, err := analysisFrontmatter(analysis)
	if err != nil {
		return missionQualityDocument{}, false, err
	}
	var nodes map[string]yaml.Node
	if err := yaml.Unmarshal(header, &nodes); err != nil {
		return missionQualityDocument{}, false, fmt.Errorf("archivist advisories: parse analysis frontmatter: %w", err)
	}
	_, decisions := nodes["decisions"]
	_, evidence := nodes["evidence"]
	if !decisions && !evidence {
		return missionQualityDocument{}, false, nil
	}
	return decodeMissionQualityDocument(header)
}

func analysisFrontmatter(analysis []byte) ([]byte, error) {
	text := bytes.TrimSpace(bytes.ReplaceAll(analysis, []byte("\r\n"), []byte("\n")))
	if len(text) == 0 {
		return nil, fmt.Errorf("archivist advisories: analysis is empty")
	}
	if !bytes.HasPrefix(text, []byte("---\n")) {
		return nil, fmt.Errorf("archivist advisories: analysis frontmatter is missing")
	}
	rest := text[len("---\n"):]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return nil, fmt.Errorf("archivist advisories: analysis frontmatter is unclosed")
	}
	return rest[:end], nil
}

func decodeMissionQualityDocument(header []byte) (missionQualityDocument, bool, error) {
	var doc missionQualityDocument
	if err := yaml.Unmarshal(header, &doc); err != nil {
		return missionQualityDocument{}, false, fmt.Errorf("archivist advisories: decode mission-quality input: %w", err)
	}
	for _, decision := range doc.Decisions {
		if err := domain.ValidateDecision(decision); err != nil {
			return missionQualityDocument{}, false, fmt.Errorf("archivist advisories: invalid decision %q: %w", decision.ID, err)
		}
	}
	for _, evidence := range doc.Evidence {
		if err := domain.ValidateEvidence(evidence); err != nil {
			return missionQualityDocument{}, false, fmt.Errorf("archivist advisories: invalid evidence %q: %w", evidence.ID, err)
		}
	}
	return doc, true, nil
}
