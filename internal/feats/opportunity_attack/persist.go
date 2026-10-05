package opportunity_attack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
	"gopkg.in/yaml.v3"
)

// Ensure evaluates and persists the Opportunity Attack manifest for a package.
func Ensure(in Input) (Manifest, error) {
	if in.PackageDigest == "" {
		return Manifest{}, fmt.Errorf("opportunity attack: package digest is required")
	}
	if existing, found, err := existingManifest(in); err != nil {
		return Manifest{}, err
	} else if found {
		return existing, nil
	}
	evaluation, err := Evaluate(in)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{SchemaVersion: SchemaVersion, MissionID: in.MissionID, PackageDigest: in.PackageDigest, EvaluatedAt: time.Now().UTC().Format(time.RFC3339Nano), Candidates: candidates(in.MissionID, evaluation)}
	return persistEnsuredManifest(in, manifest)
}

func existingManifest(in Input) (Manifest, bool, error) {
	existing, err := LoadManifest(in.StrategistRoot, in.MissionID)
	if err == nil {
		return existing, existing.PackageDigest == in.PackageDigest, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return Manifest{}, false, nil
	}
	return Manifest{}, false, err
}

func persistEnsuredManifest(in Input, manifest Manifest) (Manifest, error) {
	if err := persistAnalysis(in.RefinedPath, manifest); err != nil {
		return Manifest{}, err
	}
	// The handoff digest is authoritative. Rewriting YAML frontmatter can
	// normalize policy-fact encoding, so bind the manifest to the post-write
	// package identity before checking replay/mismatch semantics.
	actualDigest, err := handoff.PackageDigest(in.RefinedPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("opportunity attack: calculate persisted package digest: %w", err)
	}
	if manifest.PackageDigest != actualDigest {
		manifest.PackageDigest = actualDigest
		if err := persistAnalysis(in.RefinedPath, manifest); err != nil {
			return Manifest{}, err
		}
	}
	if err := persistManifest(in.StrategistRoot, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func candidates(missionID string, evaluation Evaluation) []Candidate {
	return []Candidate{
		{ID: "OA-ADR-" + missionID, Kind: KindADR, Warranted: evaluation.ADR, Status: StatusProposed, Description: "Evaluate whether a durable architecture decision should be recorded.", Strategy: "execute_later", EstimatedImpact: "medium", InvestigationRequired: evaluation.ADR},
		{ID: "OA-RUNBOOK-" + missionID, Kind: KindRunbook, Warranted: evaluation.Runbook, Status: StatusProposed, Description: "Evaluate an operational Runbook; Playbook is an alias, not a new family.", Strategy: "execute_later", EstimatedImpact: "medium", InvestigationRequired: evaluation.Runbook},
		{ID: "OA-CHEST-" + missionID, Kind: KindChest, Warranted: evaluation.Chest, Status: StatusProposed, Description: "Evaluate registering a fully covered source as a Treasure Chest.", Strategy: "execute_later", EstimatedImpact: "low", InvestigationRequired: evaluation.Chest, SourceRef: evaluation.ChestSource},
	}
}

func persistManifest(root string, manifest Manifest) error {
	if root == "" {
		return fmt.Errorf("opportunity attack: strategist root is required")
	}
	dir := filepath.Join(root, "opportunity-attacks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("opportunity attack: create manifest directory: %w", err)
	}
	path := filepath.Join(dir, manifest.MissionID+".yaml")
	if err := validateExistingManifest(path); err != nil {
		return err
	}
	raw, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("opportunity attack: encode manifest: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil { //nolint:gosec // destination is the resolved strategist root
		return fmt.Errorf("opportunity attack: write manifest: %w", err)
	}
	return nil
}

func validateExistingManifest(path string) error {
	existingRaw, err := os.ReadFile(path) //nolint:gosec // path is resolved under the Strategist root
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opportunity attack: read existing manifest: %w", err)
	}
	var existing Manifest
	if err := yaml.Unmarshal(existingRaw, &existing); err != nil {
		return fmt.Errorf("opportunity attack: parse existing manifest: %w", err)
	}
	return nil
}

// VerifyDigest rejects a manifest that is stale relative to a package being
// consumed at a later boundary.
func VerifyDigest(manifest Manifest, packageDigest string) error {
	if manifest.PackageDigest == "" || packageDigest == "" || manifest.PackageDigest != packageDigest {
		return fmt.Errorf("opportunity attack: manifest package digest mismatch: manifest %s, package %s", manifest.PackageDigest, packageDigest)
	}
	return nil
}

func persistAnalysis(refined string, manifest Manifest) error {
	path := filepath.Join(refined, "analysis.md")
	raw, err := os.ReadFile(path) //nolint:gosec // path is resolved by the mission boundary
	if err != nil {
		return fmt.Errorf("opportunity attack: read analysis for manifest: %w", err)
	}
	updated, err := updateAnalysisFrontmatter(raw, manifest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil { //nolint:gosec // path is resolved by the mission boundary
		return fmt.Errorf("opportunity attack: persist analysis manifest: %w", err)
	}
	return nil
}

// LoadManifest reads the persisted Opportunity Attack manifest.
func LoadManifest(root, missionID string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "opportunity-attacks", missionID+".yaml")) //nolint:gosec // root is resolved by the CLI boundary
	if err != nil {
		return Manifest{}, fmt.Errorf("opportunity attack: read manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("opportunity attack: parse manifest: %w", err)
	}
	return manifest, nil
}

// AttachRunbookPreparation adds advisory preparation evidence to the existing
// OA manifest. It never changes analysis.md, the canonical Runbook root, or
// the candidate's proposed status.
func AttachRunbookPreparation(root, missionID, packageDigest string, result preparerunbook.Result) (Manifest, error) {
	manifest, err := LoadManifest(root, missionID)
	if err != nil {
		return Manifest{}, err
	}
	if err := VerifyDigest(manifest, packageDigest); err != nil {
		return Manifest{}, err
	}
	if result.PackageDigest != packageDigest || result.SourceDigest != packageDigest {
		return Manifest{}, fmt.Errorf("opportunity attack: runbook preparation package digest mismatch")
	}
	if !attachPreparation(&manifest, result) {
		return Manifest{}, fmt.Errorf("opportunity attack: runbook candidate %q is not present", result.CandidateID)
	}
	if err := persistManifest(root, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func attachPreparation(manifest *Manifest, result preparerunbook.Result) bool {
	found := false
	for i := range manifest.Candidates {
		candidate := &manifest.Candidates[i]
		if candidate.Kind == KindRunbook && candidate.ID == result.CandidateID {
			candidate.RunbookPreparation = &result
			found = true
		}
	}
	return found
}
