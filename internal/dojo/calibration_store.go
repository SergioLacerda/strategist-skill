package dojo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/filesystem"
)

// ReadObservations reads a case's raw observation ledger. A missing ledger is no
// observations; malformed lines are counted, never dropped silently.
func ReadObservations(path string) ([]calibration.Observation, int, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is inside the dojo storage domain
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("dojo: read %s: %w", path, err)
	}
	observations, bad := calibration.DecodeObservations(raw)
	return observations, bad, nil
}

// AppendObservation appends one observation to the ledger.
func AppendObservation(path string, obs calibration.Observation) error {
	return appendJSONL(path, obs)
}

// Evidence is the per-run evidence of a case: written for a passing run as well as
// a failing one, and carrying no package content.
type Evidence struct {
	Schema     string `json:"schema"`
	Case       string `json:"case"`
	At         string `json:"at"`
	Layer      string `json:"layer"`
	Model      string `json:"model,omitempty"`
	Profile    string `json:"profile,omitempty"`
	ProfileVer string `json:"profile_version,omitempty"`
	Corpus     string `json:"corpus_digest"`
	Catalog    string `json:"catalog_version"`
	Calls      int    `json:"calls"`
	Skipped    int    `json:"skipped_calls"`
	// InputTokens and OutputTokens are the usage the provider reported (measured), not an estimate.
	InputTokens  int                `json:"input_tokens"`
	OutputTokens int                `json:"output_tokens"`
	Passed       bool               `json:"passed"`
	Report       calibration.Report `json:"report"`
	RecordID     string             `json:"calibration_id,omitempty"`
	// Consent is the term the operator assented to, for a run that called out.
	Consent *ConsentRecord `json:"consent,omitempty"`
}

// EvidenceSchema identifies the evidence format.
const EvidenceSchema = "strategist-dojo-evidence/v1"

// WriteEvidence replaces the case's latest evidence atomically.
func WriteEvidence(paths CasePaths, evidence Evidence) error {
	return writeJSONFile(paths.EvidencePath, evidence)
}

// WriteManifest replaces the case's corpus manifest atomically.
func WriteManifest(paths CasePaths, manifest Manifest) error {
	return writeJSONFile(paths.CorpusPath, manifest)
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return fmt.Errorf("dojo: create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("dojo: encode %s: %w", path, err)
	}
	if err := atomicWriteFile(path, data, 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo: write %s: %w", path, err)
	}
	return nil
}

// SaveCalibration publishes a record under <base_path>/dojo/calibrations/<id>.json.
// A record is never overwritten: ids are content-derived, and an existing id
// with different content is refused.
func SaveCalibration(basePath string, record calibration.Record) error {
	if err := record.Validate(); err != nil {
		return fmt.Errorf("dojo: calibration record: %w", err)
	}
	path := filepath.Join(basePath, "dojo", DirCalibrations, record.ID+".json")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("dojo: calibration %s already exists", record.ID)
	}
	return writeJSONFile(path, record)
}

// LoadCalibration reads and validates a published record by id. An id that is not
// a calibration id (including "latest") is refused before any path is built.
func LoadCalibration(basePath, id string) (calibration.Record, error) {
	if !calibration.ValidID(id) {
		return calibration.Record{}, fmt.Errorf("dojo: %q is not a calibration id", id)
	}
	raw, err := os.ReadFile(filepath.Join(basePath, "dojo", DirCalibrations, id+".json")) //nolint:gosec // G304: id is validated to a fixed shape
	if err != nil {
		return calibration.Record{}, fmt.Errorf("dojo: read calibration %s: %w", id, err)
	}
	var record calibration.Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return calibration.Record{}, fmt.Errorf("dojo: parse calibration %s: %w", id, err)
	}
	if record.ID != id {
		return calibration.Record{}, fmt.Errorf("dojo: calibration file %s holds id %s", id, record.ID)
	}
	if err := record.Validate(); err != nil {
		return calibration.Record{}, fmt.Errorf("dojo: calibration %s: %w", id, err)
	}
	return record, nil
}

// ListCalibrations returns the ids of the published records, newest first by name.
func ListCalibrations(basePath string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(basePath, "dojo", DirCalibrations))
	if filesystem.IsMissingPath(filepath.Join(basePath, "dojo", DirCalibrations), err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dojo: list calibrations: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if id := strings.TrimSuffix(e.Name(), ".json"); !e.IsDir() && calibration.ValidID(id) {
			ids = append(ids, id)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	return ids, nil
}

// ReadManifestDigest is the corpus digest of the case's last manifest, or "".
func ReadManifestDigest(paths CasePaths) string {
	raw, err := os.ReadFile(paths.CorpusPath) //nolint:gosec // G304: path is inside the dojo storage domain
	if err != nil {
		return ""
	}
	var manifest Manifest
	if json.Unmarshal(raw, &manifest) != nil {
		return ""
	}
	return manifest.Digest
}

// NewEvidence builds the evidence of one run from its outcome. It carries
// numbers and identifiers only.
func NewEvidence(name, layer string, out CheckOutcome, corpusDigest string, summary calibration.Summary, now time.Time) Evidence {
	evidence := Evidence{
		Schema: EvidenceSchema, Case: name, At: now.UTC().Format(time.RFC3339), Layer: layer, Model: out.Model, Profile: out.Profile,
		ProfileVer: out.Version, Corpus: corpusDigest, Catalog: calibration.CatalogVersion(), Calls: summary.Calls, Skipped: summary.Skipped, InputTokens: summary.InputTokens, OutputTokens: summary.OutputTokens,
		Passed: out.Result.Passed(), Report: out.Report,
	}
	if out.Result.Calibration != nil {
		evidence.RecordID = out.Result.Calibration.RecordID
	}
	return evidence
}

// WriteJSON replaces a JSON document atomically; it is how a pair case writes its evidence.
func WriteJSON(path string, value any) error { return writeJSONFile(path, value) }
