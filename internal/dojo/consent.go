package dojo

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// Data classes a consent term names.
const (
	DataSynthetic = "synthetic fixtures (no real package content)"
	DataReal      = "projected fields of real packages"
	DataProbe     = "synthetic probe and handoff text"
	DataPairs     = "requirement and task pairs authored in the case (no package content)"
)

// ConsentTerm is the term of assent for a run that calls an external provider:
// exactly what will leave the machine, to where, how much and of what kind. The
// dry run prints it and a run records it, so what was assented to is on record.
// Its id is a digest of its content: the same plan gives the same id, and any
// change of corpus, bytes, calls or endpoint gives another one.
type ConsentTerm struct {
	ID              string `json:"id"`
	Case            string `json:"case"`
	Layer           string `json:"layer"`
	Provider        string `json:"provider"`
	Endpoint        string `json:"endpoint"`
	Profile         string `json:"profile,omitempty"`
	DataClass       string `json:"data_class"`
	Fields          string `json:"fields,omitempty"`
	Corpus          string `json:"corpus_digest,omitempty"`
	Cases           int    `json:"cases"`
	Calls           int    `json:"calls"`
	Bytes           int    `json:"bytes"`
	EstimatedTokens int    `json:"estimated_input_tokens"`
}

// NewTerm seals the term of a plan.
func NewTerm(t ConsentTerm) ConsentTerm {
	t.ID = ""
	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%d|%d|%d|%d", t.Case, t.Layer, t.Provider, t.Endpoint, t.Profile, t.DataClass, t.Fields, t.Corpus, t.Cases, t.Calls, t.Bytes, t.EstimatedTokens)
	t.ID = fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))[:12]
	return t
}

// Text is the term as the operator reads it.
func (t ConsentTerm) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "consent term %s\n", t.ID)
	fmt.Fprintf(&b, "  a run of %q (layer %s) calls %s at %s\n", t.Case, t.Layer, t.Provider, t.Endpoint)
	if t.Profile != "" {
		fmt.Fprintf(&b, "  question profile: %s\n", t.Profile)
	}
	fmt.Fprintf(&b, "  data sent: %s", t.DataClass)
	if t.Fields != "" {
		fmt.Fprintf(&b, " (fields: %s)", t.Fields)
	}
	fmt.Fprintf(&b, "\n  calls: %d over %d cases; bytes: %d; estimated input tokens: about %d (an estimate)\n", t.Calls, t.Cases, t.Bytes, t.EstimatedTokens)
	b.WriteString("  to assent, run again with --consent (add --consent-term " + t.ID + " to bind the assent to exactly this plan); nothing was sent now\n")
	return b.String()
}

// ConsentRecord is the term as it was assented to, kept in the run's evidence.
type ConsentRecord struct {
	Term ConsentTerm `json:"term"`
	At   string      `json:"at"`
	// Bound is true when the operator passed the term id, so the assent covers exactly this plan.
	Bound bool `json:"bound"`
}

// Assent checks an optional bound term id against the plan and returns the record.
func Assent(term ConsentTerm, boundID string, now time.Time) (ConsentRecord, error) {
	if boundID != "" && boundID != term.ID {
		return ConsentRecord{}, fmt.Errorf("dojo: the plan changed since the term %s was read (it is now %s); read the new term with --dry-run", boundID, term.ID)
	}
	return ConsentRecord{Term: term, At: now.UTC().Format(time.RFC3339), Bound: boundID != ""}, nil
}
