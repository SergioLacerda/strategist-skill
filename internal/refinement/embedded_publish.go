package refinement

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// EmbeddedInput identifies one embedded Archivist completion and its mission.
// BasePath and PendingAnalysisPath must be resolved absolute paths. RequestID
// is the invocation request that produced the completion; it is the package's
// provenance identifier, standing where OpenSpec records its change id.
type EmbeddedInput struct {
	MissionID           string
	BasePath            string
	PendingAnalysisPath string
	ProviderID          string
	RequestID           string
	Completion          EmbeddedCompletion
	// RecordConfidence persists the Archivist boundary claim before any file is
	// published; it is required so refinement cannot finish silently.
	RecordConfidence func(domain.ConfidenceClaim, []domain.Evidence) error
	// RecordAdvisories persists optional critic and mission-quality boundary
	// observations before the package is published.
	RecordAdvisories func(AdvisoryInput) error
	// RecordPublication persists a publication event; nil is permitted for
	// library-only callers.
	RecordPublication func(PackagePublication) error
}

// EmbeddedResult describes the canonical package published by PublishEmbedded.
type EmbeddedResult struct {
	RefinedPath string
	PublishedAt time.Time
}

// PreparedEmbedded is a completion that passed every check short of writing:
// input shape, pending analysis identity, the canonical package validators and
// the typed facts. Callers commit their journal between Prepare and Publish, so
// a malformed completion never leaves a request committed to it.
type PreparedEmbedded struct {
	input            EmbeddedInput
	refined          string
	contents         map[string][]byte
	alreadyPublished bool
}

// PrepareEmbedded validates a parsed embedded completion as the canonical
// four-file package without writing anything.
func PrepareEmbedded(input EmbeddedInput) (PreparedEmbedded, error) {
	refined, err := validateEmbeddedInput(input)
	if err != nil {
		return PreparedEmbedded{}, err
	}
	published, err := alreadyPublished(input, refined)
	if err != nil || published {
		return PreparedEmbedded{input: input, refined: refined, alreadyPublished: published}, err
	}
	contents, err := embeddedContents(input)
	if err != nil {
		return PreparedEmbedded{}, err
	}
	if _, err := handoff.ValidateRefinedPackageContent(contents, input.MissionID); err != nil {
		return PreparedEmbedded{}, fmt.Errorf("embedded refinement: validate refined package: %w", err)
	}
	return PreparedEmbedded{input: input, refined: refined, contents: contents}, nil
}

// Publish writes the prepared package through the same tail as the OpenSpec
// bridge: confidence before publication, atomic and idempotent publication,
// conflicting output failing closed, and the pending analysis promoted.
func (p PreparedEmbedded) Publish() (EmbeddedResult, error) {
	if p.alreadyPublished {
		return EmbeddedResult{RefinedPath: p.refined, PublishedAt: time.Now().UTC()}, nil
	}
	err := publishTail(publishTailInput{
		MissionID: p.input.MissionID, ProviderChangeID: p.input.RequestID, Refined: p.refined,
		Pending: p.input.PendingAnalysisPath, Contents: p.contents,
		RecordConfidence: func() error { return recordEmbeddedConfidence(p.input) },
		RecordAdvisories: func(packageDigest string, analysis []byte) error {
			if p.input.RecordAdvisories == nil {
				return nil
			}
			return p.input.RecordAdvisories(AdvisoryInput{MissionID: p.input.MissionID, PackageDigest: packageDigest, Analysis: analysis})
		},
		RecordPublication: p.input.RecordPublication,
	})
	if err != nil {
		return EmbeddedResult{}, err
	}
	return EmbeddedResult{RefinedPath: p.refined, PublishedAt: time.Now().UTC()}, nil
}

// PublishEmbedded prepares and publishes in one step, for library callers
// that have no journal to commit in between.
func PublishEmbedded(input EmbeddedInput) (EmbeddedResult, error) {
	prepared, err := PrepareEmbedded(input)
	if err != nil {
		return EmbeddedResult{}, err
	}
	return prepared.Publish()
}

func validateEmbeddedInput(input EmbeddedInput) (string, error) {
	if err := validateEmbeddedIdentifiers(input); err != nil {
		return "", err
	}
	if err := validateEmbeddedPaths(input); err != nil {
		return "", err
	}
	if input.RecordConfidence == nil {
		return "", fmt.Errorf("embedded refinement: archivist confidence recorder is required")
	}
	return filepath.Join(input.BasePath, "refined", input.MissionID), validateHandoffFacts(input.Completion.HandoffFacts)
}

func validateEmbeddedIdentifiers(input EmbeddedInput) error {
	for label, value := range map[string]string{"mission id": input.MissionID, "provider id": input.ProviderID, "request id": input.RequestID} {
		if !safeComponent.MatchString(value) {
			return fmt.Errorf("embedded refinement: %s is malformed", label)
		}
	}
	return nil
}

func validateEmbeddedPaths(input EmbeddedInput) error {
	for label, value := range map[string]string{"base path": input.BasePath, "pending analysis path": input.PendingAnalysisPath} {
		if value == "" || !filepath.IsAbs(value) {
			return fmt.Errorf("embedded refinement: %s must be an absolute path", label)
		}
	}
	if err := validateContained(input.BasePath, input.PendingAnalysisPath); err != nil {
		return fmt.Errorf("embedded refinement: pending analysis path: %w", err)
	}
	return nil
}

func embeddedContents(input EmbeddedInput) (map[string][]byte, error) {
	analysis, err := readPendingAnalysis(OpenSpecInput{MissionID: input.MissionID, PendingAnalysisPath: input.PendingAnalysisPath})
	if err != nil {
		return nil, err
	}
	withMetadata, err := withPublicationMetadata(analysis, input.MissionID, publicationMetadata{
		Provider: input.ProviderID, ChangeID: input.RequestID, Runtime: "embedded",
	}, input.Completion.HandoffFacts)
	if err != nil {
		return nil, err
	}
	contents := map[string][]byte{"analysis.md": withMetadata}
	for _, name := range embeddedDocuments {
		document, ok := input.Completion.Documents[name]
		if !ok {
			return nil, fmt.Errorf("embedded refinement: document %s is missing", name)
		}
		contents[name] = document
	}
	return contents, nil
}

func alreadyPublished(input EmbeddedInput, refined string) (bool, error) {
	if _, err := os.Stat(input.PendingAnalysisPath); !os.IsNotExist(err) {
		return false, nil
	}
	analysis, found, err := readPublishedAnalysis(refined)
	if err != nil || !found {
		return false, err
	}
	if frontmatterValue(analysis, "provider_change_id") != input.RequestID {
		return false, errEmbeddedConflict
	}
	return true, requirePublishedDocuments(refined, input.Completion.Documents)
}

var errEmbeddedConflict = fmt.Errorf("embedded refinement: existing refined package conflicts with this completion")

func readPublishedAnalysis(refined string) ([]byte, bool, error) {
	analysis, err := os.ReadFile(filepath.Join(refined, "analysis.md")) //nolint:gosec // refined is contained under the validated base path
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("embedded refinement: inspect published package: %w", err)
	}
	return analysis, true, nil
}

func requirePublishedDocuments(refined string, documents map[string][]byte) error {
	for _, name := range embeddedDocuments {
		existing, err := os.ReadFile(filepath.Join(refined, name)) //nolint:gosec // refined is contained under the validated base path
		if err != nil || !bytes.Equal(existing, documents[name]) {
			return errEmbeddedConflict
		}
	}
	return nil
}
