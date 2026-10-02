package handoff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var missionStatusLine = regexp.MustCompile(`(?m)^mission_status:[ \t]*([^\r\n#]+)[ \t]*$`)

// AnalysisLifecycle is the mutable lifecycle metadata on refined analysis.md.
type AnalysisLifecycle struct {
	MissionID string
	Status    string
	ClaimedBy string
}

// ReadAnalysisLifecycle reads the structured lifecycle fields used by the
// Approval Gate and Sniper completion boundaries.
func ReadAnalysisLifecycle(path string) (AnalysisLifecycle, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller resolves the refined mission package path
	if err != nil {
		return AnalysisLifecycle{}, fmt.Errorf("read analysis lifecycle: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(raw)
	if err != nil {
		return AnalysisLifecycle{}, fmt.Errorf("analysis lifecycle: %w", err)
	}
	text := func(key string) string {
		value, ok := frontmatter[key].(string)
		if !ok {
			return ""
		}
		return value
	}
	return AnalysisLifecycle{MissionID: text("mission_id"), Status: text("mission_status"), ClaimedBy: text("claimed_by")}, nil
}

// AcceptAnalysisAtGate advances the refined package's Markdown lifecycle when
// the authoritative mission engine accepts the main Approval Gate. It returns
// the original bytes so the caller can roll the artifact back if persisting
// the engine transition fails.
func AcceptAnalysisAtGate(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller resolves the refined mission package path
	if err != nil {
		return nil, false, fmt.Errorf("read approval-gate analysis: %w", err)
	}
	updated, changed, err := acceptedAnalysisBytes(raw)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return raw, false, nil
	}
	if err := writeAnalysisAtomic(path, updated); err != nil {
		return nil, false, fmt.Errorf("write approval-gate analysis: %w", err)
	}
	return raw, true, nil
}

func acceptedAnalysisBytes(raw []byte) ([]byte, bool, error) {
	frontmatter, _, err := parseFrontmatter(raw)
	if err != nil {
		return nil, false, fmt.Errorf("approval-gate analysis: %w", err)
	}
	status, ok := frontmatter["mission_status"].(string)
	if !ok {
		status = ""
	}
	switch status {
	case "gate_analysis_accepted":
		return raw, false, nil
	case "archivist_done", "gate_pending":
	default:
		return nil, false, fmt.Errorf("approval-gate analysis: mission_status %q cannot transition to gate_analysis_accepted", status)
	}
	closing := bytes.Index(raw[4:], []byte("\n---"))
	if closing < 0 {
		return nil, false, fmt.Errorf("approval-gate analysis: frontmatter is unclosed")
	}
	closing += 4
	frontmatterRaw := raw[:closing]
	match := missionStatusLine.FindIndex(frontmatterRaw)
	if match == nil {
		return nil, false, fmt.Errorf("approval-gate analysis: mission_status line is missing or ambiguous")
	}
	updated := make([]byte, 0, len(raw)+16)
	updated = append(updated, raw[:match[0]]...)
	updated = append(updated, "mission_status: gate_analysis_accepted"...)
	updated = append(updated, raw[match[1]:]...)
	if bytes.Equal(updated, raw) {
		return nil, false, fmt.Errorf("approval-gate analysis: mission_status line is missing or ambiguous")
	}
	return updated, true, nil
}

// RestoreAnalysis restores bytes returned by AcceptAnalysisAtGate.
func RestoreAnalysis(path string, raw []byte) error {
	if err := writeAnalysisAtomic(path, raw); err != nil {
		return fmt.Errorf("restore approval-gate analysis: %w", err)
	}
	return nil
}

func writeAnalysisAtomic(path string, raw []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".analysis-status-*")
	if err != nil {
		return fmt.Errorf("create temporary analysis: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup after rename or failure
	defer func() { _ = tmp.Close() }()        //nolint:errcheck // explicit close below reports the meaningful error
	if err = tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("chmod temporary analysis: %w", err)
	}
	if _, err = tmp.Write(raw); err != nil {
		return fmt.Errorf("write temporary analysis: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary analysis: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temporary analysis: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace analysis: %w", err)
	}
	return nil
}
