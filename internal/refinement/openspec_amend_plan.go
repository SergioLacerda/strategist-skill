package refinement

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// amendableStatuses are the mission statuses at which a refined package may still be
// amended: before the gate answer, or after an analysis-only acceptance. Every other
// status, including a claimed or applied package, is refused.
var amendableStatuses = map[string]bool{
	"archivist_done": true, "gate_pending": true, "gate_revision_requested": true, "gate_analysis_accepted": true,
}

type amendmentPlan struct {
	input                           AmendInput
	refined, changeDir, snapshotDir string
	number                          int
	status, original                string
	at                              time.Time
	previous, next                  map[string][]byte
	analysisSHA, packageSHA         string
	reason, disposition             string
}

// planAmendment reads and checks everything; it writes nothing.
func planAmendment(input AmendInput) (amendmentPlan, error) {
	if err := validateAmendInput(input); err != nil {
		return amendmentPlan{}, err
	}
	refined, err := validatedRefinedPath(OpenSpecInput{MissionID: input.MissionID, BasePath: input.BasePath})
	if err != nil {
		return amendmentPlan{}, err
	}
	plan := amendmentPlan{input: input, refined: refined, at: amendTime(input)}
	if err := plan.loadPackage(); err != nil {
		return amendmentPlan{}, err
	}
	if err := plan.checkState(); err != nil {
		return amendmentPlan{}, err
	}
	if err := plan.loadChange(); err != nil {
		return amendmentPlan{}, err
	}
	return plan, plan.checkTargets()
}

func amendTime(input AmendInput) time.Time {
	if input.Now != nil {
		return input.Now().UTC()
	}
	return time.Now().UTC()
}

func validateAmendInput(input AmendInput) error {
	if err := validateAmendComponents(input); err != nil {
		return err
	}
	if err := validateAbsolutePaths(OpenSpecInput{BasePath: input.BasePath, RuntimeRoot: input.RuntimeRoot, PendingAnalysisPath: input.BasePath}); err != nil {
		return err
	}
	return validateReference(input.AuthorizationRef)
}

func validateAmendComponents(input AmendInput) error {
	for label, value := range map[string]string{"mission id": input.MissionID, "change id": input.ChangeID, "--amends": input.Amends} {
		if !safeComponent.MatchString(value) {
			return fmt.Errorf("openspec amend: %s is malformed", label)
		}
	}
	return nil
}

// validateReference requires a non-empty, single-line reference with no control
// character, so it can be recorded verbatim in frontmatter and manifest.
func validateReference(ref string) error {
	if strings.TrimSpace(ref) == "" || len(ref) > 512 || strings.IndexFunc(ref, unicode.IsControl) >= 0 {
		return fmt.Errorf("openspec amend: the authorization reference must be a non-empty single line of at most 512 characters with no control characters")
	}
	return nil
}

func (p *amendmentPlan) loadPackage() error {
	if _, err := os.Stat(p.refined); err != nil {
		return fmt.Errorf("openspec amend: no refined package at %s: %w", p.refined, err)
	}
	if err := requireFiles(p.refined, canonicalFiles); err != nil {
		return fmt.Errorf("openspec amend: incomplete refined package: %w", err)
	}
	p.previous = make(map[string][]byte, len(canonicalFiles))
	for _, name := range canonicalFiles {
		raw, err := os.ReadFile(filepath.Join(p.refined, name)) //nolint:gosec // validated canonical package
		if err != nil {
			return fmt.Errorf("openspec amend: read %s: %w", name, err)
		}
		p.previous[name] = raw
	}
	p.analysisSHA = digest(p.previous["analysis.md"])
	p.packageSHA = canonicalPackageDigest(p.previous)
	return nil
}

// checkState covers identity, status, claim, pending analysis, chain and gate label.
func (p *amendmentPlan) checkState() error {
	analysis := p.previous["analysis.md"]
	if !hasMissionIdentity(analysis, p.input.MissionID) {
		return fmt.Errorf("openspec amend: analysis.md mission_id does not match %q", p.input.MissionID)
	}
	p.status = frontmatterValue(analysis, "mission_status")
	p.original = frontmatterValue(analysis, "provider_change_id")
	if !amendableStatuses[p.status] {
		return fmt.Errorf("openspec amend: mission_status %q is not amendable (allowed: archivist_done, gate_pending, gate_revision_requested, gate_analysis_accepted)", p.status)
	}
	if frontmatterValue(analysis, "claimed_by") != "" {
		return fmt.Errorf("openspec amend: the package is claimed_by %q; it cannot be amended", frontmatterValue(analysis, "claimed_by"))
	}
	if err := p.checkPersistedState(); err != nil {
		return err
	}
	return p.checkChain()
}

func (p *amendmentPlan) checkChain() error {
	pending := filepath.Join(p.input.BasePath, "pending", p.input.MissionID+"-analysis.md")
	if _, err := os.Stat(pending); err == nil {
		return fmt.Errorf("openspec amend: a pending analysis exists for the mission (%s); resolve it first", pending)
	}
	last, number, err := lastAmendment(p.refined)
	if err != nil {
		return err
	}
	expected := p.original
	if last != "" {
		expected = last
	}
	if p.input.Amends != expected {
		return fmt.Errorf("openspec amend: --amends %q does not match the change being amended (%q)", p.input.Amends, expected)
	}
	p.number = number + 1
	p.snapshotDir = filepath.Join(p.refined, ".amendments", fmt.Sprintf("%03d", p.number))
	return nil
}

func (p *amendmentPlan) loadChange() error {
	p.changeDir = filepath.Join(p.input.RuntimeRoot, "changes", p.input.ChangeID)
	if err := validateContained(p.input.RuntimeRoot, p.changeDir); err != nil {
		return fmt.Errorf("openspec amend: change path: %w", err)
	}
	if err := requireFiles(p.changeDir, canonicalFiles[1:]); err != nil {
		return fmt.Errorf("openspec amend: incomplete change: %w", err)
	}
	p.next = make(map[string][]byte, len(canonicalFiles)-1)
	for _, name := range canonicalFiles[1:] {
		raw, err := os.ReadFile(filepath.Join(p.changeDir, name)) //nolint:gosec // contained under the validated runtime root
		if err != nil {
			return fmt.Errorf("openspec amend: read %s: %w", name, err)
		}
		p.next[name] = raw
	}
	design, err := withAcceptanceScenarios(p.next["design.md"], p.changeDir)
	p.next["design.md"] = design
	return err
}

// checkTargets refuses turning an analysis-only accepted package into one that asks
// Sniper to materialize documentation: the acceptance did not cover that.
func (p *amendmentPlan) checkTargets() error {
	if p.status == "gate_analysis_accepted" && !documentationTargetMarker.Match(p.previous["tasks.md"]) && documentationTargetMarker.Match(p.next["tasks.md"]) {
		return fmt.Errorf("openspec amend: an analysis-only accepted package may not gain a documentation target")
	}
	return nil
}
