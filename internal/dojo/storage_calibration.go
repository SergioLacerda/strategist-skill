package dojo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Calibration artifacts. They extend the storage contract for the judge
// calibration case kind without moving any existing path.
const (
	// ArtifactObservations is the raw per-call observation ledger of a case.
	ArtifactObservations ArtifactName = "observations"
	// ArtifactEvidence is the latest per-run evidence of a case, kept for passing runs too.
	ArtifactEvidence ArtifactName = "evidence"
	// ArtifactCalibration is a published, id-addressed calibration record.
	ArtifactCalibration ArtifactName = "calibration"
	// ArtifactCorpusLabels is the shared golden and candidate registry.
	ArtifactCorpusLabels ArtifactName = "corpus_labels"
	// ArtifactProposals is the operator proposal log pending approval.
	ArtifactProposals ArtifactName = "operator_proposals"
	// ArtifactPairObservations is the raw per-call observation ledger of a pair-layer case.
	ArtifactPairObservations ArtifactName = "pair_observations"
	// ArtifactPairs is the pair fixtures file of a pair-layer case.
	ArtifactPairs ArtifactName = "pairs"
	// ArtifactAudit is the audit log of generated negatives of a case.
	ArtifactAudit ArtifactName = "audit"
)

const ownerCaseOutput = "<case output>"

var calibrationPolicies = []ArtifactPolicy{
	{ArtifactObservations, DurabilityDurable, "raw observations so thresholds are recomputed without new calls", ownerCaseOutput, "included with the workspace backup", "preserve valid lines and report malformed lines before cleanup"},
	{ArtifactPairObservations, DurabilityDurable, "raw pair observations (scores per dimension, no text)", ownerCaseOutput, "included with the workspace backup", "preserve valid lines and report malformed lines before cleanup"},
	{ArtifactPairs, DurabilityDurable, "labeled requirement-to-task pair fixtures with their oracle", "<case dir>", "included with the workspace backup", "restore from the backup; never regenerate a hand-edited file silently"},
	{ArtifactEvidence, DurabilityRetained, "latest run evidence, including a passing run", ownerCaseOutput, "retained with the workspace when inspection history matters", "replace atomically on the next run"},
	{ArtifactAudit, DurabilityDurable, "human or agent audit of generated negatives", ownerCaseOutput, "included with the workspace backup", "preserve valid lines; an audit never relabels a case"},
	{ArtifactCalibration, DurabilityDurable, "calibration record pinned by id from operator configuration", "<base_path>/dojo/calibrations", "included with the workspace backup", "restore from the backup; a missing id makes the handoff use its default threshold"},
	{ArtifactCorpusLabels, DurabilityDurable, "golden promotions and regression candidates", "<base_path>/dojo/corpus", "included with the workspace backup", "restore from the backup; never recreate a promotion silently"},
	{ArtifactProposals, DurabilityDurable, "operator proposals pending approval", "<base_path>/dojo/mutations", "included with the workspace backup", "preserve until approved or rejected by a reviewed change"},
}

// CalibrationArtifactPolicies returns the calibration part of the ownership
// contract as a defensive copy.
func CalibrationArtifactPolicies() []ArtifactPolicy {
	return append([]ArtifactPolicy(nil), calibrationPolicies...)
}

// AllArtifactPolicies returns the scenario and the calibration contracts together.
func AllArtifactPolicies() []ArtifactPolicy {
	return append(ArtifactPolicies(), CalibrationArtifactPolicies()...)
}

// CasePaths is the stable path identity for one calibration case. Output is the
// case's own configurable output directory; the shared directories always live
// in the dojo domain.
type CasePaths struct {
	Case                 string
	CaseDir              string
	Output               string
	ObservationsPath     string
	PairObservationsPath string
	PairsPath            string
	EvidencePath         string
	AuditPath            string
	CorpusPath           string
	CalibrationsDir      string
	CorpusDir            string
	MutationsDir         string
}

// NewCasePaths resolves the paths of a case. An empty output means the default,
// <base_path>/dojo/<case>. A configured output must be relative to the
// workspace base path, must stay inside it, must not live under .strategist and
// must not be a package source directory.
func NewCasePaths(basePath, caseName, output string) (CasePaths, error) {
	paths, err := NewStoragePaths(basePath, caseName)
	if err != nil {
		return CasePaths{}, err
	}
	out := paths.ScenarioDir
	if strings.TrimSpace(output) != "" {
		if out, err = resolveCaseOutput(paths.BasePath, output); err != nil {
			return CasePaths{}, err
		}
	}
	return CasePaths{
		Case: caseName, CaseDir: paths.ScenarioDir, Output: out,
		ObservationsPath:     filepath.Join(out, "observations.jsonl"),
		PairObservationsPath: filepath.Join(out, "pair-observations.jsonl"),
		PairsPath:            filepath.Join(paths.ScenarioDir, "pairs.yaml"),
		EvidencePath:         filepath.Join(out, "evidence.json"),
		AuditPath:            filepath.Join(out, "audit.jsonl"),
		CorpusPath:           filepath.Join(out, "corpus.json"),
		CalibrationsDir:      filepath.Join(paths.DojoRoot, DirCalibrations),
		CorpusDir:            filepath.Join(paths.DojoRoot, DirCorpus),
		MutationsDir:         filepath.Join(paths.DojoRoot, DirMutations),
	}, nil
}

var protectedOutputRoots = []string{".strategist", "refined", "done", "pending", "todo", "archived"}

func resolveCaseOutput(basePath, output string) (string, error) {
	if isUnsafeRelPath(output) {
		return "", fmt.Errorf("dojo: case output %q must be a relative path inside the workspace", output)
	}
	clean := filepath.Clean(output)
	first := strings.Split(filepath.ToSlash(clean), "/")[0]
	for _, protected := range protectedOutputRoots {
		if first == protected {
			return "", fmt.Errorf("dojo: case output %q is a protected area (%s)", output, protected)
		}
	}
	return filepath.Join(basePath, clean), nil
}
