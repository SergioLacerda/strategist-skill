package dojo

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Promotion is a reviewed decision that a package is golden: a positive whose
// oracle (why it is good) and reviewer are recorded. It is bound to the content
// digest, so editing the package voids it.
type Promotion struct {
	Digest     string `json:"digest"`
	Package    string `json:"package"`
	ReviewedBy string `json:"reviewed_by"`
	Oracle     string `json:"oracle"`
	At         string `json:"at"`
}

// Candidate flags a package for review: a real mission where the provider's
// answer and the later outcome disagreed. It never makes a package golden.
type Candidate struct {
	Mission    string  `json:"mission"`
	Package    string  `json:"package"`
	Transition string  `json:"transition"`
	Kind       string  `json:"kind"`
	Confidence float64 `json:"confidence"`
	Threshold  float64 `json:"threshold"`
	Outcome    string  `json:"gate_outcome"`
	At         string  `json:"at"`
}

// Labels is the shared corpus registry read from <base_path>/dojo/corpus.
type Labels struct {
	Golden     map[string]Promotion
	Candidates map[string][]Candidate
	// Demoted lists the demotions read, by digest; a demoted digest is not in Golden unless promoted again later.
	Demoted map[string]Demotion
	// Malformed counts lines that could not be read; they are reported, never dropped silently.
	Malformed int
}

const (
	goldenFile     = "golden.jsonl"
	candidatesFile = "candidates.jsonl"
)

// LoadLabels reads the registry. A missing file is an empty registry.
func LoadLabels(basePath string) (Labels, error) {
	dir := filepath.Join(basePath, "dojo", DirCorpus)
	labels := Labels{Golden: map[string]Promotion{}, Candidates: map[string][]Candidate{}, Demoted: map[string]Demotion{}}
	if err := readJSONL(filepath.Join(dir, goldenFile), labels.acceptPromotion, &labels.Malformed); err != nil {
		return Labels{}, err
	}
	if err := readJSONL(filepath.Join(dir, candidatesFile), labels.acceptCandidate, &labels.Malformed); err != nil {
		return Labels{}, err
	}
	if err := readJSONL(filepath.Join(dir, demotedFile), labels.acceptDemotion, &labels.Malformed); err != nil {
		return Labels{}, err
	}
	return labels, nil
}

func (l Labels) acceptPromotion(raw []byte) bool {
	var p Promotion
	if json.Unmarshal(raw, &p) != nil || p.Digest == "" || p.ReviewedBy == "" || p.Oracle == "" {
		return false
	}
	l.Golden[p.Digest] = p
	return true
}

func (l Labels) acceptCandidate(raw []byte) bool {
	var c Candidate
	if json.Unmarshal(raw, &c) != nil || c.Package == "" {
		return false
	}
	l.Candidates[c.Package] = append(l.Candidates[c.Package], c)
	return true
}

func readJSONL(path string, accept func([]byte) bool, bad *int) error {
	file, err := os.Open(path) //nolint:gosec // G304: path is inside the dojo storage domain
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("dojo: open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // read-only file
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !accept([]byte(line)) {
			*bad++
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("dojo: read %s: %w", path, err)
	}
	return nil
}

// AppendPromotion records a reviewed promotion. A reviewer and an oracle are
// required, and a digest already promoted is not promoted twice.
func AppendPromotion(basePath string, p Promotion) error {
	if strings.TrimSpace(p.ReviewedBy) == "" || strings.TrimSpace(p.Oracle) == "" || p.Digest == "" {
		return errors.New("dojo: a promotion needs a digest, a reviewer and an oracle")
	}
	labels, err := LoadLabels(basePath)
	if err != nil {
		return err
	}
	if _, done := labels.Golden[p.Digest]; done {
		return fmt.Errorf("dojo: digest %s is already golden", p.Digest)
	}
	return appendJSONL(filepath.Join(basePath, "dojo", DirCorpus, goldenFile), p)
}

// AppendCandidate records a regression candidate for review.
func AppendCandidate(basePath string, c Candidate) error {
	if c.Mission == "" || c.Package == "" {
		return errors.New("dojo: a candidate needs a mission and a package")
	}
	return appendJSONL(filepath.Join(basePath, "dojo", DirCorpus, candidatesFile), c)
}

func appendJSONL(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("dojo: encode record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return fmt.Errorf("dojo: create %s: %w", filepath.Dir(path), err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G302/G304: dojo storage domain
	if err != nil {
		return fmt.Errorf("dojo: open %s: %w", path, err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return errors.Join(fmt.Errorf("dojo: write %s: %w", path, err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("dojo: close %s: %w", path, err)
	}
	return nil
}
