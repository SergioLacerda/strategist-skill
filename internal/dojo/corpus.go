package dojo

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// FieldReader reads the projected fields of a boundary artifact. The CLI supplies
// the adapter over the handoff consumer's field catalog, so this package stays
// independent of the provider integration.
type FieldReader interface {
	Package(tasksPath, analysisPath string) (calibration.Fields, error)
	Ranger(artifactPath string) (calibration.Fields, error)
}

// CorpusEntry is one package or artifact of the corpus. The manifest stores
// references and digests, never copies of the content.
type CorpusEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	Digest     string `json:"digest"`
	Label      string `json:"label"`
	WeakLabel  bool   `json:"weak_label"`
	HeldOut    bool   `json:"held_out,omitempty"`
	Amendments int    `json:"amendments,omitempty"`
	ReviewFlag string `json:"review_flag,omitempty"`
}

// Manifest is the deterministic corpus listing of a case.
type Manifest struct {
	Schema         string        `json:"schema"`
	Boundary       string        `json:"boundary"`
	HeldOutPercent int           `json:"held_out_percent"`
	Salt           string        `json:"salt,omitempty"`
	Entries        []CorpusEntry `json:"entries"`
	Digest         string        `json:"digest"`
	Malformed      int           `json:"malformed_label_lines,omitempty"`
}

// ManifestSchema identifies the manifest format.
const ManifestSchema = "strategist-corpus/v1"

// Before is the before snapshot of an amendment: a package state a human asked
// to revise, paired with the current package of the same entry.
type Before struct {
	Entry  string
	Digest string
	Seq    int
	Fields calibration.Fields
}

// Corpus is what a case draws from.
type Corpus struct {
	Manifest Manifest
	Sources  []calibration.Source
	Befores  []Before
}

// BuildCorpus scans the case's roots read-only and builds the manifest, the
// sources and the amendment before snapshots. It is deterministic for the same
// workspace state.
func BuildCorpus(basePath string, cal domain.DojoCalibration, reader FieldReader) (Corpus, error) {
	labels, err := LoadLabels(basePath)
	if err != nil {
		return Corpus{}, err
	}
	percent := HeldOutPercent(cal)
	found, err := scanRoots(basePath, cal, reader)
	if err != nil {
		return Corpus{}, err
	}
	corpus := Corpus{Manifest: Manifest{Schema: ManifestSchema, Boundary: cal.Boundary, HeldOutPercent: percent, Salt: cal.Corpus.Salt, Malformed: labels.Malformed}}
	for _, item := range found {
		entry := classify(item.entry, labels, percent, cal.Corpus.Salt, cal.Corpus.Synthetic)
		corpus.Manifest.Entries = append(corpus.Manifest.Entries, entry)
		corpus.Sources = append(corpus.Sources, calibration.Source{Name: entry.Name, Digest: entry.Digest, Boundary: cal.Boundary, Fields: item.fields, Golden: entry.Label == string(calibration.KindGolden), HeldOut: entry.HeldOut})
		corpus.Befores = append(corpus.Befores, loadBefores(item, reader)...)
	}
	corpus.Manifest.Digest = manifestDigest(corpus.Manifest.Entries)
	return corpus, nil
}

func classify(entry CorpusEntry, labels Labels, percent int, salt string, synthetic bool) CorpusEntry {
	entry.Label, entry.WeakLabel = string(calibration.KindCandidate), true
	if _, ok := labels.Golden[entry.Digest]; ok || synthetic {
		entry.Label, entry.WeakLabel = string(calibration.KindGolden), false
	}
	if flags := labels.Candidates[entry.Name]; len(flags) > 0 {
		entry.ReviewFlag = flags[len(flags)-1].Kind
	}
	entry.HeldOut = calibration.HeldOut(entry.Digest, salt, percent)
	return entry
}

func manifestDigest(entries []CorpusEntry) string {
	var text strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&text, "%s|%s|%s|%s|%t\n", e.Name, e.Status, e.Digest, e.Label, e.HeldOut)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(text.String())))
}
