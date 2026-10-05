package calibration

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Schema identifies the calibration record format.
const Schema = "strategist-calibration/v1"

// DefaultMaxAge is how old a record may be and still be used by the handoff.
const DefaultMaxAge = 90 * 24 * time.Hour

// Meta is the provenance a record is bound to.
type Meta struct {
	Provider       string
	Model          string
	Profile        string
	ProfileVersion string
	CorpusDigest   string
	CatalogVersion string
	Seed           int64
	Repetitions    int
	Observations   string
	// Operators is the operator list the case restricted itself to, empty for the whole catalog.
	Operators []string
	Now       time.Time
}

// Record is the strategist-calibration/v1 artifact: evidence about a judge, never
// configuration. The handoff reads it only through a pinned id.
type Record struct {
	Schema             string           `json:"schema"`
	ID                 string           `json:"id"`
	CreatedAt          string           `json:"created_at"`
	Provider           string           `json:"provider"`
	Model              string           `json:"model"`
	Profile            string           `json:"profile"`
	ProfileVersion     string           `json:"profile_version"`
	CorpusDigest       string           `json:"corpus_digest"`
	CatalogVersion     string           `json:"catalog_version"`
	OperatorFilter     []string         `json:"operator_filter,omitempty"`
	LocalScope         []string         `json:"local_scope,omitempty"`
	PolicyLevel        *PolicyLevel     `json:"policy_level,omitempty"`
	Seed               int64            `json:"seed"`
	Repetitions        int              `json:"repetitions"`
	ObservationsDigest string           `json:"observations_digest"`
	Status             Status           `json:"status"`
	FailingGate        string           `json:"failing_gate,omitempty"`
	Threshold          *float64         `json:"threshold,omitempty"`
	Notes              []string         `json:"notes,omitempty"`
	Policy             Policy           `json:"risk_policy"`
	Limits             Limits           `json:"limits"`
	Counts             Counts           `json:"counts"`
	Gates              []Gate           `json:"gates"`
	AUC                float64          `json:"auc"`
	ECE                float64          `json:"ece"`
	Brier              float64          `json:"brier"`
	Selection          *SetResult       `json:"selection,omitempty"`
	HeldOut            *SetResult       `json:"held_out,omitempty"`
	CrossValidation    *CrossValidation `json:"cross_validation,omitempty"`
	AtCurrent          SetResult        `json:"at_current"`
	Operators          []OperatorResult `json:"operators,omitempty"`
	Integrity          string           `json:"integrity,omitempty"`
}

// NewRecord seals a report with its provenance.
func NewRecord(report Report, meta Meta) (Record, error) {
	if len(report.Exploratory) > 0 {
		return Record{}, fmt.Errorf("a calibration is never published from an exploratory evaluation (excluded operators %v)", report.Exploratory)
	}
	r := Record{
		Schema: Schema, CreatedAt: meta.Now.UTC().Format(time.RFC3339), Provider: meta.Provider, Model: meta.Model,
		Profile: meta.Profile, ProfileVersion: meta.ProfileVersion, CorpusDigest: meta.CorpusDigest, CatalogVersion: meta.CatalogVersion, OperatorFilter: meta.Operators, LocalScope: report.LocalScope, PolicyLevel: report.PolicyLevel,
		Seed: meta.Seed, Repetitions: meta.Repetitions, ObservationsDigest: meta.Observations, Status: report.Status,
		FailingGate: report.FailingGate, Threshold: report.Threshold, Notes: report.Notes, Policy: report.Policy, Limits: report.Limits,
		Counts: report.Counts, Gates: report.Gates, AUC: report.AUC, ECE: report.ECE, Brier: report.Brier, Selection: report.Selection,
		HeldOut: report.HeldOut, CrossValidation: report.CrossValidation, AtCurrent: report.AtCurrent, Operators: report.Operators,
	}
	digest, err := r.digest()
	if err != nil {
		return Record{}, err
	}
	r.Integrity = digest
	r.ID = fmt.Sprintf("cal-%s-%s", meta.Now.UTC().Format("20060102"), digest[len("sha256:"):][:8])
	return r, r.Validate()
}

func (r Record) digest() (string, error) {
	r.Integrity, r.ID = "", ""
	raw, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("encode calibration record: %w", err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}

var idPattern = regexp.MustCompile(`^cal-\d{8}-[0-9a-f]{8}$`)

// ValidID reports whether id has the shape of a calibration id. "latest" never does.
func ValidID(id string) bool { return idPattern.MatchString(id) }

// Validate checks the record's shape and integrity.
func (r Record) Validate() error {
	return errors.Join(r.checkIdentity(), r.checkThreshold(), r.checkIntegrity())
}

func (r Record) checkIdentity() error {
	var problems []error
	if r.Schema != Schema {
		problems = append(problems, fmt.Errorf("schema %q is not %s", r.Schema, Schema))
	}
	if !ValidID(r.ID) {
		problems = append(problems, fmt.Errorf("id %q is not a calibration id", r.ID))
	}
	if r.Provider == "" || r.Model == "" || r.Profile == "" || r.ProfileVersion == "" {
		problems = append(problems, errors.New("provider, model, profile and profile_version are required"))
	}
	return errors.Join(problems...)
}

func (r Record) checkThreshold() error {
	recommended := r.Status == StatusRecommended
	switch {
	case recommended && r.Threshold == nil:
		return errors.New("a recommended record needs a threshold")
	case !recommended && r.Threshold != nil:
		return errors.New("only a recommended record carries a threshold")
	}
	return nil
}

func (r Record) checkIntegrity() error {
	if digest, err := r.digest(); err != nil || digest != r.Integrity {
		return errors.New("integrity digest does not match")
	}
	return nil
}

// Reason is why a record cannot serve a call; empty means it can.
type Reason string

// Reasons a calibration is not used.
const (
	ReasonValid          Reason = ""
	ReasonNotRecommended Reason = "not_recommended"
	ReasonInvalid        Reason = "invalid_record"
	ReasonModelChanged   Reason = "model_changed"
	ReasonProfileChanged Reason = "profile_changed"
	ReasonExpired        Reason = "expired"
)

// Binding is what a call is made with.
type Binding struct {
	Model          string
	Profile        string
	ProfileVersion string
}

// UsableFor reports why the record cannot serve a call with this binding, or
// ReasonValid. A record that is not recommended, tampered with, for another model
// or question profile, or older than maxAge is never used.
func (r Record) UsableFor(binding Binding, now time.Time, maxAge time.Duration) Reason {
	switch {
	case r.Validate() != nil:
		return ReasonInvalid
	case r.Status != StatusRecommended:
		return ReasonNotRecommended
	case r.Model != binding.Model:
		return ReasonModelChanged
	case r.Profile != binding.Profile || r.ProfileVersion != binding.ProfileVersion:
		return ReasonProfileChanged
	}
	created, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err != nil || now.Sub(created) > maxAge {
		return ReasonExpired
	}
	return ReasonValid
}
