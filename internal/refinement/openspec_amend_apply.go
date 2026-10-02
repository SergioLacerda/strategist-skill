package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var amendedFiles = canonicalFiles[1:]

type amendmentFile struct {
	PreviousSHA256 string `yaml:"previous_sha256"`
	NewSHA256      string `yaml:"new_sha256"`
}

// amendmentManifest is written beside the snapshot of one amendment.
type amendmentManifest struct {
	MissionID         string                   `yaml:"mission_id"`
	Amendment         int                      `yaml:"amendment"`
	ChangeID          string                   `yaml:"change_id"`
	Amends            string                   `yaml:"amends"`
	DerivedFrom       string                   `yaml:"derived_from"`
	SupersedesMission string                   `yaml:"supersedes_mission_id"`
	SourceDigest      string                   `yaml:"source_digest"`
	PackageDigest     string                   `yaml:"package_digest"`
	Reason            string                   `yaml:"reason"`
	Disposition       string                   `yaml:"disposition"`
	At                string                   `yaml:"at"`
	AuthorizationRef  string                   `yaml:"authorization_ref"`
	GateLabel         string                   `yaml:"gate_label"`
	Status            string                   `yaml:"status"`
	AnalysisSHA256    string                   `yaml:"analysis_sha256"`
	Files             map[string]amendmentFile `yaml:"files"`
}

// apply stamps the new files, snapshots the previous ones with a manifest, replaces
// the three files one by one and verifies analysis.md is byte-identical. On any
// failure after the snapshot the previous files are restored and the snapshot removed.
func (p *amendmentPlan) apply() error {
	stamped := make(map[string][]byte, len(amendedFiles))
	for _, name := range amendedFiles {
		stamped[name] = withAmendments(p.next[name], p.previous[name], p.number, p.input.ChangeID, p.input.AuthorizationRef, p.at, p)
	}
	if err := p.snapshot(stamped); err != nil {
		return err
	}
	if err := p.replaceAll(stamped); err != nil {
		return p.rollback(err)
	}
	if err := p.verifyAnalysis(); err != nil {
		return p.rollback(err)
	}
	return nil
}

func (p *amendmentPlan) snapshot(stamped map[string][]byte) error {
	if err := os.MkdirAll(p.snapshotDir, 0o755); err != nil {
		return fmt.Errorf("openspec amend: create snapshot: %w", err)
	}
	manifest := p.manifest(stamped)
	for _, name := range amendedFiles {
		if err := os.WriteFile(filepath.Join(p.snapshotDir, name), p.previous[name], 0o644); err != nil {
			return fmt.Errorf("openspec amend: snapshot %s: %w", name, err)
		}
	}
	raw, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("openspec amend: encode manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(p.snapshotDir, "manifest.yaml"), raw, 0o644); err != nil {
		return fmt.Errorf("openspec amend: write manifest: %w", err)
	}
	return nil
}

func (p *amendmentPlan) manifest(stamped map[string][]byte) amendmentManifest {
	label := p.input.GateLabel
	if label == "" {
		label = "none"
	}
	files := make(map[string]amendmentFile, len(amendedFiles))
	for _, name := range amendedFiles {
		files[name] = amendmentFile{PreviousSHA256: digest(p.previous[name]), NewSHA256: digest(stamped[name])}
	}
	return amendmentManifest{
		MissionID: p.input.MissionID, Amendment: p.number, ChangeID: p.input.ChangeID, Amends: p.input.Amends,
		DerivedFrom: p.input.Amends, SupersedesMission: p.input.SupersedesMissionID, SourceDigest: p.analysisSHA,
		PackageDigest: p.packageSHA, Reason: p.reason, Disposition: p.disposition,
		At: p.at.Format(time.RFC3339), AuthorizationRef: p.input.AuthorizationRef, GateLabel: label,
		Status: p.status, AnalysisSHA256: p.analysisSHA, Files: files,
	}
}

func (p *amendmentPlan) replaceAll(stamped map[string][]byte) error {
	for _, name := range amendedFiles {
		if err := p.replaceOne(name, stamped[name]); err != nil {
			return err
		}
	}
	return nil
}

func (p *amendmentPlan) replaceOne(name string, content []byte) error {
	if p.input.replaceHook != nil {
		if err := p.input.replaceHook(name); err != nil {
			return fmt.Errorf("openspec amend: replace %s: %w", name, err)
		}
	}
	if err := replaceFile(filepath.Join(p.refined, name), content); err != nil {
		return fmt.Errorf("openspec amend: replace %s: %w", name, err)
	}
	return nil
}

// replaceFile writes content beside path and renames it over path.
func replaceFile(path string, content []byte) error {
	tmp := path + ".amend-tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}

func (p *amendmentPlan) verifyAnalysis() error {
	raw, err := os.ReadFile(filepath.Join(p.refined, "analysis.md")) //nolint:gosec // validated canonical package
	if err != nil || digest(raw) != p.analysisSHA {
		return fmt.Errorf("openspec amend: analysis.md changed during the amendment")
	}
	return nil
}

// rollback restores the previous three files from memory, removes the snapshot and
// returns cause (joined with any restore failure).
func (p *amendmentPlan) rollback(cause error) error {
	for _, name := range amendedFiles {
		if err := replaceFile(filepath.Join(p.refined, name), p.previous[name]); err != nil {
			return fmt.Errorf("%w (restore of %s also failed: %v; the snapshot is kept at %s)", cause, name, err, p.snapshotDir)
		}
	}
	if err := os.RemoveAll(p.snapshotDir); err != nil {
		return fmt.Errorf("%w (snapshot cleanup failed: %v)", cause, err)
	}
	return cause
}

// lastAmendment returns the change id of the highest recorded amendment ("" when
// there is none) and its number.
func lastAmendment(refined string) (string, int, error) {
	numbers, err := amendmentNumbers(refined)
	if err != nil || len(numbers) == 0 {
		return "", 0, err
	}
	highest := numbers[len(numbers)-1]
	manifest, err := readManifest(refined, highest)
	return manifest.ChangeID, highest, err
}

// amendmentNumbers lists the recorded amendment numbers, ascending.
func amendmentNumbers(refined string) ([]int, error) {
	entries, err := os.ReadDir(filepath.Join(refined, ".amendments"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("openspec amend: read amendments: %w", err)
	}
	var numbers []int
	for _, entry := range entries {
		if n, convErr := strconv.Atoi(strings.TrimLeft(entry.Name(), "0")); convErr == nil && entry.IsDir() {
			numbers = append(numbers, n)
		}
	}
	sort.Ints(numbers)
	return numbers, nil
}

func readManifest(refined string, number int) (amendmentManifest, error) {
	raw, err := os.ReadFile(filepath.Join(refined, ".amendments", fmt.Sprintf("%03d", number), "manifest.yaml")) //nolint:gosec // under the validated package
	if err != nil {
		return amendmentManifest{}, fmt.Errorf("openspec amend: read amendment %03d manifest: %w", number, err)
	}
	var manifest amendmentManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return amendmentManifest{}, fmt.Errorf("openspec amend: parse amendment %03d manifest: %w", number, err)
	}
	return manifest, nil
}
